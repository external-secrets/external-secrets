# Fake Provider V2 Design

**Date:** 2026-04-13

## Goal

Add a focused v2 e2e path for the fake provider that matches the Kubernetes v2 test shape closely enough to validate namespaced `Provider`, `ClusterProvider`, and `PushSecret` behavior without a broad framework redesign.

## Scope

This design is limited to the fake provider on the v2 path.

In scope:
- Deploying `provider-fake` in the existing e2e v2 install flow.
- Creating a fake-provider-specific v2 test harness that manages `provider.external-secrets.io/v2alpha1` `Fake` resources.
- Adding v2 e2e coverage for namespaced `Provider`, `ClusterProvider`, and `PushSecret`.
- Reusing common table-driven cases where fake-provider semantics align with the existing provider-common framework.

Out of scope:
- A generic "install any v2 provider on demand" framework.
- Reworking the controller-runtime or gRPC adapter architecture.
- Non-e2e fake provider runtime changes unless tests expose a real behavior gap.

## Current State

The repository already contains most of the provider-side v2 pieces:
- `providers/v2/fake` provides a gRPC server that wraps the existing v1 fake store and generator through the v2 adapter.
- `apis/provider/fake/v2alpha1` and the fake v2 CRD are present.
- The e2e scheme registration already includes the fake v2 API.

The missing coverage is in the e2e plumbing and suites:

1. The provider v2 e2e install path only enables the Kubernetes provider today.
   - `e2e/suites/provider/suite_test.go` installs ESO in v2 mode with `addon.WithV2KubernetesProvider()`.
   - `e2e/framework/addon/eso_v2_mutators.go` has no equivalent mutator for fake.
   - `e2e/Makefile:test.v2` only builds and loads `provider-kubernetes`.

2. The fake provider e2e suite is still v1-only.
   - `e2e/suites/provider/cases/fake/provider.go` mutates a v1 `SecretStore`.
   - `e2e/suites/provider/cases/fake/regressions.go` only exercises the legacy path.

3. The reusable common cases recently added for Kubernetes v2 cluster-provider and push coverage are not yet exercised by fake v2.
   - The framework now supports reusable `DescribeTable()`-driven `ExternalSecret` and `PushSecret` cases with provider-specific setup hooks.
   - Fake v2 can reuse that machinery if it creates the correct `Provider` / `ClusterProvider` resources and exposes a compatible provider helper.

## Approaches Considered

### Recommended: Targeted fake v2 plumbing plus fake-specific suites

Add fake to the existing v2 e2e install flow, then implement fake v2 suites using the same table-driven test framework used for Kubernetes v2.

Why this is the recommendation:
- It keeps the architectural change small.
- It matches the pattern already established by the Kubernetes v2 work.
- It gets parity coverage without blocking on a generic provider registry design.
- It leaves room for later cleanup if a second non-Kubernetes v2 provider needs the same path.

### Alternative: Generic v2 provider registration framework first

Teach the addon, suite bootstrap, and Makefile layers to accept an arbitrary list of v2 providers.

Rejected for now because:
- It broadens the change far beyond the fake provider work.
- It is more abstraction than the current repository needs for one additional provider.
- It adds review and debugging surface before the fake-provider behaviors are locked down.

### Alternative: Test-local provider deployment inside fake suites

Have each fake v2 suite deploy and manage its own `provider-fake` pod and service.

Rejected because:
- It duplicates Helm/chart install knowledge in test code.
- It diverges from the global v2 provider-suite bootstrap already used by Kubernetes.
- It increases fixture churn and teardown complexity for little gain.

## Design

### 1. Extend the existing provider-v2 e2e install path with fake

Add a new addon mutator, `WithV2FakeProvider()`, beside `WithV2KubernetesProvider()` in `e2e/framework/addon/eso_v2_mutators.go`.

That mutator should:
- enable provider deployments
- populate a `providers.list[...]` entry for `fake`
- point it at `ghcr.io/external-secrets/provider-fake:$VERSION`
- keep replica count at `1` for e2e

The v2 provider suite bootstrap in `e2e/suites/provider/suite_test.go` should install both Kubernetes and fake providers in v2 mode. Installing both is acceptable because:
- the provider suite already relies on label selection to choose which specs run
- provider deployments are isolated by provider name/type
- avoiding label-sensitive install logic keeps the bootstrap predictable

The e2e v2 Makefile path should also build/load `provider-fake` alongside `provider-kubernetes` so the cluster always has the matching local image for fake v2 runs.

### 2. Add a fake-provider-specific v2 e2e harness

Create a new fake v2 helper in `e2e/suites/provider/cases/fake/provider_v2.go`.

Its responsibilities:
- create a namespaced `provider.external-secrets.io/v2alpha1` `Fake` resource for the test namespace
- optionally create a matching `external-secrets.io/v1` `Provider` that points at the fake provider service
- optionally create `ClusterProvider` wrappers for cluster-scoped tests
- expose `CreateSecret` / `DeleteSecret` methods by mutating the fake v2 CR spec data
- provide helpers to wait for namespaced `Provider` / `ClusterProvider` readiness, mirroring the Kubernetes v2 suites where useful

The important boundary is that the helper should make fake v2 look like a normal `framework.SecretStoreProvider` to table-driven cases. The cases should not need to know how the fake provider stores data internally.

### 3. Reuse common cases where the semantics match

The recent framework changes already support reusable case definitions in `e2e/suites/provider/cases/common`. Fake v2 should follow that pattern.

Specifically:
- keep fake-provider-specific suite files thin
- use `DescribeTable()` with `framework.TableFuncWithExternalSecret()` and `framework.TableFuncWithPushSecret()`
- move fake-provider-common behavior into `cases/common` when it can be shared between v1 and v2 fake implementations

Expected reuse split:
- existing fake regression coverage should be reviewed first for v1/v2 sharing
- common `ExternalSecret` sync cases that only rely on `CreateSecret` / `DeleteSecret` should be shareable
- `ClusterProvider` and `PushSecret` cases may need fake-specific common entries if the generic Kubernetes ones encode Kubernetes-only auth/scope assumptions

The design goal is reuse at the test-case level, not forced reuse of unrelated provider semantics.

### 4. Cover full fake-v2 provider parity in e2e

The fake v2 suite should cover three slices.

Namespaced `Provider`:
- basic `ExternalSecret` sync through a namespaced provider
- refresh after remote data changes
- `dataFrom.find` behavior
- existing regression coverage that is provider-agnostic

`ClusterProvider`:
- successful sync through a cluster-scoped wrapper around fake v2
- namespace-condition enforcement if the fake v2 path supports the shared cluster-provider common cases cleanly
- recovery/repair coverage only if the fake provider has a real auth/readiness transition worth testing; otherwise avoid inventing fake-only failure mechanics

`PushSecret`:
- basic push through namespaced fake v2 provider
- push through fake v2 `ClusterProvider`
- any shared common push cases that rely only on generic read-write provider semantics

Because the fake provider is synthetic, parity means parity of controller/provider integration shape, not parity of Kubernetes-specific auth behavior.

### 5. Keep runtime changes minimal and test-driven

The expectation is that most work is e2e plumbing. However, if the new v2 fake e2e tests expose a provider/runtime gap, the implementation should fix the smallest layer that owns the behavior:
- e2e bootstrap if the provider pod is missing
- fake v2 harness if only the tests are incomplete
- runtime/controller/provider code only when the actual v2 integration is broken

No runtime refactor should be included unless a failing test demonstrates the need.

## Testing Strategy

### Test-first sequence

1. Add or extend fast unit coverage around any new addon/config helper logic.
2. Add fake-provider v2 suite tests in a failing state.
3. Run focused fake v2 labels against kind to watch the new cases fail for the right reason.
4. Implement the minimal plumbing/runtime changes.
5. Re-run focused fake v2 suites.
6. Re-run the full `provider=fake` selection in both legacy and v2 modes.

### E2E verification

At minimum, verify:
- `provider=fake && !v2`
- `provider=fake && v2`

If the v2 bootstrap changes affect shared install code, also run:
- a focused non-fake v2 sanity slice, ideally Kubernetes v2, to ensure the additional provider deployment does not regress existing provider-v2 coverage

## Risks And Guardrails

- Changing the v2 suite bootstrap affects every provider-v2 e2e run, so the install path should stay additive and predictable.
- The fake provider is stateful through CR spec data; the v2 helper must patch test-owned resources carefully and avoid leaking state between specs.
- Reuse pressure can cause the wrong abstraction. If a common case starts encoding fake-only or Kubernetes-only assumptions, keep it provider-specific instead of forcing it into `cases/common`.
- `ClusterProvider` coverage should only test semantics the fake provider can actually represent; do not simulate Kubernetes auth failures just for symmetry.

## Success Criteria

- `provider-fake` is deployed automatically in e2e v2 mode.
- Fake v2 has dedicated e2e coverage for namespaced `Provider`, `ClusterProvider`, and `PushSecret`.
- Reusable fake-provider test cases live in `e2e/suites/provider/cases/common` where appropriate, with thin suite entrypoints.
- Legacy fake coverage continues to pass.
- Focused fake v2 kind-backed e2e runs pass and do not regress an existing provider-v2 sanity slice.
