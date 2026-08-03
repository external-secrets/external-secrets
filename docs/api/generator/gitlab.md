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

GitLab deploy tokens are persistent: unlike short-lived tokens they are not garbage-collected by GitLab on their own. This generator therefore records the created token ID in its generator state and **revokes the previous token** whenever the value is regenerated (on refresh) and when the consuming `ExternalSecret` is deleted.

You can also have GitLab expire the token server-side as a backstop, using exactly one of:

- `spec.expiresAt`: an absolute expiry (RFC3339 timestamp). It is fixed, so every token minted from the spec inherits the same date and it eventually has to be bumped by hand.
- `spec.expiresAfter`: a relative expiry, a `metav1.Duration` such as `720h` using the same unit syntax as `refreshInterval` (the largest unit is hours). On every generation the token's `expires_at` is computed as `now + expiresAfter`, so each minted token carries a fresh expiry with nothing to bump. The computed timestamp is sent to GitLab only; it is never written back to the `GitlabDeployToken` object, so a GitOps-managed spec does not drift.

`spec.expiresAt` and `spec.expiresAfter` are mutually exclusive (enforced by a CEL rule). Two things to keep in mind for `expiresAfter`:

- **Minimum `24h`.** GitLab stores and enforces `expires_at` to the second, so the `24h` minimum (a CEL rule) is a deliberately conservative floor, not a GitLab limitation. Only the web UI and the `expired` flag in the API response are day-granular: a token whose `expires_at` has passed is refused for authentication straight away, but keeps reporting `expired: false` until the following UTC midnight. Do not read that flag to decide whether a token is still usable.
- **Keep it larger than `refreshInterval`.** If `expiresAfter` is shorter than the consuming `ExternalSecret`'s `refreshInterval`, a token can expire before the next refresh mints its replacement, leaving a gap. Set `expiresAfter` comfortably larger than `refreshInterval`.

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
