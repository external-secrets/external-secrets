## ngrok

External Secrets Operator integrates with [ngrok](https://ngrok.com/) to sync Kubernetes secrets with [ngrok Secrets for Traffic Policy](https://ngrok.com/blog-post/secrets-for-traffic-policy).
Only pushing secrets is supported. For security reasons, retrieving the secret value is not supported at this time.

### Configuring the secret store

Verify that `ngrok` provider is listed in the `Kind=SecretStore`. The properties `vaultName` and `apiKey` secret reference are required.
All other properties are optional.

```yaml
{% include 'ngrok-secret-store.yaml' %}
```

### Pushing to an external secret

To sync a Kubernetes secret with an external ngrok secret we need to create a PushSecret, this means a `Kind=PushSecret` is needed.

```yaml
{% include 'ngrok-push-secret.yaml' %}
```
