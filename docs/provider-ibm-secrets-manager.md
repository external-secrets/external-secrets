## IBM Cloud Secret Manager

External Secrets Operator integrates with [IBM Secret Manager](https://www.ibm.com/cloud/secrets-manager) for secret management.

### Authentication

We support API key and trusted profile container authentication for this provider.

#### API key secret

To generate your key (for test purposes we are going to generate from your user), first got to your (Access IAM) page:

![iam](./pictures/screenshot_api_keys_iam.png)

On the left, click "IBM Cloud API Keys":

![iam-left](./pictures/screenshot_api_keys_iam_left.png)

Press "Create an IBM Cloud API Key":

![iam-create-button](./pictures/screenshot_api_keys_create_button.png)

Pick a name and description for your key:

![iam-create-key](./pictures/screenshot_api_keys_create.png)

You have created a key. Press the eyeball to show the key. Copy or save it because keys can't be displayed or downloaded twice.

![iam-create-success](./pictures/screenshot_api_keys_create_successful.png)

Create a secret containing your apiKey:

```shell
kubectl create secret generic ibm-secret --from-literal=apiKey='API_KEY_VALUE'
```

#### Trusted Profile Container Auth

To create the trusted profile, first got to your (Access IAM) page:

![iam](./pictures/screenshot_api_keys_iam.png)

On the left, click "Access groups":

![iam-left](./pictures/screenshot_container_auth_create_group.png)

Pick a name and description for your group:

![iam-left](./pictures/screenshot_container_auth_create_group_1.png)

Click on "Access Policies":

![iam-left](./pictures/screenshot_container_auth_create_group_2.png)

Click on "Assign Access", select "IAM services", and pick "Secrets Manager" from the pick-list:

![iam-left](./pictures/screenshot_container_auth_create_group_3.png)

Scope to "All resources" or "Resources based on selected attributes", select "SecretsReader":

![iam-left](./pictures/screenshot_container_auth_create_group_4.png)

Click "Add" and "Assign" to save the access group.

Next, on the left, click "Trusted profiles":

![iam-left](./pictures/screenshot_container_auth_iam_left.png)

Press "Create":

![iam-create-button](./pictures/screenshot_container_auth_create_button.png)

Pick a name and description for your profile:

![iam-create-key](./pictures/screenshot_container_auth_create_1.png)

Scope the profile's access.

The compute service type will be "Red Hat OpenShift on IBM Cloud".  Additional restriction can be configured based on cloud or cluster metadata, or if "Specific resources" is selected, restriction to a specific cluster.

![iam-create-key](./pictures/screenshot_container_auth_create_2.png)

Click "Add" next to the previously created access group and then "Create", to associate the necessary service permissions.

![iam-create-key](./pictures/screenshot_container_auth_create_3.png)

To use the container-based authentication, it is necessary to map the API server `serviceAccountToken` auth token to the "external-secrets" and "external-secrets-webhook" deployment descriptors. Example below:

```yaml
{% include 'ibm-container-auth-volume.yaml' %}
```

### Update secret store
Be sure the `ibm` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'ibm-secret-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `secretApiKeySecretRef` with the namespace where the secret resides.

**NOTE:** Only `secretApiKeySecretRef` or `containerAuth` should be specified, depending on authentication me
thod being used.

To find your serviceURL, under your Secrets Manager resource, go to "Endpoints" on the left.
Note: Use the url without the `/api` suffix that is presented in the UI.
See here for a list of [publicly available endpoints](https://cloud.ibm.com/apidocs/secrets-manager#getting-started-endpoints).

![iam-create-success](./pictures/screenshot_service_url.png)

### Secret Types
We support the following secret types of [IBM Secrets Manager](https://cloud.ibm.com/apidocs/secrets-manager):

* `arbitrary` 
* `username_password`
* `iam_credentials`
* `imported_cert`
* `public_cert`
* `private_cert`
* `kv` 

To define the type of secret you would like to sync you need to prefix the secret id with the desired type. If the secret type is not specified it is defaulted to `arbitrary`:

```yaml
{% include 'ibm-es-types.yaml' %}

```

The behavior for the different secret types is as following:

#### arbitrary

* `remoteRef` retrieves a string from secrets manager and sets it for specified `secretKey`
* `dataFrom` retrieves a string from secrets manager and tries to parse it as JSON object setting the key:values pairs in resulting Kubernetes secret if successful

#### username_password
* `remoteRef` requires a `property` to be set for either `username` or `password` to retrieve respective fields from the secrets manager secret and set in specified `secretKey`
* `dataFrom` retrieves both `username` and `password` fields from the secrets manager secret and sets appropriate key:value pairs in the resulting Kubernetes secret

#### iam_credentials
* `remoteRef` retrieves an apikey from secrets manager and sets it for specified `secretKey`
* `dataFrom` retrieves an apikey from secrets manager and sets it for the `apikey` Kubernetes secret key

#### imported_cert, public_cert and private_cert
* `remoteRef` requires a `property` to be set for either `certificate`, `private_key` or `intermediate` to retrieve respective fields from the secrets manager secret and set in specified `secretKey`
* `dataFrom` retrieves all `certificate`, `private_key` and `intermediate` fields from the secrets manager secret and sets appropriate key:value pairs in the resulting Kubernetes secret

#### kv
* An optional `property` field can be set to `remoteRef` to select requested key from the KV secret. If not set, the entire secret will be returned
* `dataFrom` retrieves a string from secrets manager and tries to parse it as JSON object setting the key:values pairs in resulting Kubernetes secret if successful

```json
{
  "key1": "val1",
  "key2": "val2",
  "key3": {
    "keyA": "valA",
    "keyB": "valB"
  },
  "special.key": "special-content"
}
```

```yaml
data:
- secretKey: key3_keyB
  remoteRef:
    key: 'kv/aaaaa-bbbb-cccc-dddd-eeeeee'
    property: 'key3.keyB'
- secretKey: special_key
  remoteRef:
    key: 'kv/aaaaa-bbbb-cccc-dddd-eeeeee'
    property: 'special.key'
- secretKey: key_all
  remoteRef:
    key: 'kv/aaaaa-bbbb-cccc-dddd-eeeeee'

dataFrom:
  - key: 'kv/aaaaa-bbbb-cccc-dddd-eeeeee'
    property: 'key3'
```

results in

```yaml
data:
  # secrets from data
  key3_keyB: ... #valB
  special_key: ... #special-content
  key_all: ... #{"key1":"val1","key2":"val2", ..."special.key":"special-content"}

  # secrets from dataFrom
  keyA: ... #valA
  keyB: ... #valB
```


### Creating external secret

To create a kubernetes secret from the IBM Secrets Manager, a `Kind=ExternalSecret` is needed.

```yaml
{% include 'ibm-external-secret.yaml' %}
```

Currently we can only get the secret by its id and not its name, so something like `565287ce-578f-8d96-a746-9409d531fe2a`.

### Getting the Kubernetes secret
The operator will fetch the IBM Secret Manager secret and inject it as a `Kind=Secret`
```
kubectl get secret secret-to-be-created -n <namespace> | -o jsonpath='{.data.test}' | base64 -d
```
