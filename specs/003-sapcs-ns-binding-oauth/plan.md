# Implementation Plan: SAP CS — Namespace, BTP Binding, OAuth Caching & E2E Tests

**Branch**: `003-sapcs-ns-binding-oauth` | **Date**: 2026-07-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-sapcs-ns-binding-oauth/spec.md`

## Summary

Extend the SAP Credential Store provider with four related capabilities: (1) a per-`ExternalSecret`
namespace override via a new `remoteRef.namespace` field, (2) a `serviceBindingSecretRef` on the
`SAPCredentialStoreProvider` that resolves all connection details from a BTP-injected Kubernetes
Secret, (3) a process-level OAuth token cache using `oauth2.ReuseTokenSource` held in a
`sync.Map` to eliminate per-reconcile token fetches, and (4) an end-to-end Ginkgo/Gomega test
suite and updated provider documentation.

## Technical Context

**Language/Version**: Go 1.26.3
**Primary Dependencies**: `golang.org/x/oauth2` (ReuseTokenSource, clientcredentials), `k8s.io/client-go` v0.35.0, `sigs.k8s.io/controller-runtime` v0.23.1, `github.com/go-jose/go-jose/v4` (existing JWE), `github.com/onsi/ginkgo/v2` + `gomega` (e2e)
**Storage**: N/A (in-memory token cache; no persistent storage added)
**Testing**: `go test` (unit), Ginkgo v2 / Gomega (e2e), `httptest.Server` (existing integration harness)
**Target Platform**: Linux (Kubernetes controller pod); runs as a controller-manager
**Project Type**: Kubernetes operator provider plugin (library-style, loaded via build tags)
**Performance Goals**: ≤3 OAuth token-endpoint calls per 5-minute window under 100 concurrent reconciles with token lifetime ≥5 min (SC-003); e2e suite completes in <5 min (SC-005)
**Constraints**: No new direct Go module dependencies beyond what is already in go.mod; `sync.Map` + `singleflight` are stdlib; `oauth2.ReuseTokenSource` is already a transitive dep
**Scale/Scope**: Single provider package (`providers/v1/sapcredentialstore/`), one CRD types file, one shared `ExternalSecretDataRemoteRef` field addition, one e2e suite directory

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Code Quality (Principle I)

- **Impacted packages**: `providers/v1/sapcredentialstore/` (provider.go, client.go, api/client.go), `apis/externalsecrets/v1/secretstore_sapcredentialstore_types.go`, `apis/externalsecrets/v1/externalsecret_types.go`, `config/crds/` (regenerated), `docs/provider/sap-credentials-store.md`
- **Patterns to follow**: Provider `ValidateStore` / `NewClient` pattern (see existing `provider.go`); `esmeta.SecretKeySelector` for Kubernetes Secret references (consistent with all other provider auth blocks); Ginkgo `Describe`/`It` pattern for e2e (see `e2e/suites/provider/cases/`).
- **Deviations**: None. The token cache uses stdlib `sync.Map` + `oauth2.ReuseTokenSource` (no new abstraction layer); the binding ref uses the established `SecretKeySelector`-style pattern.

### Testing (Principle II)

| Layer | Coverage required | Approach |
|-------|------------------|----------|
| Unit | `ValidateStore` with binding ref (valid, missing fields, not found); namespace precedence logic; token cache hit/miss/concurrent | `provider_test.go` using `httptest.Server` and a mock token endpoint; table-driven tests |
| Integration | Token refresh across expiry boundary | Time-injection via `oauth2.StaticTokenSource` stub in tests |
| E2E | BasicSync, NamespaceOverride, BTPBindingSecret, MissingKey, ConnectionFailure | Ginkgo suite in `e2e/suites/provider/cases/sapcredentialstore/`; gated by build tag `e2e_sapcredentialstore` |
| Regression | Existing `client_test.go` and `provider_test.go` must continue passing | No changes to existing test assertions |

### Consistency (Principle III)

- `serviceBindingSecretRef` field name follows `*SecretRef` suffix convention used elsewhere (e.g., `caSecretRef`, `tokenSecretRef` in other providers).
- `remoteRef.namespace` field name is lowercase, consistent with existing `remoteRef.key`, `remoteRef.property`.
- Error messages follow existing pattern: `"sapCredentialStore: <problem description>"`.
- Status conditions use existing `SecretStoreReady` condition type and reason strings from `esv1.ReasonInvalid` / `esv1.ReasonValid`.
- CRD marker comments use `// +optional` and `// +kubebuilder:default=...` consistent with existing fields.

### Performance (Principle IV)

- **Hot path**: `GetSecret` → `tokenCache.GetOrCreate` → `oauth2.ReuseTokenSource.Token()` → HTTP call. Token lookup is O(1) via `sync.Map`. No additional HTTP calls unless token expires.
- **Cache contention**: `oauth2.ReuseTokenSource` holds an internal mutex; concurrent callers block only for the brief token-fetch duration (< 1 s on normal network). No lock held during HTTP secret fetch.
- **Binding secret re-read**: The binding secret is re-read on every `NewClient` call (once per reconcile). This is one additional Kubernetes API call per reconcile. Acceptable — reconciles are already doing multiple API calls. Can be cached in a future iteration if needed.
- **Memory**: One `oauth2.ReuseTokenSource` per unique `(tokenURL, clientID)` pair. In practice, one per `SecretStore` config; negligible.
- **Regression check**: Existing unit benchmark (if any) must not regress; if none exists, add a simple benchmark for `GetSecret` in the unit test to establish a baseline.

### Security and Compliance (Principle V)

- **Credential logging**: `clientSecret` and `clientid` values read from the binding secret must never appear in log lines, status conditions, or error messages. Error messages reference only key names (e.g., `"missing field: clientsecret"`), not values.
- **TLS**: `InsecureSkipVerify` is not introduced. Existing `api/client.go` uses default `http.Transport`; no change.
- **RBAC**: No new ClusterRole permissions needed. The operator already has `get` on `Secrets` cluster-wide for `SecretKeySelector` resolution. `serviceBindingSecretRef` uses the same path. Documented in contracts.
- **Token cache key**: `sha256(tokenURL + clientID)` — the hash is stored as the map key, not the raw credential values. The `ReuseTokenSource` itself holds the token in memory (unavoidable for functioning auth); this is consistent with how all other ESO providers hold credentials in-process.
- **New dependencies**: None. `oauth2.ReuseTokenSource` and `sync` are already present (stdlib / existing dep).
- **CVE check**: No new direct dependencies; existing `golang.org/x/oauth2 v0.34.0` and `go-jose/go-jose/v4 v4.1.0` must be verified clean at merge time.

**Gate result**: PASS — no violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/003-sapcs-ns-binding-oauth/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── crd-extensions.md   # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
apis/externalsecrets/v1/
├── secretstore_sapcredentialstore_types.go   # Add SAPCSServiceBindingRef; extend SAPCredentialStoreProvider
└── externalsecret_types.go                   # Add Namespace field to ExternalSecretDataRemoteRef

config/crds/                                  # Regenerated by make generate

providers/v1/sapcredentialstore/
├── provider.go                               # ValidateStore: binding ref validation; NewClient: binding resolution + token cache
├── client.go                                 # GetSecret/GetAllSecrets: effectiveNamespace logic
├── provider_test.go                          # Unit tests for ValidateStore extensions
├── client_test.go                            # Unit tests for namespace override in GetSecret
├── tokencache.go                             # NEW: process-level sync.Map token cache + helper
├── tokencache_test.go                        # NEW: unit tests for cache hit/miss/concurrent/refresh
└── api/
    ├── client.go                             # No change (namespace already a parameter)
    └── types.go                              # No change

e2e/suites/provider/cases/sapcredentialstore/
├── sapcredentialstore.go                     # NEW: Ginkgo suite; build tag e2e_sapcredentialstore
└── setup.go                                  # NEW: test config from env vars; fixture creation

docs/provider/
└── sap-credentials-store.md                 # Updated: BTP binding, namespace override, auth flow, e2e guide
```

**Structure Decision**: Single provider package with one new internal file (`tokencache.go`) for the
cache. No new packages or modules. E2E tests in a new subdirectory under the existing provider
cases directory. CRD type additions in the existing types files.

## Complexity Tracking

> No constitution violations requiring justification.

| Deviation | N/A |
|-----------|-----|
| None | All new code follows established patterns. |
