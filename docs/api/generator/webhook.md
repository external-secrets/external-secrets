The Webhook generator is very similar to SecretStore generator, and provides a way to use external systems to generate sensitive information.

## HTTP Response Limit

Successful response bodies are limited to 10 MiB. A larger response fails generation with
an error instead of being loaded into controller memory. Configure the limit in bytes with
the controller's `--webhook-provider-max-response-size` flag. Helm installations can pass
the flag through `extraArgs.webhook-provider-max-response-size`.

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
