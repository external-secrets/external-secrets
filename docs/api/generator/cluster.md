`ClusterGenerator` is a generator wrapper that is available to configure a generator
cluster-wide. The purpose of this generator is that the user doesn't have to redefine
the generator in every namespace. They could define it once in the cluster and then reference that
in the consuming `ExternalSecret`.

## Limitations

- The generator will continue to create objects in the same namespace as the referencing ExternalSecret (ES) object.
  This behavior is subject to change in future updates.
- The objects referenced within the ClusterGenerator must also reside in the same namespace as the ES object that
  references them. This is due to the inherent, namespace-scoped nature of the embedded generator types.

## Example Manifest

```yaml
{% include 'generator-cluster.yaml' %}
```

Example `ExternalSecret` that references the Cluster generator:
```yaml
{% include 'generator-cluster-example.yaml' %}
```
