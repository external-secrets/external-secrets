# Provider API Removal And V2 E2E Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove `Provider` and `ClusterProvider` from core, make `ProviderStore` and `ClusterProviderStore` the only out-of-process store path, and migrate the current v2 e2e coverage to the clean store API.

**Architecture:** Delete the transitional runtime-backed CRDs and their controller/clientmanager branches, then switch the e2e helper and common harness layers to create `ProviderStore` and `ClusterProviderStore` objects that point at provider-owned backend CRs through `runtimeRef` and `backendRef`. Keep `SecretStore` / `ClusterSecretStore` compatibility with `runtimeRef` unchanged.

**Tech Stack:** Go, controller-runtime, Kubernetes CRDs, ginkgo e2e helpers, helm snapshots, protobuf-backed runtime clientmanager

---

## File Map

- Modify: `apis/externalsecrets/v1/provider_types.go`
- Modify: `apis/externalsecrets/v1/register.go`
- Modify: `apis/externalsecrets/v1/zz_generated.deepcopy.go`
- Modify: `apis/externalsecrets/v1/externalsecret_types.go`
- Modify: `apis/externalsecrets/v1beta1/externalsecret_types.go`
- Modify: `apis/externalsecrets/v1alpha1/pushsecret_types.go`
- Modify: `config/crds/bases/external-secrets.io_externalsecrets.yaml`
- Modify: `config/crds/bases/external-secrets.io_pushsecrets.yaml`
- Modify: `config/crds/bases/kustomization.yaml`
- Modify: `deploy/crds/bundle.yaml`
- Modify: `deploy/charts/external-secrets/templates/deployment.yaml`
- Modify: `deploy/charts/external-secrets/templates/crds/external-secrets.io_clusterexternalsecret.yaml`
- Modify: `deploy/charts/external-secrets/templates/crds/external-secrets.io_externalsecret.yaml`
- Modify: `deploy/charts/external-secrets/templates/crds/external-secrets.io_clusterpushsecret.yaml`
- Modify: `deploy/charts/external-secrets/templates/crds/external-secrets.io_pushsecret.yaml`
- Delete or stop generating: provider/clusterprovider CRD surfaces in `config/crds`, `deploy/crds`, and Helm snapshots
- Modify: `cmd/controller/root.go`
- Delete: `pkg/controllers/provider/controller.go`
- Delete: `pkg/controllers/provider/controller_test.go`
- Delete: `pkg/controllers/clusterprovider/controller.go`
- Delete: `pkg/controllers/clusterprovider/controller_test.go`
- Modify: `pkg/controllers/externalsecret/externalsecret_controller.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2_test.go`
- Modify: `runtime/clientmanager/manager.go`
- Modify: `runtime/clientmanager/manager_test.go`
- Modify: `runtime/clientmanager/providerstore.go`
- Modify: `runtime/clientmanager/providerstore_test.go`
- Modify: `pkg/controllers/providerstore/providerstore_controller_test.go`
- Modify: `e2e/framework/framework.go`
- Modify: `e2e/framework/v2/helpers.go`
- Modify: `e2e/suites/provider/cases/common/namespaced_provider.go`
- Modify: `e2e/suites/provider/cases/common/clusterprovider.go`
- Modify: `e2e/suites/provider/cases/common/operational_v2.go`
- Modify: `e2e/suites/provider/cases/common/push_secret.go`
- Modify: `e2e/suites/provider/cases/fake/provider_v2.go`
- Modify: `e2e/suites/provider/cases/fake/operational_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/provider_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/clusterprovider_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/push_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/operational_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/metrics_v2.go`
- Modify: `e2e/suites/provider/cases/kubernetes/capabilities_v2.go`
- Modify: `e2e/suites/provider/cases/aws/secretsmanager/provider_support.go`
- Modify: `e2e/suites/provider/cases/aws/secretsmanager/provider_v2.go`
- Modify: `e2e/suites/provider/cases/aws/secretsmanager/clusterprovider_v2.go`
- Modify: `e2e/suites/provider/cases/aws/secretsmanager/push_v2.go`
- Modify: `e2e/suites/provider/cases/aws/parameterstore/provider_support_v2.go`
- Modify: `e2e/suites/provider/cases/aws/parameterstore/clusterprovider_v2.go`
- Modify: `e2e/suites/provider/cases/aws/parameterstore/push_v2.go`

## Task 1: Remove Provider And ClusterProvider From The Core API Surface

**Files:**
- Modify: `apis/externalsecrets/v1/provider_types.go`
- Modify: `apis/externalsecrets/v1/register.go`
- Modify: `apis/externalsecrets/v1/zz_generated.deepcopy.go`
- Modify: `apis/externalsecrets/v1/externalsecret_types.go`
- Modify: `apis/externalsecrets/v1beta1/externalsecret_types.go`
- Modify: `apis/externalsecrets/v1alpha1/pushsecret_types.go`
- Modify: `config/crds/bases/kustomization.yaml`
- Modify: `deploy/crds/bundle.yaml`

- [ ] **Step 1: Write failing tests that lock the new allowed kinds**

Update `apis/externalsecrets/v1alpha1/pushsecret_crd_test.go` and the matching `externalsecret` CRD tests to assert:

- `ProviderStore` and `ClusterProviderStore` remain allowed
- `Provider` and `ClusterProvider` are no longer listed in the enum

Run:

```bash
GOWORK=$(pwd)/go.work go test ./apis/externalsecrets/v1alpha1 ./apis/externalsecrets/v1 ./apis/externalsecrets/v1beta1 -run 'PushSecretCRD|ExternalSecret' -count=1
```

Expected: FAIL because old kinds are still present.

- [ ] **Step 2: Remove old kind constants and API registration**

Delete the `Provider` / `ClusterProvider` type definitions and related constants/registration from `apis/externalsecrets/v1/provider_types.go` and `apis/externalsecrets/v1/register.go`.

Keep only:

- `SecretStore`
- `ClusterSecretStore`
- `ProviderStore`
- `ClusterProviderStore`

Then regenerate API output with the project generator command already used on this branch.

- [ ] **Step 3: Update CRD enums and generated output**

Regenerate and update:

- `config/crds/bases/*`
- `deploy/crds/bundle.yaml`
- Helm CRD templates/snapshots

Remove all `Provider` / `ClusterProvider` enum entries from `ExternalSecret` and `PushSecret` store refs.

- [ ] **Step 4: Verify the API surface**

Run:

```bash
GOWORK=$(pwd)/go.work go test ./apis/externalsecrets/v1 ./apis/externalsecrets/v1alpha1 ./apis/externalsecrets/v1beta1 -count=1
make test.crds
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apis/externalsecrets/v1 apis/externalsecrets/v1alpha1 apis/externalsecrets/v1beta1 config/crds deploy/crds deploy/charts/external-secrets/templates/crds
git commit -m "refactor: remove provider api kinds from core surface"
```

## Task 2: Remove Old Controllers And Runtime Lookup Paths

**Files:**
- Modify: `cmd/controller/root.go`
- Delete: `pkg/controllers/provider/controller.go`
- Delete: `pkg/controllers/provider/controller_test.go`
- Delete: `pkg/controllers/clusterprovider/controller.go`
- Delete: `pkg/controllers/clusterprovider/controller_test.go`
- Modify: `runtime/clientmanager/manager.go`
- Modify: `runtime/clientmanager/manager_test.go`
- Modify: `runtime/clientmanager/providerstore.go`
- Modify: `runtime/clientmanager/providerstore_test.go`
- Modify: `pkg/controllers/providerstore/providerstore_controller_test.go`

- [ ] **Step 1: Write failing runtime/controller assertions**

Add or update tests to prove:

- `runtime/clientmanager` does not accept `Provider` / `ClusterProvider`
- `ProviderStore` / `ClusterProviderStore` remain accepted without the old gate
- `cmd/controller/root.go` no longer wires `provider` / `clusterprovider` reconcilers

Run:

```bash
GOWORK=$(pwd)/go.work go test ./runtime/clientmanager ./cmd/controller -count=1
```

Expected: FAIL because the old branches and gate are still present.

- [ ] **Step 2: Remove the provider controllers from startup**

Delete the provider controller packages and remove their setup from `cmd/controller/root.go`.

Also remove:

- `enable-v2-providers` flag wiring
- `clientmanager.SetV2ProvidersEnabled(...)`
- conditional metrics registration tied to the deleted flag

- [ ] **Step 3: Remove old clientmanager branches and gate**

In `runtime/clientmanager/manager.go`:

- delete `Provider` and `ClusterProvider` fetch helpers
- delete the old cache key types and error strings
- delete the `V2ProvidersEnabled` global gate
- keep `runtimeRef` compatibility and `ProviderStore` / `ClusterProviderStore` support

Update the tests to use only clean store kinds.

- [ ] **Step 4: Verify runtime/controller behavior**

Run:

```bash
GOWORK=$(pwd)/go.work go test ./runtime/clientmanager ./pkg/controllers/providerstore ./cmd/controller -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/controller runtime/clientmanager pkg/controllers/providerstore
git commit -m "refactor: drop provider runtime controllers and lookups"
```

## Task 3: Remove Old Consumer Kind Handling

**Files:**
- Modify: `pkg/controllers/externalsecret/externalsecret_controller.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2_test.go`

- [ ] **Step 1: Write failing consumer tests**

Update the existing `pushsecret` and `externalsecret` tests to remove `Provider` / `ClusterProvider` assumptions and assert only:

- `ProviderStore`
- `ClusterProviderStore`

Run:

```bash
GOWORK=$(pwd)/go.work go test ./pkg/controllers/externalsecret ./pkg/controllers/pushsecret -count=1
```

Expected: FAIL while old branches remain.

- [ ] **Step 2: Remove old kind handling**

Update:

- `shouldSkipClusterStore`
- `shouldSkipUnmanagedStore`
- clean-store resolution in pushsecret
- inferred clean-store kind helpers

so the clean path recognizes only `ProviderStore` and `ClusterProviderStore`.

- [ ] **Step 3: Verify controller packages**

Run:

```bash
ASSETS=$(./bin/setup-envtest use 1.33.x -p path --bin-dir ./bin)
ABS_ASSETS=$(cd "$ASSETS" && pwd)
KUBEBUILDER_ASSETS="$ABS_ASSETS" GOWORK=$(pwd)/go.work go test ./pkg/controllers/externalsecret ./pkg/controllers/pushsecret -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add pkg/controllers/externalsecret pkg/controllers/pushsecret
git commit -m "refactor: narrow clean store consumers to providerstore kinds"
```

## Task 4: Migrate The E2E Helper Layer To ProviderStore

**Files:**
- Modify: `e2e/framework/framework.go`
- Modify: `e2e/framework/v2/helpers.go`
- Modify: `e2e/suites/provider/cases/common/namespaced_provider.go`
- Modify: `e2e/suites/provider/cases/common/clusterprovider.go`
- Modify: `e2e/suites/provider/cases/common/operational_v2.go`
- Modify: `e2e/suites/provider/cases/common/push_secret.go`

- [ ] **Step 1: Write/update helper tests**

Update the existing helper/unit tests in the e2e package to assert:

- default runtime-backed kinds are `ProviderStore`
- helper-created objects are `ProviderStore` / `ClusterProviderStore`
- readiness waiters read `ProviderStoreStatus`

Run the narrow e2e helper tests already present in the tree.

Expected: FAIL until helpers are migrated.

- [ ] **Step 2: Replace provider-connection helpers with store helpers**

In `e2e/framework/v2/helpers.go`:

- create `ClusterProviderClass` if needed
- create `ProviderStore` with `runtimeRef` + `backendRef`
- create `ClusterProviderStore` with `runtimeRef` + `backendRef` + `conditions`
- wait on `ProviderStoreReady`

In `e2e/framework/framework.go`, switch default clean-store kinds from `Provider` to `ProviderStore`.

- [ ] **Step 3: Update the common harness layer**

Switch all common v2 harnesses to write:

- `SecretStoreRef.Kind = ProviderStore`
- `SecretStoreRef.Kind = ClusterProviderStore`

Update expected denial messages from `ClusterProvider` to `ClusterProviderStore`.

- [ ] **Step 4: Verify helper/common packages**

Run the narrow tests that cover:

- `e2e/framework/v2`
- `e2e/suites/provider/cases/common`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add e2e/framework e2e/suites/provider/cases/common
git commit -m "test: migrate v2 e2e helper layer to providerstore"
```

## Task 5: Migrate Provider-Specific V2 E2E Suites

**Files:**
- Modify: fake, kubernetes, and aws v2 suite files listed in the File Map

- [ ] **Step 1: Migrate fake v2 suites**

Replace provider helper calls and explicit kind references in:

- `e2e/suites/provider/cases/fake/provider_v2.go`
- `e2e/suites/provider/cases/fake/operational_v2.go`

Run the local fake-specific tests already in the repo.

- [ ] **Step 2: Migrate kubernetes v2 suites**

Replace provider helper calls and explicit kind references in:

- `provider_v2.go`
- `clusterprovider_v2.go`
- `push_v2.go`
- `operational_v2.go`
- `metrics_v2.go`
- `capabilities_v2.go`

Also rename descriptions and expected messages to use `ProviderStore` / `ClusterProviderStore`.

- [ ] **Step 3: Migrate aws v2 suites**

Replace provider helper calls and explicit kind references in:

- secretsmanager v2 files
- parameterstore v2 files

Keep the provider-owned backend CR setup unchanged; only the ESO-facing runtime/store object changes.

- [ ] **Step 4: Verify the provider-suite package tests**

Run the package tests that already exist for the touched provider-suite helpers, for example:

```bash
GOWORK=$(pwd)/go.work go test ./e2e/suites/provider/cases/common ./e2e/suites/provider/cases/fake ./e2e/suites/provider/cases/kubernetes ./e2e/suites/provider/cases/aws/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add e2e/suites/provider
git commit -m "test: migrate v2 provider suites to providerstore api"
```

## Task 6: Final Verification

**Files:**
- Review only; no new files expected.

- [ ] **Step 1: Run focused repo verification**

Run:

```bash
GOWORK=$(pwd)/go.work go test ./apis/externalsecrets/v1 ./apis/externalsecrets/v1alpha1 ./apis/externalsecrets/v1beta1 ./apis/externalsecrets/v2alpha1 -count=1
GOWORK=$(pwd)/go.work go test ./runtime/clientmanager ./cmd/controller -count=1
GOWORK=$(pwd)/go.work go test ./pkg/controllers/providerstore ./pkg/controllers/externalsecret ./pkg/controllers/pushsecret -count=1
make test.crds
make helm.test
```

Expected: PASS.

- [ ] **Step 2: Run full verification**

Run:

```bash
make test
```

Expected: PASS.

- [ ] **Step 3: Review requirements coverage**

Confirm:

- no `Provider` / `ClusterProvider` API surface remains
- no old provider controllers remain
- no old clientmanager lookup path remains
- clean runtime-backed refs only use `ProviderStore` / `ClusterProviderStore`
- v2 e2e helpers and suites use the new store API
- `--enable-v2-providers` is gone

- [ ] **Step 4: Commit final fixups**

```bash
git add apis cmd config deploy e2e pkg runtime
git commit -m "refactor: remove provider api and migrate v2 tests to providerstore"
```
