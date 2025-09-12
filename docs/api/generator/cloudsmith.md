`CloudsmithAccessToken` creates a short-lived Cloudsmith access token that can be used to authenticate against Cloudsmith's container registry for pushing or pulling container images. This generator uses OIDC token exchange to authenticate with Cloudsmith using a Kubernetes service account token and generates Docker registry credentials in dockerconfigjson format.

## Output Keys and Values

| Key        | Description                                                                    |
| ---------- | ------------------------------------------------------------------------------ |
| auth       | Base64 encoded authentication string for Docker registry access.              |
| expiry     | Time when token expires in UNIX time (seconds since January 1, 1970 UTC).    |

## Authentication

To use the Cloudsmith generator, you must configure OIDC authentication between your Kubernetes cluster and Cloudsmith. Your cluster must have a publicly available [OIDC service account issuer](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery) endpoint for Cloudsmith to validate tokens against.

### Prerequisites

1. **Cloudsmith OIDC Service**: Configure an OIDC service in your Cloudsmith organization that trusts your Kubernetes cluster's OIDC issuer.
2. **Service Account**: Create a Kubernetes service account that will be used for token exchange.
3. **Proper Audiences**: The service account token must include the appropriate audience for Cloudsmith (typically `https://api.cloudsmith.io`).

### Service Account Configuration

You can determine the issuer and subject fields by creating and decoding a service account token for the service account you wish to use (this is the service account you will specify in `spec.serviceAccountRef`). For example, if using the `default` service account in the `default` namespace:

Obtain issuer:

```bash
kubectl create token default -n default | cut -d '.' -f 2 | sed 's/[^=]$/&==/' | base64 -d | jq -r '.iss'
```

Use these values when configuring the OIDC service in your Cloudsmith Workspace settings.

## Configuration Parameters

| Parameter           | Description                                                              | Required |
| ------------------- | ------------------------------------------------------------------------ | -------- |
| `apiHost`          | The Cloudsmith API host. Defaults to `api.cloudsmith.io`.               | No       |
| `orgSlug`          | The organization slug in Cloudsmith.                                    | Yes      |
| `serviceSlug`      | The OIDC service slug configured in Cloudsmith.                         | Yes      |
| `serviceAccountRef` | Reference to the Kubernetes service account for OIDC token exchange.    | Yes      |

## Example Manifest

```yaml
{% include 'generator-cloudsmith.yaml' %}
```

Example `ExternalSecret` that references the Cloudsmith generator:

```yaml
{% include 'generator-cloudsmith-example.yaml' %}
```

## Using the Generated Docker Registry Secret

Once the dockerconfigjson secret is created, you can use it to authenticate with Cloudsmith's container registry in several ways:

### In Pod Specifications

Reference the secret in your pod's `imagePullSecrets`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  imagePullSecrets:
    - name: cloudsmith-credentials
  containers:
    - name: app
      image: docker.cloudsmith.io/my-org/my-repo/my-image:latest
```

### In ServiceAccount

Add the secret to a ServiceAccount for automatic usage:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
imagePullSecrets:
  - name: cloudsmith-credentials
```

### For Docker CLI Authentication

Extract the dockerconfigjson and use it with Docker:

```bash
kubectl get secret cloudsmith-credentials -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d > ~/.docker/config.json
docker pull docker.cloudsmith.io/my-org/my-repo/my-image:latest
```

## Usage Notes

- **Container Registry Access**: The generated dockerconfigjson secret is specifically designed for authenticating with Cloudsmith's container registry to push or pull Docker images.
- **Token Lifetime**: Cloudsmith access tokens have a limited lifetime. The `expiry` field in the generated secret indicates when the token will expire.
- **Refresh Interval**: Set an appropriate `refreshInterval` in your `ExternalSecret` to ensure tokens are refreshed before expiration.
- **Permissions**: The generated token will have the same permissions as the OIDC service configured in Cloudsmith for container registry access.

## Troubleshooting

- **Token Exchange Fails**: Verify that your OIDC service in Cloudsmith is correctly configured with your cluster's issuer.
- **Invalid Audience**: Ensure the service account token includes the correct audience for Cloudsmith API.
- **Network Issues**: Check that your cluster can reach the Cloudsmith API endpoint specified in `apiHost`.
- **Container Image Pull Fails**: Verify that the generated dockerconfigjson secret is properly referenced in your pod's `imagePullSecrets` and that the image exists in your Cloudsmith container registry.
- **Registry Domain Issues**: Ensure you're using the correct registry domain format (e.g., `docker.cloudsmith.io/org/repo/image:tag`) in your image references.
- **Permissions**: Confirm that your OIDC service in Cloudsmith has the necessary permissions to pull/push container images from the specific repositories.
