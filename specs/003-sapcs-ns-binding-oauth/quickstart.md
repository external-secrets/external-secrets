# Quickstart: SAP Credential Store Provider v2 Features

**Branch**: `003-sapcs-ns-binding-oauth`
**Date**: 2026-07-14
**Audience**: Platform engineers and developers onboarding to the updated SAP Credential Store provider

---

## Prerequisites

- External Secrets Operator installed (v0.x.x or later with this feature)
- A SAP BTP Credential Store service instance created
- BTP Service Binding created and its Kubernetes Secret available (or manual credentials at hand)
- `kubectl` access to the cluster

---

## Option A: Configure using a BTP Service Binding Secret (Recommended on Kyma/BTP)

### Step 1 — Identify your BTP Service Binding Secret

The SAP BTP Operator creates a Kubernetes Secret when you bind a service instance. Find it:

```bash
kubectl get secrets -n sap-bindings -l services.cloud.sap.com/serviceBinding=sap-credstore-binding
```

Inspect the `credentials` key:

```bash
kubectl get secret sap-credstore-binding -n sap-bindings \
  -o jsonpath='{.data.credentials}' | base64 -d | jq .
```

You should see (at minimum):

```json
{
  "clientid": "sb-credential-store-...",
  "clientsecret": "...",
  "url": "https://credstore.cfapps.eu10.hana.ondemand.com/api/v1",
  "tokenurl": "https://mysubdomain.authentication.eu10.hana.ondemand.com/oauth/token"
}
```

### Step 2 — Create a ClusterSecretStore

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: sap-credstore
spec:
  provider:
    sapCredentialStore:
      namespace: default-namespace          # your default CS namespace
      serviceBindingSecretRef:
        name: sap-credstore-binding
        namespace: sap-bindings
        # credentialsKey defaults to "credentials" — omit unless your binding uses a different key
```

```bash
kubectl apply -f cluster-secret-store.yaml
kubectl get clustersecretstore sap-credstore
# STATUS: Valid
```

### Step 3 — Create an ExternalSecret

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secret
  namespace: my-app
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: sap-credstore
    kind: ClusterSecretStore
  target:
    name: my-app-secret
  data:
    - secretKey: db-password
      remoteRef:
        key: my-db-password         # credential name in SAP CS
        property: password          # credential type
```

```bash
kubectl apply -f external-secret.yaml
kubectl get externalsecret my-app-secret -n my-app
# STATUS: SecretSynced
kubectl get secret my-app-secret -n my-app -o jsonpath='{.data.db-password}' | base64 -d
```

---

## Option B: Configure using inline credentials

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: sap-credstore-inline
spec:
  provider:
    sapCredentialStore:
      serviceURL: https://credstore.cfapps.eu10.hana.ondemand.com/api/v1
      namespace: default-namespace
      auth:
        oauth2:
          tokenURL: https://mysubdomain.authentication.eu10.hana.ondemand.com/oauth/token
          clientId:
            name: sap-credstore-creds
            namespace: secrets
            key: clientId
          clientSecret:
            name: sap-credstore-creds
            namespace: secrets
            key: clientSecret
```

---

## Per-Secret Namespace Override

When credentials are stored in multiple namespaces within the same SAP CS instance, override the
namespace on a per-ExternalSecret basis without creating additional stores:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: team-a-secret
  namespace: team-a
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: sap-credstore
    kind: ClusterSecretStore
  target:
    name: team-a-secret
  data:
    - secretKey: api-key
      remoteRef:
        key: service-api-key
        property: key
        namespace: team-a-credentials   # overrides the store-level default namespace
    - secretKey: db-password
      remoteRef:
        key: main-db
        property: password
        # no namespace override → uses store-level default namespace
```

**Precedence rule**: `remoteRef.namespace` (if non-empty) > `spec.provider.sapCredentialStore.namespace`

---

## Authentication Flow Overview

```
ExternalSecret reconcile
       │
       ▼
Provider.NewClient(ctx, store, kube, namespace)
       │
       ├─ [binding ref set?] → Read binding Secret → parse credentials JSON
       │                      → extract clientid, clientsecret, url, tokenurl
       │
       ├─ [inline auth] → Read clientId/clientSecret SecretKeySelectors
       │
       ▼
OAuthTokenCache.GetOrCreate(sha256(tokenURL + clientID))
       │
       ├─ [cache hit, token valid] → return existing ReuseTokenSource
       │
       └─ [cache miss or new credentials] → create clientcredentials.TokenSource
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
  → token expires within 60s → proactive refresh before next use
  → refresh fails, token still valid → use existing token, log warning
  → refresh fails, token expired → return error on ExternalSecret
```

---

## Running Integration Tests

### Requirements

Set the following environment variables:

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

# Or directly with go test:
go test ./e2e/suites/provider/... -tags e2e_sapcredentialstore -v -timeout 10m
```

### What the suite covers

| Test case | Description |
|-----------|-------------|
| `BasicSecretSync` | Creates an ExternalSecret and verifies the Kubernetes Secret matches the CS value |
| `NamespaceOverride` | Verifies per-secret namespace override fetches from the correct CS namespace |
| `BTPBindingSecret` | Verifies a ClusterSecretStore using only a binding secret ref resolves credentials correctly |
| `MissingKey` | Verifies an ExternalSecret with a non-existent key transitions to error state with the expected message |
| `ConnectionFailure` | Verifies a store pointing at an unreachable URL transitions to error state |
