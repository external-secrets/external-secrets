# Proton Pass

The Proton Pass provider syncs secrets from a [Proton Pass](https://proton.me/pass) vault into Kubernetes `Secrets`.

It is a **read-only** provider that authenticates with a Proton Pass **Personal Access Token (PAT)** and talks directly to the Proton Pass HTTP API — no CLI, sidecar, or exec is required.

## Authentication

Proton Pass authenticates via a Personal Access Token. Create one with the `pass-cli` [`pat create`](https://protonpass.github.io/pass-cli/commands/personal-access-token/) command, grant it access to the vault(s) you want to sync, and store the resulting `pst_...::...` token in a Kubernetes `Secret`.

{% include 'proton-pass-token-secret.yaml' %}

> The PAT is `pst_<token>::<key>`. The key half is a 32-byte AES key used locally to decrypt your vault contents; it is never sent to Proton. Keep it secret.

### Scope notes

- **Group-shared vaults are skipped.** A PAT cannot unwrap their share keys (they are PGP-wrapped to user/group keys the PAT lacks), so they are invisible rather than half-broken.
- Sessions are cached per account to avoid Proton's per-account login rate limit.

## Store Configuration

{% include 'proton-pass-secret-store.yaml' %}

| Field | Description | Required |
|-------|-------------|----------|
| `auth.personalAccessTokenSecretRef` | A secret key selector pointing at the PAT (`pst_...::...`). | Yes |

## External Secret Spec

Address items by their **title**, or by `id:<ItemID>` for an unambiguous reference. The default property is `password`.

{% include 'proton-pass-external-secret.yaml' %}

### Addressing

- `ref.key` is the item **title** (e.g. `Database`). An ambiguous title (multiple matches across accessible vaults) is a hard error — use `id:<ItemID>` instead.
- `id:<ItemID>` resolves a single item by its Proton Pass Item ID.
- `ref.property` selects a single field. When omitted, it defaults to `password`.

### Available fields (Login items)

The provider projects the decrypted item into a flat key/value map:

| Key | Source |
|-----|--------|
| `title` | Item metadata name |
| `note` | Item metadata note |
| `username` | Login `item_username` |
| `password` | Login `password` (default property) |
| `email` | Login `item_email` |
| `url` | Login `urls` (joined with `,`) |
| *(custom)* | Each extra field (text/hidden) is surfaced by its field name, even when empty |

Use `dataFrom.extract` to pull every field of an item into a `Secret`.

## Capabilities

This provider is **read-only**: `PushSecret`, `DeleteSecret`, `SecretExists`, and `GetAllSecrets` (find) are not supported and return an error. `Capabilities()` reports `ReadOnly`.

## Limitations

- Only **Login** items are projected (alias, credit-card, identity, SSH-key, Wi-Fi, and custom items are decoded at the protobuf level but not projected).
- No TOTP code generation — TOTP seeds are never surfaced.
- No `find`/`GetAllSecrets` support.
