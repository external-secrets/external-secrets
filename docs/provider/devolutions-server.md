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
| `insecure` | (Optional) Set to `true` to allow plain HTTP connections. **Not recommended for production.** |
| `auth.secretRef.appId` | Reference to the secret containing the Application ID |
| `auth.secretRef.appSecret` | Reference to the secret containing the Application Secret |

**NOTE:** For `ClusterSecretStore`, ensure you specify the `namespace` in the secret references.

## Referencing Secrets

Secrets are referenced using the format: `<vault-id>/<entry-id>`

- **vault-id**: The UUID of the vault containing the entry
- **entry-id**: The UUID of the credential entry

You can find these UUIDs in the DVLS web interface by viewing the entry properties.

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
- **Name-based lookups**: Currently only UUID-based references (`vault-id/entry-id`) are supported. Path/name-based lookups are planned for future releases
- **Certificate entries**: Certificate entry types (`Document/Certificate`) are not currently supported. Only Credential entries are supported

## Troubleshooting

### Authentication Errors

If you receive authentication errors:

1. Verify the Application ID and Secret are correct
2. Ensure the application has the necessary permissions in DVLS
3. Check that the DVLS server URL is accessible from your Kubernetes cluster

### Entry Not Found

If an entry cannot be found:

1. Verify the vault UUID and entry UUID are correct
2. Ensure the application has at least read access to the vault
3. Check that the entry exists and is a Credential or Secret type entry
4. Ensure the application has at least read, view password, and connect (execute) permissions on the entry.
