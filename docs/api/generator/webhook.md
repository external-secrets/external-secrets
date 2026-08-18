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

Entries under `spec.secrets` follow the same rules as the webhook provider: each entry must set
exactly one of `secretRef` or `serviceAccountRef`. A `serviceAccountRef` entry mints a short-lived
token for the referenced (and `external-secrets.io/type: webhook` labeled) ServiceAccount via the
TokenRequest API and exposes it to templates as `{% raw %}{{ .<name>.token }}{% endraw %}`.

This will generate a kubernetes secret with the following values:
```yaml
parameter: test
```
