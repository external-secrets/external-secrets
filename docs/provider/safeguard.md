# One Identity Safeguard

External Secrets Operator can sync credentials from [One Identity Safeguard for Privileged Passwords (SPP)](https://www.oneidentity.com/products/safeguard-for-privileged-passwords/) using the official [`safeguard-go`](https://github.com/OneIdentity/safeguard-go) SDK.

The provider uses Safeguard Application-to-Application (A2A) authentication. Your application authenticates with a client certificate over mutual TLS and retrieves account credentials authorized by per-account API keys.

## Authentication

Configure A2A authentication on the `SecretStore`:

- `appliance`: Safeguard appliance hostname or `https://` URL
- `auth.a2a.certificate`: PEM client certificate
- `auth.a2a.certificateKey`: optional separate PEM private key
- `auth.a2a.certificatePassword`: optional password for encrypted private keys
- `caBundle` or `caProvider`: optional custom CA bundle for the appliance TLS certificate
- `apiVersion`: optional API version override (defaults to `v4`)

{% include 'safeguard-secret-store.yaml' %}

## External Secret Spec

`remoteRef.key` identifies the credential to retrieve. Supported formats:

| Key format | Example | Description |
| --- | --- | --- |
| A2A API key | `abc123...` | Direct API key value registered for the account |
| Account lookup | `svc-account/database-server` | Builds `AccountName ieq 'svc-account' and SystemName ieq 'database-server'` |
| OData filter | `filter:AccountName ieq 'x' and SystemName ieq 'y'` | Passes the filter directly to `RetrievableAccounts` |
| Account ID lookup | `accountId:12345` | Builds `AccountId eq 12345` |

`remoteRef.property` selects the credential type:

| Property | Description |
| --- | --- |
| `password` | Account password (default when omitted) |
| `privateKey` | SSH private key in OpenSSH format |
| `privateKey.Ssh2` | SSH private key in SSH2 format |
| `privateKey.Putty` | SSH private key in PuTTY format |
| `apiKey` | JSON array of API key credentials. When used with `dataFrom`/`GetSecretMap`, fields from the first API key are flattened into the target secret |
| `apiKey.clientId` | OAuth client ID from the first API key when multiple keys exist on the account |
| `apiKey.clientSecret` | OAuth client secret from the first API key when multiple keys exist on the account |
| `apiKey.<name>` | Client secret for the API key with the given name |

{% include 'safeguard-external-secret.yaml' %}

## Curl-equivalent workflow

If you currently use a workflow like:

1. Authenticate with a client certificate
2. `GET .../RetrievableAccounts?filter=AccountName ieq 'xxxx' and SystemName ieq 'yyyy'`
3. `GET .../Credentials?type=Password` with the returned `ApiKey`

you can configure ESO like this:

{% include 'safeguard-curl-equivalent-secret-store.yaml' %}

{% include 'safeguard-curl-equivalent-external-secret.yaml' %}

Equivalent shorthand for step 2:

```yaml
remoteRef:
  key: xxxx/yyyy
  property: password
```

Notes:

- Use `apiVersion: v3` when your appliance expects `/service/core/v3/` and `/service/a2a/v3/`.
- `SystemName` in Safeguard OData filters corresponds to the asset/system name in Safeguard.
- Account lookup uses Safeguard's `ieq` operator, so matching is case-insensitive.

## PushSecret

Password write-back is supported when the A2A registration has bidirectional access enabled. Only `property: password` is supported for push operations.

When PushSecret metadata specifies `filter` or both `accountName` and `systemName`, that metadata drives account lookup and `remoteRef.remoteKey` is ignored for discovery. `remoteRef.remoteKey` must still be set because ExternalSecret validates it.

PushSecret metadata can also drive account discovery:

```yaml
metadata:
  apiVersion: kubernetes.external-secrets.io/v1alpha1
  kind: PushSecretMetadata
  spec:
    filter: AccountName ieq 'xxxx' and SystemName ieq 'yyyy'
```

{% include 'safeguard-push-secret.yaml' %}

## Capabilities

| Feature | Supported |
| --- | --- |
| Read (GetSecret) | Yes |
| Write (PushSecret) | Password only |
| Delete | No |
| Find (GetAllSecrets) | No |

## References

- [safeguard-go SDK](https://github.com/OneIdentity/safeguard-go)
- [One Identity Safeguard documentation](https://docs.oneidentity.com/)
