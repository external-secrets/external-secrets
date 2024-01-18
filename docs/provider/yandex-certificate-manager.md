## Yandex Certificate Manager

External Secrets Operator integrates with [Yandex Certificate Manager](https://cloud.yandex.com/docs/certificate-manager/)
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
    yandexcertificatemanager:
      auth:
        authorizedKeySecretRef:
          name: yc-auth
          key: authorized-key
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in all `authorizedKeySecretRef` with the namespace where the secret resides.

### Creating external secret
To make External Secrets Operator sync a k8s secret with a Certificate Manager certificate:

* Create a Certificate Manager certificate (follow
  [the instructions](https://cloud.yandex.com/en-ru/docs/certificate-manager/operations/)), if not already created.
* Assign the [`certificate-manager.certificates.downloader`](https://cloud.yandex.com/en-ru/docs/certificate-manager/security/#roles-list) role
  for accessing the certificate content to the service account used for authentication (`*****` is the certificate ID):
```bash
yc cm certificate add-access-binding \
  --id ***** \
  --service-account-name eso-service-account \
  --role certificate-manager.certificates.downloader
```
Run the following command to ensure that the correct access binding has been added:
```bash
yc cm certificate list-access-bindings --id *****
```
* Create an [ExternalSecret](../api/externalsecret.md) pointing to `secret-store` and the certificate in Certificate Manager:
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
    template:
      type: kubernetes.io/tls
  data:
    - secretKey: tls.crt # the target k8s secret key
      remoteRef:
        key: ***** # the certificate ID
        property: chain
    - secretKey: tls.key # the target k8s secret key
      remoteRef:
        key: ***** # the certificate ID
        property: privateKey
```
The following property values are possible:
    * `chain` – to fetch PEM-encoded certificate chain
    * `privateKey` – to fetch PEM-encoded private key
    * `chainAndPrivateKey` or missing property – to fetch both chain and private key

The operator will fetch the Yandex Certificate Manager certificate and inject it as a `Kind=Secret`
```yaml
kubectl get secret k8s-secret -ojson | jq '."data"."tls.crt"' -r | base64 --decode
kubectl get secret k8s-secret -ojson | jq '."data"."tls.key"' -r | base64 --decode
```
