STSAssumeRoleToken uses `sts:AssumeRole` to obtain temporary AWS credentials.

Unlike [`STSSessionToken`](sts.md) (which calls `GetSessionToken`), this generator works with **any** type of AWS credentials — including temporary session credentials from IRSA / pod identity — making it suitable for on-premises clusters, EKS with IRSA, or any environment where the caller already holds temporary credentials.

## When to use STSAssumeRoleToken vs STSSessionToken

| Scenario | Generator |
|---|---|
| Long-term IAM credentials, need MFA enforcement | `STSSessionToken` |
| Long-term IAM credentials, need to assume a role | `STSAssumeRoleToken` |
| IRSA / pod identity (temporary credentials) + role assumption | `STSAssumeRoleToken` |
| IRSA / pod identity, no additional role assumption needed | Use [`ECRAuthorizationToken`](ecr.md) or provider directly |

## Output Keys and Values

| Key               | Description                                                                         |
|-------------------|-------------------------------------------------------------------------------------|
| access_key_id     | The access key ID of the assumed role credentials.                                  |
| secret_access_key | The secret access key of the assumed role credentials.                              |
| session_token     | The session token required to use the temporary credentials.                        |
| expiration        | Unix timestamp (seconds) at which the credentials expire. Absent if unknown.        |

## Authentication

You can use either:

* **Static credentials** via `spec.auth.secretRef` — a Kubernetes Secret containing your long-term IAM access key and secret.
* **IRSA / service account token** via `spec.auth.jwt` — uses the pod's projected service account token to call `AssumeRoleWithWebIdentity`, then chains an additional `AssumeRole` to the target role.

If neither is specified, the AWS SDK default credential chain is used (environment variables, EC2 instance metadata, etc.).

## Request Parameters

| Field                         | Description                                                                                     |
|-------------------------------|-------------------------------------------------------------------------------------------------|
| `requestParameters.sessionDuration` | Duration in seconds for the assumed role session. Range: 900–43200. Default: 3600 (1 hour). |
| `requestParameters.externalID`      | External ID for cross-account role assumption. Required when the role trust policy enforces it. |

## Example Manifest

```yaml
{% include 'generator-stsassumerole.yaml' %}
```

Example `ExternalSecret` that references the STSAssumeRoleToken generator:
```yaml
{% include 'generator-stsassumerole-example.yaml' %}
```
