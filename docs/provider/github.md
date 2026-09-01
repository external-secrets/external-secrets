# GitHub

External Secrets Operator integrates with GitHub to sync Kubernetes secrets with [GitHub Actions secrets](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions) or [Dependabot secrets](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/managing-encrypted-secrets-for-dependabot).

## Limitations

The GitHub provider is **write-only**, designed specifically to **create and update** GitHub Actions or Dependabot secrets using the
[GitHub REST API](https://docs.github.com/en/rest/actions/secrets) or the [Dependabot secrets API](https://docs.github.com/en/rest/dependabot/secrets), and does not support **fetching the secret values**.

## Configuring GitHub provider

The GitHub API requires to install the ESO app to your GitHub organisation in order to use the GitHub provider features. The same App ID, installation ID, and private key authentication are used for both secret types. Grant the app read and write access to **Secrets** for Actions secrets, or **Dependabot secrets** for Dependabot secrets, at the target organization or repository scope.

## Configuring the secret store

Verify that `github` provider is listed in the `Kind=SecretStore`. The properties `appID`, `installationID`, `organization` are required to register the provider. In addition, authentication has to be provided.

Set `secretType` to `Actions` or `Dependabot`. `Actions` is the default when the field is omitted, preserving compatibility with existing stores.

Optionally, to target Actions `repository` and `environment` secrets, the fields `repository` and `environment` need also to be added. For Dependabot repository secrets, add the `repository` field.

| Secret type | Organization | Repository | Environment |
| :-- | :--: | :--: | :--: |
| `Actions` | Yes | Yes | Yes |
| `Dependabot` | Yes | Yes | No |

Organization scope is used when `repository` is omitted. Combining `secretType: Dependabot` with `environment` is invalid.

For organization secrets, the optional `orgSecretVisibility` field controls the visibility of secrets created via PushSecret. Valid values are `all` or `private`. When unset, new secrets are created with visibility `all` and existing secrets keep whatever visibility they already have in GitHub. Updates also preserve existing selected-repository associations. Repository-scoped secrets do not use organization visibility or selected-repository associations.

```yaml
{% include 'github-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `auth.privateKey` with the namespace where the secret resides.

## Pushing to an external secret

To sync a Kubernetes secret with an external GitHub secret we need to create a PushSecret, this means a `Kind=PushSecret` is needed.

```yaml
{% include 'github-push-secret.yaml' %}
```
