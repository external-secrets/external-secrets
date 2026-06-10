# E2E V2 Migration Plan

## Goal

Reuse the existing provider e2e suite for the new V2 architecture, starting with the Kubernetes provider, and delete `e2e/suites/v2` after its reusable logic has been migrated.

The target outcome is:

- `provider.test` remains the single suite binary for provider e2e tests.
- legacy and V2 runs are selected by configuration, not by separate suites.
- existing table-driven provider assertions are reused.
- V2-only Kubernetes tests for push, capabilities, and metrics live under `e2e/suites/provider`.
- only the Kubernetes V2 provider is enabled initially.

## Agreed Decisions

- Use `provider.test` for both legacy and V2 runs.
- Replace the current `test.e2e.v2` flow so it runs `provider.test` in V2 mode instead of `v2.test`.
- Reuse the existing table-driven tests in `e2e/suites/provider/cases/kubernetes`.
- Also migrate the V2 push, capabilities, and metrics tests into `e2e/suites/provider`.
- For referent-auth equivalence in V2, use `ClusterProvider` with `AuthenticationScopeManifestNamespace`.
- In the first migration step, enable only the Kubernetes provider deployment in V2 mode.

## Proposed Design

### 1. Introduce a provider suite mode switch

Add an environment-driven mode switch for the provider suite.

Suggested env var:

- `E2E_PROVIDER_MODE=legacy|v2`

Behavior:

- default: `legacy`
- `v2`: install ESO with V2 enabled, create Provider/ClusterProvider CRDs, and enable the Kubernetes provider deployment

This keeps the suite layout stable and avoids carrying `e2e/suites/v2` as a parallel test hierarchy.

### 2. Move reusable V2 bootstrap into framework/addon

The V2 install logic currently proven in `e2e/suites/v2/suite_test.go` should be promoted into reusable addon mutators instead of staying in suite code.

Recommended shape:

- add V2 mutators in `e2e/framework/addon`
- use `addon.NewESO(...)` for both legacy and V2
- do **not** rely on the bespoke installer in `e2e/framework/addon/eso_v2.go`; either remove it later or repurpose it so there is only one supported installation path

Suggested mutators:

- `addon.WithV2Mode()`
- `addon.WithProviderNamespace("external-secrets-system")`
- `addon.WithV2Providers(...providers)` or a narrower `addon.WithKubernetesV2Provider()`

Minimum V2 Helm settings:

- `v2.enabled=true`
- `crds.createProvider=true`
- `crds.createClusterProvider=true`
- `providers.enabled=true`
- one provider entry for Kubernetes
- release namespace `external-secrets-system`
- release name `external-secrets`

### 3. Make framework defaults mode-aware

The main framework coupling today is that the default testcase builder assumes a namespaced `SecretStore` reference and no ref kind.

Add mode-aware defaults to `framework.Framework` so existing test tables can continue to build their manifests the same way.

Suggested new fields on `framework.Framework`:

- `DefaultSecretStoreRefKind string`
- `DefaultPushSecretStoreRefKind string`
- optionally `DefaultPushSecretStoreRefAPIVersion string`
- optionally `ProviderMode string`

Set them in `framework.New(...)` based on `E2E_PROVIDER_MODE`:

- legacy:
  - `DefaultSecretStoreRefKind = ""`
  - `DefaultPushSecretStoreRefKind = ""`
- v2:
  - `DefaultSecretStoreRefKind = esv1.ProviderKindStr`
  - `DefaultPushSecretStoreRefKind = esv1.ProviderKindStr`

Update default testcase creation in `e2e/framework/testcase.go` so:

- `makeDefaultExternalSecretTestCase()` sets `Spec.SecretStoreRef.Kind` from framework defaults
- `makeDefaultPushSecretTestCase()` sets `Spec.SecretStoreRefs[0].Kind` from framework defaults

This allows existing table-driven tests to reuse the same setup with no broad manifest rewrite.

### 4. Move V2 resource helpers out of `e2e/suites/v2`

Anything that is reusable test infrastructure should move out of the `v2` suite package before that package is deleted.

Recommended destination:

- new package under `e2e/framework/v2` or similar

Helpers to move first:

- cluster CA bundle lookup
- Kubernetes provider CR creation
- Provider creation
- ClusterProvider creation
- provider readiness waits
- RBAC helper for Kubernetes provider access
- metrics scraping helpers

Likely source files:

- `e2e/suites/v2/helpers.go`
- `e2e/suites/v2/metrics_helpers.go`

After migration, provider cases should import only framework packages, never `e2e/suites/v2`.

## Kubernetes Provider Migration

### 5. Add a V2 setup path to the Kubernetes provider case

Extend `e2e/suites/provider/cases/kubernetes/provider.go` so the provider setup can operate in both modes.

Recommended approach:

- keep the current `Provider` as the abstraction used by the table tests
- branch internally on provider mode
- legacy mode keeps creating `SecretStore` and `ClusterSecretStore`
- V2 mode creates:
  - namespaced `Kind=Kubernetes`
  - namespaced `Kind=Provider`
  - `ClusterProvider` for the referent-auth equivalent path

Recommended structure:

- `BeforeEach()` decides `setupLegacy()` vs `setupV2()`
- `CreateStore()` keeps legacy behavior
- add `CreateStoreV2()`
- `CreateReferentStore()` keeps legacy behavior
- add `CreateReferentStoreV2()`

### 6. Preserve existing testcase naming semantics

To minimize rewiring, keep the resource names aligned with what the table tests already expect.

For the default case in V2 mode:

- `Provider.Name = f.Namespace.Name`
- `Provider.Namespace = f.Namespace.Name`
- `ExternalSecret.Spec.SecretStoreRef.Name` remains untouched by tests and still resolves correctly

For referent-auth cases in V2 mode:

- create a `ClusterProvider` using the existing referent naming pattern
- wire it with `AuthenticationScopeManifestNamespace`

### 7. Map referent-auth behavior explicitly

Today the Kubernetes suite expresses referent auth by switching to `ClusterSecretStore`.

In V2, the semantic equivalent should be:

- `SecretStoreRef.Kind = ClusterProvider`
- `SecretStoreRef.Name = <referent-name>`
- `ClusterProvider.Spec.AuthenticationScope = ManifestNamespace`

Update `withReferentStore(...)` so it branches by mode:

- legacy: `ClusterSecretStore`
- V2: `ClusterProvider`

Use API constants where available:

- `esv1.ProviderKindStr`
- `esv1.ClusterProviderKindStr`
- `esv1.AuthenticationScopeManifestNamespace`

### 8. RBAC model for Kubernetes V2 tests

The V2 Kubernetes provider needs explicit RBAC to read or write secrets in the target namespace.

For namespaced provider tests:

- create role + rolebinding in the remote namespace
- bind to the manifest namespace service account used by the provider auth configuration

For referent/cluster-provider cases:

- ensure RBAC is granted in the namespace the provider should access
- when `AuthenticationScopeManifestNamespace` is used, bind the service account identity from the ExternalSecret/PushSecret namespace

The plan should prefer one RBAC helper with explicit parameters over many ad-hoc copies.

## Test Migration Scope

### 9. Reuse the existing table-driven Kubernetes tests

Keep the existing table-driven assertions in:

- `e2e/suites/provider/cases/kubernetes/kubernetes.go`

These should run in both modes, but V2 execution should be label-gated so we can migrate incrementally.

Recommended labeling:

- retain current `kubernetes` label
- add V2-specific coverage behind `v2`
- optionally add `legacy` label to the old suite bootstrap only if needed later

Recommended strategy:

- do not duplicate the common table entries
- instantiate the same tables with a V2-aware provider setup

### 10. Migrate V2-only Kubernetes tests into provider suite

Move the following test coverage into `e2e/suites/provider/cases/kubernetes` or a sibling under `e2e/suites/provider`:

- capabilities tests
- push tests
- metrics tests
- any cluster-provider tests that express Kubernetes V2 behavior not already covered by the legacy tables

Suggested placement:

- `e2e/suites/provider/cases/kubernetes/capabilities_v2_test.go`
- `e2e/suites/provider/cases/kubernetes/push_v2_test.go`
- `e2e/suites/provider/cases/kubernetes/metrics_v2_test.go`
- `e2e/suites/provider/cases/kubernetes/cluster_provider_v2_test.go`

Guideline:

- keep table-driven sync assertions in the existing `kubernetes.go`
- keep new V2-only behavior in separate files so the legacy flow stays readable

### 11. Metrics migration approach

The metrics coverage in `e2e/suites/v2/metrics_test.go` is useful and should remain, but it should be narrowed to Kubernetes-only assumptions for this first step.

First migration scope:

- provider readiness metrics for `Provider`
- readiness metrics for `ClusterProvider`
- gRPC client/server request metrics after an `ExternalSecret` sync
- any Kubernetes-specific cache metrics that remain stable enough for CI

Avoid in the first pass:

- fake-provider metrics dependencies
- broad provider-agnostic abstractions that are not needed yet

## CI / Build Changes

### 12. Switch `test.e2e.v2` to run the provider suite

Update the E2E execution path so V2 uses the provider suite binary instead of `v2.test`.

Planned changes:

- `e2e/Makefile`
  - keep loading the controller image
  - load only `provider-kubernetes` for V2 initially
  - run `TEST_SUITES="provider"`
  - run with `E2E_PROVIDER_MODE=v2`
  - keep `GINKGO_LABELS="v2"`
- `e2e/Dockerfile`
  - remove `ADD e2e/suites/v2/v2.test /v2.test` once the old suite is gone

Target command shape:

```bash
GINKGO_LABELS="v2" E2E_PROVIDER_MODE="v2" TEST_SUITES="provider" ./run.sh
```

### 13. Delete `e2e/suites/v2` only after parity is reached

Deletion should happen only after:

- V2 bootstrap moved out of `e2e/suites/v2/suite_test.go`
- helpers moved out of `e2e/suites/v2/helpers.go`
- metrics helpers moved out of `e2e/suites/v2/metrics_helpers.go`
- Kubernetes V2 tests live under `e2e/suites/provider`
- `test.e2e.v2` no longer depends on `v2.test`

At that point remove:

- `e2e/suites/v2/`
- `v2.test` build/copy path
- any stale docs or make targets referencing the old suite layout

## Suggested Implementation Sequence

1. Add provider-mode env handling to the provider suite bootstrap.
2. Extract V2 Helm mutators from `e2e/suites/v2/suite_test.go` into `e2e/framework/addon`.
3. Make framework testcase defaults mode-aware for `Provider` refs.
4. Move reusable V2 helpers into a framework package.
5. Extend Kubernetes provider setup to support legacy and V2.
6. Reuse the existing table-driven Kubernetes tests in V2 mode.
7. Migrate V2 capabilities tests into the provider suite.
8. Migrate V2 push tests into the provider suite.
9. Migrate V2 metrics tests into the provider suite.
10. Update `test.e2e.v2` to run `provider.test`.
11. Delete `e2e/suites/v2` and remove `v2.test` packaging.

## Acceptance Criteria

The migration is complete when all of the following are true:

- `make test.e2e` still runs the legacy provider suite unchanged.
- `make test.e2e.v2` runs `provider.test` with `E2E_PROVIDER_MODE=v2`.
- the Kubernetes table-driven tests pass in V2 mode using `Provider` / `ClusterProvider` resources.
- V2 push, capabilities, and metrics tests pass from within `e2e/suites/provider`.
- only the Kubernetes V2 provider deployment is enabled in the initial V2 run.
- `e2e/suites/v2` is deleted.
- the e2e image no longer bundles `v2.test`.

## Risks / Watchouts

- The existing `e2e/framework/addon/eso_v2.go` appears to describe a second installation approach; keeping both active will create drift. Consolidate on the Helm-mutation path.
- Metrics tests may be brittle if they assert on counters that can vary with retries or background reconciliation. Prefer existence and lower-bound assertions over exact counts.
- `ClusterProvider` resources are cluster-scoped and must be cleaned up carefully to avoid cross-test leakage.
- The default testcase ref-kind switch must not affect non-V2 suites; it should be entirely gated by `E2E_PROVIDER_MODE`.
- Referent-auth semantics should be validated carefully: for V2 this migration assumes `AuthenticationScopeManifestNamespace` is the intended equivalent.

## Out of Scope for This First Step

- Migrating all other providers to V2.
- Enabling AWS, Fake, or other provider deployments in the V2 provider-suite bootstrap.
- Building a generic V2 abstraction for every provider before the Kubernetes migration proves the pattern.
