External Secrets Operator integrates with [Device42 API](https://api.device42.com/#!/Passwords/getPassword) to sync Device42 secrets into a Kubernetes cluster.


### Authentication

`username` and `password` is required to talk to the Device42 API.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: device42-credentials
data:
  username: dGVzdA== # "test"
  password: dGVzdA== # "test"
```

### Creating a SecretStore

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: device42-secret-store
spec:
  provider:
    device42:
      host: <DEVICE42_HOSTNAME>
      auth:
        secretRef:
          credentials:
            name: <NAME_OF_KUBE_SECRET>
            key: <KEY_IN_KUBE_SECRET>
            namespace: <kube-system>
```

### Referencing Secrets

Secrets can be referenced by defining the `key` containing the Id of the secret.
The `password` field is return from device42

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: device42-external-secret
spec:
  refreshInterval: 5m
  secretStoreRef:
    kind: SecretStore
    name: device42-secret-store
  target:
    name: <K8s_SECRET_NAME_TO_MANAGE>
  data:
  - secretKey: <KEY_NAME_WITHIN_KUBE_SECRET>
    remoteRef:
      key: <DEVICE42_SECRET_ID>
```
