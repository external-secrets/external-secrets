![ClusterSecretStore](../pictures/diagrams-high-level-cluster-detail.png)

The `ClusterSecretStore` is a cluster scoped SecretStore that can be referenced by all
`ExternalSecrets` from all namespaces. Use it to offer a central gateway to your secret backend.

Different Store Providers have different stability levels, maintenance status, and support. 
To check the full list, please see [Stability Support](../introduction/stability-support.md).

!!! note "Unmaintained Stores generate events"
    Admission webhooks and controllers will emit warning events for providers without a explicit maintainer.
    To disable controller warning events, you can add `external-secrets.io/ignore-maintenance-checks: "true"` annotation to the SecretStore.
    Admission webhook warning cannot be disabled.

## Example

For a full list of supported fields see [spec](./spec.md) or dig into our [guides](../guides/introduction.md).

``` yaml
{% include 'full-cluster-secret-store.yaml' %}
```
