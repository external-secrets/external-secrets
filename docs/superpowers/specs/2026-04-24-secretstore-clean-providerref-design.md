# SecretStore Clean ProviderRef Design

**Date:** 2026-04-24

## Goal

Define a clean long-term `SecretStore` model where:

- transport/runtime selection is handled by `spec.runtimeRef`
- provider-specific configuration is handled by `spec.providerRef`
- the inline `spec.provider` union remains only as a `v1` transition path
- the gRPC API uses a single reference-based out-of-process mode

This design intentionally treats the currently implemented `runtimeRef + inline provider` model as an intermediate step, not the final API.

## Scope

In scope:

- the `v1` transition API shape
- the `v2` clean API shape
- validation rules
- namespace resolution rules
- gRPC API evolution
- controller/reconciler structure
- migration sequencing

Out of scope:

- implementing every provider-owned backend CRD
- full `esoctl` migration UX
- detailed rollout sequencing for each provider runtime chart

## Problem

The current `v1 SecretStore + runtimeRef + provider` implementation is effectively a "SecretStore 1.5":

- `runtimeRef` cleanly separates transport from config
- but core still owns the giant inline `spec.provider` union
- out-of-process mode still serializes the store spec instead of referencing a provider-owned backend object

This is a useful bridge, but it is not the desired end state.

The long-term clean model should make `SecretStore` itself stable while moving provider-specific schema ownership entirely out of core.

## Design Principles

1. `SecretStore` remains the user-facing abstraction.
2. `runtimeRef` selects only the provider runtime process.
3. provider-specific backend schema belongs to the provider, not the ESO core API.
4. `v1` is the transition surface.
5. `v2` is the cleanup surface.
6. out-of-process gRPC requests should use references, not serialized store payloads.
7. reconciliation logic should remain unified across versions.

## API Design

### 1. `v1` transition model

`external-secrets.io/v1 SecretStore` and `ClusterSecretStore` support two mutually exclusive config modes:

- legacy in-process mode via `spec.provider`
- clean out-of-process mode via `spec.runtimeRef + spec.providerRef`

Example:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: fake-store
  namespace: demo
spec:
  runtimeRef:
    name: fake-runtime
  providerRef:
    apiVersion: provider.external-secrets.io/v2alpha1
    kind: Fake
    name: fake-config
```

Rules:

- exactly one of `spec.provider` or `spec.providerRef` must be set
- if `spec.provider` is set, `spec.runtimeRef` must be absent
- if `spec.providerRef` is set, `spec.runtimeRef` is required

This means:

- `spec.provider` remains the in-process config path only
- `spec.runtimeRef + spec.providerRef` is the out-of-process path
- `spec.provider + spec.runtimeRef` is invalid

### 2. `v2` clean model

`external-secrets.io/v2alpha1 SecretStore` and `ClusterSecretStore` remove the inline provider union entirely.

Example:

```yaml
apiVersion: external-secrets.io/v2alpha1
kind: SecretStore
metadata:
  name: fake-store
  namespace: demo
spec:
  runtimeRef:
    name: fake-runtime
  providerRef:
    apiVersion: provider.external-secrets.io/v2alpha1
    kind: Fake
    name: fake-config
```

Rules:

- `spec.provider` does not exist
- `spec.runtimeRef` is required
- `spec.providerRef` is required

### 3. Naming

Use `spec.providerRef`, not `spec.backendRef`.

Rationale:

- the existing gRPC field is already named `provider_ref`
- the provider-owned CRD being referenced is the provider configuration object for the runtime
- `runtimeRef` and `providerRef` form a symmetrical API:
  - `runtimeRef`: where to dial
  - `providerRef`: what backend config to load

`providerRef` does **not** refer to the runtime process itself. It refers to the provider-owned configuration object that the runtime will load.

## Runtime Reference Semantics

Retain the existing runtime selection model:

- namespaced `SecretStore`
  - default `runtimeRef.kind` is `ProviderClass`
  - explicit `ProviderClass` resolves in the store namespace
  - explicit `ClusterProviderClass` resolves cluster-scoped

- `ClusterSecretStore`
  - default `runtimeRef.kind` is `ClusterProviderClass`
  - `ProviderClass` is invalid

This preserves the current and recently approved `ProviderClass` behavior.

## Provider Reference Resolution

`providerRef` shape:

```yaml
providerRef:
  apiVersion: provider.external-secrets.io/v2alpha1
  kind: Fake
  name: fake-config
  namespace: demo # optional
```

### Namespaced `SecretStore`

- if `providerRef.namespace` is omitted, resolve to `metadata.namespace`
- if `providerRef.namespace` is set, it must equal `metadata.namespace`
- cross-namespace provider refs are rejected

### `ClusterSecretStore`

- if `providerRef.namespace` is set, use it as-is
- if `providerRef.namespace` is omitted, resolve it to the caller namespace
  - `ExternalSecret`: its namespace
  - `PushSecret`: its namespace
- `conditions` continue to gate whether the caller namespace may use the cluster store

This keeps namespaced stores tenant-local while allowing shared cluster stores to remain flexible.

## Validation Rules

### `v1 SecretStore` / `v1 ClusterSecretStore`

- exactly one of `spec.provider` or `spec.providerRef` must be set
- if `spec.providerRef` is set, `spec.runtimeRef` is required
- if `spec.provider` is set, `spec.runtimeRef` must be absent
- `ClusterSecretStore` must reject `runtimeRef.kind: ProviderClass`
- namespaced store `providerRef.namespace` must be empty or equal to the store namespace

### `v2 SecretStore` / `v2 ClusterSecretStore`

- `spec.provider` is forbidden
- `spec.runtimeRef` is required
- `spec.providerRef` is required
- cluster-store runtime and namespace rules remain the same as in `v1`

## gRPC API Evolution

The remote contract should use a single reference-based clean mode.

### Decision

Remove `CompatibilityStore` and use `provider_ref` as the only out-of-process configuration mechanism.

This produces a simpler model:

- in-process path: `spec.provider`
- out-of-process path: `runtimeRef + providerRef` -> gRPC `provider_ref`

### Updated semantics of `provider_ref`

`provider_ref` becomes the canonical reference to the provider-owned backend configuration object loaded by the runtime.

It should no longer be described as a generic holder for old `Provider` / `ClusterProvider` transport-era semantics.

Recommended proto shape remains conceptually:

```proto
message ProviderReference {
  string api_version = 1;
  string kind = 2;
  string name = 3;
  string namespace = 4;
  string store_ref_kind = 5;
}
```

Optional future additions if needed:

- `store_name`
- `store_namespace`
- `store_uid`
- `store_generation`

Those fields should be added only if runtime-side caching, auditing, or diagnostics require them.

### Request mode

For all store RPCs:

- `provider_ref` is required for the out-of-process path
- no `compatibility_store`
- no serialized `SecretStoreSpec` transport

Affected requests:

- `GetSecretRequest`
- `GetSecretMapRequest`
- `GetAllSecretsRequest`
- `ValidateRequest`
- `PushSecretRequest`
- `DeleteSecretRequest`
- `SecretExistsRequest`
- `CapabilitiesRequest`

## Controller And Runtime Responsibilities

### Core controller

Core owns:

- `SecretStore` / `ClusterSecretStore` validation
- runtime resolution through `runtimeRef`
- provider reference resolution through `providerRef`
- namespace policy and cluster-store `conditions`
- connection pooling to runtimes
- request orchestration for `ExternalSecret` and `PushSecret`

### Provider runtime

Runtime owns:

- registering the provider-owned CRD types it serves
- loading referenced provider config CRs from Kubernetes
- validating provider-specific config semantics
- constructing backend clients from those CRs
- backend-specific caching and request execution

Core does not inspect provider-specific CR specs beyond generic object reference validation.

## Reconciliation And Controllers

Use one reconciliation core, not separate business-logic controllers for `v1` and `v2`.

Recommended structure:

- one shared `SecretStore` reconciliation engine
- one shared `ClusterSecretStore` reconciliation engine
- versioned API objects are mapped into a shared internal model such as `StoreView`

Examples:

- `v1 SecretStore(spec.provider)` -> `StoreView{Mode: InProcess}`
- `v1 SecretStore(spec.runtimeRef + spec.providerRef)` -> `StoreView{Mode: RemoteRef}`
- `v2 SecretStore` -> `StoreView{Mode: RemoteRef}`

This keeps reconciliation, readiness, runtime dialing, and request dispatch unified.

## Migration Model

Migration happens in two phases.

### Phase 1: semantic migration inside `v1`

Users move from legacy inline provider config:

```yaml
spec:
  provider: ...
```

to the clean out-of-process model:

```yaml
spec:
  runtimeRef: ...
  providerRef: ...
```

Still under `apiVersion: external-secrets.io/v1`.

This is the main user migration.

### Phase 2: API version migration to `v2`

After legacy `v1` stores using `spec.provider` are no longer present, introduce and serve clean `v2` store APIs:

- `external-secrets.io/v2alpha1 SecretStore`
- `external-secrets.io/v2alpha1 ClusterSecretStore`

At that point:

- `v2` contains only the clean model
- version migration is mostly a manifest `apiVersion` update

## Conversion Strategy

The cleanest model is to avoid simultaneous lossless conversion between:

- legacy `v1` objects using `spec.provider`
- clean `v2` objects where `spec.provider` does not exist

Therefore:

- `v1` remains the served/storage API while legacy `spec.provider` objects still exist
- `v2` should be introduced only after clusters have migrated to the clean `v1` subset

This avoids conversion hacks and preserves a single clear mental model.

## Example: Fake Provider

Namespaced runtime object:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ProviderClass
metadata:
  name: fake-runtime
  namespace: demo
spec:
  address: provider-fake.external-secrets-system.svc:8080
```

Provider-owned config object:

```yaml
apiVersion: provider.external-secrets.io/v2alpha1
kind: Fake
metadata:
  name: fake-config
  namespace: demo
spec:
  data:
  - key: hello
    value: world
```

`v1` clean-path store:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: fake-store
  namespace: demo
spec:
  runtimeRef:
    name: fake-runtime
  providerRef:
    apiVersion: provider.external-secrets.io/v2alpha1
    kind: Fake
    name: fake-config
```

Consumer remains store-oriented:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: fake-example
  namespace: demo
spec:
  secretStoreRef:
    name: fake-store
  target:
    name: fake-example-secret
  data:
  - secretKey: message
    remoteRef:
      key: hello
```

## Recommendation

Adopt the following model:

1. `v1` supports:
   - `spec.provider` for in-process
   - `spec.runtimeRef + spec.providerRef` for out-of-process
2. `v1 spec.provider + runtimeRef` is invalid
3. `v2` removes `spec.provider`
4. `providerRef` is the API name used in both `v1` clean mode and `v2`
5. gRPC removes `CompatibilityStore`
6. gRPC uses only `provider_ref` for out-of-process config lookup
7. one reconciliation core serves both versions through a shared internal model
8. `v2` is introduced only after the cluster state has migrated away from legacy `v1.spec.provider`
