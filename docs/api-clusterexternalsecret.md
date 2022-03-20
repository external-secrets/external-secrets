The `ClusterExternalSecret` is a cluster scoped resource that can be used to push an `ExternalSecret` to specific namespaces.

Using the `namespaceSelector` you can select namespaces, and any matching namespaces will have the `ExternalSecret` specified in the `externalSecretSpec` created in it.

## Example

Below is an example of the `ClusterExternalSecret` in use.

```yaml
{% include 'full-cluster-external-secret.yaml' %}
```
