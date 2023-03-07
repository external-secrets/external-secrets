## Akeyless Secrets Management Platform

External Secrets Operator integrates with the [Akeyless Secrets Management Platform](https://www.akeyless.io/).
### Create Secret Store:
SecretStore resource specifies how to access Akeyless. This resource is namespaced.

**NOTE:** Make sure the Akeyless provider is listed in the Kind=SecretStore.
If you use a customer fragment, define the value of akeylessGWApiURL as the URL of your Akeyless Gateway in the following format: https://your.akeyless.gw:8080/v2.

Akeyelss provide several Authentication Methods:

### Authentication with Kubernetes:

Options for obtaining Kubernetes credentials include:

1. Using a service account jwt referenced in serviceAccountRef
2. Using the jwt from a Kind=Secret referenced by the secretRef
3. Using transient credentials from the mounted service account token within the external-secrets operator

#### Create the Akeyless Secret Store Provider with Kubernetes Auth-Method
```yaml
{% include 'akeyless-secret-store-k8s-auth.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `serviceAccountRef` and `secretRef` according to  the namespaces where the secrets reside.


### Authentication With Cloud-Identity or Api-Access-Key

Akeyless providers require an access-id, access-type and access-Type-param
To set your SecretStore with an authentication method from Akeyless.

The supported auth-methods and their parameters are:

| accessType  | accessTypeParam                                                                                                                                                                                                                      |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `aws_iam` |   -                                                         |
| `gcp` |      The gcp audience                                                      |
| `azure_ad` |  azure object id  (optional)                                                          |
| `api_key`      | The access key.                                                                                                                                     |
| `k8s`         | The k8s configuration name |
For more information see [Akeyless Authentication Methods](https://docs.akeyless.io/docs/access-and-authentication-methods)

#### Creating an Akeyless Credentials Secret

Create a secret containing your credentials using the following example as a guide:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: akeyless-secret-creds
type: Opaque
stringData:
  accessId: "p-XXXX"
  accessType:  # gcp/azure_ad/api_key/k8s/aws_iam
  accessTypeParam:  # optional: can be one of the following: gcp-audience/azure-obj-id/access-key/k8s-conf-name
```

#### Create the Akeyless Secret Store Provider with the Credentials Secret

```yaml
{% include 'akeyless-secret-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` for `accessID`, `accessType` and `accessTypeParam`  according to the namespaces where the secrets reside.

#### Create the Akeyless Secret Store With CAs for TLS handshake
```yaml
....
spec:
  provider:
    akeyless:
      akeylessGWApiURL: "https://your.akeyless.gw:8080/v2"

      # Optional caBundle - PEM/base64 encoded CA certificate
      caBundle: "<base64 encoded cabundle>"
      # Optional caProvider:
      # Instead of caBundle you can also specify a caProvider
      # this will retrieve the cert from a Secret or ConfigMap
      caProvider:
        type: "Secret/ConfigMap" # Can be Secret or ConfigMap
        name: "<name of secret or configmap>"
        key: "<key inside secret>"
        # namespace is mandatory for ClusterSecretStore and not relevant for SecretStore
        namespace: "my-cert-secret-namespace"
  ....
```

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
kubectl get secret database-credentials -o jsonpath='{.data.db-password}' | base64 -d
```

```
kubectl get secret database-credentials-json -o jsonpath='{.data}'
```
