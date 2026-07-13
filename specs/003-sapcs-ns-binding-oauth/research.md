# Research: SAP CS — Namespace, BTP Binding, OAuth Caching & E2E Tests

**Branch**: `003-sapcs-ns-binding-oauth`
**Date**: 2026-07-14
**Status**: Complete — all NEEDS CLARIFICATION items resolved

---

## Topic 1: Per-ExternalSecret Namespace Override

### Decision
Expose the namespace override as a new optional field `namespace` on `ExternalSecretDataRemoteRef`.
Because `ExternalSecretDataRemoteRef` is a shared type used by all providers, the SAP CS provider
will read `remoteRef.namespace` (or fall through to `SAPCredentialStoreProvider.Namespace` if
empty). No new CRD top-level type is needed.

### Rationale
- `ExternalSecretDataRemoteRef` already has extension points (`property`, `version`); adding
  `namespace` keeps the override co-located with the key reference.
- The `property` field is already provider-specific (SAP CS uses it for credential type). A
  `namespace` field has the same well-understood semantics for any partitioned secrets store.
- Adding the field to the shared type makes it discoverable and consistent with how partition/path
  overrides work in other operators (e.g., Vault `mountPath` is baked into `key`).

### Alternatives considered
- **Provider-specific annotation on `ExternalSecret`**: Works but annotations are stringly-typed,
  invisible to validation webhooks, and not surfaced in CRD docs.
- **Separate provider extension CRD**: Over-engineered; the change is a single string field.
- **Encode namespace in `remoteRef.key` as `namespace/credentialName`**: Ambiguous separator; would
  require parsing and break existing key references.

---

## Topic 2: BTP Service Binding Secret — Field Mapping

### Decision
Add a new optional field `serviceBindingSecretRef` of type
`esmeta.SecretKeySelector` (name + namespace + optional key) to `SAPCredentialStoreProvider`.
When set, the provider reads the referenced Kubernetes `Secret` during `NewClient`, parses the
JSON value found at the specified key (default: `credentials`), and extracts the following fields:

| Binding JSON key | Mapped to |
|-----------------|-----------|
| `clientid`      | `auth.oauth2.clientId` (inline value, not a k8s SecretKeyRef) |
| `clientsecret`  | `auth.oauth2.clientSecret` |
| `url`           | `serviceURL` |
| `tokenurl`      | `auth.oauth2.tokenURL` |

All four fields are required; a missing field transitions the store to `SecretStoreReady=False`
with a message listing the missing keys.

### Rationale
- BTP Service Binding secrets produced by the SAP BTP Operator contain a `credentials` JSON object
  (or flat top-level keys depending on the binding type). Parsing `credentials` covers both.
- Using a single `SecretKeySelector` reference (name + namespace) is consistent with how other
  ESO providers reference external secrets (e.g., `auth.secretRef` in AWS, GCP, etc.).
- Precedence rule (binding ref overrides inline) is implemented by checking for non-nil
  `serviceBindingSecretRef` first in `ValidateStore` and `NewClient`.

### Alternatives considered
- **Require individual `SecretKeySelector` fields per BTP key**: More verbose; users would need
  four separate `SecretKeySelector` entries pointing at the same secret, which is redundant.
- **Auto-detect BTP binding by label**: Fragile; label conventions differ across BTP Operator
  versions.
- **Support both flat and nested JSON automatically**: Add complexity for marginal gain; document
  that the `credentials` key is canonical and fall back to top-level parsing as a secondary path.

---

## Topic 3: OAuth Token Caching — Cross-Reconcile Shared Cache

### Decision
Introduce a process-level `sync.Map`-based token cache in the SAP CS provider package, keyed by
`sha256(tokenURL + clientID)`. Each cache entry stores the `*oauth2.Token` value and is
invalidated when `token.Expiry.Sub(time.Now()) < refreshWindow` (default 60 s). The cache is
populated on the first call that misses, with a `singleflight.Group` guard to prevent duplicate
concurrent fetches.

### Rationale
- `golang.org/x/oauth2/clientcredentials` does cache tokens internally, but its cache is
  attached to the `*http.Client` returned by `Config.Client(ctx)`. Because `NewClient` is called
  on every reconcile and creates a fresh `Config.Client(ctx)` each time, no token is reused
  across reconciles. The fix requires a cache that outlives the per-reconcile `NewClient` call.
- `sync.Map` is appropriate for a read-heavy cache with infrequent writes (token refresh is rare
  relative to reconcile frequency).
- `singleflight.Group` prevents the "thundering herd" problem where many concurrent reconciles
  simultaneously find a cache miss and all try to fetch a new token.
- A process-level cache (package-level `var`) is the correct scope: it outlives reconciles but is
  bounded by the controller process lifetime.

### Alternatives considered
- **Re-use the existing `clientcredentials` transport cache by holding a long-lived `*http.Client`
  in provider state**: This requires the provider registration to maintain a singleton per
  credential identity. Feasible, but moving caching into the token cache layer is cleaner and more
  testable.
- **Persist tokens to a Kubernetes Secret**: Unnecessary complexity; tokens are short-lived and
  the provider already re-fetches on startup.
- **Use `golang.org/x/oauth2/tokenSource` directly**: Wrapping a `CachedTokenSource` from the
  oauth2 package is a clean alternative to a hand-rolled cache. This is the preferred approach
  because it reuses proven library code: `oauth2.ReuseTokenSource` wraps any `TokenSource` and
  reuses the cached token until it is within 10 s of expiry; combine with a package-level map of
  `oauth2.ReuseTokenSource` instances keyed by credential identity.

### Final approach
Use `oauth2.ReuseTokenSource` (from `golang.org/x/oauth2`) held in a package-level
`sync.Map` keyed by credential identity. This:
- Reuses proven cache and refresh logic from the oauth2 library.
- Is safe for concurrent use (`ReuseTokenSource` uses a mutex internally).
- Avoids duplicating retry/refresh logic.
- Is straightforward to test by injecting a `TokenSource` stub.

---

## Topic 4: End-to-End Test Framework

### Decision
Add SAP CS e2e tests under `e2e/suites/provider/cases/sapcredentialstore/` following the
established Ginkgo/Gomega pattern used by all other providers. Tests are gated by build tag
`e2e_sapcredentialstore` and require the following environment variables:

| Variable | Source |
|----------|--------|
| `SAPCS_SERVICE_URL` | From BTP service binding `url` |
| `SAPCS_TOKEN_URL`   | From BTP service binding `tokenurl` |
| `SAPCS_CLIENT_ID`   | From BTP service binding `clientid` |
| `SAPCS_CLIENT_SECRET` | From BTP service binding `clientsecret` |
| `SAPCS_NAMESPACE`   | Target namespace in the CS instance |

Test cases mirror the common ESO e2e patterns (`SimpleDataSync`, error cases) plus SAP CS-specific
cases for namespace override and BTP binding secret.

### Rationale
- Existing e2e framework already handles `SecretStore` creation, `ExternalSecret` lifecycle, and
  assertion helpers. Reusing it minimises new code.
- Build-tag gating (consistent with other provider e2e tests) keeps the default `make test` fast.

---

## Topic 5: Documentation Surfaces

### Decision
Update the following documentation files:
- `docs/provider/sap-credentials-store.md` — Add sections: Authentication Flows, BTP Service
  Binding Secret Contents, Namespace Configuration (store-level vs. per-secret precedence), and
  Running Integration Tests.
- `apis/externalsecrets/v1/secretstore_sapcredentialstore_types.go` — Add Go doc comments to new
  fields (`serviceBindingSecretRef`).
- Generated CRD YAML (`config/crds/`) — regenerated by `make generate`.

### Rationale
- The existing provider doc is sparse; the new features are non-obvious and require clear
  reference material.
- Go doc comments on CRD types appear in `kubectl explain` and generated API reference docs.

---

## Open Questions (resolved)

| # | Question | Resolution |
|---|----------|------------|
| 1 | Which field carries the per-secret namespace override? | `ExternalSecretDataRemoteRef.Namespace` (new field) |
| 2 | BTP binding JSON structure — flat or nested? | Parse `credentials` key first; fall back to top-level |
| 3 | Token cache scope — process-level or per-reconcile? | Process-level `sync.Map` of `oauth2.ReuseTokenSource` instances |
| 4 | E2E test gating mechanism | Build tag `e2e_sapcredentialstore` + env vars |
| 5 | mTLS BTP bindings in scope? | Out of scope for this iteration (documented) |
