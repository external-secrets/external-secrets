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

### Using Token Auth

Use a Grafana [Service Account Token](https://grafana.com/docs/grafana/latest/administration/service-accounts/#service-account-tokens) to authenticate.
The token must belong to a service account with `Admin` role so it can create other service accounts and tokens.
Store the token in a Kubernetes Secret and reference it via `spec.auth.token`.

```yaml
{% include 'generator-grafana.yaml' %}
```

### Using Basic Auth

Alternatively, you can use basic auth credentials (username and password) to authenticate against the Grafana instance.
The user must have sufficient permissions to manage service accounts.
Store the password in a Kubernetes Secret and reference it via `spec.auth.basic.password`.

```yaml
{% include 'generator-grafana-basicauth.yaml' %}
```

Example `ExternalSecret` that references the Grafana generator:
```yaml
{% include 'generator-grafana-example.yaml' %}
```
