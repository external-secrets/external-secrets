# GitHub

External Secrets Operator integrates with GitHub to sync Kubernetes secrets with [GitHub Actions secrets](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions).

## Limitations

The GitHub provider is **write-only**, designed specifically to **create and update** GitHub Actions secrets using the
[GitHub REST API](https://docs.github.com/en/rest/actions/secrets), and does not support **fetching the secret values**.

## Configuring GitHub provider

The GitHub API requires to install the ESO app to your GitHub organisation in order to use the GitHub provider features.

## Configuring the secret store

Verify that `github` provider is listed in the `Kind=SecretStore`. The properties `appID`, `installationID`, `organization` are required to register the provider. In addition, authentication has to be provided.

Optionally, to target `repository` and `environment` secrets, the fields `repository` and `environment` need also to be added.

For organization secrets, the optional `orgSecretVisibility` field controls the visibility of secrets created via PushSecret. Valid values are `all` or `private`. When unset, new secrets are created with visibility `all` and existing secrets keep whatever visibility they already have in GitHub.

```yaml
{% include 'github-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `auth.privateKey` with the namespace where the secret resides.

## Pushing to an external secret

To sync a Kubernetes secret with an external GitHub secret we need to create a PushSecret, this means a `Kind=PushSecret` is needed.

```yaml
{% include 'github-push-secret.yaml' %}
```
