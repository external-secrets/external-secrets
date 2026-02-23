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

The `pass-cli` binary requires glibc, so the container must use a Debian-based image rather than Alpine:

```dockerfile
# Dockerfile for external-secrets with Proton Pass provider
# Requires pass-cli binary which needs glibc, so we use Debian slim instead of Alpine/distroless
# Build: docker buildx build --push --platform linux/arm64,linux/amd64 --tag external-secrets:protonpass --file Dockerfile.protonpass .

FROM golang:1.25.7-alpine@sha256:f6751d823c26342f9506c03797d2527668d095b0a15f1862cddb4d927a7a4ced AS builder
LABEL maintainer="cncf-externalsecretsop-maintainers@lists.cncf.io" \
      description="External Secrets Operator with Proton Pass provider"
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
WORKDIR /app
COPY . /app/
RUN go mod download
RUN go build -tags protonpass -o external-secrets main.go

# Download pass-cli from Proton's official distribution
FROM debian:bookworm-slim AS pass-cli-downloader
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /tmp
# Download from Proton's official CDN - update version as needed
# Check https://proton.me/download/pass-cli/versions.json for latest version
# When updating PASS_CLI_VERSION, recompute checksums:
#   sha256sum pass-cli-linux-x86_64 pass-cli-linux-aarch64
RUN PASS_CLI_VERSION="1.5.2" && \
    if [ "${TARGETARCH}" = "amd64" ]; then \
        ARCH="x86_64"; \
        EXPECTED_SHA256="b6e02ac79cee277767023dda21b6cea276d56fdb0bf85d96eaf022ff6227debc"; \
    elif [ "${TARGETARCH}" = "arm64" ]; then \
        ARCH="aarch64"; \
        EXPECTED_SHA256="ac88e2ebf15a4799c508408c0cf81dd9180313fe45b67941567ebd30c4fbadb2"; \
    else \
        echo "Unsupported architecture: ${TARGETARCH}" && exit 1; \
    fi && \
    curl -fsSL -o pass-cli "https://proton.me/download/pass-cli/${PASS_CLI_VERSION}/pass-cli-linux-${ARCH}" && \
    echo "${EXPECTED_SHA256}  pass-cli" | sha256sum -c - && \
    chmod +x pass-cli

# Final image - using Debian slim for glibc compatibility with pass-cli
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/external-secrets /bin/external-secrets
COPY --from=pass-cli-downloader /tmp/pass-cli /usr/local/bin/pass-cli

# Run as non-root user
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
```

Build with:

```bash
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
| `vault` | Name of the Proton Pass vault to use | Yes (recommended) |
| `auth.secretRef.password` | Reference to the password secret | Yes |
| `auth.secretRef.totp` | Reference to the TOTP secret for 2FA | No |
| `auth.secretRef.extraPassword` | Reference to an extra password | No |

!!! note "Vault Configuration"
    Specifying a `vault` is strongly recommended. The provider uses the vault name when fetching item details, and some operations fail without it.

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

#### "Please provide either --share-id or --vault-name"

Add the `vault` field to your SecretStore. Use the exact vault name as it appears in Proton Pass.

#### Item not found

1. Item names are case-sensitive
2. The item must be in the configured vault

#### CLI not found

Verify the container image includes `pass-cli` at `/usr/local/bin/pass-cli`.

#### Read-only filesystem errors

Mount an `emptyDir` volume at `/tmp`. The provider uses it for session storage.
