# PushSecret dataTo

The `dataTo` field in PushSecret enables bulk pushing of secrets without requiring explicit per-key configuration. This feature is particularly useful when you need to push multiple related secrets and want to avoid verbose, repetitive YAML configurations.

## Overview

Instead of manually mapping each secret key individually in the `data` field, `dataTo` allows you to:

- **Match secrets by pattern**: Use regexp to select keys from your source Secret
- **Transform keys**: Apply regexp or template-based transformations to key names
- **Bulk push**: Push multiple secrets with minimal configuration
- **Combine with explicit data**: Override specific keys while using dataTo for the rest

## Basic Usage

### Push All Keys

Push all keys from the source Secret without any transformation:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-all-keys
spec:
  secretStoreRefs:
    - name: my-secret-store
  selector:
    secret:
      name: source-secret
  dataTo:
    - storeRef:
        name: my-secret-store  # Required: specify target store
```

This will push all keys from `source-secret` to the provider with their original names.

!!! note "storeRef is required"
    Each `dataTo` entry must specify a `storeRef` to target a specific store. This prevents
    accidental "apply to all stores" behavior which could break multi-provider setups.

### Match Keys with Pattern

Use regexp to select specific keys:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-db-secrets
spec:
  secretStoreRefs:
    - name: my-secret-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^db-.*"  # Only push keys starting with "db-"
```

If `app-secrets` contains:
```yaml
db-host: localhost
db-port: 5432
db-password: secret123
api-key: xyz789
```

Only `db-host`, `db-port`, and `db-password` will be pushed.

## Key Transformations

### Regexp Rewrite

Transform key names using regular expressions:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-with-prefix
spec:
  secretStoreRefs:
    - name: aws-secret-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    - storeRef:
        name: aws-secret-store
      match:
        regexp: "^db-.*"
      rewrite:
        - regexp:
            source: "^db-"
            target: "myapp/database/"
```

**Transformation example:**
- `db-host` → `myapp/database/host`
- `db-port` → `myapp/database/port`
- `db-password` → `myapp/database/password`

### Template Rewrite

Use Go templates with sprig functions for advanced transformations:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-with-template
spec:
  secretStoreRefs:
    - name: vault-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    - storeRef:
        name: vault-store
      rewrite:
        - transform:
            template: "app/{{ .value | upper }}"
```

**Transformation example:**
- `username` → `app/USERNAME`
- `password` → `app/PASSWORD`
- `api-key` → `app/API-KEY`

Available template functions include all [sprig functions](https://masterminds.github.io/sprig/) like `upper`, `lower`, `replace`, `trim`, etc.

### Chained Rewrites

Apply multiple transformations sequentially:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-chained-rewrites
spec:
  secretStoreRefs:
    - name: my-secret-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^db-.*"
      rewrite:
        - regexp:
            source: "^db-"
            target: ""  # First: remove "db-" prefix
        - regexp:
            source: "^"
            target: "prod/"  # Second: add "prod/" prefix
```

**Transformation example:**
- `db-host` → `host` → `prod/host`
- `db-port` → `port` → `prod/port`

Rewrites are applied in order, with each rewrite seeing the output of the previous one.

## Combining dataTo with Explicit Data

You can use both `dataTo` and `data` fields together. Explicit `data` entries override `dataTo` for the same source key:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-with-override
spec:
  secretStoreRefs:
    - name: my-secret-store
  selector:
    secret:
      name: app-secrets
  # Push all keys with original names
  dataTo:
    - storeRef:
        name: my-secret-store
  # But override how db-host is pushed
  data:
    - match:
        secretKey: db-host
        remoteRef:
          remoteKey: custom/database/hostname
```

Result:
- `db-host` → `custom/database/hostname` (from explicit `data`)
- All other keys → original names (from `dataTo`)

This is useful when you want bulk behavior for most keys but need custom handling for specific ones.

## Metadata and Conversion Strategy

You can apply metadata and conversion strategy to all matched secrets:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-with-metadata
spec:
  secretStoreRefs:
    - name: aws-secret-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    - storeRef:
        name: aws-secret-store
      match:
        regexp: "^db-.*"
      metadata:
        labels:
          app: myapp
          env: production
      conversionStrategy: ReverseUnicode
```

The metadata structure is provider-specific. Check your provider's documentation for supported metadata fields.

## Multiple dataTo Entries

You can have multiple `dataTo` entries with different patterns and rewrites:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-multiple-patterns
spec:
  secretStoreRefs:
    - name: my-secret-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    # Push db-* keys with database/ prefix
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^db-.*"
      rewrite:
        - regexp:
            source: "^db-"
            target: "database/"
    # Push api-* keys with api/ prefix
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^api-.*"
      rewrite:
        - regexp:
            source: "^api-"
            target: "api/"
```

## Error Handling

### Invalid Regexp

If you provide an invalid regexp pattern, the PushSecret will enter an error state:

```yaml
dataTo:
  - storeRef:
      name: my-store
    match:
      regexp: "[invalid(regexp"  # Invalid regexp
```

Check the PushSecret status for error details:
```bash
kubectl get pushsecret my-pushsecret -o yaml
```

### Duplicate Remote Keys

If rewrites result in duplicate remote keys, the operation will fail:

```yaml
dataTo:
  - storeRef:
      name: my-store
    rewrite:
      - regexp:
          source: ".*"
          target: "same-key"  # All keys map to "same-key" - error!
```

Error message will indicate which source keys caused the conflict.

### No Matching Keys

If the match pattern doesn't match any keys, it's not an error. A warning will be logged but the PushSecret remains in Ready state.

## Use Cases

### Migrating Secrets Between Providers

Push all secrets from Kubernetes to a cloud provider:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: migrate-to-aws
spec:
  secretStoreRefs:
    - name: aws-secrets-manager
  selector:
    secret:
      name: legacy-secrets
  dataTo:
    - storeRef:
        name: aws-secrets-manager
      rewrite:
        - regexp:
            source: "^"
            target: "migrated/"  # Add prefix to all keys
```

### Environment-Specific Secrets

Push secrets with environment prefix:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-prod-secrets
spec:
  secretStoreRefs:
    - name: vault-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    - storeRef:
        name: vault-store
      rewrite:
        - regexp:
            source: "^"
            target: "prod/myapp/"
```

### Organizing Secrets by Category

Organize different types of secrets with prefixes:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: organize-secrets
spec:
  secretStoreRefs:
    - name: my-secret-store
  selector:
    secret:
      name: app-config
  dataTo:
    # Database credentials
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^db-.*"
      rewrite:
        - regexp: {source: "^db-", target: "config/database/"}
    # API keys
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^api-.*"
      rewrite:
        - regexp: {source: "^api-", target: "config/api/"}
    # TLS certificates
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^tls-.*"
      rewrite:
        - regexp: {source: "^tls-", target: "config/tls/"}
```

## Best Practices

1. **Start Simple**: Begin with basic pattern matching before adding complex rewrites
2. **Test Patterns**: Use `kubectl get secret source-secret -o jsonpath='{.data}' | jq 'keys'` to see all keys before writing patterns
3. **Use Descriptive Patterns**: Regexp patterns should be readable and maintainable
4. **Document Transformations**: Add comments explaining complex rewrite chains
5. **Validate in Non-Prod**: Test dataTo configurations in development before production
6. **Monitor Status**: Check PushSecret status regularly for errors or warnings
7. **Combine Wisely**: Use `dataTo` for bulk operations and `data` for exceptions

## Comparison: Before and After

**Before (without dataTo):**
```yaml
spec:
  data:
    - match: {secretKey: db-host, remoteRef: {remoteKey: app/db/host}}
    - match: {secretKey: db-port, remoteRef: {remoteKey: app/db/port}}
    - match: {secretKey: db-user, remoteRef: {remoteKey: app/db/user}}
    - match: {secretKey: db-pass, remoteRef: {remoteKey: app/db/pass}}
    - match: {secretKey: db-name, remoteRef: {remoteKey: app/db/name}}
    # ... many more entries
```

**After (with dataTo):**
```yaml
spec:
  dataTo:
    - storeRef:
        name: my-secret-store
      match:
        regexp: "^db-.*"
      rewrite:
        - regexp: {source: "^db-", target: "app/db/"}
```

## See Also

- [PushSecret Guide](pushsecrets.md) - Basic PushSecret usage
- [PushSecret API Reference](../api/pushsecret.md) - Complete API specification
- [Templating Guide](templating.md) - Advanced template usage
- [ExternalSecret dataFrom](datafrom-rewrite.md) - Similar feature for pulling secrets
