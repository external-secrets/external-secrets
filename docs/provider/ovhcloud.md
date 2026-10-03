# OVHcloud Secrets Manager

External Secrets Operator integrates with [OVHcloud KMS](https://www.ovhcloud.com/en/identity-security-operations/key-management-service/).  

This guide demonstrates:

- how to set up a `ClusterSecretStore`/`SecretStore` with the OVH provider.
- `ExternalSecret` use cases with examples.
- `PushSecret` use cases with examples.

This guide assumes:

- External Secrets Operator is already installed
- You have an OKMS domain

### <u>Authentication</u>

The OVH provider talks to the OKMS *data plane*, the regional REST API exposed by your OKMS domain
(for example `https://eu-west-rbx.okms.ovh.net`). It supports the two data plane authentication methods that can
be carried by a Kubernetes Secret: a **token** (bearer) and an **access certificate** (mTLS). Both are described
below. Either one needs the correct access rights on the OKMS domain, granted through [IAM permissions](#iam-permissions).

#### Retrieve your endpoint and OKMS ID

Both are shown in the **General information** tab of your OKMS domain dashboard, and can be listed with the
[OVHcloud CLI](https://github.com/ovh/ovhcloud-cli):

```bash
$ ovhcloud okms list
┌──────────────────────────────────────┬─────────────┐
│ id                                   │ region      │
├──────────────────────────────────────┼─────────────┤
│ 734b9b45-8b1a-469c-b140-b10bd6540017 │ eu-west-rbx │
└──────────────────────────────────────┴─────────────┘
```

The `id` column is the `okmsid` field, and the region gives the `server` endpoint: `https://<region>.okms.ovh.net`.

#### Token authentication

The token is any bearer token accepted by the OKMS data plane. The recommended one is a
**Personal Access Token (PAT)** created on a local user, since it is long-lived.

Create it with the [OVHcloud CLI](https://github.com/ovh/ovhcloud-cli):

```bash
ovhcloud iam user token create <user> \
  --name pat-secretmanager-734b9b45-8b1a-469c-b140-b10bd6540017 \
  --description "PAT secret manager for domain 734b9b45-8b1a-469c-b140-b10bd6540017"
```

or through the `POST /me/identity/user/{user}/token` API call. The `token` value is returned once and never prompted
again, so store it right away.

Then store the token in a Kubernetes Secret:

```bash
kubectl create secret generic ovh-token -n my-namespace --from-literal=token="<token>"
```

!!! note
     The token is resolved from the Kubernetes Secret on every reconciliation, so rotating the credential only
     requires updating the Secret.

#### mTLS authentication

mTLS uses an [OKMS access certificate](https://docs.ovhcloud.com/en/guides/manage-and-operate/kms/okms-certificate-management),
created from the OKMS domain dashboard or the OVHcloud API, either by letting OVHcloud generate the private key or by
providing your own CSR. It yields a certificate and a private key in PEM format.

Store them in a Kubernetes Secret:

```bash
kubectl create secret tls ovh-mtls -n my-namespace --cert=ID_certificate.pem --key=ID_privatekey.pem
```

#### IAM permissions

Access rights are attached to an identity, not to the credential itself. For a token that identity is the local user or
service account the token was created on; for an access certificate it is every entry of the certificate `identityURNs`
list. The identity must be a member of a group with the ADMIN role, or be granted an
[IAM policy](https://docs.ovhcloud.com/en/guides/account-and-service-management/account-information/iam-policy-ui)
on the OKMS domain with at least the following actions:

- `okms:apiovh:secret/get`
- `okms:apikms:secret/get`
- `okms:apikms:secret/version/getData`
- `okms:apikms:secret/create`

Listing a secret is a distinct right from reading its content, so both `secret/get` and `secret/version/getData` are
required for an `ExternalSecret` to resolve. `secret/create` is only needed for `PushSecret`; add the matching
`secret/update` and `secret/delete` actions if the operator must overwrite or remove secrets (for example a
`PushSecret` with `deletionPolicy: Delete`). The full action list is documented in
[OKMS authentication methods](https://docs.ovhcloud.com/en/guides/manage-and-operate/kms/okms-authentication-methods).

The policy designates the OKMS domain by its resource URN, `urn:v1:eu:resource:okms:<okmsid>`. Create it from the
OVHcloud Control Panel, or with a `POST /v2/iam/policy`
[API call](https://docs.ovhcloud.com/en/guides/account-and-service-management/account-information/iam-policies-api).
Token and mTLS take the same policy; only `identities` changes, listing either the user or service account owning the
token, or the identities declared on the access certificate:

```json
{
  "name": "external-secrets-okms",
  "description": "External Secrets Operator access to the OKMS Secret Manager",
  "identities": [
    "urn:v1:eu:identity:user:xx1111-ovh/external-secrets"
  ],
  "resources": [
    { "urn": "urn:v1:eu:resource:okms:734b9b45-8b1a-469c-b140-b10bd6540017" }
  ],
  "permissions": {
    "allow": [
      { "action": "okms:apiovh:secret/get" },
      { "action": "okms:apikms:secret/get" },
      { "action": "okms:apikms:secret/version/getData" },
      { "action": "okms:apikms:secret/create" }
    ]
  }
}
```

### <u>SecretStore</u>

**OVH provider supports both `token` and `mTLS` authentication.**

Token authentication:
```yaml
apiVersion: external-secrets.io/v1 
kind: SecretStore
metadata:
  name: secret-store-ovh
  namespace: my-namespace
spec:
  provider:
    ovh:
      server: <kms-endpoint> # for example: "https://eu-west-rbx.okms.ovh.net"
      okmsid: <okms-id> # for example: "734b9b45-8b1a-469c-b140-b10bd6540017" 
      auth:
        token:
          tokenSecretRef:
            name: ovh-token
            key: token
---
apiVersion: v1
kind: Secret
metadata:
  name: ovh-token
  namespace: my-namespace
data:
  token: BASE64-TOKEN-VALUE-PLACEHOLDER
```
mTLS authentication:
```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-store-ovh
  namespace: my-namespace
spec:
  provider:
    ovh:
      server: <kms-endpoint> # for example: "https://eu-west-rbx.okms.ovh.net"
      okmsid: <okms-id> # for example: "734b9b45-8b1a-469c-b140-b10bd6540017" 
      auth:
        mtls:
          certSecretRef:
            name: ovh-mtls
            key: tls.crt
          keySecretRef:
            name: ovh-mtls
            key: tls.key
---
apiVersion: v1
kind: Secret
metadata:
  name: ovh-mtls
  namespace: my-namespace
type: kubernetes.io/tls
data:
  tls.crt: BASE64_CERT_PLACEHOLDER # "client certificate value"
  tls.key: BASE64_KEY_PLACEHOLDER  # "client key value"
```

Authentication fields:

| Field                  | Description                                                                          | Required         |
|------------------------|--------------------------------------------------------------------------------------|------------------|
| `token.tokenSecretRef` | Reference to the Secret key holding the bearer token                                 | Yes, for `token` |
| `mtls.certSecretRef`   | Reference to the Secret key holding the client certificate (PEM)                     | Yes, for `mtls`  |
| `mtls.keySecretRef`    | Reference to the Secret key holding the client private key (PEM)                      | Yes, for `mtls`  |
| `mtls.caBundle`        | Base64-encoded CA bundle used to validate the OKMS server certificate                | No               |
| `mtls.caProvider`      | Reference to a `Secret` or `ConfigMap` holding that CA bundle, instead of inlining it | No               |

!!! note
     Exactly one of `token` and `mtls` must be set.

!!! note
     A `ClusterSecretStore` configuration is the same except you must provide the `namespace` for `tokenSecretRef`, `certSecretRef` and `keySecretRef` according to your chosen authentication method.  

### <u>ExternalSecret</u>
 
For these examples, we will assume you have the following secret in your Secret Manager:
```json
{
  "path": "creds",
  "data": {
    "type": "credential",
    "users": {
      "kevin": {
        "token": "kevin token value"
      },
      "laura": {
        "token": "laura token value"
      }
    }
  }
}
```
`path` refers to the secret's path in OVH Secret Manager.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  data:
    - secretKey: foo
      remoteRef:
        key: creds
        version: version
        property: property
```

| Field      | Description                                                            | Required |
|------------|------------------------------------------------------------------------|----------|
| version    | Secret version to retrieve                                             | No       |
| property   | Specific key or nested key in the secret                               | No       |
| secretKey  | The key inside the Kubernetes Secret that will hold the secret's value | Yes      |

#### Fetch the whole secret

- Using `spec.data`
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  data:
    - secretKey: foo
      remoteRef:
        key: creds
```
Resulting Kubernetes Secret data:
```json
{
  "foo": {
    "type": "credential",
    "users": {
      "kevin": {
        "token": "kevin token value"
      },
      "laura": {
        "token": "laura token value"
      }
    }
  }
}
```
- Using `spec.dataFrom.extract`
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  dataFrom:
  - extract:
      key: creds
```
Resulting Kubernetes Secret data:
```json
{
  "type": "credential",
  "users": {
    "kevin": {
      "token": "kevin token value"
    },
    "laura": {
      "token": "laura token value"
    }
  }
}
```

#### Fetch scalar/nested values
- Scalar value using `data`
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  data:
    - secretKey: type
      remoteRef:
        key: creds
        property: type
```
Resulting Kubernetes Secret data:
```json
{
  "type": "credential"
}
```
- Nested value using `data`
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  data:
    - secretKey: kevin-token
      remoteRef:
        key: creds
        property: users.kevin.token
```
Resulting Kubernetes Secret data:
```json
{
  "kevin-token": "kevin token value"
}
```
- Nested value using `dataFrom.extract`
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  dataFrom:
  - extract:
      key: creds
      property: users
```
Resulting Kubernetes Secret data:
```json
{
  "kevin": {
    "token": "kevin token value"
  },
  "laura": {
    "token": "laura token value"
  }
}
```

!!! warning
     Scalar values cannot be retrieved using `dataFrom.extract` because no Kubernetes secret key can be specified, which would imply storing a value without a corresponding key.

#### Fetch multiple secrets

Extract multiple secrets, with filtering support.  
You can filter either by path or/and regular expression. Path filtering occurs first if you use both.

For these examples, we will assume you have the following secrets in your Secret Manager: `path/to/secret/secret1`, `path/to/secret/secret2`, `path/to/config/config2`, `path/to/config/config3`, `secret-example2`.
- Path filtering
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  dataFrom:
  - find:
      path: "path/to/secret"
```
Resulting Kubernetes Secret data:
```json
{
  "path/to/secret/secret1": "secret1 value",
  "path/to/secret/secret2": "secret2 value"
}
```
!!! note
     If path is left empty or is "/", every secret will be retrieved from your Secret Manager.

- Regular expression filtering
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  dataFrom:
  - find:
      name:
        regexp: "[2-3]"
```
Resulting Kubernetes Secret data:
```json
{
  "path/to/secret/secret2": "secret2 value",
  "path/to/config/config2": "config2 value",
  "path/to/config/config3": "config3 value",
  "secret-example2": "secret-example2 value"
}
```
!!! note
     If name.regexp is left empty, every secret will be retrieved from your Secret Manager.

- Combination of both
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-ovh
    kind: SecretStore
  target:
    name: secret-example
  dataFrom:
  - find:
      path: "path/to"
      name:
        regexp: "2$"
```
Resulting Kubernetes Secret data:
```json
{
  "path/to/secret/secret2": "secret2 value",
  "path/to/config/config2": "config2 value"
}
```

!!! note
     When both are combined, path filtering occurs first.

### <u>PushSecret</u>

#### Check-And-Set
Check-And-Set can be enabled/disabled (default: disabled), in the Secret Store configuration:
```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-store-ovh
  namespace: my-namespace
spec:
  provider:
    ovh:
      server: <kms-endpoint> # for example: "https://eu-west-rbx.okms.ovh.net"
      okmsid: <okms-id> # for example: "734b9b45-8b1a-469c-b140-b10bd6540017" 
      auth:
        token:
          tokenSecretRef:
            name: ovh-token
            key: token
      casRequired: true
---
apiVersion: v1
kind: Secret
metadata:
  name: ovh-token
  namespace: my-namespace
data:
  token: BASE64_TOKEN_PLACEHOLDER # "token value"
```

#### Secret Rotation
```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: Password
metadata:
  name: my-password-generator
  namespace: my-namespace
spec:
  length: 32
  digits: 5
  symbols: 5
  symbolCharacters: "-_^$%*ù/;:,?"
  noUpper: false
  allowRepeat: true
---
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-secret-ovh
  namespace: my-namespace
spec:
  refreshInterval: 6h0m0s
  secretStoreRefs:
    - name: secret-store-ovh
      kind: SecretStore
  selector:
    generatorRef:
      apiVersion: generators.external-secrets.io/v1alpha1
      kind: Password
      name: my-password-generator
  data:
    - match:
        secretKey: password # property in the generator output
        remoteRef:
          remoteKey: prod/mysql/password
```

With this configuration, the secret is automatically rotated every 6 hours in the OVH Secret Manager.

#### Secret migration
```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-store-vault
  namespace: my-namespace
spec:
  provider:
    vault:
      server: "https://my.vault.server:8200"
      path: "secret"
      version: "v2"
      auth:
        tokenSecretRef:
          name: vault-token
          key: token
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-vault
  namespace: my-namespace
spec:
  secretStoreRef:
    name: secret-store-vault
    kind: SecretStore
  refreshPolicy: Periodic
  refreshInterval: "10s"
  target:
    name: creds-secret-vault
  dataFrom:
    - extract:
        key: example
---
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-store-ovh
  namespace: my-namespace
spec:
  provider:
    ovh:
      server: <kms-endpoint> # for example: "https://eu-west-rbx.okms.ovh.net"
      okmsid: <okms-id> # for example: "734b9b45-8b1a-469c-b140-b10bd6540017" 
      auth:
        token:
          tokenSecretRef:
            name: ovh-token
            key: token
---
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-secret-ovh
  namespace: my-namespace
spec:
  secretStoreRefs:
    - name: secret-store-ovh
      kind: SecretStore
  selector:
    secret:
      name: creds-secret-vault
  refreshInterval: 10s
  data:
    - match:
        secretKey: "secretKey"
        remoteRef:
          remoteKey: "creds-secret-migrated"
```

This example demonstrates how to fetch a secret from a HashiCorp Vault KV secrets engine and sync it into OVH Secret Manager.