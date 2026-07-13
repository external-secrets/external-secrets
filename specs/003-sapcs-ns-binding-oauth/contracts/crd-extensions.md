# Contract: SAP Credential Store — CRD Extensions

**Branch**: `003-sapcs-ns-binding-oauth`
**Date**: 2026-07-14
**Contract type**: Kubernetes CRD field contracts (Go struct → CRD YAML → user-facing YAML)

This document defines the public API surface added or changed by this feature. Changes here are
backward-compatible additions only (no field removals, no semantic changes to existing fields).

---

## 1. `SAPCredentialStoreProvider` — New Field: `serviceBindingSecretRef`

### Go struct addition

```go
// SAPCredentialStoreProvider is the provider configuration for SAP Credential Store.
type SAPCredentialStoreProvider struct {
    // ... existing fields unchanged ...

    // ServiceBindingSecretRef references a Kubernetes Secret containing a BTP Service
    // Binding. When set, the provider derives serviceURL, auth.oauth2.tokenURL,
    // auth.oauth2.clientId, and auth.oauth2.clientSecret from the binding secret,
    // and these inline fields become optional.
    // If serviceBindingSecretRef and inline auth fields are both set,
    // serviceBindingSecretRef takes precedence.
    // +optional
    ServiceBindingSecretRef *SAPCSServiceBindingRef `json:"serviceBindingSecretRef,omitempty"`
}

// SAPCSServiceBindingRef references a Kubernetes Secret that contains a BTP Service Binding.
type SAPCSServiceBindingRef struct {
    // Name of the Kubernetes Secret containing the BTP Service Binding.
    Name string `json:"name"`

    // Namespace of the Secret. For ClusterSecretStore, this field is required.
    // For namespaced SecretStore, defaults to the store's own namespace.
    // +optional
    Namespace string `json:"namespace,omitempty"`

    // CredentialsKey is the key within the Secret's data map that holds the
    // JSON-encoded credentials object. Defaults to "credentials".
    // +optional
    // +kubebuilder:default=credentials
    CredentialsKey string `json:"credentialsKey,omitempty"`
}
```

### Example SecretStore YAML (binding ref)

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: sap-credstore-btp
spec:
  provider:
    sapCredentialStore:
      namespace: myapp-prod          # store-level default namespace
      serviceBindingSecretRef:
        name: sap-credstore-binding
        namespace: sap-bindings
        credentialsKey: credentials  # optional, this is the default
```

### Example BTP Binding Secret (referenced above)

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

---

## 2. `ExternalSecretDataRemoteRef` — New Field: `namespace`

### Go struct addition

```go
// ExternalSecretDataRemoteRef defines criteria for fetching a secret from a provider.
type ExternalSecretDataRemoteRef struct {
    // ... existing fields unchanged ...

    // Namespace overrides the provider-level namespace for this specific secret reference.
    // For the SAP Credential Store provider, this sets the CS namespace used in the API path.
    // When empty, the SecretStore-level namespace is used.
    // +optional
    Namespace string `json:"namespace,omitempty"`
}
```

### Example ExternalSecret YAML (namespace override)

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
```

### ExternalSecret YAML (no override — uses store default)

```yaml
  data:
    - secretKey: api-key
      remoteRef:
        key: external-api-key
        property: key
        # namespace omitted → uses store-level namespace "myapp-prod"
```

---

## 3. Precedence and Validation Rules

| Scenario | `serviceURL` source | `tokenURL` source | `namespace` used |
|----------|--------------------|--------------------|-----------------|
| Inline only | `spec.serviceURL` | `spec.auth.oauth2.tokenURL` | `spec.namespace` |
| Binding ref only | Binding `url` | Binding `tokenurl` | `spec.namespace` (required) |
| Both set | Binding `url` (warning logged) | Binding `tokenurl` | `spec.namespace` |
| Per-secret override | — | — | `remoteRef.namespace` |
| No override | — | — | `spec.namespace` |

### `ValidateStore` error conditions

| Condition | HTTP-equivalent | Error message |
|-----------|----------------|---------------|
| `serviceBindingSecretRef.name` empty | 400 | "serviceBindingSecretRef.name is required" |
| Binding secret not found | 404 | "serviceBindingSecretRef: secret <ns>/<name> not found" |
| Missing binding JSON key | 400 | "serviceBindingSecretRef: missing required fields: [clientsecret, tokenurl]" |
| Neither inline auth nor binding ref | 400 | "either auth.oauth2 or serviceBindingSecretRef must be set" |
| `namespace` empty (store level) | 400 | "namespace is required" |

---

## 4. RBAC — Additional Permission Required

When `serviceBindingSecretRef` targets a namespace other than the operator's own, the operator's
`ClusterRole` must include:

```yaml
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
  # No wildcard namespace — operator already has get/list/watch on secrets for store auth resolution
```

The existing ClusterRole already grants `get` on `secrets` cluster-wide for resolving
`SecretKeySelector` references in other providers. This feature does not add new permissions;
it uses the existing grant. Document this in the provider docs.

---

## 5. Backward Compatibility

- All new fields are `+optional` with `omitempty`; existing `SecretStore` and `ExternalSecret`
  resources without the new fields continue to work without changes.
- `ExternalSecretDataRemoteRef.Namespace` is a net-new field with no prior semantic; adding it
  cannot break existing users.
- `SAPCSServiceBindingRef` is a net-new type; existing stores without it are unaffected.
- CRD regeneration (`make generate`) is required to publish the new fields.
