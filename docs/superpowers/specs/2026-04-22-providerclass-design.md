# ProviderClass Design

**Date:** 2026-04-22

## Goal

Add a namespaced runtime object named `ProviderClass` with the same transport and readiness semantics as `ClusterProviderClass`, and make namespaced `SecretStore` runtime references default to `ProviderClass`.

## Scope

In scope:
- Defining the `ProviderClass` API shape and scope.
- Defining `runtimeRef.kind` defaults for `SecretStore` and `ClusterSecretStore`.
- Defining runtime resolution rules for namespaced and cluster-scoped stores.
- Defining validation rules that prevent `ClusterSecretStore` from using namespaced runtimes.
- Defining controller and client-manager behavior needed to support both runtime classes.

Out of scope:
- Changing the transport schema beyond what `ClusterProviderClass` already supports.
- Redesigning `ClusterProviderClass`.
- Changing classic in-process store behavior when `runtimeRef` is omitted.
- Changing clean `v2` store behavior outside the runtime-ref lookup rules shared with `v1`.

## Problem

The current runtime-split implementation supports only `ClusterProviderClass`. That forces namespaced `SecretStore` objects to reference a cluster-scoped runtime even when the runtime should be owned and managed within a tenant namespace.

The desired model is:

- namespaced stores should be able to use a namespaced runtime object
- cluster-scoped stores should continue to use a cluster-scoped runtime object
- the default runtime kind for namespaced stores should match their namespace scope

Without an explicit namespaced runtime class, the API pushes users toward broader-than-needed scope and makes namespaced ownership awkward.

## Decisions

- Introduce `external-secrets.io/v1alpha1 ProviderClass` as a namespaced resource.
- `ProviderClass` has the same schema and status semantics as `ClusterProviderClass`.
- `SecretStore.spec.runtimeRef.kind` defaults to `ProviderClass`.
- `ClusterSecretStore.spec.runtimeRef.kind` defaults to `ClusterProviderClass`.
- A `SecretStore` may explicitly reference either `ProviderClass` or `ClusterProviderClass`.
- A `ClusterSecretStore` may reference only `ClusterProviderClass`.
- When a `SecretStore` references `ProviderClass`, ESO resolves it in the `SecretStore` namespace.
- `ClusterSecretStore` never resolves a namespaced runtime class from the caller namespace.

## Approaches Considered

### Recommended: scope-sensitive defaulting with explicit runtime kinds

- `SecretStore` defaults to `ProviderClass`.
- `ClusterSecretStore` defaults to `ClusterProviderClass`.
- Explicit `runtimeRef.kind` values remain supported where valid.

Why this is recommended:
- the default matches the scope of the store object
- namespaced ownership becomes the standard case for namespaced stores
- cluster-scoped shared runtimes remain available when explicitly requested
- runtime lookup rules stay deterministic and easy to explain

### Alternative: keep `ClusterProviderClass` as the universal default

Rejected because:
- it conflicts with the desired namespaced-first UX
- it keeps namespaced runtimes as an opt-in edge path

### Alternative: infer kind by lookup order when `runtimeRef.kind` is omitted

Rejected because:
- it creates hidden precedence rules
- validation becomes weaker
- identical names across scopes would be ambiguous for users

## Proposed API

### 1. Namespaced runtime object

`ProviderClass` mirrors the existing `ClusterProviderClass` shape but is namespace-scoped.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ProviderClass
metadata:
  name: aws
  namespace: team-a
spec:
  address: provider-aws.team-a.svc:8080
status:
  conditions:
    - type: Ready
      status: "True"
```

Semantics:

- `spec.address` is the runtime dial target.
- `status.conditions` reports runtime readiness.
- TLS loading and health checks use the same rules as `ClusterProviderClass`.

### 2. Namespaced store runtime reference

`SecretStore.spec.runtimeRef.kind` defaults to `ProviderClass`.

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: aws-prod
  namespace: team-a
spec:
  runtimeRef:
    name: aws
  provider:
    fake:
      data:
        - key: db
          value: s3cr3t
```

Resolution rules:

- if `kind` is omitted, ESO treats it as `ProviderClass`
- if `kind` is `ProviderClass`, ESO resolves `ProviderClass` in `metadata.namespace`
- if `kind` is `ClusterProviderClass`, ESO resolves the cluster-scoped object with that name

### 3. Cluster-scoped store runtime reference

`ClusterSecretStore.spec.runtimeRef.kind` defaults to `ClusterProviderClass`.

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: aws-shared
spec:
  runtimeRef:
    name: aws
  provider:
    fake:
      data:
        - key: db
          value: s3cr3t
```

Resolution rules:

- if `kind` is omitted, ESO treats it as `ClusterProviderClass`
- if `kind` is `ClusterProviderClass`, ESO resolves the cluster-scoped object with that name
- if `kind` is `ProviderClass`, admission validation rejects the resource

## Controller Design

- Keep the existing `ClusterProviderClass` reconciler.
- Add a `ProviderClass` reconciler with the same health-check and status-update behavior.
- Register both controllers from `cmd/controller/root.go`.
- Keep the implementation parallel rather than introducing a shared generic reconciler in this change.

This keeps the new feature low-risk and matches the current code structure.

## Client Resolution Design

`runtime/clientmanager` should branch on both the store kind and the runtime kind.

For `SecretStore`:
- omitted `runtimeRef.kind` means `ProviderClass`
- `ProviderClass` lookup uses `types.NamespacedName{Name: runtimeRef.Name, Namespace: store.GetNamespace()}`
- `ClusterProviderClass` lookup uses `types.NamespacedName{Name: runtimeRef.Name}`

For `ClusterSecretStore`:
- omitted `runtimeRef.kind` means `ClusterProviderClass`
- `ClusterProviderClass` lookup keeps the existing behavior
- `ProviderClass` is treated as invalid input and should return a clear error if reached outside admission

Connection pooling, compatibility-store serialization, TLS loading, and adapter client creation stay unchanged after the runtime object has been resolved.

## Validation and CRD Rules

- Add `ProviderClass` to the allowed enum values for namespaced store runtime refs.
- Keep CRD defaults aligned with store scope:
  - `SecretStore`: default `ProviderClass`
  - `ClusterSecretStore`: default `ClusterProviderClass`
- Add webhook validation so `ClusterSecretStore` rejects `runtimeRef.kind: ProviderClass`.
- Preserve backward compatibility for explicit `runtimeRef.kind: ClusterProviderClass` on `SecretStore`.

## Testing

Add or update tests for:

- API defaulting and runtime-ref serialization for `SecretStore`
- API defaulting and runtime-ref serialization for `ClusterSecretStore`
- validation rejection when `ClusterSecretStore` uses `ProviderClass`
- client-manager lookup of `ProviderClass` in the `SecretStore` namespace
- client-manager lookup of explicit `ClusterProviderClass` from a `SecretStore`
- error reporting when a referenced `ProviderClass` is missing
- readiness reconciliation for `ProviderClass`
- CRD and snapshot output showing the new enum/default behavior
- e2e coverage for runtime-class resolution using the v2 Fake provider

### E2E coverage

Add end-to-end tests that exercise runtime-class resolution through real reconciliation, not only through unit tests.

Recommended test strategy:

- use the v2 Fake provider as the working runtime target because the e2e suite already provisions it
- create one valid runtime object that points at the real Fake provider service
- create a shadow runtime object with the same name in the other scope that points at an invalid address
- assert success or failure based on which runtime class ESO should resolve

Required scenarios:

1. `SecretStore` with omitted `runtimeRef.kind`
- create `ProviderClass/<ns>/<name>` with the valid Fake provider address
- create `ClusterProviderClass/<name>` with an invalid address
- assert the `ExternalSecret` sync succeeds, proving default namespaced resolution chose `ProviderClass`

2. `SecretStore` with explicit `runtimeRef.kind: ClusterProviderClass`
- create `ProviderClass/<ns>/<name>` with an invalid address
- create `ClusterProviderClass/<name>` with the valid Fake provider address
- assert the `ExternalSecret` sync succeeds, proving explicit cluster-scoped resolution chose `ClusterProviderClass`

3. `ClusterSecretStore` with omitted `runtimeRef.kind`
- create `ClusterProviderClass/<name>` with the valid Fake provider address
- assert the `ExternalSecret` sync succeeds, proving the cluster-scoped default remains `ClusterProviderClass`

These tests should live in the existing provider v2 e2e area so they can reuse the Fake provider addon and helper functions.

## Migration Impact

This changes the default meaning of omitted `SecretStore.spec.runtimeRef.kind` from `ClusterProviderClass` to `ProviderClass`.

Operationally:

- existing manifests that already set `kind: ClusterProviderClass` continue to behave the same
- existing manifests that omit `kind` and expect cluster-scoped lookup must start setting `kind: ClusterProviderClass`
- new namespaced runtime usage becomes simpler because omitted `kind` now matches the namespaced object

This is an intentional API behavior change and should be documented in generated docs and examples.
