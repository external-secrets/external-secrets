## Conjur Provider

The following sections outline what is needed to get your external-secrets Conjur provider setup.

### Pre-requirements

This section contains the list of the pre-requirements before installing the Conjur Provider.

*   Running Conjur Server
    -   These items will be needed in order to configure the secret-store
        +   Conjur endpoint - include the scheme but no trailing '/', ex: https://myapi.example.com
        +   Conjur credentials (hostid, apikey)
        +   Certificate for Conjur server is OPTIONAL -- But, **when using a self-signed cert when setting up your Conjur server, it is strongly recommended to populate "caBundle" with self-signed cert in the secret-store definition**
*   Kubernetes cluster
    -   External Secrets Operator is installed

### Create External Secret Store Definition

Recommend to save as filename: `conjur-secret-store.yaml`

```yaml
{% include 'conjur-secret-store.yaml' %}
```

### Create External Secret Definition

Important note: **Creds must live in the same namespace as a SecretStore  - the secret store may only reference secrets from the same namespace.**  When using a ClusterSecretStore this limitation is lifted and the creds can live in any namespace.

Recommend to save as filename: `conjur-external-secret.yaml`

```yaml
{% include 'conjur-external-secret.yaml' %}}
```

### Create Kubernetes Secrets

In order for the ESO **Conjur** provider to connect to the Conjur server, the creds should be stored as k8s secrets.  Please refer to <https://kubernetes.io/docs/concepts/configuration/secret/#creating-a-secret> for various methods to create secrets.  Here is one way to do it using `kubectl`

***NOTE***: "conjur-creds" is the "name" used in "userRef" and "apikeyRef" in the conjur-secret-store definition

```shell
# This is all one line
kubectl -n external-secrets create secret generic conjur-creds --from-literal=hostid=MYCONJURHOSTID --from-literal=apikey=MYAPIKEY

# Example:
# kubectl -n external-secrets create secret generic conjur-creds --from-literal=hostid=host/data/app1/host001 --from-literal=apikey=321blahblah
```

### Create the External Secrets Store

```shell
# WARNING: this will create the store configuration in the "external-secrets" namespace, adjust this to your own situation
#
kubectl apply -n external-secrets -f conjur-secret-store.yaml

# WARNING: running the delete command will delete the secret store configuration
#
# If there is a need to delete the external secretstore
# kubectl delete secretstore -n external-secrets conjur
```

### Create the External Secret

```shell
# WARNING: this will create the external-secret configuration in the "external-secrets" namespace, adjust this to your own situation
#
kubectl apply -n external-secrets -f conjur-external-secret.yaml

# WARNING: running the delete command will delete the external-secrets configuration
#
# If there is a need to delete the external secret
# kubectl delete externalsecret -n external-secrets conjur
```

### Getting the K8S Secret

* Login to your Conjur server and verify that your secret exists
* Review the value of your kubernetes secret to see that it contains the same value from Conjur

```shell
# WARNING: this command will reveal the stored secret in plain text
#
# Assuming the secret name is "secret00", this will show the value
kubectl get secret -n external-secrets conjur -o jsonpath="{.data.secret00}"  | base64 --decode && echo
```

### Support

Copyright (c) 2023 CyberArk Software Ltd. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
