## Delinea DevOps Secrets Vault

External Secrets Operator integrates with [Delinea DevOps Secrets Vault](https://docs.delinea.com/online-help/products/devops-secrets-vault/current).

Please note that the [Delinea Secret Server](https://delinea.com/products/secret-server) product is NOT in scope of this integration.

### Creating a SecretStore

You need client ID, client secret and tenant to authenticate with DSV.
Both client ID and client secret can be specified either directly in the config, or by referencing a kubernetes secret.

To acquire client ID and client secret, refer to the  [policy management](https://docs.delinea.com/dsv/current/tutorials/policy.md) and [client management](https://docs.delinea.com/dsv/current/usage/cli-ref/client.md) documentation.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secret-store
spec:
  provider:
    delinea:
      tenant: <TENANT>
      tld: <TLD>
      clientId:
        value: <CLIENT_ID>
      clientSecret:
        secretRef:
          name: <NAME_OF_KUBE_SECRET>
          key: <KEY_IN_KUBE_SECRET>
```

Both `clientId` and `clientSecret` can either be specified directly via the `value` field or can reference a kubernetes secret.

The `tenant` field must correspond to the host name / site name of your DevOps vault. If you selected a region other than the US you must also specify the TLD, e.g. `tld: eu`.

If required, the URL template (`urlTemplate`) can be customized as well.

### Referencing Secrets

Secrets can be referenced by path. Getting a specific version of a secret is not yet supported.

Note that because all DSV secrets are JSON objects, you must specify `remoteRef.property`. You can access nested values or arrays using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md).

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
          property: <JSON_PROPERTY>
```
