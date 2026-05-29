STSAssumeRoleToken generates temporary AWS credentials using the STS AssumeRole
(or AssumeRoleWithWebIdentity for IRSA) API.

Unlike [STSSessionToken](sts.md), this generator **never calls GetSessionToken**.
`GetSessionToken` requires long-term IAM user credentials and fails when called with
temporary credentials (e.g. from IRSA / service-account tokens). Use this generator
whenever:

- You authenticate via a Kubernetes service account with IRSA, **or**
- You need to assume a specific IAM role after authentication.

## Output Keys and Values

| Key               | Description                                                                         |
|-------------------|-------------------------------------------------------------------------------------|
| access_key_id     | The access key ID that identifies the temporary security credentials.               |
| secret_access_key | The secret access key that can be used to sign requests.                            |
| session_token     | The token that users must pass to the service API to use the temporary credentials. |
| expiration        | The Unix timestamp (seconds) at which the credentials expire. Only present for temporary credentials. |

## Authentication

Two authentication methods are supported:

* **Static credentials** via `spec.auth.secretRef` — provide an IAM access key ID and secret.
* **IRSA / service-account tokens** via `spec.auth.jwt.serviceAccountRef` — the Kubernetes service account must have an `eks.amazonaws.com/role-arn` annotation. The generator exchanges the service-account token for temporary AWS credentials via `AssumeRoleWithWebIdentity`.

When `spec.role` is also set, the credentials from the chosen auth method are used to call `AssumeRole` for the specified role ARN, and the resulting short-term credentials are returned.

## Spec

| Field                                   | Type   | Required | Description |
|-----------------------------------------|--------|----------|-------------|
| `spec.region`                           | string | yes      | AWS region  |
| `spec.auth.secretRef`                   | object | no       | Static IAM credentials from a Kubernetes Secret |
| `spec.auth.jwt.serviceAccountRef`       | object | no       | IRSA service-account reference |
| `spec.role`                             | string | no       | ARN of the IAM role to assume |
| `spec.roleAssumptionParameters.sessionDuration` | int32 | no | Session duration in seconds (900–max role duration, default 3600) |
| `spec.roleAssumptionParameters.externalId`      | string | no | External ID for cross-account trust |
| `spec.roleAssumptionParameters.roleSessionName` | string | no | Identifier for the assumed-role session |

## Example Manifest

```yaml
{% include 'generator-stsassumeroletokens.yaml' %}
```

Example `ExternalSecret` that references the generator:
```yaml
{% include 'generator-stsassumeroletokens-example.yaml' %}
```
