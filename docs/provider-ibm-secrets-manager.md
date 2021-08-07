## IBM Cloud Secret Manager

External Secrets Operator integrates with [IBM Secret Manager](https://www.ibm.com/cloud/secrets-manager) for secret management.

### Authentication

At the moment, we only support API key authentication for this provider. To generate your key (for test purposes we are going to generate from your user), first got to your (Access IAM) page:

![iam](./pictures/screenshot_api_keys_iam.png)

On the left, click "IBM Cloud API Keys":

![iam-left](./pictures/screenshot_api_keys_iam_left.png)

Press "Create an IBM Cloud API Key":

![iam-create-button](./pictures/screenshot_api_keys_create_button.png)

Pick a name and description for your key:

![iam-create-key](./pictures/screenshot_api_keys_create.png)

You have created a key. Press the eyeball to show the key. Copy or save it because keys can't be displayed or downloaded twice.

![iam-create-success](./pictures/screenshot_api_keys_create_successful.png)



#### API key secret

Create a secret containing your apiKey:

```shell
kubectl create secret generic ibm-secret --from-literal=apiKey='API_KEY_VALUE'
```

### Update secret store
Be sure the `ibm` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'ibm-secret-store.yaml' %}
```

To find your serviceURL, under your Secrets Manager resource, go to "Endpoints" on the left:


![iam-create-success](./pictures/screenshot_service_url.png)

### Secret Types
We support all secret types of [IBM Secrets Manager](https://cloud.ibm.com/apidocs/secrets-manager): `arbitrary`, `username_password`, `iam_credentials` and `imported_cert`. To define the type of secret you would like to sync you need to prefix the secret id with the desired type. If the secret type is not specified it is defaulted to `arbitrary`:

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

#### imported_cert
* `remoteRef` requires a `property` to be set for either `certificate`, `private_key` or `intermediate` to retrieve respective fields from the secrets manager secret and set in specified `secretKey`
* `dataFrom` retrieves all `certificate`, `private_key` and `intermediate` fields from the secrets manager secret and sets appropriate key:value pairs in the resulting Kubernetes secret


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
