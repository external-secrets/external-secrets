
The Azure Container Registry (ACR) generator creates a short-lived refresh or access token for accessing ACR.
The token is generated for a particular ACR registry defined in `spec.registry`.

## Output Keys and Values

| Key      | Description |
| -------- | ----------- |
| username | username for the `docker login` command |
| password | password for the `docker login` command |


## Authentication

You must choose one out of three authentication mechanisms:

- service principal
- managed identity
- workload identity

The generated token will inherit the permissions from the assigned policy. I.e. when you assign a read-only policy all generated tokens will be read-only.

You can scope tokens to a particular repository using `spec.scope`.

## Scope

First, an Azure Active Directory access token is obtained with the desired authentication method.
This AAD access token will be used to authenticate against ACR to issue a refresh token or access token.
If `spec.scope` if it is defined it obtains an ACR access token. If  `spec.scope` is missing it obtains an ACR refresh token:

- access tokens are scoped to a specific repository or action (pull,push)
- refresh tokens can are scoped to whatever policy is attached to the identity that creates the acr refresh token

The Scope grammar is defined in the [Docker Registry spec](https://docs.docker.com/registry/spec/auth/scope/).
Note: You **can not** use a wildcards in the scope parameter, you can match exactly one repository and defined multiple actions like `pull` or `push`.

Example scopes:

```
repository:my-repository:pull,push
repository:my-repository:pull
```

## Example Manifest

```yaml
{% include 'generator-acr.yaml' %}
```

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: azurecr-credentials
spec:
  dataFrom:
    - sourceRef:
        generatorRef:
          apiVersion: generators.external-secrets.io/v1alpha1
          kind: ACRAccessToken
          name: myregistry.azurecr.io
  refreshInterval: 12h
  target:
    name: azurecr-credentials
    template:
      type: kubernetes.io/dockerconfigjson
      data:
        .dockerconfigjson: |
          {
            "auths": {
              "myregistry.azurecr.io": {
                "username": "{{ .username }}",
                "identitytoken": "{{ .password }}",
              }
            }
          }
```