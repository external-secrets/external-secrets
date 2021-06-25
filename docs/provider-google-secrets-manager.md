## Google Cloud Secret Manager

External Secrets Operator integrates with [GCP Secret Manager](https://cloud.google.com/secret-manager) for secret management.

### Authentication

At the moment, we only support [service account key](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) authentication.

#### Service account key authentication

A service account key is created and the JSON keyfile is stored in a `Kind=Secret`. The `project_id` and `private_key` should be configured for the project.

```yaml
{% include 'gcpsm-credentials-secret.yaml' %}
```

### Update secret store
Be sure the `gcpsm` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'gcpsm-secret-store.yaml' %}
```

### Creating external secret

To create a kubernetes secret from the GCP Secret Manager secret a `Kind=ExternalSecret` is needed.

```yaml
{% include 'gcpsm-external-secret.yaml' %}
```

The operator will fetch the GCP Secret Manager secret and inject it as a `Kind=Secret`
```
kubectl get secret secret-to-be-created -n <namespace> | -o jsonpath='{.data.dev-secret-test}' | base64 -d
```
