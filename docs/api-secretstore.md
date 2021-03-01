![SecretStore](./pictures/diagrams-high-level-ns-detail.png)


The `SecretStore` is namespaced and specifies how to access the external API.
The SecretStore maps to exactly one instance of an external API.

``` yaml
{% include 'full-secret-store.yaml' %}
```
