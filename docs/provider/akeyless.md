## Akeyless Vault

External Secrets Operator integrates with [Akeyless API](https://docs.akeyless.io/reference#v2).

### Authentication

The API requires an access-id, access-type and access-Type-param.

The supported auth-methods and their params are:

| accessType  | accessTypeParam                                                                                                                                                                                                                      |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `api_key`      | The access key.                                                                                                                                     |
| `k8s`         | The k8s configuration name |
| `aws_iam` |   -                                                         |
| `gcp` |      The gcp audience                                                      |
| `azure_ad` |  azure object id  (optional)                                                          |

form more information about [Akeyless Authentication Methods](https://docs.akeyless.io/docs/access-and-authentication-methods)

### Akeless credentials secret

Create a secret containing your credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: akeylss-secret-creds
type: Opaque
stringData:
  accessId: "p-XXXX"
  accessType:  # k8s/aws_iam/gcp/azure_ad/api_key
  accessTypeParam:  # can be one of the following: k8s-conf-name/gcp-audience/azure-obj-id/access-key
```

### Update secret store
Be sure the `akeyless` provider is listed in the `Kind=SecretStore` and the `akeylessGWApiURL` is set (def: "https://api.akeless.io".

```yaml
{% include 'akeyless-secret-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `accessID`, `accessType` and `accessTypeParam` with the namespaces where the secrets reside.

### Authentication with Kubernetes

options of obtaining credentials:

- by using a service account jwt referenced in serviceAccountRef
- by using the jwt from a Kind=Secret referenced by the secretRef
- by using transient credentials from the mounted service account token within the external-secrets operator

```yaml
{% include 'akeyless-secret-store-k8s-auth.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `serviceAccountRef` and `secretRef` with the namespaces where the secrets reside.


### Creating external secret

To get a secret from Akeyless and secret it on the Kubernetes cluster, a `Kind=ExternalSecret` is needed.

```yaml
{% include 'akeyless-external-secret.yaml' %}
```


#### Using DataFrom

DataFrom can be used to get a secret as a JSON string and attempt to parse it.

```yaml
{% include 'akeyless-external-secret-json.yaml' %}
```

### Getting the Kubernetes secret
The operator will fetch the secret and inject it as a `Kind=Secret`.
```
kubectl get secret akeyless-secret-to-create -o jsonpath='{.data.secretKey}' | base64 -d
```

```
kubectl get secret akeyless-secret-to-create-json -o jsonpath='{.data}'
```
