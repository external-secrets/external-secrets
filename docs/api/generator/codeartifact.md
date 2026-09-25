CodeArtifactAuthorizationTokenSpec uses the [GetAuthorizationToken](https://docs.aws.amazon.com/codeartifact/latest/APIReference/API_GetAuthorizationToken.html) API to retrieve an authorization token for AWS CodeArtifact.
The authorization token is a temporary bearer token that can be used to authenticate package manager clients (`pip`, `npm`, `maven`, `gradle`, etc.) against a CodeArtifact repository. For more information, see [CodeArtifact authentication and tokens](https://docs.aws.amazon.com/codeartifact/latest/ug/tokens-authentication.html) in the AWS CodeArtifact User Guide.

The token is valid for up to 12 hours (the maximum allowed by the CodeArtifact API).

## Spec Fields

| Field             | Required | Description                                                                                                                                                                                                                                    |
| ----------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `region`          | yes      | AWS region to call `GetAuthorizationToken` in.                                                                                                                                                                                                 |
| `domain`          | yes      | Name of the CodeArtifact domain.                                                                                                                                                                                                               |
| `domainOwner`     | yes      | 12-digit AWS account ID that owns the CodeArtifact domain.                                                                                                                                                                                     |
| `auth`            | no       | Authentication method (see [Authentication](#authentication) below). When omitted, the controller's default AWS credentials chain is used.                                                                                                     |
| `role`            | no       | ARN of an IAM role to assume before calling `GetAuthorizationToken`.                                                                                                                                                                           |
| `durationSeconds` | no       | Token lifetime in seconds. Valid values are `0` or any integer between `900` (15 minutes) and `43200` (12 hours). `0` matches the caller's temporary-credentials expiration. AWS defaults to `43200` when omitted. Validated by the API server. |

## Output Keys and Values

| Key                  | Description                                                               |
| -------------------- | ------------------------------------------------------------------------- |
| `authorizationToken` | The bearer token used to authenticate against CodeArtifact.               |
| `expiration`         | Time when the token expires in UNIX time (seconds since 1970-01-01 UTC).  |

## Authentication

You can choose from three authentication mechanisms:

* static credentials using `spec.auth.secretRef`
* point to an IRSA Service Account with `spec.auth.jwt`
* use credentials from the [SDK default credentials chain](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default) from the controller environment

## Example Manifest

```yaml
{% include 'generator-codeartifact.yaml' %}
```

Example `ExternalSecret` that references the CodeArtifact generator:
```yaml
{% include 'generator-codeartifact-example.yaml' %}
```
