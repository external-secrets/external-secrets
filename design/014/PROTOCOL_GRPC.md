# ESO ⇄ Provider Contract: gRPC Binding

How the contract in [PROTOCOL.md](./PROTOCOL.md) travels over gRPC. A runtime config
selects it with `spec.protocol: grpc`, which is the default and the only
protocol implemented in milestone 1. This document adds no meaning to the
contract; where the two disagree, PROTOCOL.md is right.

## Summary

| Contract concept | gRPC encoding |
| --- | --- |
| contract version `v1alpha1` | protobuf package `externalsecrets.provider.v1alpha1` |
| Runtime service | `externalsecrets.provider.v1alpha1.Runtime` |
| SecretStore service | `externalsecrets.provider.v1alpha1.SecretStoreProvider` |
| Generator service | `externalsecrets.provider.v1alpha1.GeneratorProvider` |
| Health service | standard `grpc.health.v1.Health`, `Check` with an empty `service` |
| operation address | `/externalsecrets.provider.v1alpha1.<Service>/<Operation>`, one unary RPC per operation |
| messages | protobuf binary, from `proto/provider/v1alpha1/provider.proto` |
| error code | gRPC status code (`NOT_FOUND`, ...) |
| error detail | `ProviderError` packed in `google.rpc.Status.details` (`grpc-status-details-bin` trailer) |
| request id | metadata `x-eso-request-id` |
| deadline | native gRPC deadline (`grpc-timeout`) |
| bearer token *(reserved)* | metadata `authorization: Bearer <token>` |
| message size limit (16 MiB) | receive and send limit on both sides (grpc-go: `MaxCallRecvMsgSize`/`MaxRecvMsgSize`; connect-go: `WithReadMaxBytes`/`WithSendMaxBytes`) |
| HTTP version | HTTP/2 only: TLS with ALPN `h2`, h2c in the `HTTP(dev)` composition, or a Unix socket |

## Protobuf

The schema lives in `proto/provider/v1alpha1/provider.proto`, in the `proto/`
root module (README D4). Both bindings are generated from this one file
(PROTOCOL.md P10).

```proto
syntax = "proto3";

package externalsecrets.provider.v1alpha1;

option go_package = "github.com/external-secrets/external-secrets/proto/provider/v1alpha1;providerv1alpha1";

// Runtime is served by every provider runtime. Health is grpc.health.v1.
service Runtime {
  rpc Describe(DescribeRequest) returns (DescribeResponse);
}

// SecretStoreProvider is served by runtimes that list kinds in Describe.serves.
service SecretStoreProvider {
  // Configuration.
  rpc Validate(ValidateRequest) returns (ValidateResponse);
  rpc Capabilities(CapabilitiesRequest) returns (CapabilitiesResponse);

  // Read.
  rpc GetSecret(GetSecretRequest) returns (GetSecretResponse);
  rpc GetSecretMap(GetSecretMapRequest) returns (GetSecretMapResponse);
  rpc GetAllSecrets(GetAllSecretsRequest) returns (GetAllSecretsResponse);

  // Write.
  rpc PushSecret(PushSecretRequest) returns (PushSecretResponse);
  rpc DeleteSecret(DeleteSecretRequest) returns (DeleteSecretResponse);
  rpc SecretExists(SecretExistsRequest) returns (SecretExistsResponse);
}

// GeneratorProvider is served by runtimes that generate. Its Describe lists the
// generator kinds; a runtime without generators answers UNIMPLEMENTED.
service GeneratorProvider {
  // Configuration.
  rpc Describe(DescribeGeneratorsRequest) returns (DescribeGeneratorsResponse);
  rpc Validate(ValidateGeneratorRequest) returns (ValidateGeneratorResponse);

  // Generation.
  rpc Generate(GenerateRequest) returns (GenerateResponse);
  rpc Cleanup(CleanupRequest) returns (CleanupResponse);
}

// ---------------------------------------------------------------- addressing

// ProviderConfigReference mirrors SecretStore/Generator spec.providerConfig:
// exactly one member is set. New members (key_ref, uuid_ref, label_selector)
// are added as new oneof cases.
message ProviderConfigReference {
  oneof ref {
    GVKReference gvk_ref = 1;
  }
}

message GVKReference {
  string api_version = 1;
  string kind        = 2;
  string name        = 3;
  string namespace   = 4;
}

message ObjectReference {
  string api_version = 1;
  string kind        = 2;
  string namespace   = 3;
  string name        = 4;
  string uid         = 5;
  int64  generation  = 6;
}

message RequestContext {
  ObjectReference request                  = 1;
  ObjectReference source                   = 2;
  string caller_namespace                  = 3;
  string controller_class                  = 4;
  map<string, string> request_labels       = 5;
  // Only keys starting with "provider.external-secrets.io/" (PROTOCOL.md P8).
  map<string, string> request_annotations  = 6;
  map<string, string> attributes           = 7;
  string credential_namespace              = 8;
  bool   allow_explicit_credential_namespace = 9;
}

message Resolved {
  repeated string credential_namespaces = 1;
  string provider_uid                   = 2;
  int64  provider_generation            = 3;
}

// ------------------------------------------------- runtime and configuration

message DescribeRequest {}
message DescribeResponse {
  string provider_name              = 1;
  string provider_version           = 2;
  repeated string protocol_versions = 3;
  repeated ServedKind serves        = 4;
  repeated string capabilities      = 5;
  reserved 6; // was generates; now GeneratorProvider.Describe
}
message ServedKind {
  string api_version = 1;
  string kind        = 2;
}

message ValidateRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
}
message ValidateResponse {
  ValidationResult result  = 1;
  repeated string warnings = 2;
  Resolved resolved        = 3;
}
// UNSPECIFIED is never a valid answer: the core treats it as INTERNAL, so a
// provider that forgets to set the field does not report a store as valid.
enum ValidationResult {
  VALIDATION_RESULT_UNSPECIFIED = 0;
  VALIDATION_RESULT_VALID       = 1;
  VALIDATION_RESULT_UNKNOWN     = 2;
}

message CapabilitiesRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
}
message CapabilitiesResponse {
  repeated string capabilities = 1;
  Resolved resolved            = 2;
}

// ---------------------------------------------------------------------- read

message RemoteRef {
  string key               = 1;
  string version           = 2;
  string property          = 3;
  string decoding_strategy = 4;
  string metadata_policy   = 5;
}

message GetSecretRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
  RemoteRef remote_ref           = 3;
}
message GetSecretResponse {
  bytes value       = 1;
  Resolved resolved = 2;
}

message GetSecretMapRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
  RemoteRef remote_ref           = 3;
}
message GetSecretMapResponse {
  map<string, bytes> data = 1;
  Resolved resolved       = 2;
}

message Find {
  optional string path       = 1; // unset and "" differ, as v1's *string does
  FindName name              = 2;
  map<string, string> tags   = 3;
  string conversion_strategy = 4;
  string decoding_strategy   = 5;
}
message FindName {
  string regexp = 1;
}

message GetAllSecretsRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
  Find find                      = 3;
}
message GetAllSecretsResponse {
  map<string, bytes> data = 1;
  Resolved resolved       = 2;
}

// --------------------------------------------------------------------- write

message PushSecretData {
  string secret_key = 1;
  string remote_key = 2;
  string property   = 3;
  bytes  metadata   = 4;
}
message PushSecretRequest {
  ProviderConfigReference provider_config         = 1;
  RequestContext context                 = 2;
  PushSecretData data                    = 3;
  map<string, bytes>  secret_data        = 4;
  string secret_type                     = 5;
  map<string, string> secret_labels      = 6;
  map<string, string> secret_annotations = 7;
}
message PushSecretResponse {
  Resolved resolved = 1;
}

message PushRemoteRef {
  string remote_key = 1;
  string property   = 2;
}
message DeleteSecretRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
  PushRemoteRef remote_ref       = 3;
}
message DeleteSecretResponse {
  Resolved resolved = 1;
}
message SecretExistsRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
  PushRemoteRef remote_ref       = 3;
}
message SecretExistsResponse {
  bool exists       = 1;
  Resolved resolved = 2;
}

// ----------------------------------------------------------------- generator

message DescribeGeneratorsRequest {}
message DescribeGeneratorsResponse {
  repeated ServedKind generates = 1;
}

message ValidateGeneratorRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext          context         = 2;
}
message ValidateGeneratorResponse {
  ValidationResult result   = 1;
  repeated string warnings  = 2;
  repeated string outputs   = 3;
  bool stateful             = 4;
  Resolved resolved         = 5;
}

message GenerateRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
  // uid of the GeneratorState the core wrote before the call; the same on every
  // retry. Always set.
  string operation_id            = 3;
}
message GenerateResponse {
  map<string, bytes> data = 1;
  // Opaque to the core; stored in the GeneratorState and sent back on Cleanup.
  bytes state             = 2;
  Resolved resolved       = 3;
}

message CleanupRequest {
  ProviderConfigReference provider_config = 1;
  RequestContext context         = 2;
  bytes state                    = 3; // empty when the Generate response was never recorded
  string operation_id            = 4; // as sent to Generate
}
message CleanupResponse {
  Resolved resolved = 1;
}

// --------------------------------------------------------------------- error

// Packed into google.rpc.Status.details. Optional (PROTOCOL.md P3).
message ProviderError {
  string code                 = 1;
  string message              = 2;
  map<string, string> details = 3;
}
```

Field numbers are part of the contract. They never change and are never
reused within a version. Fields that exist in the PoC proto keep their
numbers. The PoC's separate `generator.proto` is folded into this file, so
that one package, and one negotiated version, covers all services.

## Errors

| Contract code | gRPC status code |
| --- | --- |
| `NotFound` | `NOT_FOUND` (5) |
| `InvalidArgument` | `INVALID_ARGUMENT` (3) |
| `FailedPrecondition` | `FAILED_PRECONDITION` (9) |
| `PermissionDenied` | `PERMISSION_DENIED` (7) |
| `Unauthenticated` | `UNAUTHENTICATED` (16) |
| `Unimplemented` | `UNIMPLEMENTED` (12) |
| `AlreadyExists` | `ALREADY_EXISTS` (6) |
| `ResourceExhausted` | `RESOURCE_EXHAUSTED` (8) |
| `Unavailable` | `UNAVAILABLE` (14) |
| `DeadlineExceeded` | `DEADLINE_EXCEEDED` (4) |
| `Internal` | `INTERNAL` (13) |
| `Unknown` | `UNKNOWN` (2), and any code not listed |

The detail is a `google.rpc.Status` whose `details` holds one `Any`-packed
`externalsecrets.provider.v1alpha1.ProviderError`. With connect-go (the SDK, ADAPTER.md) this is
`connect.NewError(connect.CodeFailedPrecondition, err)` plus `AddDetail`; with grpc-go,
`status.New(codes.FailedPrecondition, msg).WithDetails(&ProviderError{...})`.
The core reads the first `ProviderError` among the details and ignores other
detail types.

Transport-level failures (connection refused, TLS handshake failure, the call
deadline expiring) come back from the gRPC client as `UNAVAILABLE` or
`DEADLINE_EXCEEDED`, and are handled as the contract says for those codes.
The one exception is a handshake the provider rejects because of the client
certificate (a TLS alert from the server such as `bad_certificate`,
`certificate_required` or `unknown_ca`). The core's client classifies it as
`UNAUTHENTICATED`, as [PROTOCOL.md](./PROTOCOL.md) "Authentication" requires,
so a rejected identity is never reported as an unreachable provider.

## Metadata, deadline, size

- `x-eso-request-id`: always set by the core. Lower case, as gRPC metadata keys
  are.
- `authorization`: `Bearer <projected token>`, only with
  `auth.serviceAccountToken` *(reserved)*. The core never sends it over a
  connection without TLS, which matches the CEL rule "auth requires TLS".
- Deadline: `context.WithTimeout(ctx, RuntimeConfig.spec.timeout)`, which gRPC carries
  as `grpc-timeout`.
- Size: both sides set 16 MiB for receive and send (grpc-go's default receive
  limit is 4 MiB). The server's limit matters too, because a `PushSecret`
  request carries the whole rendered source Secret.
- Compression: none required. A server must accept uncompressed requests. The
  core does not compress.

Any other metadata the core's gRPC library adds (`user-agent`, `te`, ...) is
not part of the contract (P11).

## Connections and compositions

- **Network endpoints.** The core dials `https://<endpoint.address>` with HTTP/2
  (ALPN `h2`). TLS, server verification and client certificates come from the
  runtime config composition ([API.md](./API.md)), and are checked as
  [PROTOCOL.md](./PROTOCOL.md) "Authentication" describes. The server name for SNI and
  verification is `tls.serverName`, or the host part of the address.
- **IPC.** HTTP/2 without TLS (h2c) over the Unix socket at `endpoint.socket`
  (a `ClusterRuntimeConfig` only), with `:authority: localhost`. Providers must
  not reject that authority.
- **h2c** (`HTTP(dev)`). `http://<endpoint.address>` with HTTP/2 without TLS.
- **Connection reuse.** The core keeps one client connection per runtime config
  (connect-go over `net/http`, ADAPTER.md), keyed by everything that changes the
  dial: the runtime config generation, the negotiated contract version, the
  `resourceVersion` of the client certificate Secret, and that of the `caRef`
  object, if any. A connection older than one hour is re-dialed, so rotated
  certificates take effect ([PROTOCOL.md](./PROTOCOL.md) "Authentication"). The connection is never shared across runtime configs, even when
  addresses coincide, because identities differ (ADAPTER.md).

## Health

The core calls `grpc.health.v1.Health/Check` with `service: ""`.

- `SERVING` is healthy.
- `NOT_SERVING`, `UNKNOWN` and `SERVICE_UNKNOWN` are reported as the runtime config
  reason `Unreachable` with the status in the message.
- An error code is handled as for any other operation.

The SDK registers the standard health server and sets `SERVING` once the
server certificate is loaded.

## Examples

The examples use the call from PROTOCOL.md "Example: one call, fully
populated". Messages are shown in protobuf text format; on the wire they are
binary.

### GetSecret, success

```
:method   POST
:path     /externalsecrets.provider.v1alpha1.SecretStoreProvider/GetSecret
:authority provider-aws.external-secrets-system.svc:8443
content-type      application/grpc
grpc-timeout      30000m
x-eso-request-id  01JD3X8Q4M7R2N5V6W9Y0Z1A2B
```

```textproto
# GetSecretRequest
provider_config { gvk_ref { api_version: "provider.external-secrets.io/v1alpha1" kind: "SecretsManager"
                         namespace: "external-secrets-system" name: "aws-shared" } }
context {
  request { api_version: "external-secrets.io/v1" kind: "ExternalSecret" namespace: "team-b"
            name: "app-config" uid: "6f1c2d3e-0000-4000-8000-00000000000b" generation: 4 }
  source  { api_version: "external-secrets.io/v1" kind: "ClusterSecretStore"
            name: "aws-shared" uid: "9a8b7c6d-0000-4000-8000-0000000000aa" generation: 2 }
  caller_namespace: "team-b"
  request_labels { key: "app" value: "billing" }
  request_labels { key: "team" value: "payments" }
  request_annotations { key: "provider.external-secrets.io/purpose" value: "db" }
  credential_namespace: "team-b"
  allow_explicit_credential_namespace: true
}
remote_ref { key: "/team-b/prod/db/password" }
```

```textproto
# response: grpc-status 0, GetSecretResponse
value: "s3cr3t-p4ss"
resolved { credential_namespaces: "team-b"
           provider_uid: "3c4d5e6f-0000-4000-8000-0000000000cc" provider_generation: 7 }
```

### GetSecret, remote secret missing

```
# response trailers
grpc-status          5
grpc-message         secret /team-b/prod/db/password not found
grpc-status-details-bin  <base64 google.rpc.Status>
```

```textproto
# google.rpc.Status, decoded
code: 5
message: "secret /team-b/prod/db/password not found"
details {
  [type.googleapis.com/externalsecrets.provider.v1alpha1.ProviderError] {
    code: "ResourceNotFoundException"
    message: "Secrets Manager can't find the specified secret."
  }
}
```

The core maps this to `esv1.NoSecretErr`. The `ExternalSecret`'s
`deletionPolicy` decides what happens to the target Secret.

### Validate through a SecretStore with a forbidden explicit namespace

```textproto
# ValidateRequest (no caller: request and caller_namespace are empty)
provider_config { gvk_ref { api_version: "provider.external-secrets.io/v1alpha1" kind: "SecretsManager"
                         namespace: "team-a" name: "aws-prod" } }
context {
  source { api_version: "external-secrets.io/v1" kind: "SecretStore" namespace: "team-a"
           name: "aws-prod" uid: "..." generation: 1 }
  credential_namespace: "team-a"
  allow_explicit_credential_namespace: false
}
```

```textproto
# response: grpc-status 9 (FAILED_PRECONDITION), detail:
code: "ExplicitNamespaceNotAllowed"
message: "auth.jwt.serviceAccountRef.namespace \"team-b\" is not allowed through a SecretStore in \"team-a\""
```

The store becomes `Ready=False / ProviderValidationFailed` with the provider's
message.

### Generate and Cleanup

The Vault example from PROTOCOL.md "Generating and cleaning up".

```textproto
# GenerateRequest, to /externalsecrets.provider.v1alpha1.GeneratorProvider/Generate
provider_config { gvk_ref { api_version: "provider.external-secrets.io/v1alpha1" kind: "VaultDynamicSecret"
                         namespace: "team-a" name: "db-creds" } }
context {
  request { api_version: "external-secrets.io/v1" kind: "ExternalSecret" namespace: "team-a"
            name: "db" uid: "..." generation: 1 }
  source  { api_version: "external-secrets.io/v1" kind: "Generator"
            namespace: "team-a" name: "db-creds" uid: "..." generation: 1 }
  caller_namespace: "team-a"
  credential_namespace: "team-a"
}
operation_id: "5d1c9e2a-..."   # uid of the GeneratorState written before the call
```

```textproto
# GenerateResponse
data { key: "username" value: "v-app-8f2k" }
data { key: "password" value: "..." }
state: "{\"address\":\"https://vault.internal:8200\",\"authRef\":\"team-a/vault-approle\",\"leaseId\":\"database/creds/app/9c1e...\"}"
resolved { credential_namespaces: "team-a"
           provider_uid: "..." provider_generation: 3 }
```

The `CleanupRequest` carries the `provider_config` recorded in the
`GeneratorState`, a `context` rebuilt from it (without request labels or
annotations), the same `operation_id`, plus `state` byte for byte.
`state` is empty when the `Generate` response was never recorded; the provider
then revokes what it created under `operation_id`. A lease that has already
expired is not an error: `Cleanup` answers success.

### Debugging with grpcurl

```bash
grpcurl \
  -import-path proto -proto provider/v1alpha1/provider.proto \
  -cacert ca.crt -cert tls.crt -key tls.key \
  -H 'x-eso-request-id: debug-1' -max-time 30 \
  -d '{"providerConfig":{"gvkRef":{"apiVersion":"provider.external-secrets.io/v1alpha1","kind":"SecretsManager","namespace":"team-a","name":"aws-prod"}},"context":{"request":{"apiVersion":"external-secrets.io/v1","kind":"ExternalSecret","namespace":"team-a","name":"db","uid":"<externalsecret-uid>","generation":"1"},"source":{"apiVersion":"external-secrets.io/v1","kind":"SecretStore","namespace":"team-a","name":"aws-prod","uid":"<secretstore-uid>","generation":"1"},"callerNamespace":"team-a","credentialNamespace":"team-a"},"remoteRef":{"key":"/team-a/prod/db/password"}}' \
  provider-aws.external-secrets-system.svc:8443 \
  externalsecrets.provider.v1alpha1.SecretStoreProvider/GetSecret
```

Over IPC, drop the TLS flags, add `-plaintext -unix`, and use the socket path
`/var/run/external-secrets/aws.sock` as the address. Server reflection is
optional. It is an SDK option, off by default, so that a production provider
does not advertise its surface.
