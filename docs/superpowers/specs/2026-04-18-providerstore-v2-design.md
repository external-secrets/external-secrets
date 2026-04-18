# ProviderStore V2 Design

**Date:** 2026-04-18

## Goal

Define a clean store API for out-of-process providers that removes the inline `spec.provider` union from core, keeps runtime transport concerns separate from backend configuration, and preserves the current `v1` store APIs as the low-friction compatibility path.

## Scope

In scope:
- Defining the clean `external-secrets.io/v2alpha1` store API.
- Defining the long-term role of `ClusterProviderClass`.
- Defining `backendRef` semantics and namespace rules.
- Defining how `ExternalSecret` and `PushSecret` reference the clean store kinds.
- Defining the controller-to-runtime contract for clean `v2` stores.

Out of scope:
- Implementing every provider-owned backend CRD.
- Removing `v1` `SecretStore` or `ClusterSecretStore`.
- Designing `esoctl` migration UX in detail.
- Adding transport capability discovery metadata to `ClusterProviderClass`.

## Decisions

The clean path is intentionally separate from the compatibility path.

- `v1 SecretStore` and `v1 ClusterSecretStore` remain the compatibility surface.
- Clean `v2` stores are separate CRDs, not additional versions on the existing CRDs.
- Clean `v2` stores remove `spec.provider`.
- Clean `v2` stores require both `runtimeRef` and `backendRef`.
- `backendRef` is a full Kubernetes object reference: `apiVersion`, `kind`, `name`, optional `namespace`.
- The provider runtime resolves the backend CR itself. ESO sends only the effective backend reference over gRPC.
- `ClusterProviderClass` remains transport-only and does not advertise supported backend kinds.
- `ClusterProviderClass` status keeps readiness conditions only.
- `ClusterProviderStore` keeps namespace `conditions`, like today's `ClusterSecretStore`.

## Problem

The compatibility design solves migration, but it does not give core a clean long-term store API. If we stop at `v1 SecretStore + runtimeRef`, the ESO API still permanently owns the giant inline provider union. That is the opposite of the "minimal core" goal.

At the same time, reusing the existing `SecretStore` and `ClusterSecretStore` kinds for the clean path would create a poor migration surface:

- the new shape removes `spec.provider`
- the new shape requires `backendRef`
- `ExternalSecret` and `PushSecret` store references only carry `name` and `kind`, not `apiVersion`
- conversion between the old and new store shapes would be lossy and confusing

The clean path therefore needs its own store kinds.

## Approaches Considered

### Recommended: new clean store kinds

Introduce `ProviderStore` and `ClusterProviderStore` in `external-secrets.io/v2alpha1`.

Why this is recommended:
- makes the clean path explicit
- lets `v1` and clean `v2` coexist without conversion tricks
- avoids overloading `SecretStore` with incompatible semantics
- keeps the compatibility migration simple: existing users can stay on `v1` and only add `runtimeRef`

### Alternative: reuse `SecretStore` names and add `apiVersion` to every store ref

Rejected because:
- `ExternalSecret`, `PushSecret`, generators, validation, and controller lookup all need broader API changes
- it spreads version-selection complexity into every consumer object
- it still creates awkward semantics for two incompatible store shapes with the same kind

### Alternative: move backend selection into `ClusterProviderClass`

Rejected because:
- `ClusterProviderClass` is a transport object, not a backend instance selector
- one runtime must be able to serve many backend CRs
- it couples runtime routing and provider-owned configuration

## Proposed API

### 1. Transport Object

`ClusterProviderClass` remains the transport and readiness object for provider runtimes.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ClusterProviderClass
metadata:
  name: aws
spec:
  address: provider-aws.external-secrets-system.svc:8080
status:
  conditions:
    - type: Ready
      status: "True"
```

Semantics:

- `spec.address` is the dial target for the runtime.
- `status.conditions` reports runtime reachability and readiness.
- No backend-type registry lives here.
- No read/write capability metadata lives here.

### 2. Namespaced Clean Store

`ProviderStore` is namespace-scoped and always resolves a backend object in the same namespace.
Its resource name is `providerstores`.

```yaml
apiVersion: external-secrets.io/v2alpha1
kind: ProviderStore
metadata:
  name: aws-prod
  namespace: team-a
spec:
  runtimeRef:
    kind: ClusterProviderClass
    name: aws
  backendRef:
    apiVersion: aws.external-secrets.io/v1alpha1
    kind: SecretsManagerStore
    name: prod
```

Validation rules:

- `spec.runtimeRef` is required.
- `spec.backendRef` is required.
- `spec.provider` does not exist.
- `spec.backendRef.namespace` must be empty or equal to `metadata.namespace`.
- ESO treats an omitted `backendRef.namespace` as `metadata.namespace`.

### 3. Cluster Clean Store

`ClusterProviderStore` is cluster-scoped and can point at a shared backend object or at a namespace-relative backend object.
Its resource name is `clusterproviderstores`.

```yaml
apiVersion: external-secrets.io/v2alpha1
kind: ClusterProviderStore
metadata:
  name: aws-shared
spec:
  runtimeRef:
    kind: ClusterProviderClass
    name: aws
  backendRef:
    apiVersion: aws.external-secrets.io/v1alpha1
    kind: SecretsManagerStore
    name: shared
  conditions:
    - namespaces:
        - team-a
        - team-b
```

Validation rules:

- `spec.runtimeRef` is required.
- `spec.backendRef` is required.
- `spec.provider` does not exist.
- `spec.conditions` keeps the current `ClusterSecretStore` meaning.
- `spec.backendRef.namespace` is optional.
- If `spec.backendRef.namespace` is set, ESO always uses it.
- If `spec.backendRef.namespace` is omitted, ESO resolves it to the `ExternalSecret` or `PushSecret` namespace before sending the request to the provider runtime.

This defaulting keeps one cluster-scoped store reusable across namespaces without forcing the store object itself to hardcode a backend namespace.

### 4. Backend Reference Shape

The clean store API uses a generic reference shape:

```yaml
backendRef:
  apiVersion: aws.external-secrets.io/v1alpha1
  kind: SecretsManagerStore
  name: prod
  namespace: team-a # optional
```

Semantics:

- `apiVersion` and `kind` identify the provider-owned backend CRD type.
- `name` identifies the backend instance.
- `namespace` is resolved by ESO according to the store kind rules above.
- ESO does not interpret the backend spec payload.

### 5. Consumer References

`ExternalSecret` and `PushSecret` continue to reference a store object, but the allowed store kinds expand.

New allowed kinds:
- `ProviderStore`
- `ClusterProviderStore`

This means clean `v2` store adoption is explicit in consumer manifests:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-config
  namespace: team-a
spec:
  secretStoreRef:
    name: aws-prod
    kind: ProviderStore
  data:
    - secretKey: password
      remoteRef:
        key: /prod/db/password
```

Implication:

- the compatibility path remains the "minimal change" option for existing users
- the clean `v2` path is explicit and may require consumer `kind` updates

## Runtime Contract

### Compatibility mode

For `v1 SecretStore` and `v1 ClusterSecretStore` with `runtimeRef`:

1. ESO resolves the referenced `ClusterProviderClass`.
2. ESO serializes the legacy inline provider config.
3. ESO sends that compatibility payload to the runtime.
4. The runtime serves the request without ESO needing provider-specific backend CRDs.

This mode exists to keep migration cheap.

### Clean `v2` mode

For `ProviderStore` and `ClusterProviderStore`:

1. ESO resolves the referenced `ClusterProviderClass`.
2. ESO validates store scoping and cluster-store `conditions`.
3. ESO resolves the effective backend namespace:
   - `ProviderStore`: store namespace
   - `ClusterProviderStore` with explicit backend namespace: that explicit namespace
   - `ClusterProviderStore` with omitted backend namespace: caller namespace
4. ESO sends the effective `backendRef` over gRPC.
5. The provider runtime loads the referenced backend CR from Kubernetes using its own API scheme and credentials.
6. The provider runtime rejects unknown backend kinds, missing backend objects, or invalid backend specs.

This keeps provider-specific schema ownership fully with the provider runtime.

## Provider Responsibilities

The provider runtime owns:

- registering and reconciling provider-owned backend CRDs
- validating backend spec semantics
- loading backend objects by reference
- constructing provider clients from backend config
- caching backend-specific clients if useful

ESO core owns:

- store-level validation and namespace policy
- runtime lookup and connection management
- effective backend namespace resolution
- request orchestration for `ExternalSecret` and `PushSecret`

## Migration

### Existing users

Existing users do not need to adopt the clean store API to use out-of-process providers.

Their path remains:

1. install provider runtime
2. install or create `ClusterProviderClass`
3. add `runtimeRef` to existing `SecretStore` or `ClusterSecretStore`

`ExternalSecret` and `PushSecret` manifests remain unchanged on this path.

### New users

New users that want the clean minimal core API should use:

- provider-owned backend CRDs
- `ProviderStore` or `ClusterProviderStore`
- `kind: ProviderStore` or `kind: ClusterProviderStore` in consumer refs

### Tool-assisted migration

A follow-up `esoctl` flow can generate:

- provider-owned backend CRs from legacy inline provider config
- clean `ProviderStore` or `ClusterProviderStore` objects
- optional consumer manifest updates when the user wants to leave the compatibility path

## Consequences

Positive:

- core gets a clean store API with no inline provider union
- `ClusterProviderClass` stays small and focused
- provider teams fully own backend CRD evolution
- one runtime can serve many backend objects cleanly
- existing users still have a low-friction migration path

Trade-offs:

- the clean path introduces new store kinds, not just a new version
- `ExternalSecret` and `PushSecret` must opt into those new kinds
- ESO must support both compatibility and clean request modes for a transition period

## Recommendation

Adopt the dual-track model:

1. keep `v1 SecretStore` and `ClusterSecretStore` as the compatibility path
2. keep `ClusterProviderClass` transport-only
3. introduce `external-secrets.io/v2alpha1 ProviderStore`
4. introduce `external-secrets.io/v2alpha1 ClusterProviderStore`
5. require explicit `runtimeRef` and `backendRef` in the clean API
6. let provider runtimes load provider-owned backend CRs directly from Kubernetes

This produces the cleanest long-term API without sacrificing the low-friction migration path that existing users need.
