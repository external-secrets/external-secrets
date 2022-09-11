## Akeyless Vault

External Secrets Operator integrates with the [Akeyless API](https://docs.akeyless.io/reference#v2).

### Authentication

To operate the API first define an access-id, access-type and access-Type-param.

The supported auth-methods and their parameters are:

| accessType  | accessTypeParam                                                                                                                                                                                                                      |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `api_key`      | The access key.                                                                                                                                     |
| `k8s`         | The k8s configuration name |
| `aws_iam` |   -                                                         |
| `gcp` |      The gcp audience                                                      |
| `azure_ad` |  azure object id  (optional)                                                          |

For more information see [Akeyless Authentication Methods](https://docs.akeyless.io/docs/access-and-authentication-methods)

### Creating an Akeyless Ccredentials Secret

Create a secret containing your credentials using the following example as a guide:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: akeyless-secret-creds
type: Opaque
stringData:
  accessId: "p-XXXX"
  accessType:  # k8s/aws_iam/gcp/azure_ad/api_key
  accessTypeParam:  # can be one of the following: k8s-conf-name/gcp-audience/azure-obj-id/access-key
```

### Update Secret Store
Be sure the `akeyless` provider is listed in the `Kind=SecretStore` and the `akeylessGWApiURL` is set (def: "https://api.akeless.io").

```yaml
{% include 'akeyless-secret-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` for `accessID`, `accessType` and `accessTypeParam`  according to the namespaces where the secrets reside.

### Authentication with Kubernetes

Options for obtaining Kubernetes credentials include:

1. Using a service account jwt referenced in serviceAccountRef
2. Using the jwt from a Kind=Secret referenced by the secretRef
3. Using transient credentials from the mounted service account token within the external-secrets operator

```yaml
{% include 'akeyless-secret-store-k8s-auth.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `serviceAccountRef` and `secretRef` according to  the namespaces where the secrets reside.


### Creating an external secret

To get a secret from Akeyless and create it as a secret on the Kubernetes cluster, a `Kind=ExternalSecret` is needed.

```yaml
{% include 'akeyless-external-secret.yaml' %}
```


#### Using DataFrom

DataFrom can be used to get a secret as a JSON string and attempt to parse it.

```yaml
{% include 'akeyless-external-secret-json.yaml' %}
```

### Getting the Kubernetes Secret
The operator will fetch the secret and inject it as a `Kind=Secret`.
```
kubectl get secret akeyless-secret-to-create -o jsonpath='{.data.secretKey}' | base64 -d
```

```
kubectl get secret akeyless-secret-to-create-json -o jsonpath='{.data}'
```
