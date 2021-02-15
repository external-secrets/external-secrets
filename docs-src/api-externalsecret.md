The `ExternalSecret` describes what data should be fetched, how the data should be transformed and saved as a `Kind=Secret`:

* tell the operator what secrets should be synced by using `spec.data` to explicitly sync individual keys or use `spec.dataFrom` to get all values from the external API.
* you can specify how the secret should look like by specifying a `spec.target.template`

## Example

Take a look at an annotated example to understand the design behind the `ExternalSecret`.

``` yaml
{% include 'full-external-secret.yaml' %}
```
