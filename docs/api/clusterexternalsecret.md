![ClusterExternalSecret](../pictures/diagrams-cluster-external-secrets.png)

The `ClusterExternalSecret` is a cluster scoped resource that can be used to manage `ExternalSecret` resources in specific namespaces.

With `namespaceSelectors` you can select namespaces in which the ExternalSecret should be created.
If there is a conflict with an existing resource the controller will error out.

## Example

Below is an example of the `ClusterExternalSecret` in use.

```yaml
{% include 'full-cluster-external-secret.yaml' %}
```

## Synchronizing corresponding ExternalSecrets

Regular refreshes can be controlled using the `refreshPolicy` and
`refreshInterval` fields. Adhoc synchronizations can be triggered by
setting, updating or deleting the annotation `external-secrets.io/force-sync`
on the ClusterExternalSecret:

```
kubectl annotate ces my-ces external-secrets.io/force-sync=$(date +%s) --overwrite
```

Changes to this annotation will be synchronized to all ExternalSecrets
owned by the ClusterExternalSecret.

## Deprecations

### namespaceSelector

The field `namespaceSelector` has been deprecated in favor of `namespaceSelectors` and will be removed in a future
version.
