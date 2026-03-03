## BeyondTrust Secrets Manager

External Secrets Operator integrates with BeyondTrust Secrets Manager for secret management.

The provider supports static key-value secrets stored in folders. For dynamic secret generation (e.g., temporary AWS credentials), refer to the [BeyondTrust Secrets Manager Generator](../api/generator/beyondtrustsecrets.md).

### Example

First, create a SecretStore with a BeyondTrust Secrets Manager backend. You'll need an API token and the server configuration:

```yaml
{% include 'beyondtrustsecrets-secret-store.yaml' %}
```

Create the API token secret:
```bash
kubectl create secret generic api-token \
  --from-literal=token=<YOUR_API_TOKEN> \
  -n external-secrets
```

If using self-signed certificates, create a CA bundle secret:
```bash
kubectl create secret generic my-ca-bundle \
  --from-file=ca.crt="/path/to/root.crt" \
  -n external-secrets
```

Now create an ExternalSecret that uses the above SecretStore:

```yaml
{% include 'beyondtrustsecrets-external-secret.yaml' %}
```

This will automatically create a Kubernetes Secret with the synced data.

### Fetching Secret Properties

#### Single Property Retrieval

You can fetch a specific property from a secret by specifying the `property` field:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: postgres-credentials
  namespace: external-secrets
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: beyondtrustsecrets-ss
    kind: SecretStore
  target:
    name: postgres-creds
  data:
    - secretKey: username
      remoteRef:
        key: postgresCreds
        property: username
    - secretKey: password
      remoteRef:
        key: postgresCreds
        property: password
```
This creates a secret with individual keys for `username` and `password`.

#### Fetching All Properties as JSON

If you omit the `property` field, you'll get all key-value pairs as a single JSON string:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: postgres-credentials-json
  namespace: external-secrets
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: beyondtrustsecrets-ss
    kind: SecretStore
  target:
    name: postgres-creds-json
  data:
    - secretKey: credentials
      remoteRef:
        key: postgresCreds
```

This returns: `{"username":"user","password":"pass"}` as the value of `credentials`.

#### Extracting All Properties as Separate Keys

To sync all properties of a secret as individual keys in the target Kubernetes secret, use `dataFrom` with `extract`:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: postgres-credentials-extracted
  namespace: external-secrets
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: beyondtrustsecrets-ss
    kind: SecretStore
  target:
    name: postgres-creds-extracted
  dataFrom:
    - extract:
        key: postgresCreds
```

This creates a secret with:
```yaml
data:
  username: dXNlcg==     # base64("user")
  password: cGFzcw==     # base64("pass")
```

### Getting Multiple Secrets

You can extract multiple secrets from a folder by using `dataFrom.find`.

Given a folder `eso/static` with these secrets:
- `anotherSecret`: `{"someKey": "value1"}`
- `mySecret`: `{"myKey": "value2", "someKey": "value3"}`
- `postgresCreds`: `{"username": "user", "password": "pass"}`

#### Fetch All Secrets in Folder

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: all-secrets
  namespace: external-secrets
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: beyondtrustsecrets-ss
    kind: SecretStore
  target:
    name: all-folder-secrets
  dataFrom:
    - find:
        name:
          regexp: ".*"
```

This merges all key-value pairs from all secrets in the folder into a single Kubernetes secret.

#### Regex Pattern Filtering

To sync only secrets matching a specific pattern:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: filtered-secrets
  namespace: external-secrets
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: beyondtrustsecrets-ss
    kind: SecretStore
  target:
    name: filtered-folder-secrets
  dataFrom:
    - find:
        name:
          regexp: "Secret$"
```

This will only sync secrets whose names end with "Secret" (e.g., `anotherSecret`, `mySecret`).

#### Specifying a Different Folder

By default, `find` uses the `folderPath` from the SecretStore. To search a different folder, use the `path` field:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: subfolder-secrets
  namespace: external-secrets
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: beyondtrustsecrets-ss
    kind: SecretStore
  target:
    name: subfolder-data
  dataFrom:
    - find:
        path: "eso/production"  # Override folder path
        name:
          regexp: ".*"           # Get all secrets in this folder
```

This will list all secrets in the `eso/production` folder, regardless of the `folderPath` configured in the SecretStore.

**Note:** The `path` field specifies a folder path, not a path to a specific secret. To fetch a single secret, use `data` with `extract` or individual `remoteRef` entries.

### Handling Source Secret Deletion

By default, when a source secret is deleted from BeyondTrust Secrets Manager, the managed Kubernetes secret is retained. You can change this behavior using `deletionPolicy`:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: secret-with-deletion
  namespace: external-secrets
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: beyondtrustsecrets-ss
    kind: SecretStore
  target:
    name: managed-secret
    deletionPolicy: Delete  # Delete the Kubernetes secret when source is removed
  data:
    - secretKey: myKey
      remoteRef:
        key: mySecret
```

Valid values:
- `Retain` (default): Keep the Kubernetes secret even if the source is deleted
- `Delete`: Remove the Kubernetes secret when the source is deleted

### Authentication

BeyondTrust Secrets Manager uses API key authentication. The API key is stored in a Kubernetes Secret and referenced in the SecretStore:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: beyondtrustsecrets-ss
  namespace: external-secrets
spec:
  provider:
    beyondtrustsecrets:
      auth:
        apikey:
          token:
            name: api-token  # Name of the Kubernetes Secret
            key: token            # Key within the Secret
      server:
        apiUrl: "https://api.beyondtrust.io/site"
        siteId: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      folderPath: "eso/static"
```

```yaml
auth:
  apikey:
    token:
      name: api-token
      key: token
```

### Server Configuration

The server configuration consists of:
- `apiUrl`: The base URL of your BeyondTrust Secrets Manager API
- `siteId`: Your BeyondTrust site identifier (UUID format)

The provider automatically constructs the full API endpoint as: `{apiUrl}/{siteId}/secrets`

### Certificate Trust

BeyondTrust Secrets Manager typically uses certificates signed by public CAs, requiring no additional configuration.

If using self-signed certificates, configure trust using either `caBundle` or `caProvider`:

#### Using caProvider (Recommended)

```yaml
spec:
  provider:
    beyondtrustsecrets:
      # ... other config ...
      caProvider:
        type: Secret
        name: my-ca-bundle
        key: ca.crt
        namespace: external-secrets  # Required for ClusterSecretStore
```

First create the CA bundle secret:
```bash
kubectl create secret generic my-ca-bundle \
  --from-file=ca.crt="/path/to/ca.crt" \
  -n external-secrets
```

#### Using caBundle

Alternatively, embed the base64-encoded PEM certificate directly:

```yaml
spec:
  provider:
    beyondtrustsecrets:
      # ... other config ...
      server:
        apiUrl: "https://api.beyondtrust.io/site"
        siteId: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      caBundle: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0t..."  # base64-encoded PEM
```

To generate the base64 string:
```bash
cat /path/to/ca.crt | base64 -w 0
```

### Folder Path

The `folderPath` specifies the default folder containing your secrets. This should be a folder path, not the full path to a specific secret.

For example, if your secrets are stored at:
- `eso/static/secret1`
- `eso/static/secret2`
- `eso/static/secret3`

Set `folderPath: "eso/static"` in your SecretStore.

When using `data` or `dataFrom.extract`, secret names are relative to this folder. When using `dataFrom.find`, this folder is searched by default (unless overridden with the `path` field).
### Refresh Interval
The `refreshInterval` controls how often the ExternalSecret checks for updates:

```yaml
spec:
  refreshInterval: 5m  # Check every 5 minutes
```

Supported units: `s` (seconds), `m` (minutes), `h` (hours).

**Best Practice:** Balance between keeping secrets up-to-date and minimizing API calls. For most use cases, `1m` to `15m` is appropriate.

### ClusterSecretStore

To use a ClusterSecretStore (accessible across all namespaces):

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: beyondtrustsecrets-css
spec:
  provider:
    beyondtrustsecrets:
      auth:
        apikey:
          token:
            name: api-token
            key: token
            namespace: external-secrets  # Required: specify where the token secret lives
      server:
        apiUrl: "https://api.beyondtrust.io/site"
        siteId: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      folderPath: "eso/static"
```

Reference it in an ExternalSecret:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-secret
  namespace: my-app
spec:
  secretStoreRef:
    name: beyondtrustsecrets-css
    kind: ClusterSecretStore  # Specify ClusterSecretStore
  # ... rest of spec
```