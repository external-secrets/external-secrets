## Conjur Provider

This section describes how to set up the Conjur provider for External Secrets Operator (ESO). For a working example, see the [Accelerator-K8s-External-Secrets repo](https://github.com/conjurdemos/Accelerator-K8s-External-Secrets).

### Prerequisites

Before installing the Conjur provider, you need:

* A running Conjur Server ([OSS](https://github.com/cyberark/conjur),
[Enterprise](https://www.cyberark.com/products/secrets-manager-enterprise/), or
[Cloud](https://www.cyberark.com/products/multi-cloud-secrets/)), with:
  * An accessible Conjur endpoint (for example: `https://myapi.example.com`).
  * Your configured Conjur authentication info (such as `hostid`, `apikey`, or JWT service ID). For more information on configuring Conjur, see [Policy statement reference](https://docs.cyberark.com/conjur-open-source/Latest/en/Content/Operations/Policy/policy-statement-ref.htm).
  * Support for your authentication method (`apikey` is supported by default, `jwt` requires additional configuration).
  * **Optional**: Conjur server certificate (see [below](#conjur-server-certificate)).
* A Kubernetes cluster with ESO installed.

### Conjur server certificate

If you set up your Conjur server with a self-signed certificate, we recommend that you populate the `caBundle` field with the Conjur self-signed certificate in the secret-store definition. The certificate CA must be referenced in the secret-store definition using either `caBundle` or `caProvider`:

```yaml
{% include 'conjur-ca-bundle.yaml' %}
```

### External secret store

The Conjur provider is configured as an external secret store in ESO. The Conjur provider supports these two methods to authenticate to Conjur:

* [`apikey`](#option-1-external-secret-store-with-apikey-authentication): uses a Conjur `hostid` and `apikey` to authenticate with Conjur
* [`jwt`](#option-2-external-secret-store-with-jwt-authentication): uses a JWT to authenticate with Conjur

#### Option 1: External secret store with apiKey authentication

This method uses a Conjur `hostid` and `apikey` to authenticate with Conjur. It is the simplest method to set up and use because your Conjur instance requires no additional configuration.

##### Step 1: Define an external secret store

!!! Tip
    Save as the file as: `conjur-secret-store.yaml`

```yaml
{% include 'conjur-secret-store-apikey.yaml' %}
```

##### Step 2: Create Kubernetes secrets for Conjur credentials

To connect to the Conjur server, the **ESO Conjur provider** needs to retrieve the `apikey` credentials from K8s secrets.

!!! Note
    For more information about how to create K8s secrets, see [Creating a secret](https://kubernetes.io/docs/concepts/configuration/secret/#creating-a-secret).

Here is an example of how to create K8s secrets using the `kubectl` command:

```shell
# This is all one line
kubectl -n external-secrets create secret generic conjur-creds --from-literal=hostid=MYCONJURHOSTID --from-literal=apikey=MYAPIKEY

# Example:
# kubectl -n external-secrets create secret generic conjur-creds --from-literal=hostid=host/data/app1/host001 --from-literal=apikey=321blahblah
```

!!! Note
    `conjur-creds` is the `name` defined in the `userRef` and `apikeyRef` fields of the `conjur-secret-store.yml` file.


##### Step 3: Create the external secrets store

!!! Important
    Unless you are using a [ClusterSecretStore](../api/clustersecretstore.md), credentials must reside in the same namespace as the SecretStore.

```shell
# WARNING: creates the store in the "external-secrets" namespace, update the value as needed
#
kubectl apply -n external-secrets -f conjur-secret-store.yaml

# WARNING: running the delete command will delete the secret store configuration
#
# If there is a need to delete the external secretstore
# kubectl delete secretstore -n external-secrets conjur
```

#### Option 2: External secret store with JWT authentication

This method uses JWT tokens to authenticate with Conjur. You can use the following methods to retrieve a JWT token for authentication:

* JWT token from a referenced Kubernetes service account
* JWT token stored in a Kubernetes secret

##### Step 1: Define an external secret store

When you use JWT authentication, the following must be specified in the `SecretStore`:

* `account` -  The name of the Conjur account
* `serviceId` - The ID of the JWT Authenticator `WebService` configured in Conjur that is used to authenticate the JWT token

You can retrieve the JWT token from either a referenced service account or a Kubernetes secret.

For example, to retrieve a JWT token from a referenced Kubernetes service account, the following secret store definition can be used:

```yaml
{% include 'conjur-secret-store-jwt-service-account-ref.yaml' %}
```

!!! Important
    This method is only supported in Kubernetes 1.22 and above as it uses the [TokenRequest API](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/) to get the JWT token from the referenced service account. Audiences can be defined in the [Conjur JWT authenticator](https://docs.conjur.org/Latest/en/Content/Integrations/k8s-ocp/k8s-jwt-authn.htm).

Alternatively, here is an example where a secret containing a valid JWT token is referenced:

```yaml
{% include 'conjur-secret-store-jwt-secret-ref.yaml' %}
```

The JWT token must identify your Conjur host, be compatible with your configured Conjur JWT authenticator, and meet all the [Conjur JWT guidelines](https://docs.conjur.org/Latest/en/Content/Operations/Services/cjr-authn-jwt-guidelines.htm#Best).

You can use an external JWT issuer or the Kubernetes API server to create the token. For example, a Kubernetes service account token can be created with this command:

```shell
kubectl create token my-service-account --audience='https://conjur.company.com' --duration=3600s
```

Save the secret store file as `conjur-secret-store.yaml`.

##### Step 2: Create the external secrets store

```shell
# WARNING: creates the store in the "external-secrets" namespace, update the value as needed
#
kubectl apply -n external-secrets -f conjur-secret-store.yaml

# WARNING: running the delete command will delete the secret store configuration
#
# If there is a need to delete the external secretstore
# kubectl delete secretstore -n external-secrets conjur
```

### Define an external secret

After you have configured the Conjur provider secret store, you can fetch secrets from Conjur.

Here is an example of how to fetch a single secret from Conjur:

```yaml
{% include 'conjur-external-secret.yaml' %}
```

Save the external secret file as `conjur-external-secret.yaml`.

#### Find by Name and Find by Tag

The Conjur provider also supports the Find by Name and Find by Tag ESO features. This means that
you can use a regular expression or tags to dynamically fetch multiple secrets from Conjur.

```yaml
{% include 'conjur-external-secret-find.yaml' %}
```

If you use these features, we strongly recommend that you limit the permissions of the Conjur host
to only the secrets that it needs to access. This is more secure and it reduces the load on
both the Conjur server and ESO.

### Create the external secret

```shell
# WARNING: creates the external-secret in the "external-secrets" namespace, update the value as needed
#
kubectl apply -n external-secrets -f conjur-external-secret.yaml

# WARNING: running the delete command will delete the external-secrets configuration
#
# If there is a need to delete the external secret
# kubectl delete externalsecret -n external-secrets conjur
```

### Get the K8s secret

* Log in to your Conjur server and verify that your secret exists
* Review the value of your Kubernetes secret to verify that it contains the same value as the Conjur server

```shell
# WARNING: this command will reveal the stored secret in plain text
#
# Assuming the secret name is "secret00", this will show the value
kubectl get secret -n external-secrets conjur -o jsonpath="{.data.secret00}"  | base64 --decode && echo
```

### See also

* [Accelerator-K8s-External-Secrets repo](https://github.com/conjurdemos/Accelerator-K8s-External-Secrets)
* [Configure Conjur JWT authentication](https://docs.cyberark.com/conjur-open-source/Latest/en/Content/Operations/Services/cjr-authn-jwt-guidelines.htm)

### License

Copyright (c) 2023-2024 CyberArk Software Ltd. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
