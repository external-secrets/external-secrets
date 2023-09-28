## Scaleway Secret Manager

External Secrets Operator integrates with [Scaleway's Secret Manager](https://developers.scaleway.com/en/products/secret_manager/api/v1alpha1/).

### Creating a SecretStore

You need an api key (access key + secret key) to authenticate with the secret manager.
Both access and secret keys can be specified either directly in the config, or by referencing
a kubernetes secret.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secret-store
spec:
  provider:
    scaleway:
      region: <REGION>
      projectId: <PROJECT_UUID>
      accessKey:
        value: <ACCESS_KEY>
      secretKey:
        secretRef:
          name: <NAME_OF_KUBE_SECRET>
          key: <KEY_IN_KUBE_SECRET>
```

### Referencing Secrets

Secrets can be referenced by name, id or path, using the prefixes `"name:"`, `"id:"` and `"path:"` respectively.

A PushSecret resource can only use a name reference.

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
          key: id:<SECRET_UUID>
          version: latest_enabled
```
