# Out-of-Process Provider Adapter Pattern

## Context

External Secrets Operator reconciles `ExternalSecret` and `PushSecret` resources by fetching or pushing secrets to external secret management systems through provider implementations. Historically, all providers run in-process within the controller binary. This architecture requires provider code to be statically linked at compile time and limits deployment flexibility.

We are introducing a v2 provider architecture where providers run as separate gRPC server processes, enabling:
- Independent deployment and scaling of providers
- Heterogeneous language support for provider implementations
- Reduced controller binary size and memory footprint
- Runtime provider discovery and configuration

This architectural shift introduces a network hop between the controller and secret providers. A critical requirement is maintaining a single codebase for provider implementations—we cannot fork provider implementations into separate "in-process" and "out-of-process" versions.

## Problem Description

The provider abstraction is defined by the `SecretsClient` interface, which provides methods for secret operations (`GetSecret`, `PushSecret`, `DeleteSecret`, etc.). The `clientmanager` is responsible for instantiating and caching these clients for use during reconciliation.

Introducing out-of-process providers creates two challenges:

1. **Interface Compatibility:** The controller expects all providers to implement `SecretsClient`, but out-of-process providers communicate via gRPC rather than direct method calls.

2. **Code Reuse:** Provider implementations must work both as standalone gRPC servers and as libraries usable by in-process controllers without maintaining duplicate codebases.

## Decision

Implement a **bidirectional adapter pattern** with two complementary components:

### Client-Side Adapter

A client-side adapter wraps a gRPC client and implements the `esv1.SecretsClient` interface. When the `clientmanager` requests a provider client, it receives this adapter which:

1. Accepts method calls matching the `SecretsClient` interface
2. Converts parameters to protobuf messages
3. Sends gRPC requests to the remote provider server
4. Converts protobuf responses back to expected return types

The adapter is transparent to the reconciliation logic. Controllers interact with remote providers using the same interface as in-process providers.

Integration point:

```go
// clientmanager/manager.go
func (m *Manager) getV2ProviderClient(ctx context.Context, providerName, namespace string) (esv1.SecretsClient, error) {
    // Get gRPC connection from pool
    grpcClient, err := pool.Get(ctx, address, tlsConfig)
    
    // Wrap with client-side adapter
    wrappedClient := adapterstore.NewClient(grpcClient, providerRef, authNamespace)
    
    // Cache and return - reconciler sees SecretsClient interface
    return wrappedClient, nil
}
```

### Server-Side Adapter

The server-side adapter receives gRPC requests and translates them into `SecretsClient` interface calls. The adapter:

1. Implements gRPC service interfaces (`SecretStoreProviderServer`, `GeneratorProviderServer`)
2. Receives protobuf request messages
3. Constructs provider instances using existing v1 provider factories
4. Converts protobuf parameters to interface types
5. Invokes methods on the provider's `SecretsClient` implementation
6. Converts results to protobuf responses

Provider implementations remain unchanged—they implement `ProviderInterface.NewClient()` and return `SecretsClient` instances exactly as they do for in-process use.

Integration point:

```go
// providers/v2/aws/main.go (generated)
func main() {
    // Existing v1 provider implementation
    v1Provider := store.NewProvider()
    
    // Map provider by GVK
    providerMapping := adapterstore.ProviderMapping{
        schema.GroupVersionKind{...}: v1Provider,
    }
    
    // Adapter wraps v1 provider as gRPC server
    adapterServer := adapter.NewServer(kubeClient, scheme, providerMapping, specMapper, generatorMapping)
    pb.RegisterSecretStoreProviderServer(grpcServer, adapterServer)
}
```

### Data Flow

1. Controller reconciles `ExternalSecret` referencing a v2 `Provider`
2. `clientmanager.Get()` detects v2 provider kind
3. Manager creates client-side adapter wrapping gRPC connection
4. Reconciler calls `client.GetSecret(ctx, ref)`
5. Client-side adapter converts call to `pb.GetSecretRequest`
6. gRPC request sent to remote provider server
7. Server-side adapter receives request
8. Server-side adapter constructs v1 provider client
9. Server-side adapter calls `client.GetSecret(ctx, ref)` on v1 implementation
10. Server-side adapter converts result to `pb.GetSecretResponse`
11. Client-side adapter converts response to `[]byte`
12. Reconciler receives secret data

### Connection Management

The architecture employs a global connection pool (`grpc.ConnectionPool`) to enable connection reuse across reconciliations. The `clientmanager` tracks pooled connections and releases them on `Close()`, not closing the underlying connection but returning it to the pool for subsequent use.

## Consequences

### Positive

- **Single Codebase:** Provider implementations exist once and work both in-process and out-of-process through adapters
- **Interface Stability:** Reconciliation logic remains unchanged; the adapter pattern is transparent
- **Flexibility:** Providers can be deployed in-process (legacy), out-of-process (v2), or mixed
- **Testability:** v1 provider implementations can be tested directly without gRPC infrastructure
- **Gradual Migration:** Existing providers migrate individually without disrupting others

### Negative

- **Performance Overhead:** Network hop adds latency compared to in-process calls (mitigated by connection pooling and client caching)
- **Serialization Cost:** Data must be serialized/deserialized at adapter boundaries
- **Complexity:** Additional layer of indirection requires understanding adapter pattern for debugging
- **Error Propagation:** gRPC errors must be properly mapped to provider errors for consistent behavior

### Neutral

- **Interface Constraints:** The adapter pattern requires protobuf definitions to match the `SecretsClient` interface capabilities
- **Versioning:** Changes to `SecretsClient` interface require coordinated updates to protobuf definitions and both adapters
