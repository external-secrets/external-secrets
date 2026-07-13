# Data Model: SAP CS — Namespace, BTP Binding, OAuth Caching & E2E Tests

**Branch**: `003-sapcs-ns-binding-oauth`
**Date**: 2026-07-14

---

## Entities

### 1. `SAPCredentialStoreProvider` (extended)

*Represents the provider-level configuration block within a `SecretStore` or `ClusterSecretStore`.*

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serviceURL` | string | Yes* | Base URL of the SAP Credential Store REST API |
| `namespace` | string | Yes* | Default CS namespace used when no per-secret override is present |
| `auth` | `SAPCSAuth` | Yes* | Authentication configuration (OAuth2 or mTLS) |
| `encryption` | `SAPCSEncryption` | No | Optional JWE payload decryption keys |
| `serviceBindingSecretRef` | `SAPCSServiceBindingRef` | No | Reference to a BTP Service Binding Kubernetes Secret; when set, overrides `serviceURL`, `auth.oauth2`, and derivable `namespace` fields |

\*Fields marked required become optional (omitempty) when `serviceBindingSecretRef` is set, because
all values can be derived from the binding secret. `ValidateStore` enforces that either
(`serviceURL` + `auth`) OR `serviceBindingSecretRef` is present — not both partially filled.

**Validation rules**:
- `serviceBindingSecretRef` and inline `auth.oauth2` fields are mutually exclusive; if both are
  set, `serviceBindingSecretRef` wins and a warning is recorded.
- If `serviceBindingSecretRef` is set, `serviceURL` and `auth` are ignored; derived values from
  the binding are used instead.
- `namespace` from the store spec is always used as the default when no per-secret override is
  present; if the store uses `serviceBindingSecretRef` and `namespace` is empty, the provider
  returns an error on first use (namespace is always required for API calls).

---

### 2. `SAPCSServiceBindingRef` (new)

*A reference to a Kubernetes `Secret` produced by the SAP BTP Operator that contains the service
binding credentials.*

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the Kubernetes Secret |
| `namespace` | string | No | Namespace of the Secret (defaults to the SecretStore's own namespace for namespaced stores; must be explicit for ClusterSecretStore) |
| `credentialsKey` | string | No | JSON key inside the Secret's data that holds the credentials object (default: `credentials`) |

**BTP Service Binding Secret — required JSON fields** (under `credentialsKey`):

| JSON key | Mapped provider field | Description |
|----------|-----------------------|-------------|
| `clientid` | `auth.oauth2.clientId` | OAuth2 client identifier |
| `clientsecret` | `auth.oauth2.clientSecret` | OAuth2 client secret |
| `url` | `serviceURL` | SAP CS REST API base URL |
| `tokenurl` | `auth.oauth2.tokenURL` | OAuth2 token endpoint |

All four keys are required. A missing key produces a `SecretStoreReady=False` condition with the
message listing the missing keys.

**State transitions for the `SecretStore`**:

```
Pending
  → Ready          (binding secret found, all required keys present, OAuth token obtained)
  → NotReady/Error (binding secret not found)
  → NotReady/Error (required keys missing from binding JSON)
  → NotReady/Error (OAuth token fetch failed)
```

---

### 3. `ExternalSecretDataRemoteRef` (extended)

*Per-secret reference that identifies which credential to fetch; extended with a namespace override.*

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Yes | Credential name within the SAP CS namespace |
| `property` | string | No | Credential type (`password`, `key`, `certificate`, `certificate/key`); default: `password` |
| `version` | string | No | Credential version (not used by current SAP CS API; reserved) |
| `namespace` | string | No | **(new)** SAP CS namespace override; when non-empty, takes precedence over the store-level namespace |

**Namespace precedence rule** (evaluated at fetch time):
```
effectiveNamespace = remoteRef.namespace (if non-empty)
                  ?? secretStore.spec.provider.sapCredStore.namespace
```

---

### 4. `OAuthTokenCache` (process-level, in-memory only)

*Not a Kubernetes resource; a package-level data structure in the provider binary.*

| Attribute | Description |
|-----------|-------------|
| Key | `sha256(tokenURL + ":" + clientID)` — stable, non-secret identifier |
| Value | `oauth2.ReuseTokenSource` wrapping the underlying `clientcredentials.TokenSource` |
| Scope | Process lifetime (controller pod) |
| Concurrency | Safe: `sync.Map` for the map; `oauth2.ReuseTokenSource` uses an internal mutex |
| Refresh window | Token is considered stale when `expiry - now < 60s` (configurable via env var `SAPCS_TOKEN_REFRESH_WINDOW`) |
| Eviction | Entries are replaced when credentials rotate (new `clientID` or `tokenURL` → new key) |

---

### 5. E2E Test Fixtures

*Not Kubernetes resources; Go test data structures used by the e2e suite.*

| Entity | Fields |
|--------|--------|
| `SAPCSTestConfig` | `ServiceURL`, `TokenURL`, `ClientID`, `ClientSecret`, `Namespace` (all from env vars) |
| `SAPCSNamespaceOverrideFixture` | Store with default namespace + ExternalSecret with override namespace |
| `SAPCSBindingSecretFixture` | Kubernetes Secret containing simulated BTP binding JSON |

---

## Relationships

```
ClusterSecretStore
  └─ spec.provider.sapCredentialStore  (SAPCredentialStoreProvider)
       ├─ serviceURL                    (string, from binding if binding ref set)
       ├─ namespace                     (string, store-level default)
       ├─ auth.oauth2                   (SAPCSOAuth2Auth, from binding if binding ref set)
       └─ serviceBindingSecretRef       (SAPCSServiceBindingRef)  ──→  Kubernetes Secret
                                                                         └─ credentials JSON
                                                                              ├─ clientid
                                                                              ├─ clientsecret
                                                                              ├─ url
                                                                              └─ tokenurl

ExternalSecret
  └─ spec.data[].remoteRef             (ExternalSecretDataRemoteRef)
       ├─ key                           (credential name)
       ├─ property                      (credential type)
       └─ namespace                     (NEW: per-secret CS namespace override)
            │
            ▼
       effectiveNamespace = remoteRef.namespace ?? store.namespace

OAuthTokenCache (process-level sync.Map)
  └─ keyed by sha256(tokenURL + clientID)
       └─ oauth2.ReuseTokenSource
            └─ automatically refreshes token before expiry
```
