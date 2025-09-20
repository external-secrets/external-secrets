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

In case you have defined multiple environments in Gitlab, the secret store should be constrained to a specific `environment_scope`.

#### Environment Scope Fallback Behavior

The GitLab provider implements an intelligent fallback mechanism for environment scopes:

1. **Primary lookup**: When you configure a specific `environment` in your SecretStore (example: `environment: "production"`), the provider first tries to find variables with that exact environment scope.
2. **Automatic fallback**: If no variable is found with the specific environment scope, the provider automatically falls back to variables with "All environments" scope (`*` wildcard).
3. **Priority order**: Variables with specific environment scopes take precedence over wildcard variables when both exist.

**Example**: If your SecretStore has `environment: "production"` but your GitLab variable is set to "All environments", the variable will still be successfully retrieved through the fallback mechanism.

> **Implementation Note**: This fallback behavior is implemented in the [`getVariables` function](https://github.com/external-secrets/external-secrets/blob/636ce0578dda4a623a681066def8998a68b051a6/pkg/provider/gitlab/provider.go#L134-L151) where the provider automatically retries with `EnvironmentScope: "*"` when the initial lookup with the specific environment scope returns a 404 Not Found response.

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

#### Using DataFrom

DataFrom can be used to get a variable as a JSON string and attempt to parse it.

```yaml
{% include 'gitlab-external-secret-json.yaml' %}
```

### Getting the Kubernetes secret
The operator will fetch the project variable and inject it as a `Kind=Secret`.
```
kubectl get secret gitlab-secret-to-create -o jsonpath='{.data.secretKey}' | base64 -d
```
