![ClusterSecretStore](./pictures/diagrams-high-level-cluster-detail.png)

The `ClusterSecretStore` is a cluster scoped SecretStore that can be referenced by all
`ExternalSecrets` from all namespaces. Use it to offer a central gateway to your secret backend.

``` yaml
{% include 'full-secret-store.yaml' %}
```
