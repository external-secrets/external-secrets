# OpenStack Barbican

External Secrets Operator integrates with [OpenStack Barbican](https://docs.openstack.org/barbican/latest/) for secret management.

Barbican is OpenStack's Key Manager service that provides secure storage, provisioning and management of secret data. This includes keys, passwords, certificates, and other sensitive data. The Barbican provider for External Secrets Operator allows you to retrieve secrets stored in Barbican and synchronize them with Kubernetes secrets.

## Authentication

The Barbican provider supports two OpenStack Keystone authentication modes:

- `password` (default): Username + password.
- `applicationCredential`: OpenStack Application Credentials.

### Required provider fields

- **authURL**: OpenStack Keystone authentication endpoint.
- **region**: OpenStack region (optional).
- **tenantName**: OpenStack project/tenant (optional, depending on your Keystone setup).
- **domainName**: OpenStack domain (required for password auth in environments that require domain scoping).

## Example User Name/Password Authentication

First, create a secret containing your OpenStack credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: barbican-secret
type: Opaque
data:
  username: bXl1c2VybmFtZQ== # base64 encoded "myusername"
  password: bXlwYXNzd29yZA== # base64 encoded "mypassword"
```

Then create a SecretStore with the Barbican backend:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: barbican-backend
spec:
  provider:
    barbican:
      authURL: "https://keystone.example.com:5000/v3"
      tenantName: "my-project"
      domainName: "default"
      region: "RegionOne"
      auth:
        username:
          secretRef:
            name: "barbican-secret"
            key: "username"
        password:
          secretRef:
            name: "barbican-secret"
            key: "password"
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` for the `secretRef` with the namespace of the secret that contains the credentials.

## Example Application Credential Authentication

You can authenticate using OpenStack Application Credentials by setting `auth.authType: applicationCredential`.

Create a secret with Application Credential ID and credential secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: barbican-appcred
type: Opaque
data:
  appCredID: YXBwLWNyZWQtaWQ= # base64 encoded app credential ID
  appCredSecret: YXBwLWNyZWQtc2VjcmV0 # base64 encoded app credential secret
```

Use it in a `SecretStore`:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: barbican-backend-appcred-id
spec:
  provider:
    barbican:
      authURL: "https://keystone.example.com:5000/v3"
      region: "RegionOne"
      auth:
        authType: applicationCredential
        applicationCredentialID:
          secretRef:
            name: "barbican-appcred"
            key: "appCredID"
        applicationCredentialSecret:
          secretRef:
            name: "barbican-appcred"
            key: "appCredSecret"
```

## Creating an ExternalSecret

Now you can create an ExternalSecret that uses the Barbican provider to retrieve secrets:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: barbican-example
spec:
  secretStoreRef:
    name: barbican-backend
    kind: SecretStore
  target:
    name: example-secret
    creationPolicy: Owner
  data:
  - secretKey: password
    remoteRef:
      key: "my-secret-uuid"
```

The `remoteRef.key` should be the UUID of the secret in Barbican. You can find this by listing secrets in Barbican:

```bash
openstack secret list
```

## Referencing a property within a secret

If a Barbican secret stores a JSON object as its payload, you can select a single top-level key with `remoteRef.property`:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: barbican-property
spec:
  secretStoreRef:
    name: barbican-backend
    kind: SecretStore
  target:
    name: example-secret
    creationPolicy: Owner
  data:
  - secretKey: token
    remoteRef:
      key: "my-secret-uuid"
      property: "token" # selects the "token" key from the JSON payload
```

To expand a whole JSON payload into multiple Kubernetes secret keys at once, use `dataFrom.extract`:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: barbican-extract
spec:
  secretStoreRef:
    name: barbican-backend
    kind: SecretStore
  target:
    name: example-secret
    creationPolicy: Owner
  dataFrom:
  - extract:
      key: "my-secret-uuid"
```

Both `property` and `extract` require the secret payload to be a JSON object. Without `property`, `remoteRef` returns the raw payload unchanged.

## Finding Secrets by Name

You can retrieve secrets with the `find` feature, matching on the secret name.

Despite the field being named `regexp`, the value is passed to Barbican's secret listing API as a `name` filter, which performs an exact name match. Regular-expression metacharacters are **not** interpreted, so a value like `^db-.*` matches only a secret literally named `^db-.*`. Provide the exact secret name.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: barbican-find-secret
spec:
  secretStoreRef:
    name: barbican-backend
    kind: SecretStore
  target:
    name: found-secrets
    creationPolicy: Owner
  dataFrom:
  - find:
      name:
        regexp: "database" # exact secret name, not a pattern
```

Because Barbican allows several secrets to share a name, this can return more than one secret. The keys of the resulting Kubernetes secret are the Barbican secret UUIDs (not the names), and each value is the corresponding payload.

## ClusterSecretStore

For a ClusterSecretStore, you need to specify the namespace where the credentials secret is located:

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: barbican-cluster-backend
spec:
  provider:
    barbican:
      authURL: "https://keystone.example.com:5000/v3"
      tenantName: "my-project"
      domainName: "default"
      region: "RegionOne"
      auth:
        username:
          secretRef:
            name: "barbican-secret"
            key: "username"
            namespace: "default"  # Required for ClusterSecretStore
        password:
          secretRef:
            name: "barbican-secret"
            key: "password"
            namespace: "default"  # Required for ClusterSecretStore
```
The same `namespace` rule applies to `applicationCredentialID.secretRef` and `applicationCredentialSecret.secretRef` when using `ClusterSecretStore`.

## Configuration Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `authURL` | string | Yes | OpenStack Keystone authentication endpoint URL |
| `tenantName` | string | No | OpenStack tenant/project name |
| `domainName` | string | No | OpenStack domain name |
| `region` | string | No | OpenStack region |
| `auth` | BarbicanAuth | Yes | Authentication credentials |

### BarbicanAuth

The `BarbicanAuth` type contains the authentication information:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `authType` | BarbicanAuthType | No | Auth mode: `password` (default) or `applicationCredential` |
| `username` | BarbicanProviderUsernameRef | Conditional | Required for `password` |
| `password` | BarbicanProviderPasswordRef | Conditional | Required for `password` |
| `applicationCredentialID` | BarbicanProviderAppCredIDRef | Conditional | Required for `applicationCredential` |
| `applicationCredentialSecret` | BarbicanProviderAppCredSecretRef | Conditional | Required for `applicationCredential` |

### BarbicanAuthType

| Value | Description |
|-------|-------------|
| `password` | Keystone username/password authentication |
| `applicationCredential` | Keystone Application Credential authentication |

### BarbicanProviderUsernameRef

The `BarbicanProviderUsernameRef` type allows you to specify username either as a literal or reference to a Kubernetes secret:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `value` | string | No | Literal value (not recommended for sensitive data) |
| `secretRef` | SecretKeySelector | No | Reference to a Kubernetes secret |

### BarbicanProviderPasswordRef

The `BarbicanProviderPasswordRef` type requires a reference to a Kubernetes secret:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `secretRef` | SecretKeySelector | Yes | Reference to a Kubernetes secret |

### BarbicanProviderAppCredIDRef

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `value` | string | No | Literal Application Credential ID |
| `secretRef` | SecretKeySelector | No | Reference to a Kubernetes secret containing the Application Credential ID |

### BarbicanProviderAppCredSecretRef

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `secretRef` | SecretKeySelector | Yes | Reference to a Kubernetes secret containing the Application Credential secret |

## Limitations

- The Barbican provider is **read-only**. Creating, updating, or deleting secrets is not supported (`PushSecret` and `DeletionPolicy: Delete` will fail).
- The credentials used must have access to the secrets being retrieved.
- `find` matches the exact secret name only; `find.path` and `find.tags` are not supported.
- Barbican secrets are immutable, so `remoteRef.version` is ignored.
- Secret metadata is not exposed (`metadataPolicy: Fetch` is not supported); only the payload is returned.

## Troubleshooting

### Authentication Issues

If you encounter authentication errors, verify:

1. The `authURL` is correct and accessible
2. The credentials are valid and have appropriate permissions
3. The `tenantName` and `domainName` (if used) are correct
4. Network connectivity to the OpenStack endpoints

### Secret Not Found

If a secret cannot be found:

1. Verify the secret UUID exists in Barbican: `openstack secret get -p https://barbican-url/v1/secrets/<uuid>`
2. Check that the user has permission to access the secret
3. Ensure the secret is in the correct project/tenant

### Network Connectivity

Ensure your Kubernetes cluster can reach:

- The OpenStack Keystone endpoint (for authentication)
- The Barbican service endpoint (for secret retrieval)

Check firewall rules and network policies that might block access.
