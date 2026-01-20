# PushSecret dataFrom Examples

This page provides practical examples of using the `dataFrom` field in PushSecret to bulk-push secrets to external providers.

## Prerequisites

Before using these examples, ensure you have:

- External Secrets Operator installed in your cluster
- A configured SecretStore (or ClusterSecretStore)
- A source Kubernetes Secret with the data you want to push

## Example 1: Basic Database Credentials Push

Push all database-related secrets with organized naming.

**Source Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
  namespace: myapp
type: Opaque
stringData:
  db-host: "prod-db.example.com"
  db-port: "5432"
  db-username: "app_user"
  db-password: "super-secret-password"
  db-database: "myapp_db"
  db-ssl-mode: "require"
```

**PushSecret with dataFrom:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-db-credentials
  namespace: myapp
spec:
  refreshInterval: 1h
  secretStoreRefs:
    - name: aws-secrets-manager
      kind: SecretStore
  selector:
    secret:
      name: db-credentials
  dataFrom:
    - match:
        regexp: "^db-.*"
      rewrite:
        - regexp:
            source: "^db-"
            target: "myapp/production/database/"
```

**Result in AWS Secrets Manager:**
- `myapp/production/database/host`
- `myapp/production/database/port`
- `myapp/production/database/username`
- `myapp/production/database/password`
- `myapp/production/database/database`
- `myapp/production/database/ssl-mode`

## Example 2: Multi-Environment Configuration

Push the same secrets to different environments with different prefixes.

**Source Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-config
  namespace: myapp
type: Opaque
stringData:
  api-key: "abc123xyz"
  api-secret: "secret456"
  redis-url: "redis://cache:6379"
  postgres-url: "postgres://db:5432/mydb"
```

**Development Environment:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-dev-config
  namespace: myapp
spec:
  secretStoreRefs:
    - name: vault-dev
  selector:
    secret:
      name: app-config
  dataFrom:
    - rewrite:
        - regexp:
            source: "^"
            target: "dev/myapp/"
```

**Production Environment:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-prod-config
  namespace: myapp
spec:
  secretStoreRefs:
    - name: vault-prod
  selector:
    secret:
      name: app-config
  dataFrom:
    - rewrite:
        - regexp:
            source: "^"
            target: "prod/myapp/"
```

## Example 3: Organizing Secrets by Category

Push different types of secrets to organized paths.

**Source Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mixed-secrets
  namespace: myapp
type: Opaque
stringData:
  db-host: "database.local"
  db-password: "dbpass"
  api-github-token: "ghp_xxx"
  api-stripe-key: "sk_live_xxx"
  tls-cert: "-----BEGIN CERTIFICATE-----"
  tls-key: "-----BEGIN PRIVATE KEY-----"
```

**PushSecret with Multiple Patterns:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: organize-secrets
  namespace: myapp
spec:
  secretStoreRefs:
    - name: vault-store
  selector:
    secret:
      name: mixed-secrets
  dataFrom:
    # Database credentials -> config/database/*
    - match:
        regexp: "^db-.*"
      rewrite:
        - regexp:
            source: "^db-"
            target: "config/database/"

    # API keys -> config/api/*
    - match:
        regexp: "^api-.*"
      rewrite:
        - regexp:
            source: "^api-"
            target: "config/api/"

    # TLS certificates -> config/tls/*
    - match:
        regexp: "^tls-.*"
      rewrite:
        - regexp:
            source: "^tls-"
            target: "config/tls/"
```

**Result:**
- `config/database/host`
- `config/database/password`
- `config/api/github-token`
- `config/api/stripe-key`
- `config/tls/cert`
- `config/tls/key`

## Example 4: Template Transformation

Use Go templates to transform key names with advanced logic.

**Source Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: service-keys
  namespace: myapp
type: Opaque
stringData:
  payment-gateway-key: "pk_xxx"
  email-service-key: "es_xxx"
  storage-service-key: "ss_xxx"
```

**PushSecret with Template:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-service-keys
  namespace: myapp
spec:
  secretStoreRefs:
    - name: gcp-secret-manager
  selector:
    secret:
      name: service-keys
  dataFrom:
    - rewrite:
        - transform:
            template: "services/{{ .value | upper | replace \"-\" \"_\" }}"
```

**Result:**
- `services/PAYMENT_GATEWAY_KEY`
- `services/EMAIL_SERVICE_KEY`
- `services/STORAGE_SERVICE_KEY`

## Example 5: Chained Transformations

Apply multiple transformations sequentially for complex key restructuring.

**Source Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: legacy-secrets
  namespace: myapp
type: Opaque
stringData:
  old-db-primary-host: "db1.old.local"
  old-db-replica-host: "db2.old.local"
  old-cache-redis-url: "redis://old-cache:6379"
```

**PushSecret with Chained Rewrites:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: migrate-legacy-secrets
  namespace: myapp
spec:
  secretStoreRefs:
    - name: aws-secrets-manager
  selector:
    secret:
      name: legacy-secrets
  dataFrom:
    - rewrite:
        # First: Remove "old-" prefix
        - regexp:
            source: "^old-"
            target: ""
        # Second: Add "migrated/" prefix
        - regexp:
            source: "^"
            target: "migrated/"
        # Third: Replace hyphens with slashes for hierarchy
        - regexp:
            source: "-"
            target: "/"
```

**Result:**
- `migrated/db/primary/host`
- `migrated/db/replica/host`
- `migrated/cache/redis/url`

## Example 6: Override Specific Keys

Use both dataFrom and explicit data to handle exceptions.

**Source Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
  namespace: myapp
type: Opaque
stringData:
  db-host: "database.local"
  db-port: "5432"
  db-user: "app"
  db-password: "secret123"
  db-admin-password: "admin-secret"  # Should go to different location
```

**PushSecret with Override:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-with-override
  namespace: myapp
spec:
  secretStoreRefs:
    - name: vault-store
  selector:
    secret:
      name: app-secrets
  # Push all db-* keys to app/database/*
  dataFrom:
    - match:
        regexp: "^db-.*"
      rewrite:
        - regexp:
            source: "^db-"
            target: "app/database/"

  # Except db-admin-password which goes to admin/
  data:
    - match:
        secretKey: db-admin-password
        remoteRef:
          remoteKey: admin/database/password
```

**Result:**
- `app/database/host` (from dataFrom)
- `app/database/port` (from dataFrom)
- `app/database/user` (from dataFrom)
- `app/database/password` (from dataFrom)
- `admin/database/password` (from explicit data override)

## Example 7: AWS Secrets Manager with Metadata

Push secrets with AWS-specific metadata tags.

**PushSecret with Metadata:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-with-aws-tags
  namespace: myapp
spec:
  secretStoreRefs:
    - name: aws-secrets-manager
  selector:
    secret:
      name: app-config
  dataFrom:
    - match:
        regexp: "^prod-.*"
      rewrite:
        - regexp:
            source: "^prod-"
            target: "myapp/prod/"
      metadata:
        tags:
          - key: Environment
            value: production
          - key: Application
            value: myapp
          - key: ManagedBy
            value: external-secrets-operator
```

## Example 8: Vault with KV Version 2

Push secrets to HashiCorp Vault KV v2 engine with proper paths.

**Source Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vault-secrets
  namespace: myapp
type: Opaque
stringData:
  service-a-key: "key-a"
  service-b-key: "key-b"
  shared-secret: "shared"
```

**PushSecret for Vault:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-to-vault
  namespace: myapp
spec:
  secretStoreRefs:
    - name: vault-kv-v2
  selector:
    secret:
      name: vault-secrets
  dataFrom:
    # Service-specific secrets
    - match:
        regexp: "^service-.*-key$"
      rewrite:
        - regexp:
            source: "^service-(.*)-key$"
            target: "services/$1/api-key"  # Use capture group

    # Shared secrets
    - match:
        regexp: "^shared-.*"
      rewrite:
        - regexp:
            source: "^shared-"
            target: "shared/"
```

**Result:**
- `services/a/api-key`
- `services/b/api-key`
- `shared/secret`

## Example 9: Azure Key Vault

Push secrets to Azure Key Vault with naming constraints (alphanumeric and hyphens only).

**PushSecret for Azure:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-to-azure
  namespace: myapp
spec:
  secretStoreRefs:
    - name: azure-key-vault
  selector:
    secret:
      name: app-secrets
  dataFrom:
    - rewrite:
        # Azure Key Vault only allows alphanumeric and hyphens
        # Convert underscores to hyphens
        - regexp:
            source: "_"
            target: "-"
        # Add prefix
        - regexp:
            source: "^"
            target: "myapp-"
```

## Example 10: Migration from One Provider to Another

Backup secrets from AWS to GCP while maintaining structure.

**Step 1: Pull from AWS using ExternalSecret:**
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: pull-from-aws
  namespace: backup
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
  target:
    name: aws-backup-secrets
  dataFrom:
    - find:
        name:
          regexp: "^myapp/.*"
```

**Step 2: Push to GCP with dataFrom:**
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: backup-to-gcp
  namespace: backup
spec:
  secretStoreRefs:
    - name: gcp-secret-manager
  selector:
    secret:
      name: aws-backup-secrets
  dataFrom:
    - rewrite:
        # Maintain structure but add backup prefix
        - regexp:
            source: "^"
            target: "backup-from-aws/"
```

## Troubleshooting

### Check PushSecret Status

```bash
kubectl get pushsecret <name> -n <namespace> -o yaml
```

Look for the `status.conditions` field for error messages.

### View Synced Secrets

```bash
kubectl get pushsecret <name> -n <namespace> -o jsonpath='{.status.syncedPushSecrets}' | jq
```

### Common Issues

**1. No keys matched:**
- Verify the source Secret has keys matching your pattern
- Check regex syntax: `kubectl get secret <name> -o jsonpath='{.data}' | jq 'keys'`

**2. Invalid regex error:**
- Validate your regex using an online regex tester
- Ensure special characters are properly escaped

**3. Duplicate remote keys:**
- Check if your rewrites produce unique keys
- Adjust patterns or use explicit data overrides

## Best Practices

1. **Start with match-all to verify**: Test with `dataFrom: [{}]` first
2. **Test regex patterns**: Use `kubectl get secret -o jsonpath='{.data}' | jq 'keys'`
3. **Use descriptive patterns**: Make regex patterns self-documenting
4. **Monitor status**: Check PushSecret status after creation
5. **Version control**: Keep PushSecret manifests in git
6. **Document transformations**: Add comments explaining complex rewrites

## See Also

- [PushSecret dataFrom Guide](../guides/pushsecret-datafrom.md)
- [PushSecret API Reference](../api/pushsecret.md)
- [Provider Documentation](../provider/)
