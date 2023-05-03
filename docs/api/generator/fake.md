The Fake generator provides hard-coded key/value pairs. The intended use is just for debugging and testing.
The key/value pairs defined in `spec.data` is returned as-is.

## Example Manifest

```yaml
{% include 'generator-fake.yaml' %}
```

Example `ExternalSecret` that references the Fake generator:
```yaml
{% include 'generator-fake-example.yaml' %}
```
