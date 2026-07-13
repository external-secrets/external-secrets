# SAP Credential Store Provider

The External Secrets Operator supports [SAP Credential Store](https://help.sap.com/docs/credential-store), the native secrets service on SAP Business Technology Platform (BTP).

## Features

| Feature | Supported |
|---------|-----------|
| `ExternalSecret` (read) | ✅ |
| `PushSecret` (write) | ✅ |
| `dataFrom` bulk sync | ✅ |
| OAuth2 authentication | ✅ |
| Mutual TLS (mTLS) authentication | ✅ |

## Authentication

The provider supports two authentication modes that correspond to the service binding formats issued by BTP.

### OAuth2 Client Credentials (recommended)

Store the `clientid` and `clientsecret` from the BTP service binding in a Kubernetes Secret:

```bash
kubectl create secret generic sap-cs-oauth2 \
  --from-literal=clientid='<your-client-id>' \
  --from-literal=clientsecret='<your-client-secret>'
```

Reference them in the `SecretStore`:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: sap-credential-store
spec:
  provider:
    sapCredentialStore:
      serviceURL: https://<instance>.credstore.cfapps.<region>.hana.ondemand.com
      namespace: <credential-store-namespace>
      auth:
        oauth2:
          tokenURL: https://<subaccount>.authentication.<region>.hana.ondemand.com/oauth/token
          clientId:
            name: sap-cs-oauth2
            key: clientid
          clientSecret:
            name: sap-cs-oauth2
            key: clientsecret
```

### Mutual TLS (mTLS)

Store the PEM-encoded certificate and private key from the BTP service binding:

```bash
kubectl create secret generic sap-cs-mtls \
  --from-file=tls.crt=client.crt \
  --from-file=tls.key=client.key
```

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: sap-credential-store
spec:
  provider:
    sapCredentialStore:
      serviceURL: https://<instance>.credstore.cfapps.<region>.hana.ondemand.com
      namespace: <credential-store-namespace>
      auth:
        mtls:
          certificate:
            name: sap-cs-mtls
            key: tls.crt
          privateKey:
            name: sap-cs-mtls
            key: tls.key
```

> Use a `ClusterSecretStore` when the ESO controller runs in a different namespace than the auth secrets. Set the `namespace` field on each `SecretKeySelector` in that case.

## Reading Credentials (ExternalSecret)

SAP Credential Store has three credential types: `password`, `key`, and `certificate`.

The `remoteRef.key` field is the credential **name**. The `remoteRef.property` field is the **type** (defaults to `password` when omitted).

### Password credential

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: db-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: sap-credential-store
    kind: SecretStore
  target:
    name: db-secret
  data:
    - secretKey: password
      remoteRef:
        key: db-password        # credential name
        property: password      # credential type (default when omitted)
```

### Key credential

```yaml
  data:
    - secretKey: api-token
      remoteRef:
        key: my-api-key
        property: key
```

### Certificate credential

Accessing the certificate PEM:

```yaml
  data:
    - secretKey: tls.crt
      remoteRef:
        key: my-service-cert
        property: certificate
```

Accessing the private key PEM (use the special `certificate/key` property):

```yaml
  data:
    - secretKey: tls.key
      remoteRef:
        key: my-service-cert
        property: certificate/key
```

### Fetching all fields of a credential (GetSecretMap)

Use `dataFrom` with `extract` to get all fields of a credential as separate keys:

```yaml
spec:
  dataFrom:
    - extract:
        key: db-password
        property: password
```

This produces a Kubernetes Secret with keys `name`, `value`, and `username` (for password credentials).

## Bulk Sync (dataFrom find)

Use `dataFrom` with `find` to sync **all credentials** in the SAP CS namespace into a single Kubernetes Secret. Keys are formatted as `<type>/<name>` (for example, `password/db-pass`, `key/api-key`, `certificate/tls-cert`).

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: all-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: sap-credential-store
    kind: SecretStore
  target:
    name: all-creds-secret
  dataFrom:
    - find: {}
```

> Each key in the resulting Kubernetes Secret maps to the primary `value` field of the credential. For certificate private keys, use individual `data` entries with `property: certificate/key`.

## Writing Credentials (PushSecret)

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-db-password
spec:
  secretStoreRefs:
    - name: sap-credential-store
      kind: SecretStore
  selector:
    secret:
      name: my-k8s-secret
  data:
    - match:
        secretKey: password        # key within the Kubernetes Secret
        remoteRef:
          remoteKey: db-password   # credential name in SAP CS
          property: password       # credential type (defaults to "password")
```

### Credential type mapping for PushSecret

| `property` value | SAP CS type |
|------------------|-------------|
| `password` (or empty) | password |
| `key` | key |
| `certificate` | certificate |

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `ExternalSecret` stuck in `SecretSyncedError` with "credential not found" | Wrong credential name or type | Verify `remoteRef.key` and `remoteRef.property` match the SAP CS credential |
| `SecretStore` not ready: "resolving clientId" | Kubernetes Secret or key not found | Check `auth.oauth2.clientId.name` and `key` match the actual Kubernetes Secret |
| `SecretStore` not ready: "failed to parse mTLS key pair" | Invalid certificate/key PEM | Verify the PEM data from the BTP service binding is complete and unmodified |
| `PushSecret` fails with unexpected status 403 | OAuth2 credentials lack write permission | Ensure the BTP service plan includes write access to Credential Store |

---

## Authentication Flows

The provider supports two ways to supply credentials. In both cases, OAuth2 tokens are cached
in a process-level store (one `oauth2.ReuseTokenSource` per unique `(tokenURL, clientID)` pair)
so that no extra token endpoint call is made on each reconcile while the token remains valid.

```
ExternalSecret reconcile
       │
       ▼
Provider.NewClient(ctx, store, kube, namespace)
       │
       ├─ [serviceBindingSecretRef set?]
       │     → Read binding Secret from Kubernetes
       │     → Parse credentials JSON (clientid, clientsecret, url, tokenurl)
       │
       ├─ [inline auth.oauth2]
       │     → Read clientId/clientSecret SecretKeySelectors
       │
       ▼
OAuthTokenCache.GetOrCreate(sha256(tokenURL + clientID))
       │
       ├─ [cache hit, token valid]   → return existing ReuseTokenSource
       │
       └─ [cache miss]               → create clientcredentials.TokenSource
                                     → wrap in oauth2.ReuseTokenSource
                                     → store in sync.Map
       │
       ▼
HTTP request to SAP CS API
  GET /api/v1/credentials/{type}/{name}?namespace={effectiveNamespace}
  Authorization: Bearer <token from ReuseTokenSource>
       │
       ▼
Token refresh (handled by ReuseTokenSource automatically)
  → token expiring soon → proactive refresh before next use
  → refresh fails, token still valid → use existing token
  → refresh fails, token expired     → ExternalSecret transitions to error
```

**Performance note**: Under 100 concurrent reconciles with a token lifetime ≥ 5 minutes, the
token endpoint is called at most 2–3 times per token lifetime window regardless of concurrency.
The `sync.Map` + `LoadOrStore` pattern prevents thundering-herd duplicate token fetches.

---

## BTP Service Binding Secret

On Kyma/BTP clusters the SAP BTP Operator injects a Kubernetes Secret containing the full
service binding as a JSON object. You can point the provider directly at that Secret via
`serviceBindingSecretRef` — no need to extract individual fields.

### Required JSON fields

The Secret value under `credentialsKey` (default: `credentials`) must contain:

| JSON key | Maps to | Description |
|----------|---------|-------------|
| `clientid` | OAuth2 client ID | Client identifier for token requests |
| `clientsecret` | OAuth2 client secret | Client secret for token requests |
| `url` | `serviceURL` | SAP CS REST API base URL |
| `tokenurl` | OAuth2 token endpoint | Token issuer URL |

A missing field causes the store to transition to `SecretStoreReady=False` with a message
listing the missing key names (never the values).

### Example binding Secret (produced by SAP BTP Operator)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: sap-credstore-binding
  namespace: sap-bindings
type: Opaque
stringData:
  credentials: |
    {
      "clientid": "sb-credential-store-...",
      "clientsecret": "...",
      "url": "https://credstore.cfapps.eu10.hana.ondemand.com/api/v1",
      "tokenurl": "https://mysubdomain.authentication.eu10.hana.ondemand.com/oauth/token"
    }
```

### ClusterSecretStore using a binding Secret

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: sap-credstore-btp
spec:
  provider:
    sapCredentialStore:
      namespace: myapp-prod          # store-level default CS namespace (required)
      serviceBindingSecretRef:
        name: sap-credstore-binding
        namespace: sap-bindings
        credentialsKey: credentials  # optional — "credentials" is the default
```

**Precedence**: when both `serviceBindingSecretRef` and inline `auth.oauth2` are set,
`serviceBindingSecretRef` takes precedence and a warning is logged.

**RBAC**: The operator already holds cluster-wide `get` on `Secrets` for `SecretKeySelector`
resolution in other providers. No new permissions are required for `serviceBindingSecretRef`.

---

## Namespace Configuration

SAP Credential Store partitions credentials into namespaces within a single instance.
ESO lets you set a store-level default and override it per-`ExternalSecret`.

### Store-level namespace (required)

```yaml
spec:
  provider:
    sapCredentialStore:
      namespace: myapp-prod   # all ExternalSecrets default to this namespace
```

### Per-secret namespace override

Add `remoteRef.namespace` to an individual data entry to fetch from a different CS namespace:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: team-a-db-password
  namespace: team-a
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: sap-credstore-btp
    kind: ClusterSecretStore
  target:
    name: team-a-db-secret
  data:
    - secretKey: password
      remoteRef:
        key: db-password
        property: password
        namespace: team-a-credentials   # overrides the store-level "myapp-prod" namespace
    - secretKey: api-key
      remoteRef:
        key: external-api-key
        property: key
        # namespace omitted → uses store-level namespace "myapp-prod"
```

**Precedence rule**: `remoteRef.namespace` (if non-empty after trimming) >
`spec.provider.sapCredentialStore.namespace`.

> This field is SAP CS-specific. Other providers ignore `remoteRef.namespace`.

---

## Running Integration Tests

The SAP CS e2e suite validates the full reconcile path from `ExternalSecret` creation to
Kubernetes Secret population against a live SAP Credential Store instance.

### Required environment variables

| Variable | Where to find it |
|----------|-----------------|
| `SAPCS_SERVICE_URL` | Binding JSON field `url` |
| `SAPCS_TOKEN_URL` | Binding JSON field `tokenurl` |
| `SAPCS_CLIENT_ID` | Binding JSON field `clientid` |
| `SAPCS_CLIENT_SECRET` | Binding JSON field `clientsecret` |
| `SAPCS_NAMESPACE` | Target namespace in the CS instance to use for tests |

```bash
export SAPCS_SERVICE_URL="https://credstore.cfapps.eu10.hana.ondemand.com/api/v1"
export SAPCS_TOKEN_URL="https://mysubdomain.authentication.eu10.hana.ondemand.com/oauth/token"
export SAPCS_CLIENT_ID="sb-credential-store-..."
export SAPCS_CLIENT_SECRET="..."
export SAPCS_NAMESPACE="e2e-test-namespace"
```

### Run the suite

```bash
# From repo root
make e2e-sapcredentialstore

# Or directly:
GOWORK=off go -C ./e2e test ./suites/provider/cases/sapcredentialstore/... \
  -tags e2e_sapcredentialstore -v -timeout 10m
```

If any required environment variable is absent the suite skips gracefully via Ginkgo `Skip`.

### What each test validates

| Test case | Description |
|-----------|-------------|
| `BasicSecretSync` | Creates a `ClusterSecretStore` with inline OAuth2 auth and an `ExternalSecret`; asserts the synced Kubernetes Secret is non-empty |
| `NamespaceOverride` | Creates two `ExternalSecret` resources — one with `remoteRef.namespace` override, one without; asserts each resolves from its respective CS namespace |
| `BTPBindingSecret` | Creates a `ClusterSecretStore` using only `serviceBindingSecretRef`; asserts `SecretSynced` and correct value |
| `MissingKey` | Creates an `ExternalSecret` referencing a non-existent credential; asserts the `ExternalSecret` transitions to error state |
| `ConnectionFailure` | Creates a `ClusterSecretStore` with an unreachable `serviceURL`; asserts the `ExternalSecret` transitions to error state |
