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
| Account lookup | `svc-account/database-server` | Resolves the API key from retrievable accounts by account and asset name |
| Account ID lookup | `accountId:12345` | Resolves the API key from retrievable accounts by account ID |

`remoteRef.property` selects the credential type:

| Property | Description |
| --- | --- |
| `password` | Account password (default when omitted) |
| `privateKey` | SSH private key in OpenSSH format |
| `privateKey.Ssh2` | SSH private key in SSH2 format |
| `privateKey.Putty` | SSH private key in PuTTY format |
| `apiKey` | JSON array of API key credentials |
| `apiKey.clientId` | OAuth client ID from the first API key |
| `apiKey.clientSecret` | OAuth client secret from the first API key |
| `apiKey.<name>` | Client secret for the API key with the given name |

{% include 'safeguard-external-secret.yaml' %}

## PushSecret

Password write-back is supported when the A2A registration has bidirectional access enabled. Only `property: password` is supported for push operations.

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
