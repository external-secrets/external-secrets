## Devolutions Server (DVLS)

External Secrets Operator integrates with [Devolutions Server](https://devolutions.net/server/) (DVLS) for secret management.

DVLS is a self-hosted privileged access management solution that provides secure password management, role-based access control, and credential injection for teams and enterprises.

## Authentication

DVLS authentication uses Application ID and Application Secret credentials.

### Creating an Application in DVLS

1. Log into your DVLS web interface
2. Navigate to **Administration > Applications identities**
3. Click **+ (Add)** to create a new application
4. Configure the application with appropriate permissions to access the vaults and entries you need
5. Save the **Application ID** and **Application Secret**

### Creating the Kubernetes Secret

Create a Kubernetes secret containing your DVLS credentials:

```bash
kubectl create secret generic dvls-credentials \
  --from-literal=app-id="your-application-id" \
  --from-literal=app-secret="your-application-secret"
```

### Creating a SecretStore

```yaml
{% include 'dvls-secret-store.yaml' %}
```

| Field | Description |
|-------|-------------|
| `serverUrl` | The URL of your DVLS instance (e.g., `https://dvls.example.com`) |
| `vault` | (Optional) The name or UUID of the vault to fetch secrets from. When omitted, the vault must be specified in the secret key using the legacy `<vault-id>/<entry-id>` format. |
| `insecure` | (Optional) Set to `true` to allow plain HTTP connections. **Not recommended for production.** |
| `auth.secretRef.appId` | Reference to the secret containing the Application ID |
| `auth.secretRef.appSecret` | Reference to the secret containing the Application Secret |

**NOTE:** For `ClusterSecretStore`, ensure you specify the `namespace` in the secret references.

## Referencing Secrets

Entries can be referenced by **UUID** or **name**:

| Format | Example |
|--------|---------|
| Entry UUID | `7c9e6679-7425-40de-944b-e07fc1f90ae7` |
| Entry name | `db-credentials` |
| Entry name with folder path | `infrastructure/databases/db-credentials` |
| Folder path with backslashes | `infrastructure\databases\db-credentials` |

The vault is configured in the SecretStore's `vault` field (name or UUID), so the key only needs to identify the entry.

### Folder paths

If an entry is inside a folder, you can include the folder path before the entry name. Both forward slashes (`/`) and backslashes (`\`) are accepted as path separators:

```text
folder/subfolder/entry-name
folder\subfolder\entry-name
```

**Note:** When using backslashes in YAML, you must escape them with a double backslash (`\\`):

```yaml
key: "folder\\subfolder\\entry-name"
```

Forward slashes do not need escaping and are recommended for simplicity.

**Important:** Entry names containing forward slashes (`/`) or backslashes (`\`) are not supported with name-based lookups, as those characters are interpreted as path separators. Use the entry UUID instead.

The folder path is **optional**. Without a path, the provider searches across all folders in the vault. If multiple entries share the same name in different folders, you can either specify the folder path or use the entry UUID to disambiguate.

**Name-based lookups** resolve the name to a UUID at runtime via an API call. If multiple credential entries match, an error is returned. For write-heavy scenarios (frequent `PushSecret` operations), prefer UUID references to avoid the extra lookup per operation.

You can find UUIDs in the DVLS web interface by viewing the entry properties.

## Supported Credential Types

DVLS supports multiple credential types. The provider maps each type to specific properties:

| Credential Type | DVLS Entry Type | Available Properties |
|-----------------|-----------------|---------------------|
| **Default** | Credential | `username`, `password`, `domain` |
| **Access Code** | Secret | `password` |
| **API Key** | Credential | `api-id`, `api-key`, `tenant-id` |
| **Azure Service Principal** | Credential | `client-id`, `client-secret`, `tenant-id` |
| **Connection String** | Credential | `connection-string` |
| **Private Key** | Credential | `username`, `password`, `private-key`, `public-key`, `passphrase` |

All entries also include `entry-id` and `entry-name` metadata properties.

**Note:** When no `property` is specified, the `password` field is returned by default.

**Note:** In the DVLS web interface, "Secret" entries appear as a distinct entry type and are mapped to the Access Code credential subtype internally.

## Examples

### Fetching Individual Properties

To fetch specific properties from a credential entry:

```yaml
{% include 'dvls-external-secret.yaml' %}
```

### Using dataFrom to Extract All Fields

When using `dataFrom.extract`, all available properties from the credential entry will be synced to the Kubernetes secret.

## Push Secrets

The DVLS provider supports pushing secrets back to DVLS:

```yaml
{% include 'dvls-push-secret.yaml' %}
```

**Note:** Push secret updates an existing entry's password field. The entry must already exist in DVLS.

## Limitations

- **GetAllSecrets**: The `find` operation for discovering secrets is not currently supported
- **Custom CA Certificates**: Custom TLS certificates for self-signed DVLS instances are not yet supported. Use the `SSL_CERT_FILE` environment variable as a workaround
- **Certificate entries**: Certificate entry types (`Document/Certificate`) are not currently supported. Only Credential entries are supported

## Troubleshooting

### Authentication Errors

If you receive authentication errors:

1. Verify the Application ID and Secret are correct
2. Ensure the application has the necessary permissions in DVLS
3. Check that the DVLS server URL is accessible from your Kubernetes cluster

### Entry Not Found

If an entry cannot be found:

1. Verify the vault and entry references are correct (UUID or name)
2. Ensure the application has at least read access to the vault
3. Check that the entry exists and is a Credential or Secret type entry
4. Ensure the application has at least read, view password, and connect (execute) permissions on the entry

### Multiple Entries Found

If you receive a "multiple entries found" error when using name-based references, it means more than one credential entry shares the same name in the vault. Use the entry UUID instead of the name to target the correct entry.
