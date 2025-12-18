# OpenStack Barbican

External Secrets Operator integrates with [OpenStack Barbican](https://docs.openstack.org/barbican/latest/) for secret management.

Barbican is OpenStack's Key Manager service that provides secure storage, provisioning and management of secret data. This includes keys, passwords, certificates, and other sensitive data. The Barbican provider for External Secrets Operator allows you to retrieve secrets stored in Barbican and synchronize them with Kubernetes secrets.

## Authentication

The Barbican provider uses OpenStack Keystone authentication. You need to provide:

- **AuthURL**: The OpenStack Keystone authentication endpoint
- **TenantName**: The OpenStack tenant/project name
- **DomainName**: The OpenStack domain name (optional)
- **Region**: The OpenStack region (optional)
- **Username**: OpenStack username (stored in a Kubernetes secret)
- **Password**: OpenStack password (stored in a Kubernetes secret)

## Example

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

## Creating an ExternalSecret

Now you can create an ExternalSecret that uses the Barbican provider to retrieve secrets:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: barbican-secret
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

## Finding Secrets by Name

You can also retrieve secrets by using the `find` feature to search by name.

It doesnt really support regexp, its exact string matching, so you need to provide the exact name of the secret.

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
        regexp: "database"
```

This will find all secrets in Barbican whose name exactly matches the string.

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

## Configuration Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `authURL` | string | Yes | OpenStack Keystone authentication endpoint URL |
| `tenantName` | string | Yes | OpenStack tenant/project name |
| `domainName` | string | No | OpenStack domain name |
| `region` | string | No | OpenStack region |
| `auth` | BarbicanAuth | Yes | Authentication credentials |

### BarbicanAuth

The `BarbicanAuth` type contains the authentication information:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | BarbicanProviderUsernameRef | Yes | OpenStack username (from secret or literal value) |
| `password` | BarbicanProviderPasswordRef | Yes | OpenStack password (from secret only) |

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

## Limitations

- The Barbican provider is **read-only**. It does not support creating or updating secrets in Barbican.
- Used credentials has to have access to the provided secret.
- It will retrieve all secret types by default.

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
