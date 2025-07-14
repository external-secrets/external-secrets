![PushSecret](../pictures/diagrams-pushsecret-basic.png)

The `PushSecret` is namespaced and it describes what data should be pushed to the secret provider.

* tells the operator what secrets should be pushed by using `spec.selector`.
* you can specify what secret keys should be pushed by using `spec.data`.
* you can also template the resulting property values using [templating](#templating).

## Example

Below is an example of the `PushSecret` in use.

``` yaml
{% include 'full-pushsecret.yaml' %}
```

The result of the created Secret object will look like:

```yaml
# The destination secret that will be templated and pushed by PushSecret.
apiVersion: v1
kind: Secret
metadata:
  name: destination-secret
stringData:
  best-pokemon-dst: "PIKACHU is the really best!"
```

## Template

When the controller reconciles the `PushSecret` it will use the `spec.template` as a blueprint to construct a new property.
You can use golang templates to define the blueprint and use template functions to transform the defined properties.
You can also pull in `ConfigMaps` that contain golang-template data using `templateFrom`.
See [advanced templating](../guides/templating.md) for details.
