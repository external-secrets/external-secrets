The `ClusterPushSecret` is a cluster scoped resource that can be used to manage `PushSecret` resources in specific namespaces.

With `namespaceSelectors` you can select namespaces in which the PushSecret should be created.
If there is a conflict with an existing resource the controller will error out.

## Example

Below is an example of the `ClusterPushSecret` in use.

```yaml
{% include 'full-cluster-push-secret.yaml' %}
```
