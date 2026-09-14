# OpenBao

External Secrets Operator integrates with [OpenBao](https://openbao.org) for secret management by using the [HashiCorp Vault Provider](./hashicorp-vault.md).

The integration was tested with [External Secrets Operator v0.16.1](https://github.com/external-secrets/external-secrets/releases/tag/v0.16.1) and [OpenBao v2.2.0](https://github.com/openbao/openbao/releases/tag/v2.2.0)

Please refer to the [HashiCorp Vault Provider](./hashicorp-vault.md) documentation for further information.

## JWT/OIDC authentication

OpenBao JWT/OIDC authentication uses either a JWT token stored in a `Kind=Secret` and referenced by `secretRef`, or a temporary Kubernetes service account token retrieved via the `TokenRequest` API.

```yaml
{% include 'openbao-jwt-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` in `secretRef` or `kubernetesServiceAccountToken.serviceAccountRef`.
