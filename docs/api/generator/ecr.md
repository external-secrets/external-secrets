ECRAuthorizationTokenSpec uses the GetAuthorizationToken API to retrieve an authorization token.
The authorization token is valid for 12 hours. For more information, see [registry authentication](https://docs.aws.amazon.com/AmazonECR/latest/userguide/Registries.html#registry_auth) in the Amazon Elastic Container Registry User Guide.


## Output Keys and Values

| Key            | Description                                                                       |
| -------------- | --------------------------------------------------------------------------------- |
| username       | username for the `docker login` command.                                          |
| password       | password for the `docker login` command.                                          |
| proxy_endpoint | The registry URL to use for this authorization token in a `docker login` command. |
| expires_at     | time when token expires in UNIX time (seconds since January 1, 1970 UTC).         |

## Authentication

You can choose from three authentication mechanisms:

* static credentials using `spec.auth.secretRef`
* point to a IRSA Service Account with `spec.auth.jwt`
* use credentials from the [SDK default credentials chain](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default) from the controller environment

## Example Manifest

```yaml
{% include 'generator-ecr.yaml' %}
```

Example `ExternalSecret` that references the ECR generator:
```yaml
{% include 'generator-ecr-example.yaml' %}
```