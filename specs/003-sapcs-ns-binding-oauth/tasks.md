# Tasks: SAP CS — Namespace, BTP Binding, OAuth Caching & E2E Tests

**Input**: Design documents from `/specs/003-sapcs-ns-binding-oauth/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓, quickstart.md ✓

**Tests**: Test tasks are included for every behavior change per constitution Principle II.
Unit tests use `httptest.Server` + table-driven patterns matching the existing test files.
E2E tests use Ginkgo v2 / Gomega matching the existing provider suite structure.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)

---

## Phase 1: Setup (Baseline Verification)

**Purpose**: Confirm the existing test baseline before any changes.

- [X] T001 Run `go test ./providers/v1/sapcredentialstore/...` and record all passing tests as the regression baseline

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: CRD type additions and code-generation that both US1 and US2 depend on, plus cross-cutting gates.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 [P] Add `Namespace string \`json:"namespace,omitempty"\`` field with `// +optional` marker to `ExternalSecretDataRemoteRef` in `apis/externalsecrets/v1/externalsecret_types.go`; add Go doc comment: "Namespace overrides the provider-level namespace for this specific secret reference"
- [X] T003 [P] Add `SAPCSServiceBindingRef` struct and `ServiceBindingSecretRef *SAPCSServiceBindingRef \`json:"serviceBindingSecretRef,omitempty"\`` field to `SAPCredentialStoreProvider` in `apis/externalsecrets/v1/secretstore_sapcredentialstore_types.go`; include `Name`, `Namespace`, `CredentialsKey` fields with `// +kubebuilder:default=credentials` on `CredentialsKey`
- [X] T004 Run `make generate` to regenerate CRD YAML in `config/crds/` and commit the updated generated files
- [X] T005 [P] Document consistency constraints: grep existing `providers/v1/sapcredentialstore/provider.go` and `client.go` for error message prefix pattern, status reason strings (`esv1.ReasonInvalid`, `esv1.ReasonValid`), and log field conventions; record findings as a comment in `providers/v1/sapcredentialstore/provider.go` header or a brief inline note for use in subsequent tasks
- [X] T006 [P] Verify security compliance gate: confirm no credential value logging exists in `providers/v1/sapcredentialstore/`; confirm no `InsecureSkipVerify` in `providers/v1/sapcredentialstore/api/client.go`; run `go mod tidy` and check `go.sum` for no new direct dependencies; record result

**Checkpoint**: CRD types extended, code generated, gates documented — user story work can begin.

---

## Phase 3: User Story 1 — Per-ExternalSecret Namespace Override (Priority: P1) 🎯 MVP

**Goal**: Allow each `ExternalSecret` to specify a SAP CS namespace that overrides the store-level default, without creating additional stores.

**Independent Test**: Deploy one `ClusterSecretStore` with `namespace: ns-default` and two `ExternalSecret` resources — one with `remoteRef.namespace: ns-a`, one with no override. Verify that the first resolves from `ns-a` and the second from `ns-default`.

### Tests for User Story 1 ⚠️

> **Write these tests FIRST, ensure they FAIL before implementation (T008–T009)**

- [X] T007 [P] [US1] Add table-driven unit tests in `providers/v1/sapcredentialstore/client_test.go` for `GetSecret` covering: (a) `remoteRef.namespace` non-empty → uses override namespace in API call, (b) `remoteRef.namespace` empty → uses store-level namespace, (c) `remoteRef.namespace` whitespace-only → falls back to store namespace; use existing `httptest.Server` harness

### Implementation for User Story 1

- [X] T008 [US1] Implement `effectiveNamespace(storeNS, remoteRefNS string) string` helper in `providers/v1/sapcredentialstore/client.go`: returns `remoteRefNS` if `strings.TrimSpace(remoteRefNS) != ""`, else `storeNS`
- [X] T009 [US1] Wire `effectiveNamespace` into `GetSecret` in `providers/v1/sapcredentialstore/client.go`: replace the hard-coded store namespace argument in the `api.Client` call with `effectiveNamespace(c.storeNamespace, ref.Namespace)`
- [X] T010 [US1] Wire `effectiveNamespace` into `GetAllSecrets` in `providers/v1/sapcredentialstore/client.go` for the list call namespace argument
- [X] T011 [US1] Confirm all existing tests plus T007 tests now pass: run `go test ./providers/v1/sapcredentialstore/...`

**Checkpoint**: US1 fully functional — namespace override works independently of all other stories.

---

## Phase 4: User Story 2 — BTP Service Binding Secret (Priority: P1)

**Goal**: Allow a `ClusterSecretStore` to be configured using only a BTP Service Binding Kubernetes Secret reference, deriving all connection details from it automatically.

**Independent Test**: Create a `ClusterSecretStore` with only `serviceBindingSecretRef` (no `serviceURL`, no `auth`); confirm `ValidateStore` succeeds; sync an `ExternalSecret`; confirm the correct credential is returned.

### Tests for User Story 2 ⚠️

> **Write these tests FIRST, ensure they FAIL before implementation (T014–T016)**

- [X] T012 [P] [US2] Add table-driven unit tests for `ValidateStore` in `providers/v1/sapcredentialstore/provider_test.go` covering: binding ref with all required fields → valid; binding ref with missing `clientsecret` → invalid with message listing missing field; binding ref pointing to non-existent secret → invalid with "not found" message; both binding ref and inline auth set → valid with warning; neither binding ref nor inline auth → invalid
- [X] T013 [P] [US2] Add unit test for `NewClient` in `providers/v1/sapcredentialstore/provider_test.go` covering: binding ref set → credentials resolved from Secret JSON; binding secret updated between reconciles → new credentials used on next `NewClient` call

### Implementation for User Story 2

- [X] T014 [US2] Implement `resolveBindingSecret(ctx, kube, ref SAPCSServiceBindingRef) (clientID, clientSecret, tokenURL, serviceURL string, err error)` in `providers/v1/sapcredentialstore/provider.go`: read the referenced Kubernetes Secret, parse the `credentialsKey` (default `credentials`) JSON value, map `clientid`, `clientsecret`, `tokenurl`, `url` fields; return a clear error listing any missing required keys
- [X] T015 [US2] Update `ValidateStore` in `providers/v1/sapcredentialstore/provider.go`: when `serviceBindingSecretRef` is non-nil, call `resolveBindingSecret` to validate existence and completeness; when both binding ref and inline auth are present, emit a warning log line (do NOT log secret values) and proceed with binding ref values; when neither is set, return existing error
- [X] T016 [US2] Update `NewClient` in `providers/v1/sapcredentialstore/provider.go`: when `ServiceBindingSecretRef` is non-nil, call `resolveBindingSecret` to get credentials and override `s.ServiceURL`, `tokenURL`, `clientID`, `clientSecret` before constructing the OAuth2 client
- [X] T017 [US2] Add `SecretStore` error-condition path: when `resolveBindingSecret` fails in `NewClient` (secret missing or fields absent), return a `NoSecretError` / provider error that surfaces as `SecretStoreReady=False` with the error message in `providers/v1/sapcredentialstore/provider.go`
- [X] T018 [US2] Run `go test ./providers/v1/sapcredentialstore/...` and confirm T012–T013 tests pass, existing tests unaffected

**Checkpoint**: US2 fully functional — BTP binding secret resolution works independently of the cache (US3) and e2e (US4).

---

## Phase 5: User Story 3 — Robust OAuth Token Caching (Priority: P2)

**Goal**: Eliminate per-reconcile OAuth token fetches by maintaining a process-level token cache that proactively refreshes tokens before expiry and is safe for concurrent use.

**Independent Test**: Run `go test -count=1 -run TestTokenCache ./providers/v1/sapcredentialstore/...` — all cache hit/miss/concurrent/refresh tests pass. Additionally, a benchmark confirms that 100 concurrent `GetSecret` calls with a warm cache result in zero token-endpoint calls.

### Tests for User Story 3 ⚠️

> **Write these tests FIRST, ensure they FAIL before implementation (T021–T023)**

- [X] T019 [P] [US3] Create `providers/v1/sapcredentialstore/tokencache_test.go` and write failing unit tests for: cache miss on first call creates a new `oauth2.ReuseTokenSource`; cache hit on subsequent call with same credential identity returns the same source (not a new one); two goroutines with the same identity do not race (run with `-race`)
- [X] T020 [P] [US3] Add failing unit test in `providers/v1/sapcredentialstore/tokencache_test.go` for stale-token fallback: inject a `TokenSource` that returns `token.Expiry = time.Now().Add(30s)` (within refresh window); verify `GetOrCreate` returns the wrapped `ReuseTokenSource`; inject a failing `TokenSource` and a still-valid token; verify `ReuseTokenSource` continues serving the old token

### Implementation for User Story 3

- [X] T021 [US3] Create `providers/v1/sapcredentialstore/tokencache.go`: declare package-level `var tokenCacheMu sync.Map`; define `cacheKey(tokenURL, clientID string) string` using `fmt.Sprintf("%x", sha256.Sum256(...))` (import `crypto/sha256`); no external dependencies
- [X] T022 [US3] Implement `GetOrCreateTokenSource(tokenURL, clientID, clientSecret string) oauth2.TokenSource` in `providers/v1/sapcredentialstore/tokencache.go`: check `tokenCacheMu.Load(key)`; on miss, create `clientcredentials.Config{...}.TokenSource(context.Background())`, wrap with `oauth2.ReuseTokenSource(nil, ts)`, store with `tokenCacheMu.LoadOrStore(key, rts)` to handle concurrent miss; return the stored value
- [X] T023 [US3] Update `NewClient` in `providers/v1/sapcredentialstore/provider.go`: replace `cfg.Client(ctx).Transport` with `oauth2.NewClient(ctx, GetOrCreateTokenSource(tokenURL, clientID, clientSecret))` transport; remove per-call `clientcredentials.Config.Client(ctx)` instantiation
- [X] T024 [US3] Add benchmark `BenchmarkGetSecretConcurrent` in `providers/v1/sapcredentialstore/client_test.go`: 100 parallel goroutines each calling `GetSecret` against an `httptest.Server`; assert via a counter that the mock token endpoint is called ≤ 2 times total during the benchmark run
- [X] T025 [US3] Run `go test -race ./providers/v1/sapcredentialstore/...` and confirm T019–T020 tests pass and no race conditions detected

**Checkpoint**: US3 fully functional — token cache tested in isolation; concurrent safety verified with `-race`.

---

## Phase 6: User Story 4 — End-to-End Integration Tests (Priority: P2)

**Goal**: Provide a Ginkgo e2e test suite that exercises the full path from `ExternalSecret` creation to Kubernetes `Secret` population against a live SAP Credential Store instance, gated by build tag and env vars.

**Independent Test**: `SAPCS_SERVICE_URL=... SAPCS_TOKEN_URL=... SAPCS_CLIENT_ID=... SAPCS_CLIENT_SECRET=... SAPCS_NAMESPACE=... go test ./e2e/suites/provider/... -tags e2e_sapcredentialstore -v -timeout 10m` — all 5 test cases pass against a live instance.

### Implementation for User Story 4

- [X] T026 [US4] Create `e2e/suites/provider/cases/sapcredentialstore/setup.go` (build tag `e2e_sapcredentialstore`): `SAPCSTestConfig` struct; `NewConfigFromEnv()` that reads `SAPCS_SERVICE_URL`, `SAPCS_TOKEN_URL`, `SAPCS_CLIENT_ID`, `SAPCS_CLIENT_SECRET`, `SAPCS_NAMESPACE` and skips the suite (via `Skip`) if any are missing; fixture helpers `CreateBindingSecret`, `CreateClusterSecretStore`, `CreateExternalSecret`
- [X] T027 [US4] Create `e2e/suites/provider/cases/sapcredentialstore/sapcredentialstore.go` (build tag `e2e_sapcredentialstore`): Ginkgo `Describe("SAP Credential Store", ...)` suite entrypoint following the pattern in `e2e/suites/provider/cases/`; `BeforeSuite` calls `NewConfigFromEnv()`
- [X] T028 [P] [US4] Implement `BasicSecretSync` `It` block in `sapcredentialstore.go`: create a `ClusterSecretStore` (inline auth) + `ExternalSecret`; wait for `SecretSynced`; assert the Kubernetes Secret value matches the pre-seeded CS credential
- [X] T029 [P] [US4] Implement `NamespaceOverride` `It` block in `sapcredentialstore.go`: create one `ClusterSecretStore` with `namespace: A`; create two `ExternalSecret` resources — one with `remoteRef.namespace: B`, one without; assert each resolves from its respective namespace
- [X] T030 [P] [US4] Implement `BTPBindingSecret` `It` block in `sapcredentialstore.go`: call `CreateBindingSecret` to create a Secret with valid BTP binding JSON; create a `ClusterSecretStore` with only `serviceBindingSecretRef`; sync an `ExternalSecret`; assert `SecretSynced` and correct value
- [X] T031 [P] [US4] Implement `MissingKey` `It` block in `sapcredentialstore.go`: create an `ExternalSecret` referencing a non-existent credential name; assert the `ExternalSecret` transitions to `SecretSyncedError` with a message containing the key name
- [X] T032 [P] [US4] Implement `ConnectionFailure` `It` block in `sapcredentialstore.go`: create a `ClusterSecretStore` with `serviceURL` set to an unreachable address; assert the store transitions to `SecretStoreReady=False` with a connection-error message within the reconcile timeout
- [X] T033 [US4] Add `e2e-sapcredentialstore` target to `Makefile`: `go test ./e2e/suites/provider/... -tags e2e_sapcredentialstore -v -timeout 10m`; document required env vars in a comment above the target

**Checkpoint**: US4 fully functional — e2e suite skips gracefully when env vars absent; passes against a live instance.

---

## Phase 7: User Story 5 — Comprehensive Documentation (Priority: P3)

**Goal**: Give any user the information needed to understand authentication flows, namespace precedence, BTP binding structure, and how to run the integration tests — in the provider documentation.

**Independent Test**: A reviewer unfamiliar with the provider can follow the updated docs to configure a `ClusterSecretStore` using a BTP binding secret, configure a namespace override, and run the e2e suite — using only the documentation.

### Implementation for User Story 5

- [X] T034 [P] [US5] Update `docs/provider/sap-credentials-store.md` — add **Authentication Flows** section documenting: OAuth2 client credentials grant, token URL discovery, process-level token cache, proactive refresh, and stale-token fallback behavior; include the ASCII flow diagram from `quickstart.md`
- [X] T035 [P] [US5] Update `docs/provider/sap-credentials-store.md` — add **BTP Service Binding Secret** section documenting: required JSON keys (`clientid`, `clientsecret`, `url`, `tokenurl`), `credentialsKey` default and override, example BTP binding Secret YAML, precedence over inline auth, and warning when both are set
- [X] T036 [P] [US5] Update `docs/provider/sap-credentials-store.md` — add **Namespace Configuration** section documenting: store-level `namespace` field (required), per-secret `remoteRef.namespace` override, precedence rule, example `ExternalSecret` YAML with and without override
- [X] T037 [US5] Update `docs/provider/sap-credentials-store.md` — add **Running Integration Tests** section documenting: required environment variables (with descriptions of where to find each in a BTP binding), `make e2e-sapcredentialstore` command, what each of the 5 test cases validates, and how to skip the suite when credentials are unavailable
- [X] T038 [US5] Add or update Go doc comments on `SAPCSServiceBindingRef`, `ServiceBindingSecretRef` field of `SAPCredentialStoreProvider`, and `Namespace` field of `ExternalSecretDataRemoteRef` in `apis/externalsecrets/v1/secretstore_sapcredentialstore_types.go` and `externalsecret_types.go`

**Checkpoint**: US5 complete — documentation covers all four required topics; doc comments appear in `kubectl explain`.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final consistency pass, generation, security audit, and quickstart validation.

- [X] T039 Cross-story consistency review: grep `providers/v1/sapcredentialstore/` for log messages, error strings, and status reason strings; confirm all follow the existing `"sapCredentialStore: ..."` prefix and use `esv1.ReasonInvalid` / `esv1.ReasonValid` reason constants; fix any deviations
- [X] T040 Run final `make generate` pass and confirm regenerated CRD YAML in `config/crds/` is committed and diff-clean; verify `ExternalSecretDataRemoteRef` and `SAPCredentialStoreProvider` show new fields in the generated CRD
- [X] T041 Run `go test -race -count=1 ./providers/v1/sapcredentialstore/...` — all tests pass, no race conditions
- [X] T042 [P] Security hardening audit: grep all changed files (`git diff --name-only main...HEAD`) for patterns that could log credential values (`clientSecret`, `clientsecret`, `password`, `secret`); confirm no `InsecureSkipVerify: true`; run `govulncheck ./providers/v1/sapcredentialstore/...` and confirm no CVSS ≥ 7.0 findings; document outcome
- [ ] T043 Run quickstart.md end-to-end validation on a BTP/Kyma cluster: follow Option A (BTP binding), configure namespace override, verify sync succeeds within 30 minutes without external help

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Foundational; no dependency on US2–US5
- **US2 (Phase 4)**: Depends on Foundational; no dependency on US1 (but benefits from baseline test passing)
- **US3 (Phase 5)**: Depends on Foundational; builds on `NewClient` touched by US2 — **start US3 after US2 completes**
- **US4 (Phase 6)**: Depends on US1, US2, US3 all complete (tests exercise all three features)
- **US5 (Phase 7)**: Depends on US1–US3 implementation complete; can overlap with US4
- **Polish (Phase 8)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (P1)**: Independent — can start after Foundational
- **US2 (P1)**: Independent — can start after Foundational; parallel with US1
- **US3 (P2)**: Depends on US2 (`NewClient` refactor must be in place first)
- **US4 (P2)**: Depends on US1 + US2 + US3 (e2e covers all three)
- **US5 (P3)**: Depends on US1 + US2 + US3 implementation (docs must reflect final API)

### Within Each User Story

- Tests written first and confirmed failing before implementation
- Implementation tasks in dependency order within the story
- `go test` run after each story to confirm no regression
- Documentation task last within each story (or in US5 phase)

### Parallel Opportunities

- T002 + T003: Different files — run in parallel (both Foundational CRD additions)
- T005 + T006: Read-only gates — run in parallel
- US1 + US2: Different files entirely — run in parallel once Foundational is done
- T012 + T013: Different test functions, same file — can be written in parallel
- T019 + T020: Different test functions, same new file — can be written in parallel
- T028–T032: Different `It` blocks in same file — write in parallel
- T034 + T035 + T036: Different sections of same doc file — write in parallel

---

## Parallel Example: User Story 2

```bash
# Write failing tests first (parallel — different test functions):
Task T012: ValidateStore tests in providers/v1/sapcredentialstore/provider_test.go
Task T013: NewClient tests in providers/v1/sapcredentialstore/provider_test.go

# Then implement (sequential — all touch provider.go):
Task T014: resolveBindingSecret helper
Task T015: ValidateStore binding ref logic
Task T016: NewClient binding ref wiring
Task T017: Error condition path
```

## Parallel Example: User Story 4 (E2E)

```bash
# After setup.go and suite entrypoint exist (T026, T027):
Task T028: BasicSecretSync It block
Task T029: NamespaceOverride It block
Task T030: BTPBindingSecret It block
Task T031: MissingKey It block
Task T032: ConnectionFailure It block
# All five can be written concurrently — independent Ginkgo It() blocks
```

---

## Implementation Strategy

### MVP First (US1 + US2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRD types + generate)
3. Complete Phase 3: US1 — Namespace override
4. Complete Phase 4: US2 — BTP Binding secret
5. **STOP and VALIDATE**: Both P1 stories independently testable
6. Deploy/demo: namespace override and BTP binding working end-to-end

### Incremental Delivery

1. Setup + Foundational → CRD types ready
2. US1 → namespace override ships → test independently
3. US2 → BTP binding ships → test independently
4. US3 → token caching ships → verified with `-race` and benchmark
5. US4 → e2e suite ships → run against live instance
6. US5 + Polish → documentation and audit complete → ready for PR merge

### Parallel Team Strategy

With two developers after Foundational is done:
- Developer A: US1 (client.go namespace override)
- Developer B: US2 (provider.go BTP binding)
- Both finish → Developer A starts US4 setup; Developer B starts US3 token cache
- US3 complete → merge into US4 branch for full e2e coverage
- US5 documentation can overlap with US3/US4

---

## Notes

- `[P]` tasks touch different files or have no file-level dependencies
- `[Story]` label maps each task to its user story for traceability
- Run `go test -race` after US3 to confirm `sync.Map` cache is race-free
- `make generate` must be re-run any time CRD types change (after T002/T003 and as final T040)
- Integration tests (US4) require live SAP CS credentials; skip gracefully via `Skip()` when env vars absent
- Commit after each user story phase or logical group to keep diffs reviewable
- Constitution gate (T006) is a prerequisite for all user story work — do not skip
