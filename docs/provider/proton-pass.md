## Proton Pass

External Secrets Operator integrates with [Proton Pass](https://proton.me/pass) to sync
items from your Proton Pass vaults into Kubernetes Secrets.

The provider is pure Go and talks directly to the Proton Pass HTTP API — there is no CLI,
sidecar, or external process. It authenticates with a Proton Pass **Personal Access Token
(PAT)** and uses it to unwrap the vault, item, and content keys locally (AES-256-GCM); no
account password, SRP login, or PGP key is involved.

### Behavior

* How a Proton Pass **item** maps to an `ExternalSecret`:
    * `remoteRef.key` is the item **title**, or `id:<ItemID>` to address an item by its
      immutable ID.
    * `remoteRef.property` is a **field label** within the item. It defaults to `password`.
      The reserved property `totp` returns the current one-time-password code (see
      [TOTP](#totp)).
    * `remoteRef.version` is not supported.
* An item's fields are surfaced as a flat label→value map: the typed fields of the item
  (e.g. a login's `username`/`password`/`url`, a credit card's `number`, …) plus every
  custom field. The TOTP seed is never included in this map.
* Ambiguity is a hard error, never a silent pick. If a title matches more than one item
  across the in-scope vaults, the lookup fails and asks you to use `id:<ItemID>`.
* **Group-shared vaults are skipped.** Their share key is wrapped to keys a PAT cannot
  access, so they are invisible to this provider rather than half-broken.
* `dataFrom`:
    * `find.name.regexp` matches item **titles**; each match contributes its default field
      (`password`).
    * `find.path` optionally restricts the search to a single vault, by vault name.
    * `find.tags` is not supported (Proton Pass items have no tags).

### Authentication

Create a Personal Access Token in the Proton Pass app or CLI. A token has the form
`pst_<token>::<key>`. The token's role determines the store's capabilities:

* a **viewer** token yields a read-only store;
* an **editor** or **manager** token additionally enables `PushSecret`.

Store the full token string in a Kubernetes Secret and reference it from a `SecretStore`
(or `ClusterSecretStore`):

```yaml
{% include 'proton-pass-secret-store.yaml' %}
```

> **NOTE:** When using a `ClusterSecretStore`, set `namespace` on
> `auth.personalAccessTokenSecretRef` — this provider does not support referent
> authentication.

### Vault scope

By default the store uses every vault the token can access. Set `spec.provider.protonpass.vaults`
to an allow-list of vault names to narrow that set. Because an ambiguous title is a hard
error, the list does not assign priorities; resolve a deliberately-duplicated title with
`id:<ItemID>` instead.

Writes (`PushSecret`/`DeleteSecret`) require the scope to resolve to exactly **one** writable
vault. If the token can write to several vaults, set `vaults` to a single one.

### Fetching secrets

Fetch individual fields with `data`, an entire item with `dataFrom.extract`, or a set of
items discovered by title with `dataFrom.find`:

```yaml
{% include 'proton-pass-external-secret.yaml' %}
```

### TOTP

For a login item that has TOTP configured, the reserved property `totp` returns the
**current** RFC 6238 code, regenerated on each reconcile:

```yaml
data:
  - secretKey: mfa-code
    remoteRef:
      key: my-login
      property: totp
```

The underlying `otpauth://` seed is deliberately never returned and never appears in
`GetSecretMap`/`dataFrom` output. The seed is a long-lived secret (possession allows
indefinite code generation), so the provider exposes only the short-lived derived code.

### PushSecret

With an editor/manager token you can push a Kubernetes Secret value into Proton Pass. If no
item with the target title exists, a new item is created with the value stored as a hidden
field; otherwise the named field on the existing item is updated:

```yaml
{% include 'proton-pass-push-secret.yaml' %}
```

`match.secretKey` (the source key to push) is required — pushing an entire Secret at once is
not supported. `remoteRef.remoteKey` is the item title and `remoteRef.property` is the field
label (default `password`). With `deletionPolicy: Delete`, removing the `PushSecret` deletes
the pushed field; if no `property` is given, the whole item is moved to trash.

> **NOTE:** Writes operate on an item's top-level fields. A field that Proton Pass stores
> inside a custom-item *section* is readable, but pushing or deleting it by that label is
> refused with a clear error rather than silently creating a duplicate.

### Secret key conversion

Field labels are returned as-is; the operator applies your `ExternalSecret`
`conversionStrategy` (e.g. `Default` or `Unicode`) when converting them into valid
Kubernetes Secret keys.
