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

You can choose from these authentication mechanisms:

* static credentials using `spec.auth.secretRef`
* point to an IRSA Service Account with `spec.auth.jwt`
* use credentials from the [SDK default credentials chain](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default) from the controller environment (includes [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html) when the External Secrets controller ServiceAccount is associated with a Pod Identity)

If `spec.role` is set, ESO assumes that role using whatever controller credentials are already available (Pod Identity, IRSA on the controller ServiceAccount, or static keys). You do **not** need `spec.auth.jwt` for Pod Identity.

### AWS Pod Identity

Configure [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html) on the External Secrets controller ServiceAccount, then omit `spec.auth`. Optionally set `spec.role` to a workload role that the controller identity can assume:

```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: ECRAuthorizationToken
metadata:
  name: ecr-gen
spec:
  region: eu-west-1
  # optional: assume this role using the controller's Pod Identity
  role: "arn:aws:iam::111122223333:role/ecr-token-role"
```

## Example Manifest

```yaml
{% include 'generator-ecr.yaml' %}
```

Example `ExternalSecret` that references the ECR generator:
```yaml
{% include 'generator-ecr-example.yaml' %}
```
