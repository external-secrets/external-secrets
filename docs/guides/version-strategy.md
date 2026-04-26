# Version Strategy

The `versionStrategy` field controls how ESO handles secret versioning when fetching secrets from a provider.
It can be placed under `spec.data.remoteRef` or `spec.dataFrom.find`, and configures the versioning behavior for that specific operation.

## Strategies

### None (default)

ESO fetches the default version (e.g. latest available) of the secret or the specific version indicated by `spec.data.remoteRef.version`.
No additional filtering is applied.

### Available

ESO filters out all secret version data that has been soft or hard deleted.

- For **`spec.data.remoteRef`**: If a `version` was also explicitly specified the Resource fails.
  If no `version` was specified, all available secret versions get fetched.

## Examples

### Fetching a single Vault secret, skipping deleted and destroyed versions

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: vault-available-version
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: my-secret
  data:
    - secretKey: password
      remoteRef:
        key: secret/my-app/db
        versionStrategy: Available
```

If Secret Manager contains a secret `my-app/db` with versions 1, 2 (destroyed) and 3, the resulting Kubernetes Secret will contain:

```yaml
data:
  backend-path/secret/my-app/db?version=1: <base64-encoded value at version 1>
  backend-path/secret/my-app/db?version=3: <base64-encoded value at version 3>
```

Version 2 is omitted because it has been destroyed.

### Enumerating all available GCP secrets/versions with `dataFrom.find`

Using `versionStrategy: Available` in a `find` operation causes ESO to return enabled secret versions.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: gcp-all-available-versions
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: gcp-backend
    kind: SecretStore
  target:
    name: my-secret-versions
  dataFrom:
    - find:
        name:
          regexp: "my-.*"
        versionStrategy: Available
```

If Secret Manager contains a secret `my-app` with versions 1, 2 (disabled) and 3, the resulting Kubernetes Secret will contain:

```yaml
data:
  projects/my-project/secrets/my-app/versions/1: <base64-encoded value at version 1>
  projects/my-project/secrets/my-app/versions/3: <base64-encoded value at version 3>
```

Version 2 is omitted because it has been disabled.
