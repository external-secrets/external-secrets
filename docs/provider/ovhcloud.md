## Secrets Manager

External Secrets Operator integrates with [OVHcloud KMS](https://www.ovhcloud.com/en/identity-security-operations/key-management-service/).  

This guide demonstrates:
- how to set up a `ClusterSecretStore`/`SecretStore` with the OVH provider.
- `ExternalSecret` use cases with examples.
- `PushSecret` use cases with examples.

This guide assumes:
- External Secrets Operator is already installed
- You have access to OVHcloud Secret Manager
- Required credentials are already created

### <u>SecretStore</u>

**OVH provider supports both `token` and `mTLS` authentication.**

Token authentication:
```yaml
apiVersion: external-secrets.io/v1 
kind: SecretStore
metadata:
  name: secret-store-ovh
  namespace: default
spec:
  provider:
    ovh:
      server: <kms-endpoint>
      okmsid: <okms-id>
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
data:
  token: BASE64-TOKEN-VALUE-PLACEHOLDER
```
mTLS authentication:
```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-store-ovh
  namespace: default
spec:
  provider:
    ovh:
      server: "https://eu-west-rbx.okms.ovh.net"
      okmsid: "734b9b45-8b1a-469c-b140-b10bd6540017"
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
  namespace: default
type: kubernetes.io/tls
data:
  tls.crt: BASE64_CERT_PLACEHOLDER # "client certificate value"
  tls.key: BASE64_KEY_PLACEHOLDER  # "client key value"
```

!!! note
     A `ClusterSecretStore` configuration is the same except you have to provide the `tokenSecretRef` `namespace`.  

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
  namespace: default
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
  namespace: default
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
  namespace: default
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
  namespace: default
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
        property: type
```
Resulting Kubernetes Secret data:
```json
{
  "foo": "credential"
}
```
- Nested value using `data`
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: external-secret-ovh
  namespace: default
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
  namespace: default
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
  namespace: default
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
  namespace: default
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
  namespace: default
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
  namespace: default
spec:
  provider:
    ovh:
      server: <kms-endpoint>
      okmsid: <okms-id>
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
data:
  token: BASE64_TOKEN_PLACEHOLDER # "token value"
```

#### Secret Rotation
```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: Password
metadata:
  name: my-password-generator
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
  namespace: default
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
  namespace: default
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
  namespace: default
spec:
  provider:
    ovh:
      server: <kms-endpoint>
      okmsid: <okms-id>
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