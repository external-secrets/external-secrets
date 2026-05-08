RDSIAMAuthToken builds an AWS RDS IAM authentication token that can be used as the database password for IAM database authentication.

The AWS principal used by the generator must be allowed to connect to the target DB user with `rds-db:connect`. This permission belongs in IAM policy, not in the generator spec.

`spec.controller` is an ESO controller-class selector. It is useful when multiple ESO controllers run in the same cluster and only one controller should process a given generator. It does not change the AWS principal or grant database access.

## Output Keys and Values

| Key        | Description                                                                     |
| ---------- | ------------------------------------------------------------------------------- |
| username   | Database user used when building the token.                                     |
| password   | Generated RDS IAM authentication token.                                         |
| token      | Same value as `password`, provided for consumers that prefer an explicit name.  |
| hostname   | RDS endpoint hostname.                                                          |
| port       | RDS endpoint port.                                                              |
| endpoint   | RDS endpoint in `hostname:port` form.                                           |
| expires_at | Time when the token expires in UNIX time (seconds since January 1, 1970 UTC).   |

## Token Lifetime and Refresh

AWS RDS IAM authentication tokens are valid for 15 minutes. Set the `ExternalSecret` `refreshInterval` below that lifetime so ESO refreshes the target Secret before the generated token expires. A `10m` refresh interval is a reasonable default; use `5m` when you want a larger safety margin for controller delays or slow consumers.

The `expires_at` output key is written for consumers, templates, and observability. ESO does not use `expires_at` to schedule the next refresh; refresh scheduling is controlled by the `ExternalSecret` refresh policy and `refreshInterval`.

Avoid `CreatedOnce`, `OnChange`, or `refreshInterval: 0` for this generator unless another process refreshes the Secret before the RDS IAM token expires.

## Authentication

You can choose from three authentication mechanisms:

* static credentials using `spec.auth.secretRef`
* point to an IRSA Service Account with `spec.auth.jwt`
* use credentials from the SDK default credentials chain from the controller environment

For EKS Pod Identity, run the ESO controller under the Kubernetes ServiceAccount associated with the desired IAM role and omit `spec.auth`.

## Example Manifest

```yaml
{% include 'generator-rdsiam.yaml' %}
```

Example `ExternalSecret` that references the RDS IAM generator:

```yaml
{% include 'generator-rdsiam-example.yaml' %}
```
