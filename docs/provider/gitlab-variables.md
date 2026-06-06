## GitLab Variables

External Secrets Operator integrates with GitLab to sync [GitLab Project Variables API](https://docs.gitlab.com/ee/api/project_level_variables.html) and/or [GitLab Group Variables API](https://docs.gitlab.com/ee/api/group_level_variables.html) to secrets held on the Kubernetes cluster.

### Configuring GitLab
The GitLab API requires an access token, project ID and/or groupIDs.

To create a new access token, go to your user settings and select 'access tokens'. Give your token a name, expiration date, and select the permissions required (Note 'api' is required).

![token-details](../pictures/screenshot_gitlab_token.png)

Click 'Create personal access token', and your token will be generated and displayed on screen. Copy or save this token since you can't access it again.
![token-created](../pictures/screenshot_gitlab_token_created.png)


### Access Token secret

Create a secret containing your access token:

```yaml
{% include 'gitlab-credentials-secret.yaml' %}
```

### Configuring the secret store
Be sure the `gitlab` provider is listed in the `Kind=SecretStore` and the ProjectID is set. If you are not using `https://gitlab.com`, you must set the `url` field as well.

In order to sync group variables `inheritFromGroups` must be true or `groupIDs` have to be defined.

> **Note**: `inheritFromGroups` and `groupIDs` are mutually exclusive. Setting both fields at the same time causes a validation error. Use `groupIDs` to sync from a fixed list of groups, or `inheritFromGroups: true` to automatically discover all parent groups of the project.

In case you have defined multiple environments in Gitlab, the secret store should be constrained to a specific `environment_scope`.

#### Environment Scope Fallback Behavior

The GitLab provider implements an intelligent fallback mechanism for environment scopes:

1. **Primary lookup**: When you configure a specific `environment` in your SecretStore (example: `environment: "production"`), the provider first tries to find variables with that exact environment scope.
2. **Automatic fallback**: If no variable is found with the specific environment scope, the provider automatically falls back to variables with "All environments" scope (`*` wildcard).
3. **Priority order**: Variables with specific environment scopes take precedence over wildcard variables when both exist.

**Example**: If your SecretStore has `environment: "production"` but your GitLab variable is set to "All environments", the variable will still be successfully retrieved through the fallback mechanism.

> **Implementation Note**: This fallback behavior is implemented in the `getVariables` function in `providers/v1/gitlab/provider.go`, where the provider automatically retries with `EnvironmentScope: "*"` when the initial lookup with the specific environment scope returns a 404 Not Found response. The same fallback applies to group variable lookups in `getGroupVariables`.

```yaml
{% include 'gitlab-secret-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `accessToken` with the namespace where the secret resides.

Your project ID can be found on your project's page.
![projectID](../pictures/screenshot_gitlab_projectID.png)

### Creating external secret

To sync a GitLab variable to a secret on the Kubernetes cluster, a `Kind=ExternalSecret` is needed.

```yaml
{% include 'gitlab-external-secret.yaml' %}
```

#### Key normalisation

When using `data:` to look up a single variable by name, hyphens in `remoteRef.key` are silently replaced with underscores before the GitLab API call. For example, `key: my-secret` will look up the GitLab variable named `my_secret`. This normalisation does not apply to `dataFrom`.

#### Using DataFrom

DataFrom can be used to get a variable as a JSON string and attempt to parse it.

```yaml
{% include 'gitlab-external-secret-json.yaml' %}
```

#### DataFrom find limitations

When using `dataFrom` with `find`, the following restrictions apply:

- `find.name` is mandatory. The provider requires a name regexp to select which variables to sync.
- `find.tags` only supports the `environment_scope` key. Any other tag key causes an error. Setting `find.tags.environment_scope` while the SecretStore already has an `environment` configured also causes an error, as the two would conflict.
- `find.path` is not implemented in the GitLab provider and returns an error if set.

### Getting the Kubernetes secret
The operator will fetch the project variable and inject it as a `Kind=Secret`.
```
kubectl get secret gitlab-secret-to-create -o jsonpath='{.data.secretKey}' | base64 -d
```
