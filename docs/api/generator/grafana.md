The Grafana generator creates short-lived [Grafana Service Account Tokens](https://grafana.com/docs/grafana/latest/administration/service-accounts/).
It creates or reuses a Grafana service account (not a Kubernetes ServiceAccount) and generates a new API token for it.
When the ExternalSecret is deleted, the generated token is cleaned up automatically. Note that the Grafana service account itself is not deleted.

## Authentication

You can authenticate against the Grafana instance using either a service account token or basic auth credentials.
The credentials must have sufficient permissions to create service accounts and tokens.
See the [Grafana RBAC documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/) for details on required roles.

## Output Keys

The generator produces two keys:

| Key     | Description                                |
|---------|--------------------------------------------|
| `login` | The login name of the created Grafana service account |
| `token` | The generated Grafana service account token         |

## Example Manifests

Regardless of the authentication method, the credentials (token or user) must have permissions to manage service accounts and tokens in Grafana.
The simplest approach is to use the `Admin` role.
Alternatively, with Grafana's [fine-grained RBAC](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/), you can grant a non-Admin role the following permissions: `serviceaccounts:read`, `serviceaccounts:write`, `serviceaccounts.tokens:write`, and `serviceaccounts.tokens:delete`.

### Using Token Auth

Use a Grafana [Service Account Token](https://grafana.com/docs/grafana/latest/administration/service-accounts/#service-account-tokens) stored in a Kubernetes Secret, referenced via `spec.auth.token`.

```yaml
{% include 'generator-grafana.yaml' %}
```

### Using Basic Auth

Use a Grafana user's username and password. The password is stored in a Kubernetes Secret and referenced via `spec.auth.basic.password`, while the username is set directly in the spec.

```yaml
{% include 'generator-grafana-basicauth.yaml' %}
```

### Example ExternalSecret

An `ExternalSecret` that references the Grafana generator:

```yaml
{% include 'generator-grafana-example.yaml' %}
```
