# Kubernetes Provider V2 Push Polish Design

**Date:** 2026-04-11

## Goal

Finish the remaining Kubernetes-provider-only v2 push path work before expanding e2e coverage, without redesigning the current controller -> clientmanager -> gRPC -> adapter architecture.

## Scope

This design is limited to the Kubernetes provider on the v2 path.

In scope:
- Reverse-adapter `PushSecret` correctness for Kubernetes writes.
- `PushSecretStoreRef` kind inference when `kind` is omitted for v2 `Provider` and `ClusterProvider`.
- Unit and e2e coverage needed to lock those semantics down.

Out of scope:
- Other v2 providers.
- Broad runtime refactors in `clientmanager`, provider controllers, or the read path.
- New API behavior beyond what the existing v1 Kubernetes provider already expects.

## Current State

The read path is largely in place:
- `ExternalSecret` reconciliation reaches `runtime/clientmanager/manager.go`.
- The v2 clientmanager resolves provider references and uses the shared gRPC transport.
- `providers/v2/adapter/store/server.go` maps protobuf requests back into synthetic v1 stores and delegates to the v1 Kubernetes provider.

The remaining gaps are both on the write path:

1. `PushSecret` loses Kubernetes secret semantics across the reverse adapter.
   - `providers/v2/adapter/store/client.go` sends only `secret.Data`.
   - `providers/v2/common/proto/provider/secretstore.proto` only carries `secret_data`.
   - `providers/v2/adapter/store/server.go` reconstructs a synthetic secret as `Opaque` with no labels or annotations.
   - The v1 Kubernetes provider uses source secret type, labels, and annotations during `PushSecret`, so the v2 path silently drops real behavior.

2. Omitted `PushSecretStoreRef.kind` can collapse back to `SecretStore` on the v2 path.
   - `pkg/controllers/pushsecret/pushsecret_controller_v2.go:isV2SecretStore()` can correctly identify a v2 `Provider` or `ClusterProvider` even when `kind` is omitted.
   - `pkg/controllers/pushsecret/pushsecret_controller.go:resolvedStoreInfo()` falls back to `SecretStore` for generic `client.Object` values when `ref.Kind == ""`.
   - That incorrect kind can then be used for `mgr.Get()` in the v2 push and delete paths.

## Approaches Considered

### Recommended: Minimal transport extension plus controller normalization fix

Extend the gRPC `PushSecret` request with the subset of secret shape the v1 Kubernetes provider actually uses on write:
- secret type
- labels
- annotations

Then preserve the resolved v2 store kind through the push/delete controller path.

Why this is the recommendation:
- Fixes the known Kubernetes correctness bug directly.
- Keeps the existing architecture intact.
- Avoids provider-specific side channels outside the transport contract.
- Keeps the change small enough for this WIP PR.

### Alternative: Kubernetes-only adapter side channel

Teach the adapter to look up source metadata by some out-of-band mechanism.

Rejected because:
- It leaks Kubernetes-specific policy into the wrong boundary.
- It makes later providers harder, not easier.
- It is harder to test and reason about than an explicit request contract.

### Alternative: Full secret object over gRPC

Send a full secret representation over gRPC.

Rejected for now because:
- It is broader than required for the current bug.
- It increases transport churn before the provider model settles.
- The current Kubernetes need is satisfied by a smaller metadata subset.

## Design

### 1. Extend the v2 `PushSecret` transport contract

Update `providers/v2/common/proto/provider/secretstore.proto` so `PushSecretRequest` carries:
- `secret_data`
- `secret_type`
- `secret_labels`
- `secret_annotations`
- existing `push_secret_data`
- existing `provider_ref`
- existing `source_namespace`

This keeps the request aligned with what the v1 Kubernetes provider consumes today during `mergePushSecretData()`.

The Go-facing contract must be updated consistently:
- `providers/v2/common/types.go`
- `providers/v2/common/grpc/client.go`
- `providers/v2/adapter/store/client.go`
- `providers/v2/adapter/store/server.go`

The reverse adapter client should forward the source secret type and metadata from the controller-side `*corev1.Secret`.

The reverse adapter server should reconstruct a synthetic `*corev1.Secret` using all transported fields instead of forcing `Opaque`.

### 2. Preserve resolved v2 store kind when `kind` is omitted

Once a `PushSecretStoreRef` has been resolved as a v2 `Provider` or `ClusterProvider`, that resolved kind must remain authoritative for:
- `PushSecretToProvidersV2()`
- `DeleteSecretFromProvidersV2()`
- `validateDataToMatchesResolvedStores()`
- any later `mgr.Get()` call that rebuilds `esv1.SecretStoreRef`

The implementation should normalize kind based on the resolved object type rather than the raw ref when the raw ref leaves `kind` empty.

Accepted outcomes:
- namespaced v2 refs with omitted `kind` behave like `Provider`
- cluster-scoped v2 refs with omitted `kind` behave like `ClusterProvider`
- v1 stores still default to `SecretStore` / `ClusterSecretStore` according to existing behavior

### 3. Keep Kubernetes semantics delegated to the v1 provider

This design does not copy Kubernetes merge logic into the v2 adapter.

The adapter stays thin:
- controller-side code provides the original secret shape
- gRPC transports that shape
- server-side adapter reconstructs the secret object
- v1 Kubernetes provider still decides how type, labels, annotations, remote namespace override, and property-based writes behave

That keeps one source of truth for Kubernetes push behavior.

## Testing Strategy

### Unit coverage

1. Adapter client tests
- prove `PushSecret` forwards type, labels, annotations, metadata, provider ref, and source namespace

2. Adapter server tests
- prove `PushSecret` rebuilds a secret with the transported type, labels, annotations, and data
- prove existing validation behavior remains unchanged

3. gRPC client tests
- prove the generated request contains the added fields

4. PushSecret controller tests
- prove omitted-kind refs resolve to `Provider` / `ClusterProvider`
- prove push and delete paths still obtain the correct v2 store client
- prove label-selector validation matches normalized kinds

### E2E coverage

Add focused Kubernetes v2 push specs that prove:
- pushed secrets preserve source secret type
- pushed secrets preserve labels/annotations relevant to the Kubernetes provider behavior
- omitted-kind `PushSecretStoreRef` still works for namespaced provider flow
- omitted-kind `PushSecretStoreRef` still works for cluster provider flow if the existing suite structure supports it without excessive fixture churn

The e2e slice should stay provider-scoped and run through the existing Makefile knobs already used for Kubernetes-only coverage.

## Risks And Guardrails

- The protobuf change touches generated files, so the plan must include regeneration and focused verification.
- The `PushSecret` interface change affects both adapter and gRPC client code, so tests need to land before the implementation change is considered done.
- Kind normalization must not regress v1 `PushSecret` behavior; defaulting rules for v1 stores remain unchanged.

## Success Criteria

- Kubernetes v2 `PushSecret` no longer drops source secret type, labels, or annotations.
- Omitted-kind v2 store refs work consistently on push and delete paths.
- Unit tests cover both fixes directly.
- Kubernetes-only e2e covers the new v2 push semantics before broader provider rollout.
