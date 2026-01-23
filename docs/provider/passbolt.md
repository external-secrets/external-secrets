External Secrets Operator integrates with [Passbolt API](https://www.passbolt.com/) to sync Passbolt to secrets held on the Kubernetes cluster.



### Creating a Passbolt secret store

Be sure the `passbolt` provider is listed in the `Kind=SecretStore` and auth and host are set.
The API requires a password and private key provided in a secret.

```yaml
{% include 'passbolt-secret-store.yaml' %}
```


### Creating an external secret

To sync a Passbolt secret to a Kubernetes secret, a `Kind=ExternalSecret` is needed.
By default the secret contains name, username, uri, password and description.

To only select a single property add the `property` key.

```yaml
{% include 'passbolt-external-secret-example.yaml' %}
```

The above external secret will lead to the creation of a secret in the following form:

```yaml
{% include 'passbolt-secret-example.yaml' %}
```


### Finding a secret by name

Instead of retrieving secrets by ID you can also use `dataFrom` to search for secrets by name.

```yaml
{% include 'passbolt-external-secret-findbyname.yaml' %}
```
