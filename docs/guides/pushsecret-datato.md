# PushSecret dataTo

The `dataTo` field in PushSecret enables bulk pushing of secrets without requiring explicit
per-key configuration. Instead of listing every key manually in `data`, you point `dataTo` at a
store and optionally filter or transform the keys that get pushed.

## Overview

`dataTo` supports two distinct modes. Which one to use depends entirely on your **provider's
secret model**:

| Mode | When to use | `remoteKey` |
|---|---|---|
| **Per-key** | Provider uses one named variable/entry per secret (GitHub Actions, Doppler) | not set |
| **Bundle** | Provider stores structured config as a single named secret (AWS SM, Azure KV, GCP SM, Vault) | **required** |

## Choosing the right mode

### Per-key mode (env-var providers)

Providers like **GitHub Actions** and **Doppler** model secrets as individual named
variables — each key in your Kubernetes Secret maps to exactly one variable in the provider.
Do **not** set `remoteKey` in this case; the key names themselves become the provider variable names.

```yaml
# GitHub Actions / Doppler — one variable per key
dataTo:
  - storeRef:
      name: github-store
    # no remoteKey — each K8s key becomes its own GitHub secret
    match:
      regexp: "^APP_"
```

Result in GitHub Actions (assuming the K8s Secret has `APP_TOKEN` and `APP_ENV`):
```
APP_TOKEN → value of APP_TOKEN
APP_ENV   → value of APP_ENV
```

### Bundle mode (named-secret providers)

Providers like **AWS Secrets Manager**, **Azure Key Vault**, **GCP Secret Manager**, and
**HashiCorp Vault** model secrets as a single named object that holds a JSON payload. Use
`remoteKey` to name that object — all matched keys are bundled into it as a JSON object.

```yaml
# AWS SM / Azure KV / GCP SM / Vault — all keys → one named secret
dataTo:
  - storeRef:
      name: aws-store
    remoteKey: my-app/config    # the AWS Secrets Manager secret name
    match:
      regexp: "^DB_"
```

Result in AWS Secrets Manager:
```
my-app/config → {"DB_HOST":"localhost","DB_USER":"admin","DB_PASS":"s3cr3t"}
```

!!! warning "Without `remoteKey` on named-secret providers"
    If you omit `remoteKey` on a provider like AWS Secrets Manager, `dataTo` falls back to
    per-key mode and creates **one AWS secret per matched key**
    (`DB_HOST`, `DB_USER`, `DB_PASS` each become separate secrets).
    This is rarely what you want on AWS — always set `remoteKey` when targeting AWS SM,
    Azure KV, GCP SM, or Vault.

## Provider reference

| Provider | Secret model | Use `remoteKey`? | Notes |
|---|---|---|---|
| AWS Secrets Manager | Named secret (JSON) | **Yes** | `remoteKey` = secret name; store `prefix` is prepended |
| AWS Parameter Store | Named parameter | **Yes** | `remoteKey` = parameter path |
| Azure Key Vault | Named secret/key/cert | **Yes** | `remoteKey` = object name |
| GCP Secret Manager | Named secret | **Yes** | `remoteKey` = secret ID |
| HashiCorp Vault | Named path (JSON) | **Yes** | `remoteKey` = Vault path |
| Oracle Vault | Named secret | **Yes** | `remoteKey` = secret name |
| Kubernetes | Named secret | **Yes** | `remoteKey` = target Secret name |
| Bitwarden | Named item | **Yes** | `remoteKey` = item key |
| GitHub Actions | Env-var (one per key) | **No** | Key name = Actions secret name |
| Doppler | Env-var (one per key) | **No** | Key name = Doppler variable name |
| Webhook | Configurable | Depends | Check your webhook implementation |

## Examples by provider

### AWS Secrets Manager

!!! warning "Prefix + remoteKey = concatenated name"
    The AWS SecretStore `prefix` is **prepended** to every `remoteKey`. If your store has
    `prefix: myapp/` and your `dataTo` has `remoteKey: db-config`, the resulting AWS secret
    name is `myapp/db-config` — not `db-config`.

    A common mistake is setting `prefix: secrets-sync-temp/` and `remoteKey: secrets-sync-temp`,
    which creates `secrets-sync-temp/secrets-sync-temp` — not `secrets-sync-temp`.
    If you want the secret name to be exactly `secrets-sync-temp`, either remove the prefix
    from the store or set `remoteKey` to the suffix portion only.

!!! tip "Make the value visible in the AWS Console"
    By default ESO stores secret values as **binary** (`SecretBinary`). The AWS Console
    may show binary secrets as blank or unreadable. Add `secretPushFormat: string` to the
    `metadata` to store the JSON as a readable `SecretString` instead.

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: aws-store
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      # No prefix — remoteKey is the full secret name.
      # If you add a prefix, the final name is: prefix + remoteKey.
---
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-to-aws
spec:
  secretStoreRefs:
    - name: aws-store
      kind: SecretStore
  selector:
    secret:
      name: app-secrets    # K8s Secret with DB_HOST, DB_USER, DB_PASS
  dataTo:
    - storeRef:
        name: aws-store
      remoteKey: my-app/db-config   # → AWS secret named exactly "my-app/db-config"
      match:
        regexp: "^DB_"
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          secretPushFormat: string    # store as SecretString (readable in console)
```

Result in AWS Secrets Manager:
```
my-app/db-config → {"DB_HOST":"localhost","DB_USER":"admin","DB_PASS":"s3cr3t"}
```

!!! warning "Metadata requires the full PushSecretMetadata wrapper"
    The `metadata` field is not a plain key-value map. It must be a valid
    `PushSecretMetadata` object with `apiVersion`, `kind`, and `spec`. Putting
    `secretPushFormat: string` directly under `metadata:` will cause a parse error.

**With a store prefix:**

```yaml
# SecretStore has prefix: myapp/
# dataTo remoteKey: db-config
# → AWS secret name: myapp/db-config
```

### Azure Key Vault

```yaml
dataTo:
  - storeRef:
      name: azure-store
    remoteKey: app-db-config    # Azure Key Vault secret name
    match:
      regexp: "^DB_"
```

### GCP Secret Manager

```yaml
dataTo:
  - storeRef:
      name: gcp-store
    remoteKey: projects/my-project/secrets/app-db-config
    match:
      regexp: "^DB_"
```

### HashiCorp Vault

```yaml
dataTo:
  - storeRef:
      name: vault-store
    remoteKey: secret/data/myapp/db    # Vault path (KV v2 style)
    match:
      regexp: "^DB_"
```

### GitHub Actions

```yaml
dataTo:
  - storeRef:
      name: github-store
    # No remoteKey — each K8s key becomes its own Actions secret
    match:
      regexp: "^DEPLOY_"
```

Result: individual GitHub Actions secrets named `DEPLOY_TOKEN`, `DEPLOY_ENV`, etc.

### Doppler

```yaml
dataTo:
  - storeRef:
      name: doppler-store
    # No remoteKey — each K8s key becomes its own Doppler variable
```

## Filtering with `match`

Use `match.regexp` to push only a subset of keys. When omitted, all keys are included.

```yaml
dataTo:
  - storeRef:
      name: aws-store
    remoteKey: myapp/db-secrets
    match:
      regexp: "^DB_"      # only keys starting with DB_
```

```yaml
dataTo:
  - storeRef:
      name: aws-store
    remoteKey: myapp/all-secrets
    # no match → all keys in the source Secret
```

## Key transformations with `rewrite`

`rewrite` only applies in **per-key mode** (no `remoteKey`). It transforms the key name before it
becomes the provider variable/secret name. Two rewrite types are available:

### Regexp rewrite

```yaml
dataTo:
  - storeRef:
      name: github-store
    match:
      regexp: "^db-"
    rewrite:
      - regexp:
          source: "^db-"
          target: "DATABASE_"   # db-host → DATABASE_host
```

### Template rewrite

{% raw %}
```yaml
dataTo:
  - storeRef:
      name: github-store
    rewrite:
      - transform:
          template: "{{ .value | upper }}"   # db-host → DB-HOST
```
{% endraw %}

### Chained rewrites

Multiple rewrites are applied in order — each sees the output of the previous:

```yaml
dataTo:
  - storeRef:
      name: github-store
    match:
      regexp: "^prod-db-"
    rewrite:
      - regexp: {source: "^prod-", target: ""}        # prod-db-host → db-host
      - regexp: {source: "^db-", target: "DATABASE_"} # db-host      → DATABASE_host
```

!!! tip "Rewrites are ignored in bundle mode"
    When `remoteKey` is set, key names are not used as provider paths — only their values
    appear in the JSON object. Rewrite entries are silently ignored in this case.

## Multiple `dataTo` entries

Split matched keys across different targets in the same PushSecret:

```yaml
# AWS: two separate secrets, each scoped to a category
dataTo:
  - storeRef:
      name: aws-store
    remoteKey: myapp/database
    match:
      regexp: "^DB_"
  - storeRef:
      name: aws-store
    remoteKey: myapp/api
    match:
      regexp: "^API_"
```

```yaml
# GitHub: separate env-var groups pushed to different stores
dataTo:
  - storeRef:
      name: github-prod-store
    match:
      regexp: "^PROD_"
  - storeRef:
      name: github-staging-store
    match:
      regexp: "^STAGING_"
```

## Combining `dataTo` with explicit `data`

Explicit `data` entries **always override** `dataTo` for the same source key. Use this to apply
bulk defaults and then carve out exceptions:

```yaml
spec:
  dataTo:
    - storeRef:
        name: aws-store
      remoteKey: myapp/config       # all keys bundled here by default
  data:
    - match:
        secretKey: MASTER_PASSWORD
        remoteRef:
          remoteKey: myapp/security/master-password   # this key gets its own secret
```

## Conversion strategy

`conversionStrategy: ReverseUnicode` decodes Unicode-escaped key names before matching and
pushing. Applied before `match` and `rewrite`:

```yaml
dataTo:
  - storeRef:
      name: aws-store
    remoteKey: myapp/config
    conversionStrategy: ReverseUnicode
```

## Error handling

| Situation | Behavior |
|---|---|
| Invalid regexp in `match` | PushSecret enters error state; check `.status.conditions` |
| Rewrite produces empty key | Reconciliation fails with the offending source key named |
| Two entries produce the same remote key | Reconciliation fails listing all conflicting sources |
| `match` matches no keys | Not an error; info log, PushSecret stays Ready |
| `storeRef` not in `secretStoreRefs` | Validation error on apply |

## Best practices

1. **Always set `remoteKey` for named-secret providers** (AWS SM, Azure KV, GCP SM, Vault) — omitting it creates one secret per key, which is almost never what you want on these providers
2. **Never set `remoteKey` for env-var providers** (GitHub Actions, Doppler) — the key name IS the variable name
3. **Filter before you bundle** — use `match.regexp` to be explicit about which keys end up in a bundle; avoids accidentally including sensitive keys
4. **Test patterns first** — inspect your source Secret's keys before writing patterns:
   ```bash
   kubectl get secret my-secret -o jsonpath='{.data}' | jq 'keys'
   ```
5. **Combine with `data` for exceptions** — use `dataTo` for the common case, explicit `data` entries for keys that need custom paths or properties
6. **Monitor status** — check `kubectl get pushsecret <name> -o yaml` for sync errors

## See Also

- [PushSecret Guide](pushsecrets.md) - Basic PushSecret usage
- [PushSecret API Reference](../api/pushsecret.md) - Complete API specification
- [Templating Guide](templating.md) - Advanced template usage
- [ExternalSecret dataFrom](datafrom-rewrite.md) - The mirror image: pulling secrets from providers
