# GCP Provider V2 Design

**Goal:** onboard GCP Secret Manager to the provider v2 architecture, reuse the existing generator-based provider shell, and add v2 e2e coverage for static credentials, workload identity, and referenced service-account auth flows.

## Scope

In scope:

- Add a new v2 GCP provider API under `apis/provider/gcp/v2alpha1`.
- Add a new generated provider module under `providers/v2/gcp`.
- Reuse the existing `providers/v1/gcp/secretmanager` implementation for store behavior and auth execution.
- Add v2 e2e coverage for:
  - namespaced provider with static service-account JSON credentials
  - namespaced provider with workload identity
  - cluster provider flows through shared v2 cluster-provider cases
  - managed v2 flows for pod-identity and referenced service-account auth
- Wire the new provider into build, image load, and verification flows that already power `make check-diff` and v2 e2e.

Out of scope:

- A new native v2 GCP store implementation.
- Generator support for GCP.
- Managed test restructuring beyond what is required to run the GCP provider in v2 mode.
- Additional auth modes that are not already covered by the approved scope above.

## Current State

The repository currently has:

- legacy GCP Secret Manager provider code in `providers/v1/gcp/secretmanager`
- legacy GCP e2e cases in `e2e/suites/provider/cases/gcp`
- provider v2 API + binary support only for `aws`, `fake`, and `kubernetes`
- provider shell generation via `providers/v2/hack/generate-provider-main.go`
- existing generated-file enforcement through `make reviewable`, `make verify-providers`, and `make check-diff`

The main gap is that GCP has not been onboarded to the v2 API/config/binary model at all.

## Design

### 1. API Layer

Add `apis/provider/gcp/v2alpha1` with a single `SecretManager` kind.

The v2 spec should represent only the GCP Secret Manager fields needed for this onboarding:

- `projectID`
- `location`
- static secret-based auth
- workload identity auth

The shape should stay close to the legacy `esv1.GCPSMProvider` and `esv1.GCPSMAuth` structures so the v2 config mapper is mechanical and low-risk.

Referenced service-account auth for cluster-scoped usage does not need a separate auth model in the GCP provider config itself. It is expressed by the v2 `ClusterProvider` / `Provider` connection layer and the referenced namespace + authentication scope, while the underlying GCP auth remains workload identity.

### 2. Provider Runtime

Create `providers/v2/gcp` with:

- `provider.yaml`
- `config.go`
- `go.mod` / `go.sum`
- generated `main.go`
- generated `Dockerfile`

`provider.yaml` is the source of truth for the provider shell and must be used to generate `main.go` and `Dockerfile` via `make generate-providers`.

`config.go` provides the `GetSpecMapper` function expected by the generator-owned shell. It reads the v2 `SecretManager` resource and maps it into a legacy `v1.SecretStoreSpec` containing `Provider.GCPSM`.

The generated provider shell should register one store:

- `provider.external-secrets.io/v2alpha1`, kind `SecretManager`

Its target implementation should be the existing `providers/v1/gcp/secretmanager` provider constructor. This keeps secret access, push behavior, and token sourcing logic in one place.

### 3. Generation and Verification Discipline

The provider shell must be bootstrapped with the existing generation tooling, not created manually.

The workflow is:

1. add `provider.yaml`
2. add hand-written `config.go`
3. run `make generate-providers`
4. verify with `make verify-providers`

No additional custom “provider-v2-generate” CI step is needed. The repository already enforces generated-file freshness because:

- `make reviewable` includes `generate-providers` and `verify-providers`
- `make check-diff` runs `reviewable` and then fails on any resulting diff

The only required change is to ensure GCP participates in those existing paths by existing as a normal `providers/v2/<name>` provider with `provider.yaml`.

### 4. Build and Image Wiring

Root and e2e build flows must be extended so GCP is treated like the other v2 providers.

Required wiring:

- add a root docker build target for `provider-gcp`
- ensure v2 e2e builds the GCP provider image
- ensure v2 e2e loads `ghcr.io/external-secrets/provider-gcp:<VERSION>` into kind
- ensure any v2 addon/install helpers that explicitly enumerate provider images can include GCP when requested

The desired result is that `make test.e2e.v2` and focused GCP v2 runs execute against the generated provider shell and the real provider image, not a local special case.

### 5. E2E Design

Add GCP v2 e2e coverage by reusing the current shared v2 framework patterns rather than copying legacy SecretStore-based setup.

#### Namespaced Provider

Add a v2 namespaced-provider suite under `e2e/suites/provider/cases/gcp` that:

- uses the existing GCP backend helper to create/delete real GCP secrets
- creates a v2 `SecretManager` config object in the test namespace
- creates a namespaced `Provider` connection to the `gcp` provider service
- reuses shared common v2 cases where they fit, especially:
  - simple sync
  - refresh
  - dataFrom/find
  - status regression checks where applicable

Auth variants:

- static service-account JSON
- workload identity

#### Cluster Provider

Add a v2 cluster-provider suite that reuses the shared common cluster-provider cases already used by AWS and Kubernetes.

The GCP harness should:

- create the v2 `SecretManager` config in the configured namespace
- create the `ClusterProvider` connection
- wait for readiness
- hand a provider backend implementation to the shared runtime

This keeps cluster-provider behavior reusable and consistent across providers.

#### Managed Coverage

Add managed v2 coverage that mirrors the current legacy GCP managed intent:

- pod-identity flow where the provider pod itself runs with the GKE-linked service account
- referenced service-account flow where the provider uses workload identity with an explicitly referenced Kubernetes service account

The managed suites should keep the existing pattern of installing an isolated ESO instance when needed, but switch the provider path to v2 mode and use the GCP provider service instead of legacy SecretStore-only wiring.

### 6. Error Handling and Readiness Expectations

Provider config creation and connection setup should follow the existing v2 readiness contract:

- create config object
- create `Provider` or `ClusterProvider` connection
- wait for connection readiness before running shared cases

Mapping or auth configuration errors should fail during provider connection readiness, not deep inside the shared secret-sync assertions. This keeps failures local to setup and consistent with the current v2 provider model.

### 7. Testing Strategy

Unit coverage:

- v2 API type round-trip / deepcopy generation
- `providers/v2/gcp/config.go` mapping tests
- any e2e helper unit tests for config construction if they contain non-trivial auth translation

Generation/build verification:

- `make generate-providers`
- `make verify-providers`

Focused Go tests:

- new GCP e2e helper tests
- any updated framework/addon tests if new provider wiring touches shared code

End-to-end verification:

- targeted v2 GCP e2e run using available real GCP credentials
- `make check-diff` after generation and code changes

## File Plan

Expected new areas:

- `apis/provider/gcp/v2alpha1/*`
- `providers/v2/gcp/*`
- `e2e/suites/provider/cases/gcp/*_v2.go`
- `e2e/suites/provider/cases/gcp/*_test.go` where helper tests are needed

Expected shared updates:

- root `Makefile`
- `e2e/Makefile`
- any shared addon/install helpers that enumerate v2 providers
- any registration or scheme wiring needed so the new v2 API is recognized by tests and provider processes

## Risks

- GCP auth has more runtime-specific behavior than fake/kubernetes and more branchy setup than AWS static auth.
- Workload identity and managed flows can fail in ways that look like generic readiness issues unless the harness waits on the right objects.
- Generated provider files are easy to drift if hand-edited; this is why the generator must remain the source of truth.

## Recommendation

Use the thin-adapter approach:

- model the v2 GCP API explicitly
- map it into the legacy provider shape
- generate the provider shell from `provider.yaml`
- reuse the shared v2 e2e framework and common cases

This gives full provider-v2 onboarding with the smallest correctness risk and the least duplicate auth logic.
