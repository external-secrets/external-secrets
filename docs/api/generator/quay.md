`QuayAccessToken` creates a short-lived Quay Access token that can be used to authenticate against quay.io or a self-hosted instance of Quay in order to push or pull images. This requires a [Quay Robot Account configured to federate](https://docs.projectquay.io/manage_quay.html#setting-robot-federation) with a Kubernetes service account.

## Output Keys and Values

| Key        | Description                                                                    |
| ---------- | ------------------------------------------------------------------------------ |
| registry   | Domain name of the registry you are authenticating to (defaults to `quay.io`). |
| auth       | Base64 encoded authentication string.                                          |
| expiry     | Time when token expires in UNIX time (seconds since January 1, 1970 UTC).      |

## Authentication

To configure Robot Account federation, your cluster must have a publicly available [OIDC service account issuer](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery) endpoint for Quay to validate tokens against against. You can determine the issuer and subject fields by creating and decoding a service account token for the service account you wish to federate with (this is the service account you will use in `spec.serviceAccountRef`). For example, if federating with the `default` service account in the `default` namespace:

Obtain issuer:

```bash
kubectl create token default -n default | cut -d '.' -f 2 | sed 's/[^=]$/&==/' | base64 -d | jq -r '.iss'
```

Obtain subject:

```bash
kubectl create token default -n default | cut -d '.' -f 2 | sed 's/[^=]$/&==/' | base64 -d | jq -r '.sub'
```

Then use the instructions [here](https://docs.projectquay.io/manage_quay.html#setting-robot-federation) to set up a robot account and federation.

## Example Manifest

```yaml
{% include 'generator-quay.yaml' %}
```

Example `ExternalSecret` that references the Quay generator:

```yaml
{% include 'generator-quay-example.yaml' %}
```
