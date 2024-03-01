External Secrets Operator integrates with [Password Depot API](https://www.password-depot.de/) to sync Password Depot to secrets held on the Kubernetes cluster.

### Authentication

The API requires a username and password. 


```yaml
{% include 'password-depot-credentials-secret.yaml' %}
```

### Update secret store
Be sure the `passworddepot` provider is listed in the `Kind=SecretStore` and host and database are set.

```yaml
{% include 'passworddepot-secret-store.yaml' %}
```


### Creating external secret

To sync a Password Depot variable to a secret on the Kubernetes cluster, a `Kind=ExternalSecret` is needed.

```yaml
{% include 'passworddepot-external-secret.yaml' %}
```

#### Using DataFrom

DataFrom can be used to get a variable as a JSON string and attempt to parse it.

```yaml
{% include 'passworddepot-external-secret-json.yaml' %}
```

### Getting the Kubernetes secret
The operator will fetch the project variable and inject it as a `Kind=Secret`.
```
kubectl get secret passworddepot-secret-to-create -o jsonpath='{.data.secretKey}' | base64 -d
```
