![HCP Vault](./pictures/diagrams-provider-vault.png)

## Hashicorp Vault

External Secrets Operator integrates with [HashiCorp Vault](https://www.vaultproject.io/) for secret
management. Vault itself implements lots of different secret engines, as of now we only support the
[KV Secrets Engine](https://www.vaultproject.io/docs/secrets/kv).

### Authentication

We support three different modes for authentication:
[token-based](https://www.vaultproject.io/docs/auth/token),
[appRole](https://www.vaultproject.io/docs/auth/approle) and
[kubernetes-native](https://www.vaultproject.io/docs/auth/kubernetes), each one comes with it's own
trade-offs. Depending on the authentication method you need to adapt your environment.

#### Token-based authentication

A static token is stored in a `Kind=Secret` and is used to authenticate with vault.

```yaml
{% include 'vault-token-store.yaml' %}
```

#### AppRole authentication example

[AppRole authentication](https://www.vaultproject.io/docs/auth/approle) reads the secret id from a
`Kind=Secret` and uses the specified `roleId` to aquire a temporary token to fetch secrets.

```yaml
{% include 'vault-approle-store.yaml' %}
```

#### Kubernetes authentication

[Kubernetes-native authentication](https://www.vaultproject.io/docs/auth/kubernetes) has three
options of optaining credentials for vault:

1.  by using a service account jwt referenced in `serviceAccountRef`
2.  by using the jwt from a `Kind=Secret` referenced by the `secretRef`
3.  by using transient credentials from the mounted service account token within the
    external-secrets operator

```yaml
{% include 'vault-kubernetes-store.yaml' %}
```
