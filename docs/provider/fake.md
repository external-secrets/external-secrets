We provide a `fake` implementation to help with testing. This provider returns static key/value pairs and nothing else.
To use the `fake` provider simply create a `SecretStore` or `ClusterSecretStore` and configure it like in the following example:

!!! note inline end
    The provider returns static data configured in `value`. You can define a `version`, too. If set the `remoteRef` from an ExternalSecret must match otherwise no value is returned.

```yaml
{% include 'fake-provider-store.yaml' %}
```

Please note that `value` is intended for exclusive use with `data` for `dataFrom`. You can use the `data` to set a `JSON` compliant value to be used as `dataFrom`.

Here is an example `ExternalSecret` that displays this behavior:

!!! warning inline end
    This provider supports specifying different `data[].version` configurations. However, `data[].property` is ignored.

```yaml
{% include 'fake-provider-es.yaml' %}
```

This results in the following secret:


```yaml
{% include 'fake-provider-secret.yaml' %}
```
