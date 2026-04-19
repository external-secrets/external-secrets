## Netwrix Password Secure

External Secrets Operator integrates with [Netwrix Password Secure](https://netwrix.com/en/products/password-secure/) for secret management. The provider supports both read and write operations, including PushSecret and configurable deletion policies.

## Authentication

Netwrix Password Secure uses API Key authentication.

### Creating an API Key

1. Log in to your Netwrix Password Secure instance
2. Navigate to the API key management section
3. Generate a new API key for the user

### Storing the API Key

Create a Kubernetes Secret containing your NPWS API key:

```sh
kubectl create secret generic npws-api-key \
    --from-literal apiKey="{your-api-key}"
```

## SecretStore Configuration

### SecretStore

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: npws-secret-store
  namespace: default
spec:
  provider:
    npws:
      host: "https://npws.example.com/api"
      auth:
        secretRef:
          apiKey:
            name: npws-api-key
            key: apiKey
```

### ClusterSecretStore

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: npws-cluster-secret-store
spec:
  provider:
    npws:
      host: "https://npws.example.com/api"
      auth:
        secretRef:
          apiKey:
            name: npws-api-key
            key: apiKey
            namespace: external-secrets
```

**NOTE:** When using a `ClusterSecretStore`, always set `namespace` in `secretRef.apiKey` to reference the correct Kubernetes Secret.

## Fetching Secrets

### Behavior

- `remoteRef.key` is the Container ID (GUID) or the container's display name
- `remoteRef.property` is resolved in the following order:
    - Empty: returns the first password item (priority: Password > PasswordMemo > OTP)
    - GUID: matches by item ID
    - `type:<typename>` prefix: matches the first item of that type
    - Otherwise: matches by item name

### Fetch a Secret

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: npws-single-secret
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: npws-secret-store
    kind: SecretStore
  target:
    name: my-k8s-secret
  data:
    - secretKey: password
      remoteRef:
        key: "my-container-name"   # container display name or GUID
        property: "Password"        # item name, item ID, or type prefix
```


### Fetch a Secret - Complete Example

The following example demonstrates all `remoteRef.property` resolution modes against a single container called `"db-credentials"`. This assumes the container holds items such as a password, an OTP secret, an email field, and a memo — each accessible via a different property selector.

The `type:` prefix selects the first item of a specific type instead of matching by name. Supported types: `text`, `password`, `date`, `check`, `url`, `email`, `phone`, `list`, `memo`, `passwordmemo`, `int`, `decimal`, `username`, `ip`, `hostname`, `otp`.


```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: npws-complete-example
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: npws-secret-store
    kind: SecretStore
  target:
    name: db-credentials-secret
  data:

    # 1. key as container GUID instead of display name (preferred way)
    - secretKey: password-by-container-id
      remoteRef:
        key: "f0e1d2c3-b4a5-6789-0abc-def123456789" # Id of the secret

    # 2. property omitted → returns the first password item from container "db-credentials"
    #    (priority: Password > PasswordMemo > OTP)
    - secretKey: default-password
      remoteRef:
        key: "db-credentials" # Name of the secret

    # 3. property as item name → matches the item named "Password"
    - secretKey: password
      remoteRef:
        key: "db-credentials"
        property: "Password"  # Name of the field

    # 4. property as GUID → matches the item by its ID
    - secretKey: specific-item
      remoteRef:
        key: "db-credentials"
        property: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"  # Id of the field

    # 5. property with type: prefix → matches the first item of that type
    - secretKey: username
      remoteRef:
        key: "db-credentials"
        property: "type:username" # First item with field type: username

```

### Fetch All Fields from a Container

Use `dataFrom` with `extract` to retrieve all items from a container as key-value pairs. Header items are excluded automatically.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: npws-all-fields
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: npws-secret-store
    kind: SecretStore
  target:
    name: my-k8s-secret-map
  dataFrom:
    - extract:
        key: "my-container-name"
```

### Fetch Multiple Containers by Name

Use `dataFrom` with `find.name.regexp` to match containers by display name. The provider returns the first password value from each matching container.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: npws-find-secrets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: npws-secret-store
    kind: SecretStore
  target:
    name: my-k8s-bulk-secrets
  dataFrom:
    - find:
        name:
          regexp: "^prod-.*"
```

**NOTE:** Regex is evaluated client-side — all containers are fetched first. This can be slow on large databases.

## PushSecret

The NPWS provider supports pushing Kubernetes Secrets to Netwrix Password Secure.

- `remoteRef.remoteKey` is the container display name (creates a new container if it does not exist) or container ID (GUID, must already exist)
- `remoteRef.property` is the item name (creates a new item if it does not exist) or item ID (GUID, must already exist). If empty, defaults to an item named `"Password"`
- Updates are idempotent: if the value has not changed, no write is performed

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: npws-push-secret
spec:
  refreshInterval: 1h
  secretStoreRefs:
    - name: npws-secret-store
      kind: SecretStore
  selector:
    secret:
      name: my-k8s-source-secret
  data:
    - match:
        secretKey: password
        remoteRef:
          remoteKey: "my-container-name"
          property: "Password"
```

When pushing to a container that does not exist, the provider creates a new container with a `Name` text field (set to the container name) and the specified password field.

**NOTE:** The provider rejects any operation that would change a container's display name. This prevents accidental key mismatch scenarios.


## Deletion Policy

The `deletionPolicyWholeEntry` field on the SecretStore controls how secrets are deleted when using `deletionPolicy: Delete` on a PushSecret:

| `deletionPolicyWholeEntry` | Behavior |
|---|---|
| `false` (default) | Only the specified field is removed from the container |
| `true` | The entire container is deleted |

To enable whole-entry deletion:

```yaml
spec:
  provider:
    npws:
      host: "https://npws.example.com/api"
      deletionPolicyWholeEntry: true
      auth:
        secretRef:
          apiKey:
            name: npws-api-key
            key: apiKey
```

**NOTE:** When removing a single field, the provider automatically deletes the entire container if it would be left empty or with only the display-name field remaining. Deletion is idempotent — if the container or field is already missing, the operation succeeds silently.

## Important Notes

- **Client-side decryption:** All encrypted values (passwords, password memos, OTP secrets) are decrypted client-side
- **Session cleanup:** The provider properly logs out and cleans up the session after each reconciliation cycle.