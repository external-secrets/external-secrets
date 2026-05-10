## Gitea Actions Secrets

External Secrets Operator integrates with [Gitea](https://gitea.com) to sync Kubernetes secrets with Gitea Actions secrets and variables.

### Features and Limitations

The Gitea provider supports **read and write** operations:

- **Read** (`ExternalSecret`): reads Gitea Actions Variables at org or repo scope.
- **Write** (`PushSecret`): creates or updates Gitea Actions Secrets at org or repo scope.

The following features are **not supported** by this provider:

- `findByTags` — Gitea Actions secrets and variables have no tag concept.
- `metadataPolicy: Fetch` — metadata is not exposed by the Gitea API.

### Authentication

The provider authenticates with a [Gitea personal access token](https://gitea.com/user/settings/applications). Create a Kubernetes secret holding the token:

```bash
kubectl create secret generic gitea-token --from-literal=token=<your-token>
```

The token needs the `write:organization` or `write:repository` scope depending on your target.

### Configuring the SecretStore

```yaml
{% include 'gitea-secret-store.yaml' %}
```

Set `organization` for org-scoped secrets. Add `repository` to scope down to a specific repo within the org. At least `organization` is required.

**NOTE:** For `ClusterSecretStore`, add `namespace` to the `auth.secretRef` pointing to the namespace where the token secret lives.

### Reading a secret (ExternalSecret)

Gitea Actions Variables are readable via `ExternalSecret`. The `remoteRef.key` maps to the variable name.

```yaml
{% include 'gitea-external-secret.yaml' %}
```

If the variable value is a JSON object you can use `remoteRef.property` to extract a specific key, or use `dataFrom` with `GetSecretMap` to expand all keys into separate secret entries.

### Pushing a secret (PushSecret)

Gitea Actions Secrets (write-only, encrypted) are the push target. The `remoteRef.remoteKey` maps to the secret name in Gitea.

```yaml
{% include 'gitea-push-secret.yaml' %}
```
