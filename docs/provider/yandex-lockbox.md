## Yandex Lockbox

External Secrets Operator integrates with [Yandex Lockbox](https://cloud.yandex.com/docs/lockbox/)
for secret management.

### Prerequisites
* [External Secrets Operator installed](../introduction/getting-started.md#installing-with-helm)
* [Yandex.Cloud CLI installed](https://cloud.yandex.com/docs/cli/quickstart)

### Authentication
At the moment, [authorized key](https://cloud.yandex.com/docs/iam/concepts/authorization/key) authentication is only supported:

* Create a [service account](https://cloud.yandex.com/docs/iam/concepts/users/service-accounts) in Yandex.Cloud:
```bash
yc iam service-account create --name eso-service-account
```
* Create an authorized key for the service account and save it to `authorized-key.json` file:
```bash
yc iam key create \
  --service-account-name eso-service-account \
  --output authorized-key.json
```
* Create a k8s secret containing the authorized key saved above:
```bash
kubectl create secret generic yc-auth --from-file=authorized-key=authorized-key.json
```
* Create a [SecretStore](../api/secretstore.md) pointing to `yc-auth` k8s secret:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secret-store
spec:
  provider:
    yandexlockbox:
      auth:
        authorizedKeySecretRef:
          name: yc-auth
          key: authorized-key
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in all `authorizedKeySecretRef` with the namespace where the secret resides.
### Creating external secret
To make External Secrets Operator sync a k8s secret with a Lockbox secret:

* Create a Lockbox secret, if not already created:
```bash
yc lockbox secret create \
  --name lockbox-secret \
  --payload '[{"key": "password","textValue": "p@$$w0rd"}]'
```
* Assign the [`lockbox.payloadViewer`](https://cloud.yandex.com/docs/lockbox/security/#roles-list) role
  for accessing the `lockbox-secret` payload to the service account used for authentication:
```bash
yc lockbox secret add-access-binding \
  --name lockbox-secret \
  --service-account-name eso-service-account \
  --role lockbox.payloadViewer
```
Run the following command to ensure that the correct access binding has been added:
```bash
yc lockbox secret list-access-bindings --name lockbox-secret
```
* Create an [ExternalSecret](../api/externalsecret.md) pointing to `secret-store` and `lockbox-secret`:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: external-secret
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: secret-store
    kind: SecretStore
  target:
    name: k8s-secret # the target k8s secret name
  data:
  - secretKey: password # the target k8s secret key
    remoteRef:
      key: ***** # ID of lockbox-secret
      property: password # (optional) payload entry key of lockbox-secret
```

The operator will fetch the Yandex Lockbox secret and inject it as a `Kind=Secret`
```yaml
kubectl get secret k8s-secret -n <namespace> | -o jsonpath='{.data.password}' | base64 -d
```
