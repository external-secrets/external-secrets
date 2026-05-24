---

description: "Task list template for feature implementation"
---

# Tasks: SAP Credential Store Provider

**Input**: Design documents from `/specs/001-sap-credstore-provider/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Tests are REQUIRED per the constitution (Principle II) and plan — all behavior
changes must be verified by automated tests at the appropriate level.

**Organization**: Tasks are grouped by user story to enable independent implementation
and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Exact file paths included in every task description

## Path Conventions

Repository root: `/Users/I576753/Documents/repos/external-secrets/`

New module root: `providers/v1/sapcredentialstore/`

---

## Phase 1: Setup (Module Initialization)

**Purpose**: Create the Go module skeleton and directory structure before any code is written.

- [x] T001 Create `providers/v1/sapcredentialstore/` directory structure: top-level, `api/`, and `fake/` subdirectories
- [x] T002 Create `providers/v1/sapcredentialstore/go.mod` with module path `github.com/external-secrets/external-secrets/providers/v1/sapcredentialstore`, Go version `1.26.3`, and `replace` directives for `apis` and `runtime` pointing at `../../../apis` and `../../../runtime`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: API types, HTTP client, and fake must be in place before any user story
can be implemented. All user story phases depend on this phase.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T003 Create `apis/externalsecrets/v1/secretstore_sapcredentialstore_types.go` with `SAPCredentialStoreProvider`, `SAPCSAuth`, `SAPCSOAuth2Auth`, and `SAPCSMTLSAuth` structs exactly as defined in `specs/001-sap-credstore-provider/data-model.md`, including all `+kubebuilder` markers
- [x] T004 Add `SAPCredentialStore *SAPCredentialStoreProvider \`json:"sapCredentialStore,omitempty"\`` field to the `SecretStoreProvider` struct in `apis/externalsecrets/v1/secretstore_types.go`, following the alphabetical or existing ordering of provider fields
- [ ] T005 Run `make generate` from the repository root to regenerate `apis/externalsecrets/v1/zz_generated.deepcopy.go` and `config/crds/bases/external-secrets.io_secretstores.yaml`; confirm the new types appear in both files without errors
- [x] T006 [P] Create `providers/v1/sapcredentialstore/api/types.go` with `Credential`, `CredentialMeta`, and `CredentialBody` structs as defined in `specs/001-sap-credstore-provider/data-model.md`
- [x] T007 Create `providers/v1/sapcredentialstore/api/client.go` with the `SAPCSClientInterface` interface (`GetCredential`, `ListCredentials`, `PutCredential`, `DeleteCredential`, `CredentialExists`) and the `httpClient` struct implementing it — OAuth2 transport via `golang.org/x/oauth2/clientcredentials`; mTLS via `crypto/tls` `tls.Config` with `InsecureSkipVerify: false`
- [x] T008 [P] Create `providers/v1/sapcredentialstore/fake/fake.go` with `FakeClient` struct implementing `SAPCSClientInterface` with configurable per-method functions (matching the doppler fake pattern in `providers/v1/doppler/fake/fake.go`)
- [x] T009 [P] Capture consistency constraints, documentation touchpoints, and performance validation approach from `specs/001-sap-credstore-provider/plan.md`
- [x] T010 [P] Verify security and compliance gate from `specs/001-sap-credstore-provider/plan.md`: confirm no credential logging paths exist, TLS configurations are valid (`InsecureSkipVerify` absent), no new RBAC rules are introduced, and new dependencies in `providers/v1/sapcredentialstore/go.mod` are CVE-clean

**Checkpoint**: Foundation ready — API types generated, HTTP client and fake implemented, security gate verified. User story phases can now begin.

---

## Phase 3: User Story 1 — Sync SAP Credentials into Kubernetes (Priority: P1) 🎯 MVP

**Goal**: Platform engineers can sync any SAP Credential Store credential (password, key,
or certificate) into a Kubernetes Secret using a `SecretStore` + `ExternalSecret` with
either OAuth2 or mTLS authentication.

**Independent Test**: Apply the example `SecretStore` and `ExternalSecret` from
`specs/001-sap-credstore-provider/quickstart.md`. Verify `ExternalSecret` status shows
`SecretSynced` and the Kubernetes Secret contains the correct credential value.

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation (T013–T017)**

- [x] T011 [P] [US1] Write integration tests in `providers/v1/sapcredentialstore/client_test.go` using `httptest.NewServer` to mock the SAP CS REST API: cover `GetSecret` for all three credential types (`password`, `key`, `certificate`), the `certificate/key` property sub-field, and OAuth2 path — confirm all tests FAIL before T014 is implemented
- [x] T012 [P] [US1] Write unit tests in `providers/v1/sapcredentialstore/provider_test.go` for `ValidateStore`: cover missing `serviceURL`, missing `namespace`, neither auth mode set, both auth modes set, missing `tokenURL` for OAuth2, missing ref fields for mTLS — confirm all tests FAIL before T013 is implemented

### Implementation for User Story 1

- [x] T013 [US1] Implement `Provider` struct in `providers/v1/sapcredentialstore/provider.go`: `Capabilities()` returning `SecretStoreReadWrite`, `ValidateStore()` enforcing all rules from `data-model.md` validation table, `NewClient()` resolving auth secrets via `resolvers.SecretKeyRef`, building `*httpClient` with OAuth2 transport or mTLS `tls.Config`, and `Close()` stub — add compile-time assertion `var _ esv1.Provider = &Provider{}`
- [x] T014 [US1] Implement `Client.GetSecret()` in `providers/v1/sapcredentialstore/client.go`: parse `ref.Property` to determine credential type (default `password`), call `api.GetCredential`, route to correct field (`value` or `key` for `certificate/key`), wrap 404 response as `esv1.NoSecretError`, emit `metrics.ObserveAPICall("sapCredentialStore", "GetCredential", err)` — add compile-time assertion `var _ esv1.SecretsClient = &Client{}`
- [x] T015 [US1] Implement `Client.GetSecretMap()` in `providers/v1/sapcredentialstore/client.go`: call `api.GetCredential`, return all non-metadata fields as `map[string][]byte` (`name`, `value`, `username` if non-empty, `key` if non-empty)
- [x] T016 [US1] Implement `Client.Validate()` (call `api.GetCredential` on a known path or return `ValidationResultUnknown`) and `Client.Close()` (no-op) in `providers/v1/sapcredentialstore/client.go`
- [x] T017 [US1] Create `pkg/register/sapcredentialstore.go` with build tag `//go:build sapcredentialstore || all_providers` and `init()` calling `esv1.Register(sapcredstore.NewProvider(), sapcredstore.ProviderSpec(), sapcredstore.MaintenanceStatus())`
- [x] T018 [P] [US1] Create `docs/provider/sap-credential-store.md` with the provider guide: `SecretStore` setup (OAuth2 and mTLS variants), `ExternalSecret` examples for all three credential types and `certificate/key` sub-field, troubleshooting table from `specs/001-sap-credstore-provider/quickstart.md`

**Checkpoint**: US1 fully functional and independently testable. Apply `SecretStore` + `ExternalSecret` and verify `SecretSynced` status.

---

## Phase 4: User Story 2 — Push Kubernetes Secrets to SAP Credential Store (Priority: P2)

**Goal**: Platform engineers can push a Kubernetes Secret value into SAP Credential Store
via `PushSecret`, with the credential type determined by `PushSecretData.GetProperty()`.

**Independent Test**: Create a Kubernetes Secret, apply the `PushSecret` example from
`specs/001-sap-credstore-provider/quickstart.md`. Verify the credential appears in SAP
Credential Store with the correct type and value.

### Tests for User Story 2 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation (T020–T022)**

- [x] T019 [P] [US2] Write integration tests in `providers/v1/sapcredentialstore/client_test.go` using `httptest.NewServer`: cover `PushSecret` creation, `PushSecret` update, `DeleteSecret`, `SecretExists` returning true and false, and credential type routing (property → credType) — confirm all tests FAIL before T020 is implemented

### Implementation for User Story 2

- [x] T020 [US2] Implement `Client.PushSecret()` in `providers/v1/sapcredentialstore/client.go`: extract credential name from `data.GetRemoteKey()`, type from `data.GetProperty()` (default `password`), value from the source Kubernetes Secret, call `api.PutCredential`, emit metrics
- [x] T021 [US2] Implement `Client.DeleteSecret()` in `providers/v1/sapcredentialstore/client.go`: call `api.DeleteCredential` with the type and name from `remoteRef`; respect ESO `deletionPolicy` convention (only delete when called by the controller)
- [x] T022 [US2] Implement `Client.SecretExists()` in `providers/v1/sapcredentialstore/client.go`: call `api.CredentialExists` (HEAD request); return `(false, nil)` on 404, `(true, nil)` on 200, and `(false, err)` on other errors
- [x] T023 [P] [US2] Update `docs/provider/sap-credential-store.md` to add `PushSecret` usage section, credential type mapping table, and deletion policy notes

**Checkpoint**: US1 and US2 both independently functional and testable.

---

## Phase 5: User Story 3 — Bulk Sync All Credentials from a Namespace (Priority: P3)

**Goal**: Platform engineers can use `dataFrom` on an `ExternalSecret` to sync all
credentials in the SAP Credential Store namespace into a single Kubernetes Secret,
with keys formatted as `<type>/<name>`.

**Independent Test**: Create multiple credentials of different types, apply an
`ExternalSecret` with `dataFrom.find`, verify the resulting Kubernetes Secret contains
all credentials keyed as `password/name`, `key/name`, `certificate/name`.

### Tests for User Story 3 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation (T025)**

- [x] T024 [P] [US3] Write integration tests in `providers/v1/sapcredentialstore/client_test.go` using `httptest.NewServer`: cover `GetAllSecrets` with mixed credential types, empty namespace (expect empty map, no error), and correct `<type>/<name>` key formatting — confirm tests FAIL before T025 is implemented

### Implementation for User Story 3

- [x] T025 [US3] Implement `Client.GetAllSecrets()` in `providers/v1/sapcredentialstore/client.go`: iterate over `credTypePassword`, `credTypeKey`, `credTypeCertificate`; for each type call `api.ListCredentials` then `api.GetCredential` for each item; return combined `map[string][]byte` keyed as `"<type>/<name>"` with the credential's primary `value`; emit metrics per `ListCredentials` call
- [x] T026 [P] [US3] Update `docs/provider/sap-credential-store.md` to add `dataFrom` bulk-sync section with example manifest and explanation of `<type>/<name>` key format

**Checkpoint**: All three user stories independently functional and testable.

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Validate correctness across all stories, consistency with repo conventions, and security compliance.

- [ ] T027 [P] Run `go test ./...` from `providers/v1/sapcredentialstore/` and confirm zero test failures
- [x] T028 Cross-story consistency review in `providers/v1/sapcredentialstore/`: verify all `metrics.ObserveAPICall` calls use `"sapCredentialStore"` as provider name, all error messages follow `fmt.Errorf("context: %w", err)` format, and all status condition Reason strings match patterns used in `providers/v1/infisical/` and `providers/v1/doppler/`
- [x] T029 Security hardening and compliance audit: grep `providers/v1/sapcredentialstore/` for any logging of `Credential.Value` or auth secret values; confirm `InsecureSkipVerify` is absent from all `tls.Config` uses; confirm `pkg/register/sapcredentialstore.go` introduces no new RBAC rules
- [ ] T030 [P] Run `make generate` from repository root to confirm `apis/externalsecrets/v1/zz_generated.deepcopy.go` and CRD YAML reflect the final API types with no uncommitted diffs
- [ ] T031 Run quickstart.md validation steps (`specs/001-sap-credstore-provider/quickstart.md`) against the implementation: apply all example manifests and verify each success criterion from spec.md SC-001 through SC-005 is met

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user story phases
- **User Stories (Phases 3–5)**: All depend on Foundational completion
  - US1 (Phase 3): No dependency on US2 or US3 — can start once Foundational is done
  - US2 (Phase 4): Depends on Foundational; independent of US1 (separate client methods)
  - US3 (Phase 5): Depends on Foundational; builds on `ListCredentials` from api/client.go
- **Polish (Phase N)**: Depends on all desired user story phases being complete

### User Story Dependencies

- **US1 (P1)**: Can start immediately after Foundational — no dependency on US2 or US3
- **US2 (P2)**: Can start immediately after Foundational — adds to `client.go` without touching US1 methods
- **US3 (P3)**: Can start immediately after Foundational — adds a single method to `client.go`

### Within Each User Story

- Tests (T011/T012, T019, T024) MUST be written and confirmed to FAIL before implementation tasks run
- `api/client.go` (T007) before `provider.go` (T013) — `NewClient` depends on `httpClient`
- `fake/fake.go` (T008) before unit tests (T012) — `ValidateStore` tests use the fake
- `provider.go` (T013) before `pkg/register/` (T017) — registration imports `NewProvider`
- Core method implementation before documentation

### Parallel Opportunities

- T001 and T002 can run in parallel (different files)
- T006 and T008 can run in parallel after T007 (types needed by fake)
- T009 and T010 can run in parallel (documentation review tasks)
- T011 and T012 can run in parallel (different test files)
- T014, T015, T016 can run in parallel after T013 (different methods in client.go; coordinate on file edits)
- T018 (docs) can run in parallel with T013–T016 (implementation)
- T019 can run in parallel with T020–T022 after T011 is complete (tests inform impl but are separate files)
- T023 (docs), T026 (docs) can run in parallel with their respective implementations

---

## Parallel Example: User Story 1

```bash
# Write tests together (T011 + T012 in parallel — different files):
Task T011: "Integration tests in providers/v1/sapcredentialstore/client_test.go"
Task T012: "Unit tests in providers/v1/sapcredentialstore/provider_test.go"

# Then implement (T013 blocks T014–T017 which can run in parallel after T013):
Task T013: "Implement Provider struct in provider.go"
# Once T013 is done:
Task T014: "Implement Client.GetSecret in client.go"
Task T015: "Implement Client.GetSecretMap in client.go"
Task T016: "Implement Client.Validate and Close in client.go"
Task T018: "Create docs/provider/sap-credential-store.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T002)
2. Complete Phase 2: Foundational (T003–T010) — CRITICAL
3. Write US1 tests (T011–T012) — confirm they FAIL
4. Complete Phase 3: US1 (T013–T018)
5. **STOP and VALIDATE**: `go test ./...` passes; apply example manifests; ExternalSecret shows `SecretSynced`
6. Open draft PR to gather early feedback before proceeding to US2/US3

### Incremental Delivery

1. Setup + Foundational → module skeleton, API types, HTTP client ready
2. US1 → Read path functional → submit MVP PR
3. US2 → Write path functional → update PR
4. US3 → Bulk sync functional → update PR
5. Polish → all tests green, docs complete, security audited → ready for review

### Task Count Summary

| Phase | Tasks | Parallel Opportunities |
|-------|-------|----------------------|
| Setup | 2 | T001, T002 |
| Foundational | 8 | T006, T008, T009, T010 |
| US1 (P1) | 8 | T011, T012, T018 |
| US2 (P2) | 5 | T019, T023 |
| US3 (P3) | 3 | T024, T026 |
| Polish | 5 | T027, T030 |
| **Total** | **31** | |

---

## Notes

- `[P]` tasks operate on different files; coordinate if multiple agents touch `client.go` simultaneously
- Tests must FAIL before their corresponding implementation tasks — never skip this verification
- `client.go` grows across US1, US2, and US3 phases; each story adds distinct methods with no overlap
- Run `make generate` after any change to `apis/` files (T005, T030)
- The `fake/fake.go` follows the pattern in `providers/v1/doppler/fake/fake.go` exactly
- `pkg/register/sapcredentialstore.go` is the only file touching the main module's build graph — keep it minimal
