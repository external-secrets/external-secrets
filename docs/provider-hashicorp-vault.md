![HCP Vault](./pictures/diagrams-provider-vault.png)

## Hashicorp Vault

External Secrets Operator integrates with [HashiCorp Vault](https://www.vaultproject.io/) for secret
management. Vault itself implements lots of different secret engines, as of now we only support the
[KV Secrets Engine](https://www.vaultproject.io/docs/secrets/kv).

### Example

First, create a SecretStore with a vault backend. For the sake of simplicity we'll use a static token `root`:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: SecretStore
metadata:
  name: vault-backend
spec:
  provider:
    vault:
      server: "http://my.vault.server:8200"
      path: "secret"
      version: "v2"
      auth:
        # points to a secret that contains a vault token
        # https://www.vaultproject.io/docs/auth/token
        tokenSecretRef:
          name: "vault-token"
          key: "token"
---
apiVersion: v1
kind: Secret
metadata:
  name: vault-token
data:
  token: cm9vdA== # "root"
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `tokenSecretRef` with the namespace of the secret that we just created.

Then create a simple k/v pair at path `secret/foo`:

```
vault kv put secret/foo my-value=s3cr3t
```

Now create a ExternalSecret that uses the above SecretStore:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
metadata:
  name: vault-example
spec:
  refreshInterval: "15s"
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: example-sync
  data:
  - secretKey: foobar
    remoteRef:
      key: secret/foo
      property: my-value
---
# will create a secret with:
kind: Secret
metadata:
  name: example-sync
data:
  foobar: czNjcjN0
```

#### Fetching Raw Values

You can fetch all key/value pairs for a given path If you leave the `remoteRef.property` empty. This returns the json-encoded secret value for that path.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
metadata:
  name: vault-example
spec:
  # ...
  data:
  - secretKey: foobar
    remoteRef:
      key: /dev/package.json
```

#### Nested Values

Vault supports nested key/value pairs. You can specify a [gjson](https://github.com/tidwall/gjson) expression at `remoteRef.property` to get a nested value.

Given the following secret - assume its path is `/dev/config`:
```json
{
  "foo": {
    "nested": {
      "bar": "mysecret"
    }
  }
}
```

You can set the `remoteRef.property` to point to the nested key using a [gjson](https://github.com/tidwall/gjson) expression.
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
metadata:
  name: vault-example
spec:
  # ...
  data:
  - secretKey: foobar
    remoteRef:
      key: /dev/config
      property: foo.nested.bar
---
# creates a secret with:
# foobar=mysecret
```

If you would set the `remoteRef.property` to just `foo` then you would get the json-encoded value of that property: `{"nested":{"bar":"mysecret"}}`.

#### Multiple nested Values

You can extract multiple keys from a nested secret using `dataFrom`.

Given the following secret - assume its path is `/dev/config`:
```json
{
  "foo": {
    "nested": {
      "bar": "mysecret",
      "baz": "bang"
    }
  }
}
```

You can set the `remoteRef.property` to point to the nested key using a [gjson](https://github.com/tidwall/gjson) expression.
```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
metadata:
  name: vault-example
spec:
  # ...
  dataFrom:
  - key: /dev/config
    property: foo.nested
```

That results in a secret with these values:
```
bar=mysecret
baz=bang
```

### Authentication

We support five different modes for authentication:
[token-based](https://www.vaultproject.io/docs/auth/token),
[appRole](https://www.vaultproject.io/docs/auth/approle),
[kubernetes-native](https://www.vaultproject.io/docs/auth/kubernetes),
[ldap](https://www.vaultproject.io/docs/auth/ldap) and
[jwt/odic](https://www.vaultproject.io/docs/auth/jwt), each one comes with it's own
trade-offs. Depending on the authentication method you need to adapt your environment.

#### Token-based authentication

A static token is stored in a `Kind=Secret` and is used to authenticate with vault.

```yaml
{% include 'vault-token-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `tokenSecretRef` with the namespace where the secret resides.

#### AppRole authentication example

[AppRole authentication](https://www.vaultproject.io/docs/auth/approle) reads the secret id from a
`Kind=Secret` and uses the specified `roleId` to aquire a temporary token to fetch secrets.

```yaml
{% include 'vault-approle-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `secretRef` with the namespace where the secret resides.

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
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `serviceAccountRef` or in `secretRef`, if used.

#### LDAP authentication

[LDAP authentication](https://www.vaultproject.io/docs/auth/ldap) uses
username/password pair to get an access token. Username is stored directly in
a `Kind=SecretStore` or `Kind=ClusterSecretStore` resource, password is stored
in a `Kind=Secret` referenced by the `secretRef`.

```yaml
{% include 'vault-ldap-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `secretRef` with the namespace where the secret resides.

#### JWT/OIDC authentication

[JWT/OIDC](https://www.vaultproject.io/docs/auth/jwt) uses a
[JWT](https://jwt.io/) token stored in a `Kind=Secret` and referenced by the
`secretRef`. Optionally a `role` field can be defined in a `Kind=SecretStore`
or `Kind=ClusterSecretStore` resource.

```yaml
{% include 'vault-jwt-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `secretRef` with the namespace where the secret resides.

### Vault Enterprise and Eventual Consistency

When using Vault Enterprise with [performance standby nodes](https://www.vaultproject.io/docs/enterprise/consistency#performance-standby-nodes),
any follower can handle read requests immediately after the provider has
authenticated. Since Vault becomes eventually consistent in this mode, these
requests can fail if the login has not yet propagated to each server's local
state.

Below are two different solutions to this scenario. You'll need to review them
and pick the best fit for your environment and Vault configuration.

#### Read Your Writes

The simplest method is simply utilizing the `X-Vault-Index` header returned on
all write requests (including logins). Passing this header back on subsequent
requests instructs the Vault client to retry the request until the server has an
index greater than or equal to that returned with the last write.

Obviously though, this has a performance hit because the read is blocked until
the follower's local state has caught up.

#### Forward Inconsistent

In addition to the aforementioned `X-Vault-Index` header, Vault also supports
proxying inconsistent requests to the current cluster leader for immediate
read-after-write consistency. This is achieved by setting the `X-Vault-Inconsistent`
header to `forward-active-node`. By default, this behavior is disabled and must
be explicitly enabled in the server's [replication configuration](https://www.vaultproject.io/docs/configuration/replication#allow_forwarding_via_header).
