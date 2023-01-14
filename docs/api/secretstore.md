![SecretStore](../pictures/diagrams-high-level-ns-detail.png)


The `SecretStore` is namespaced and specifies how to access the external API.
The SecretStore maps to exactly one instance of an external API.

By design, SecretStores are bound to a namespace and can not reference resources across namespaces.
If you want to design cross-namespace SecretStores you must use [ClusterSecretStores](./clustersecretstore.md) which do not have this limitation.

## Example

For a full list of supported fields see [spec](./spec.md) or dig into our [guides](../guides/introduction.md).

``` yaml
{% include 'full-secret-store.yaml' %}
```
