## Proton Pass

[Proton Pass](https://proton.me/pass) is an end-to-end encrypted password manager from Proton. This provider uses the `pass-cli` command-line tool to retrieve secrets from Proton Pass vaults.

!!! warning "Experimental"
    This provider is experimental and requires a custom Docker image with the `pass-cli` binary bundled.

### Prerequisites

1. A Proton account with Proton Pass enabled
2. The `pass-cli` binary installed in your ESO container
3. Credentials stored in a Kubernetes Secret
4. A writable volume for CLI session storage

### Building a Custom Image

The `pass-cli` binary requires glibc, so the container must use a Debian-based image rather than Alpine/distroless. The `Dockerfile.protonpass` in the repository expects a pre-built binary at `bin/external-secrets-${TARGETOS}-${TARGETARCH}`.

Build the Go binaries first, then build the container image:

```bash
# Build the container image
docker buildx build --push --platform linux/arm64,linux/amd64 \
    --tag your-registry/external-secrets:protonpass \
    --file Dockerfile.protonpass .
```

### Deployment Requirements

The provider needs a writable filesystem for session data. Add an `emptyDir` volume:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-secrets
spec:
  template:
    spec:
      containers:
      - name: external-secrets
        # ... other settings ...
        volumeMounts:
        - name: tmp
          mountPath: /tmp
      volumes:
      - name: tmp
        emptyDir: {}
```

Helm chart equivalent:

```yaml
extraVolumes:
  - name: tmp
    emptyDir: {}

extraVolumeMounts:
  - name: tmp
    mountPath: /tmp
```

### Session Cache

!!! warning "Experimental"
    The session cache is an experimental feature. Its flags and behavior may change in future releases.

By default, the provider logs in to Proton Pass on every reconciliation. This is slow and can fail with TOTP-based accounts because each login consumes a time-based code.

With the session cache enabled, authenticated sessions are kept in an in-memory LRU cache keyed by SecretStore identity. Cached sessions are evicted when the SecretStore's `resourceVersion` changes or when the cache reaches its size limit.

Flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--experimental-enable-protonpass-session-cache` | `false` | Enable the session cache |
| `--experimental-protonpass-session-cache-size` | `100` | Maximum number of cached sessions (LRU) |

Example Helm values:

```yaml
extraArgs:
  experimental-enable-protonpass-session-cache: true
  experimental-protonpass-session-cache-size: 50
```

On eviction the provider logs out and removes the on-disk session directory.

!!! note
    Item lists are still refreshed on every reconciliation, so changes in Proton Pass are picked up without restarting ESO. Only the login session is reused.

### Authentication

Proton Pass supports three authentication methods:

1. **Password only** - no 2FA
2. **Password + TOTP** - TOTP-based 2FA
3. **Password + Extra Password** - extra password configured in Proton

#### Setting up TOTP Secret

If your account uses TOTP 2FA, provide the TOTP secret (not a generated code). The provider generates codes automatically.

To get your TOTP secret:

1. When setting up 2FA in Proton, copy the secret key (a base32 string like `JBSWY3DPEHPK3PXP`)
2. Store it in your Kubernetes Secret

If you already have 2FA configured, some authenticator apps (Aegis, andOTP) can export the secret. You can also decode the QR code with `zbarimg`:
  ```bash
  # If you have the QR code image
  zbarimg -q --raw qrcode.png
  # Output: otpauth://totp/Proton:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Proton
  # The secret is the value after "secret="
  ```

!!! note
    Use the raw base32 secret key, not the full `otpauth://` URI.

### Creating the Credentials Secret

Create a Secret with your Proton credentials:

```yaml
{% include 'protonpass-credentials-secret.yaml' %}
```

For accounts without 2FA, you can omit the `totp-secret` key.

### SecretStore Configuration

```yaml
{% include 'protonpass-secret-store.yaml' %}
```

| Field | Description | Required |
|-------|-------------|----------|
| `username` | Your Proton account email | Yes |
| `vault` | Name of the Proton Pass vault to use | Yes |
| `auth.secretRef.password` | Reference to the password secret | Yes |
| `auth.secretRef.totp` | Reference to the TOTP secret for 2FA | No |
| `auth.secretRef.extraPassword` | Reference to an extra password | No |

### Fetching Secrets

#### GetSecret

Use the `key` field to specify the item name and optionally a field:

```yaml
{% include 'protonpass-external-secret.yaml' %}
```

Key format: `<itemName>` or `<itemName>/<fieldName>`

- If only the item name is provided, the `password` field is returned by default
- Use `property` to override the field name

Available fields depend on the item type:

**All item types:**

- `note` - Notes attached to the item
- Any extra fields added to the item (matched by field name)

**Login items:**

- `password` (default)
- `username`
- `email`
- `totpSecret`

#### GetSecretMap (dataFrom)

To fetch all fields from an item as key-value pairs:

```yaml
{% include 'protonpass-external-secret-datafrom.yaml' %}
```

This creates a Secret with all available fields from the item.

#### GetAllSecrets (find)

To fetch multiple secrets matching a pattern:

```yaml
{% include 'protonpass-external-secret-find.yaml' %}
```

This retrieves the password field from all **login** items matching the name pattern. Note items are skipped.

### Limitations

- **Read-only**: PushSecret is not supported.
- **Item types**: Only Login and Note items are supported. Other Proton Pass item types are not currently implemented.
- **GetAllSecrets**: Only returns login items with a password. Use GetSecret or GetSecretMap for note items.

### Troubleshooting

#### Login failures (422 Unprocessable Entity)

If authentication fails with a 422 error:

1. **TOTP timing**: Ensure the container's clock is synchronized. TOTP codes are time-sensitive.
2. **Rate limiting**: If you've made many login attempts, wait a few minutes before retrying.
3. **TOTP secret format**: Ensure you're providing the base32 TOTP secret, not a generated code.

#### Login failures (401 Unauthorized)

1. Verify your password is correct
2. Check if your account has been temporarily locked due to failed attempts
3. Ensure the credentials secret is correctly referenced

#### Item not found

1. Item names are case-sensitive
2. The item must be in the configured vault

#### CLI not found

Verify the container image includes `pass-cli` on the PATH somewhere.

#### Read-only filesystem errors

Mount an `emptyDir` volume at `/tmp`. The provider uses it for session storage.
