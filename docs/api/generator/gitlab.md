## GitLab Deploy Token Generator

The GitLab Deploy Token generator creates [GitLab deploy tokens](https://docs.gitlab.com/user/project/deploy_tokens/) for a project or a group. A deploy token gives read or write access to a project's repository, container registry, and package registry, which makes it well suited for pulling images or packages from automation.

The generated secret contains two keys:

- `username`: the deploy token username (the value of `spec.username`, or the `gitlab+deploy-token-{n}` value GitLab assigns when `username` is omitted).
- `token`: the deploy token value.

### Authentication

The generator authenticates against the GitLab API with an access token (personal, group, or project) that has the `api` scope and at least the **Maintainer** role on the target project (or **Owner** on the target group). Store that token in a Kubernetes secret and reference it from `spec.auth.token.secretRef`.

```bash
kubectl create secret generic gitlab-api-token --from-literal=token=glpat-xxxxxxxxxxxx
```

### Target

Set exactly one of `spec.projectID` or `spec.groupID`. Both accept either a numeric ID or an unescaped path such as `group/project`, the generator URL-escapes paths before calling the API, so do not pre-encode them. Setting both, neither, or an empty string is rejected by the CRD.

### Scopes

`spec.scopes` requires at least one of: `read_repository`, `read_registry`, `write_registry`, `read_package_registry`, `write_package_registry`. Projects additionally support `read_virtual_registry` and `write_virtual_registry`.

### Token lifecycle

GitLab deploy tokens are persistent: unlike short-lived tokens they are not garbage-collected by GitLab on their own. This generator therefore records the created token ID in its generator state and **revokes the previous token** whenever the value is regenerated (on refresh) and when the consuming `ExternalSecret` is deleted. Set `spec.expiresAt` if you also want GitLab to expire the token server-side as a backstop.

### Example Manifest

```yaml
{% include 'generator-gitlab.yaml' %}
```

Example `ExternalSecret` that references the generator:

```yaml
{% include 'generator-gitlab-example.yaml' %}
```

### Notes

- The access token used for authentication is never written to the target secret; only the generated deploy token is.
- Each refresh creates a new deploy token and revokes the prior one, so the token value rotates on every `refreshInterval`.
