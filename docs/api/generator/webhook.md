The Webhook generator is very similar to SecretStore generator, and provides a way to use external systems to generate sensitive information.

## Output Keys and Values

Webhook calls are expected to produce valid JSON objects. All keys within that JSON object will be exported as keys to the kubernetes Secret.

## Example Manifest

```yaml
{% include 'generator-webhook.yaml' %}
```

Example `ExternalSecret` that references the Webhook generator using an internal `Secret`:
```yaml
{% include 'generator-webhook-example.yaml' %}
```

This will generate a kubernetes secret with the following values:
```yaml
parameter: test
```
