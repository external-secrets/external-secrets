# SecretStore V2 Runtime Split Design

**Date:** 2026-04-18

## Goal

Move providers out of process without forcing users to adopt a new extra indirection layer in day-to-day usage.

The user-facing object that `ExternalSecret` and `PushSecret` reference should remain a store. Transport concerns should move into a separate runtime object, and provider-specific backend schemas should be owned by the provider implementation rather than by the ESO core API.

## Scope

In scope:
- Defining the long-term API shape for out-of-process providers.
- Preserving compatibility for the current `v1` `SecretStore` and `ClusterSecretStore`.
- Defining a clean `v2` store API with no inline provider union in core.
- Defining the controller-to-provider runtime boundary.
- Defining a migration path that minimizes user work.

Out of scope:
- Implementing every provider backend CRD in this phase.
- Removing `v1` store APIs immediately.
- Reworking unrelated `ExternalSecret` or `PushSecret` behavior.
- Finalizing every protobuf field for all operations in this document.

## Problems With The Current Provider Split

The current `Provider` and `ClusterProvider` design introduces a new user-facing object that overlaps heavily with the existing purpose of `SecretStore`.

Today the shape is effectively:

`ExternalSecret -> Provider -> provider-specific CRD -> synthetic SecretStore -> legacy provider client`

This has four problems:

1. It adds a second user-facing abstraction where users previously only needed to understand stores.
2. It keeps provider-specific configuration split away from the store abstraction, which makes authoring and debugging more convoluted.
3. The provider pod currently acts largely as an RPC translation shim instead of a true long-lived runtime.
4. The design does not produce a clean minimal core API because provider-specific concerns are still entangled with the migration path.

## Design Principles

1. `SecretStore` remains the user-facing abstraction.
2. Transport and runtime concerns are separate from backend configuration.
3. Provider-specific schema belongs to the provider, not the ESO core API.
4. Existing users should be able to adopt out-of-process providers with the smallest possible manifest change.
5. New users should have a cleaner API than the current `v1` inline provider union.
6. The provider runtime should be a real runtime with caching and backend ownership, not a per-request translation shim.

## Approaches Considered

### Recommended: Extend `v1`, introduce clean `v2`

- Extend current `v1` `SecretStore` and `ClusterSecretStore` with a runtime reference.
- Introduce a clean `v2` store API that references provider-owned backend CRDs.
- Introduce a transport-only runtime object for provider process connection details.

Why this is recommended:
- Existing users can adopt the out-of-process model with a one-line store change.
- New users get a clean store API without the giant inline provider union.
- `ExternalSecret` continues to point to stores, preserving the existing mental model.
- Provider-specific configuration is owned by the provider and can evolve independently.

### Alternative: Only extend `v1`

Rejected as the long-term direction because:
- It preserves the current provider union in core forever.
- It meets compatibility goals but not the minimal clean API goal.

### Alternative: Keep `Provider` / `ClusterProvider` as the main transport object

Rejected because:
- It duplicates `SecretStore` semantics.
- It adds an extra indirection layer for users.
- It keeps the UX more complex than necessary.

## Proposed Architecture

### 1. Object model

The design introduces a transport-only runtime object:

- `ClusterProviderClass`
  - Core-owned.
  - Describes how ESO reaches a provider runtime.
  - Contains address, TLS mode, and supported backend kinds.
  - Does not contain backend-specific configuration.

The store objects become:

- `external-secrets.io/v1 SecretStore`
  - Compatibility path.
  - Keeps existing inline `spec.provider`.
  - Gains `spec.runtimeRef`.

- `external-secrets.io/v1 ClusterSecretStore`
  - Compatibility path.
  - Keeps existing inline `spec.provider`.
  - Gains `spec.runtimeRef`.

- `external-secrets.io/v2alpha1 SecretStore`
  - Clean end state.
  - Contains no inline provider union.
  - References a provider runtime and a provider-owned backend CRD.

- `external-secrets.io/v2alpha1 ClusterSecretStore`
  - Clean cluster-scoped equivalent.

Provider-owned objects become backend configuration resources, for example:

- `provider.aws.external-secrets.io/v1alpha1 SecretsManagerStore`
- `provider.aws.external-secrets.io/v1alpha1 ParameterStoreStore`

### 2. API sketches

#### 2.1 Transport-only runtime object

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ClusterProviderClass
metadata:
  name: aws
spec:
  address: provider-aws.external-secrets-system.svc:8080
  tls:
    mode: Managed
  supports:
  - apiVersion: provider.aws.external-secrets.io/v1alpha1
    kind: SecretsManagerStore
  - apiVersion: provider.aws.external-secrets.io/v1alpha1
    kind: ParameterStoreStore
status:
  conditions:
  - type: Ready
    status: "True"
  capabilities: ReadWrite
```

Semantics:

- `address` is the dial target for the provider runtime.
- `tls` describes how ESO authenticates and verifies the provider runtime.
- `supports` advertises which backend CRD kinds the runtime serves.
- This object is operator-owned and commonly chart-managed.

#### 2.2 `v1` compatibility store

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: aws-prod
spec:
  runtimeRef:
    kind: ClusterProviderClass
    name: aws
  provider:
    aws:
      service: SecretsManager
      region: eu-central-1
      auth:
        jwt:
          serviceAccountRef:
            name: app-secrets
```

Semantics:

- If `runtimeRef` is omitted, current in-process behavior remains unchanged.
- If `runtimeRef` is set, ESO resolves the legacy inline provider config locally and serves it through the referenced external runtime.
- This is the migration bridge, not the long-term clean API.

#### 2.3 Clean `v2` store

```yaml
apiVersion: external-secrets.io/v2alpha1
kind: SecretStore
metadata:
  name: aws-prod
spec:
  runtimeRef:
    kind: ClusterProviderClass
    name: aws
  backendRef:
    apiVersion: provider.aws.external-secrets.io/v1alpha1
    kind: SecretsManagerStore
    name: aws-prod
```

Semantics:

- `runtimeRef` selects the provider runtime process.
- `backendRef` selects the provider-owned backend configuration resource.
- Core does not understand the internals of `backendRef`.

#### 2.4 Provider-owned backend resource

```yaml
apiVersion: provider.aws.external-secrets.io/v1alpha1
kind: SecretsManagerStore
metadata:
  name: aws-prod
spec:
  region: eu-central-1
  auth:
    jwt:
      serviceAccountRef:
        name: app-secrets
  role: arn:aws:iam::123456789012:role/eso-prod
  prefix: /prod/
```

#### 2.5 `ExternalSecret` remains unchanged

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-config
spec:
  secretStoreRef:
    name: aws-prod
  data:
  - secretKey: password
    remoteRef:
      key: /prod/db/password
```

## Runtime Contract

### 1. Controller responsibilities

The ESO core controller owns:

- `ExternalSecret`, `PushSecret`, and their reconciliation behavior.
- `SecretStore` and `ClusterSecretStore` API validation at the core level.
- Namespace policy for which store objects may be referenced.
- Runtime lookup through `runtimeRef`.
- Connection pooling to provider runtimes.

### 2. Provider runtime responsibilities

The provider runtime owns:

- Provider-specific backend CRDs.
- Provider-specific backend validation.
- Backend client construction and caching.
- Interpretation of provider-specific auth material and provider-specific semantics.
- Provider-specific readiness and capability reporting.

### 3. Request modes

The controller should support two runtime request modes over gRPC.

#### 3.1 `v1` compatibility mode

Used when a `v1` store with inline `spec.provider` also has a `runtimeRef`.

Flow:

1. ESO loads the existing store object.
2. ESO resolves the inline provider config locally into a normalized provider payload.
3. ESO dials the provider runtime via `runtimeRef`.
4. ESO sends the normalized payload plus request context to the runtime.
5. The runtime serves the request without ESO introducing extra user-facing objects.

Properties:

- Existing users only add `runtimeRef`.
- No provider backend CRD is required for the first migration step.
- This path exists for compatibility and migration, not as the clean end state.

#### 3.2 `v2` backend reference mode

Used when a `v2` store references `backendRef`.

Flow:

1. ESO loads the `v2` store.
2. ESO dials the runtime via `runtimeRef`.
3. ESO sends `backendRef` identity plus call context.
4. The provider runtime loads the backend CRD it owns.
5. The provider runtime validates access, builds or reuses a backend client, and serves the request.

Properties:

- Core does not deserialize provider-specific backend schema.
- Provider runtimes can evolve backend schemas independently.
- This is the long-term target architecture.

### 4. Runtime behavior requirements

The provider runtime must be a real long-lived runtime.

Required behavior:

- Cache resolved backend configuration by object UID and generation.
- Cache or pool backend API clients where safe for the provider.
- Avoid creating and destroying a new backend client on every single RPC.
- Avoid reconstructing synthetic legacy store objects per request in the steady state.
- Surface provider-native validation and capability errors clearly.

This is a deliberate change from the current translation-shim approach.

## TLS and Runtime Identity

This design assumes the runtime object is also the source of transport identity.

Requirements:

1. TLS lookup must be explicit from runtime identity, not guessed from service DNS.
2. The common chart-managed path should remain automatic.
3. Arbitrary addresses must remain possible through explicit TLS configuration.
4. Per-runtime TLS isolation is preferred over namespace-wide shared TLS secrets.

The exact per-runtime TLS shape can reuse the previously designed direction:

- managed mode for chart-managed in-cluster runtimes
- explicit secret reference mode for arbitrary external runtimes

This design does not require `SecretStore` users to understand TLS details in the common case.

## Validation Model

### `ClusterProviderClass`

Should validate:

- address and TLS wiring
- transport connectivity
- runtime capability discovery
- supported backend kinds

### `v1` stores with `runtimeRef`

Should validate:

- the referenced runtime exists and is ready
- the inline provider type is supported by that runtime
- current store semantics still hold

### `v2` stores

Should validate:

- the referenced runtime exists and is ready
- the referenced backend kind is advertised by the runtime
- core-level namespace rules

Provider-specific backend validation remains with the runtime.

## Migration Plan

### Phase 1: Compatibility rollout

Add `spec.runtimeRef` to current `v1` `SecretStore` and `ClusterSecretStore`.

Behavior:

- No `runtimeRef`: current in-process path.
- `runtimeRef` set: out-of-process provider runtime path.

User migration:

- Existing `ExternalSecret` and `PushSecret` manifests do not change.
- Existing store manifests only add a runtime reference.

Example:

```yaml
spec:
  runtimeRef:
    kind: ClusterProviderClass
    name: aws
```

### Phase 2: Chart-managed runtime installation

Provider charts should create matching `ClusterProviderClass` resources automatically.

For example, installing the AWS provider chart should also install:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ClusterProviderClass
metadata:
  name: aws
```

This keeps the common documentation path simple:

1. install the provider
2. add `runtimeRef`

### Phase 3: Introduce clean `v2` stores

Add clean `v2alpha1` `SecretStore` and `ClusterSecretStore` resources.

These:

- remove the inline provider union from core
- require `runtimeRef`
- require `backendRef`

New installations should prefer `v2`.

### Phase 4: Tool-assisted migration

Add an `esoctl migrate store` command.

The command should:

1. Read an existing `v1` store.
2. Generate the provider-owned backend CRD.
3. Generate the new `v2` store.
4. Preserve names where possible to avoid `ExternalSecret` changes.
5. Print the output manifests or apply them explicitly by user choice.

### Phase 5: Documentation-led deprecation

- Keep `v1` stores supported for a long compatibility window.
- Recommend `v2` stores in docs and examples.
- Do not require users to rewrite stores merely to adopt out-of-process providers.

## Consequences

Positive:

- Users keep referencing stores, preserving the current UX.
- Existing users can adopt the new runtime path with minimal change.
- Core gets a clean `v2` store API with no inline provider union.
- Provider teams fully own provider-specific backend schema.
- The runtime boundary becomes clearer and more maintainable.

Trade-offs:

- Core temporarily supports both compatibility and clean `v2` request modes.
- There is still a staged migration period where both in-process and out-of-process paths coexist.
- Provider teams must define and maintain backend CRDs for the clean `v2` path.

## Recommendation

Adopt the dual-track design:

1. Extend current `v1` stores with `runtimeRef` for compatibility.
2. Introduce `ClusterProviderClass` as the transport-only runtime object.
3. Introduce clean `v2` stores with `backendRef`.
4. Migrate users in stages rather than forcing an immediate object model rewrite.

This gives the cleanest long-term architecture while keeping migration simple enough to be realistic.
