`ArtifactoryAccessToken` creates short-lived JFrog Artifactory access tokens suitable for authenticating against container registries. The generator supports OIDC token exchange with Kubernetes service accounts or scoped token creation from bootstrap credentials stored in a Secret.

## Output Keys and Values

| Key              | Description                                                                 |
| ---------------- | --------------------------------------------------------------------------- |
| registry         | Docker registry hostname (from `spec.registry` or parsed from `spec.url`). |
| access_token     | Full Artifactory access token from the API response.                        |
| reference_token  | Shortened reference token when requested and returned by the API.           |
| username         | Username from the API or OIDC response; falls back to the JWT `sub` claim.  |
| auth             | Base64-encoded `username:token` for dockerconfigjson templates.             |
| expiry           | Token expiry as UNIX timestamp (seconds).                                   |

The `auth` key prefers `reference_token` when both tokens are present, since reference tokens are shorter and commonly used for Docker login.

## Authentication

### OIDC (recommended)

Configure a Generic OIDC provider in JFrog Platform that trusts your Kubernetes cluster's service account issuer. ESO exchanges a Kubernetes service account token for an Artifactory access token via `POST /access/api/v1/oidc/token`.

Prerequisites:

1. JFrog Platform OIDC provider configured for your Kubernetes issuer.
2. Identity mappings that map Kubernetes subjects to Artifactory users or groups.
3. A Kubernetes service account referenced in `spec.auth.oidc.serviceAccountRef`.

Obtain the cluster issuer:

```bash
kubectl create token default -n default | cut -d '.' -f 2 | sed 's/[^=]$/&==/' | base64 -d | jq -r '.iss'
```

### Reference token bootstrap

When OIDC is not available, provide a bootstrap identity or access token in a Kubernetes Secret. ESO creates a scoped token via `POST /access/api/v1/tokens` using `Authorization: Bearer`.

## Configuration Parameters

| Parameter | Description | Required |
| --------- | ----------- | -------- |
| `url` | JFrog Platform base URL (e.g. `https://acme.jfrog.io`). | Yes |
| `registry` | Docker registry hostname for templates. Defaults to URL hostname. | No |
| `auth.oidc.providerName` | OIDC provider name in JFrog Platform. | Yes (OIDC) |
| `auth.oidc.serviceAccountRef` | Kubernetes service account for token exchange. | Yes (OIDC) |
| `auth.oidc.providerType` | OIDC provider type. Defaults to `GenericOidc`. | No |
| `auth.referenceToken.token` | Secret reference to bootstrap token. | Yes (reference) |
| `auth.referenceToken.scope` | Token scope (e.g. `applied-permissions/user`). | Yes (reference) |
| `auth.referenceToken.expiresIn` | Token lifetime in seconds. | No |

The table above lists the main fields. Optional reference-token parameters include `applicationKey`, `projectKey`, `identityMappingName`, `includeReferenceToken`, `refreshable`, and `description`.

## Example Manifests

OIDC configuration:

```yaml
{% include 'generator-artifactory-oidc.yaml' %}
```

Reference-token bootstrap:

```yaml
{% include 'generator-artifactory-reference.yaml' %}
```

Example `ExternalSecret` that references the Artifactory generator:

```yaml
{% include 'generator-artifactory-example.yaml' %}
```

## Using the Generated Docker Registry Secret

Reference the secret in pod `imagePullSecrets`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  imagePullSecrets:
    - name: artifactory-credentials
  containers:
    - name: app
      image: acme.jfrog.io/docker-local/my-image:latest
```
