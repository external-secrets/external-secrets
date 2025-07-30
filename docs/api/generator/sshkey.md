# SSHKey Generator

The SSHKey generator provides SSH key pairs that you can use for authentication in your applications. It supports generating RSA and Ed25519 keys with configurable key sizes and comments.

## Output Keys and Values

| Key        | Description                     |
| ---------- | ------------------------------- |
| privateKey | the generated SSH private key   |
| publicKey  | the generated SSH public key    |

## Parameters

| Parameter | Description                                                        | Default | Required |
| --------- | ------------------------------------------------------------------ | ------- | -------- |
| keyType   | SSH key type (rsa, ed25519, ecdsa, dsa)                            | rsa     | No       |
| keySize   | Key size for RSA keys (2048, 3072, 4096); ignored for ed25519      | 2048    | No       |
| comment   | Optional comment for the SSH key                                   | ""      | No       |

## Example Manifest

```yaml
{% include 'generator-sshkey.yaml' %}
```

Example `ExternalSecret` that references the SSHKey generator:

```yaml
{% include 'generator-sshkey-example.yaml' %}
```

This will generate a `Kind=Secret` with keys called 'privateKey' and 'publicKey' containing the SSH key pair.

## Supported Key Types

### RSA Keys

- Supports key sizes: 2048, 3072, 4096 bits
- Default key size: 2048 bits
- Good compatibility with older systems

### Ed25519 Keys

- Fixed key size (keySize parameter ignored)
- Modern, secure, and efficient
- Recommended for new deployments

## Security Considerations

- Generated keys are cryptographically secure using Go's crypto/rand
- Private keys are stored in OpenSSH format
- Keys are generated fresh on each reconciliation unless cached
- Consider key rotation policies for production use
