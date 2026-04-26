CodeArtifactAuthorizationTokenSpec uses the [GetAuthorizationToken](https://docs.aws.amazon.com/codeartifact/latest/APIReference/API_GetAuthorizationToken.html) API to retrieve an authorization token for AWS CodeArtifact.
The authorization token is a temporary bearer token that can be used to authenticate package manager clients (`pip`, `npm`, `maven`, `gradle`, etc.) against a CodeArtifact repository. For more information, see [CodeArtifact authentication and tokens](https://docs.aws.amazon.com/codeartifact/latest/ug/tokens-authentication.html) in the AWS CodeArtifact User Guide.

The token is valid for up to 12 hours when assuming a role, and up to 12 hours otherwise (the maximum allowed by the CodeArtifact API).

## Output Keys and Values

| Key                  | Description                                                               |
| -------------------- | ------------------------------------------------------------------------- |
| authorizationToken   | The bearer token used to authenticate against CodeArtifact.               |
| expiration           | Time when the token expires in UNIX time (seconds since 1970-01-01 UTC).  |

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
