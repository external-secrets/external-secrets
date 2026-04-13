The Grafana generator creates short-lived [Grafana Service Account Tokens](https://grafana.com/docs/grafana/latest/administration/service-accounts/).
It creates or reuses a service account and generates a new token for it. When the ExternalSecret is deleted, the generated token is cleaned up automatically.

## Authentication

You can authenticate against the Grafana instance using either a service account token or basic auth credentials.
The credentials must have sufficient permissions to create service accounts and tokens.
See the [Grafana RBAC documentation](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/) for details on required roles.

## Output Keys

The generator produces two keys:

| Key     | Description                                |
|---------|--------------------------------------------|
| `login` | The login name of the created service account |
| `token` | The generated service account token         |

## Example Manifests

### Using Token Auth

```yaml
{% include 'generator-grafana.yaml' %}
```

### Using Basic Auth

```yaml
{% include 'generator-grafana-basicauth.yaml' %}
```

### ExternalSecret referencing the Grafana generator

```yaml
{% include 'generator-grafana-example.yaml' %}
```
