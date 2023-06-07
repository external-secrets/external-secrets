## Delinea DevOps Secrets Vault

External Secrets Operator integrates with [Delinea DevOps Secrets Vault](https://docs.delinea.com/online-help/products/devops-secrets-vault/current).

### Creating a SecretStore

You need client ID, client secret and tenant to authenticate with DSV.
Both client ID and client secret can be specified either directly in the config, or by referencing a kubernetes secret.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secret-store
spec:
  provider:
    delinea:
      tenant: <TENANT>
      clientId:
        value: <CLIENT_ID>
      clientSecret:
        secretRef:
          name: <NAME_OF_KUBE_SECRET>
          key: <KEY_IN_KUBE_SECRET>
```

### Referencing Secrets

Secrets can be referenced by path. Getting a specific version of a secret is not yet supported.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
    name: secret
spec:
    refreshInterval: 20s
    secretStoreRef:
        kind: SecretStore
        name: secret-store
    data:
      - secretKey: <KEY_IN_KUBE_SECRET>
        remoteRef:
          key: <SECRET_PATH>
```
