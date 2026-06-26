## Akeyless Secrets Management Platform

External Secrets Operator integrates with the [Akeyless Secrets Management Platform](https://www.akeyless.io/).

### Create Secret Store

SecretStore resource specifies how to access Akeyless. This resource is namespaced.

**NOTE:** Make sure the Akeyless provider is listed in the Kind=SecretStore.
If you use a customer fragment, define the value of akeylessGWApiURL as the URL of your Akeyless Gateway in the following format: https://your.akeyless.gw:8080/v2.

Akeyless provides several Authentication Methods:

### Authentication with Kubernetes

Options for obtaining Kubernetes credentials include:

1. Using a service account jwt referenced in serviceAccountRef
2. Using the jwt from a Kind=Secret referenced by the secretRef
3. Using transient credentials from the mounted service account token within the external-secrets operator

#### Create the Akeyless Secret Store Provider with Kubernetes Auth-Method

```yaml
{% include 'akeyless-secret-store-k8s-auth.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` for `serviceAccountRef` and `secretRef` according to the namespaces where the secrets reside.

### Authentication with Cloud-Identity or Api-Access-Key

Akeyless providers require an access-id, access-type and access-type-param
to set your SecretStore with an authentication method from Akeyless.

The supported auth-methods and their parameters are:

| accessType     | accessTypeParam                    |
| -------------- | ---------------------------------- |
| `aws_iam`      | -                                  |
| `gcp`          | The GCP audience                   |
| `azure_ad`     | Azure object ID (optional)         |
| `api_key`      | The access key                     |
| `access_key`   | The access key (alias for api_key) |
| `k8s`          | The k8s configuration name         |

For `azure_ad` on AKS Workload Identity, set `authSecretRef.serviceAccountRef` to a ServiceAccount annotated with `azure.workload.identity/client-id` and `azure.workload.identity/tenant-id`. This field is only used when `accessType` is `azure_ad`; other access types ignore it.

```yaml
{% include 'akeyless-secret-store-azure-ad-wi.yaml' %}
```

For `ClusterSecretStore`, set `serviceAccountRef.namespace` when the ServiceAccount is not in the same namespace as the consuming `ExternalSecret`; otherwise the ServiceAccount is resolved from that namespace. Sovereign Azure clouds (US Government, China) are supported when `AZURE_ENVIRONMENT` or `AZURE_CLOUD` is set accordingly, matching the non-WI `GetCloudId` path.

For more information see [Akeyless Authentication Methods](https://docs.akeyless.io/docs/access-and-authentication-methods)

#### Creating an Akeyless Credentials Secret

Create a secret containing your credentials using the following example as a guide:

```yaml
{% include 'akeyless-credentials-secret.yaml' %}
```

#### Create the Akeyless Secret Store Provider with the Credentials Secret

```yaml
{% include 'akeyless-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` for `accessID`, `accessType` and `accessTypeParam` according to the namespaces where the secrets reside.

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
        type: Secret  # Can be Secret or ConfigMap
        name: "<name of secret or configmap>"
        key: "<key inside secret>"
        # namespace is mandatory for ClusterSecretStore and not relevant for SecretStore
        namespace: "my-cert-secret-namespace"
  ....
```

### Supported Secret Types

The provider supports the following Akeyless item types:

- **Static Secret** -- standard key/value secret
- **Dynamic Secret** -- ephemeral credentials generated on demand
- **Rotated Secret** -- automatically rotated credentials
- **Certificate** -- TLS/SSH certificates

### Creating an external secret

To get a secret from Akeyless and create it as a secret on the Kubernetes cluster, a `Kind=ExternalSecret` is needed.

```yaml
{% include 'akeyless-external-secret.yaml' %}
```

#### Fetching a specific version

Use `remoteRef.version` to pin a specific secret version (integer). Omit the field or set it to `0` to get the latest version.

```yaml
data:
  - secretKey: password
    remoteRef:
      key: /path/to/secret
      version: "3"  # fetch version 3 specifically
```

#### Extracting a property from a JSON secret

If the secret value is a JSON object, use `remoteRef.property` to extract a single key. Nested keys can be addressed with dot notation; literal dots in key names are escaped with a backslash (`key\.with\.dots`).

```yaml
data:
  - secretKey: db-password
    remoteRef:
      key: /path/to/json-secret
      property: password  # extracts {"password": "..."} from the JSON value
```

#### Using DataFrom

DataFrom can be used to get a secret as a JSON string and attempt to parse it, creating one Kubernetes secret key per JSON field.

```yaml
{% include 'akeyless-external-secret-json.yaml' %}
```

#### Finding secrets by name or tag

Use `dataFrom.find` to bulk-fetch secrets matching a name pattern or tag:

```yaml
# by name regex
dataFrom:
  - find:
      path: /my/path/         # optional path prefix
      name:
        regexp: ".*db.*"

# by tag
dataFrom:
  - find:
      tags:
        env: production
```

### Getting the Kubernetes Secret

The operator will fetch the secret and inject it as a `Kind=Secret`.

```bash
kubectl get secret database-credentials -o jsonpath='{.data.db-password}' | base64 -d
```

```bash
kubectl get secret database-credentials-json -o jsonpath='{.data}'
```

### Pushing a secret

To push a secret from Kubernetes cluster and create it as a secret to Akeyless, a `Kind=PushSecret` resource is needed.

```yaml
{% include 'akeyless-push-secret.yaml' %}
```

Then when you create a matching secret as follows:

```bash
kubectl create secret generic --from-literal=cache-pass=mypassword k8s-created-secret
```

Then it will create a secret in akeyless `eso-created/my-secret` with value `{"cache-pass":"mypassword"}`
