## Conjur Provider

The following sections outline what is needed to get your external-secrets Conjur provider setup.

### Pre-requirements

This section describes the pre-requirements before installing the Conjur Provider

* Running Conjur Server
  * Conjur endpoint - include the scheme but no trailing '/', ex: https://myapi.example.com
  * Conjur credentials (hostid, apikey)
  * Cert for conjur server is OPTIONAL, unless using a **self-signed cert**, then this **field** is **required**
* Kubernetes cluster
  * External Secrets Operator is installed

### Create External Secret Store Definition

Recommend to save as filename: `conjur-secret-store.yaml`

```yaml
{% include 'conjur-secret-store.yaml' %}
```

### Create External Secret Definition

Recommend to save as filename: `conjur-external-secret.yaml`

```yaml
{% include 'conjur-external-secret.yaml' %}}
```

### Create Kubernetes Secrets

In order for the ESO **Conjur** provider to connect to the Conjur server, the creds should be stored as k8s secrets.  Here is one way to do it using `kubectl`

***NOTE***: "conjur-creds" is the "name" used in "userRef" and "apikeyRef" in the definition for the conjur-secret-store

```shell
# This is all one line
kubectl -n external-secrets create secret generic conjur-creds --from-literal=hostid=MYCONJURHOSTID --from-literal=apikey=MYAPIKEY

# Example:
# kubectl -n external-secrets create secret generic conjur-creds --from-literal=hostid=host/data/app1/host001 --from-literal=apikey=321blahblah
```

### Create the External Secrets Store

```shell
kubectl apply -n external-secrets -f conjur-secret-store.yaml

# To delete the external secretstore
# kubectl delete secretstore -n external-secrets conjur
```

### Create the External Secret

```shell
kubectl apply -n external-secrets -f conjur-external-secret.yaml

# To delete the external secret
# kubectl delete externalsecret -n external-secrets conjur
```

### Getting the K8S Secret

* Login to your Conjur server and verify that your secret exists
* Review the value of your kubernetes secret to see that it contains the same value from Conjur

```shell
# Assuming the secret name is "secret00", this will show the value
kubectl get secret -n external-secrets conjur -o jsonpath="{.data.secret00}"  | base64 --decode && echo
```

### Document License