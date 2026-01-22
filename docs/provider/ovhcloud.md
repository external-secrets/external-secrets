## Secrets Manager

External Secrets Operator integrates with [OVHcloud KMS](https://www.ovhcloud.com/fr/identity-security-operations/key-management-service/).  

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
{% include 'ovh-token-secret-store.yaml' %}
```
mTLS authentication:
```yaml
{% include 'ovh-mtls-secret-store.yaml' %}
```

!!! note
     A `ClusterSecretStore` configuration is the same except you have to provide the `tokenSecretRef` `namespace`.  
     `ExternalSecret` objects must be in the same `namespace` as the `tokenSecretRef` provided to your `ClusterSecretStore`.

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
{% include 'ovh-external-secret-example.yaml' %}
```

| Field      | Description                                                            | Required |
|------------|------------------------------------------------------------------------|----------|
| version    | Secret version to retrieve                                             | No       |
| property   | Specific key or nested key in the secret                               | No       |
| secretKey  | The key inside the Kubernetes Secret that will hold the secret's value | Yes      |

#### Fetch the whole secret

- Using `spec.data`
```yaml
{% include 'ovh-external-secret-data.yaml' %}
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
{% include 'ovh-external-secret-dataFrom-extract.yaml' %}
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
{% include 'ovh-external-secret-data-property.yaml' %}
```
Resulting Kubernetes Secret data:
```json
{
  "foo": "credential"
}
```
- Nested value using `data`
```yaml
{% include 'ovh-external-secret-data-nested-property.yaml' %}
```
Resulting Kubernetes Secret data:
```json
{
  "kevin-token": "kevin token value"
}
```
- Nested value using `dataFrom.extract`
```yaml
{% include 'ovh-external-secret-dataFrom-extract-property.yaml' %}
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
{% include 'ovh-external-secret-dataFrom-find-bypath.yaml' %}
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
{% include 'ovh-external-secret-dataFrom-find-byregexp.yaml' %}
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
{% include 'ovh-external-secret-dataFrom-find-byboth.yaml' %}
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
Check-And-Set can be enabled/disabled (default: enabled), in the Secret Store configuration:
```yaml
{% include 'ovh-secret-store-cas.yaml' %}
```

#### Secret Rotation
```yaml
{% include 'ovh-push-secret-rotation.yaml' %}
```

With this configuration, the secret is automatically rotated every 6 hours in the OVH Secret Manager.

#### Secret migration
```yaml
{% include 'ovh-push-secret-migration.yaml' %}
```

This example demonstrates how to fetch a secret from a HashiCorp Vault KV secrets engine and sync it into OVH Secret Manager.