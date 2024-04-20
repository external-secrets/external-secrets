
![aws sm](../pictures/eso-az-kv-aws-sm.png)

## Secrets Manager

A `SecretStore` points to AWS Secrets Manager in a certain account within a
defined region. You should define Roles that define fine-grained access to
individual secrets and pass them to ESO using `spec.provider.aws.role`. This
way users of the `SecretStore` can only access the secrets necessary.

``` yaml
{% include 'aws-sm-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `accessKeyIDSecretRef` and `secretAccessKeySecretRef`  with the namespaces where the secrets reside.
### IAM Policy

Create a IAM Policy to pin down access to secrets matching `dev-*`.

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetResourcePolicy",
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret",
        "secretsmanager:ListSecretVersionIds"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-west-2:111122223333:secret:dev-*"
      ]
    }
  ]
}
```

#### Permissions for PushSecret

If you're planning to use `PushSecret`, ensure you also have the following permissions in your IAM policy:

``` json
{
  "Effect": "Allow",
  "Action": [
    "secretsmanager:CreateSecret",
    "secretsmanager:PutSecretValue",
    "secretsmanager:TagResource",
    "secretsmanager:DeleteSecret"
  ],
  "Resource": [
    "arn:aws:secretsmanager:us-west-2:111122223333:secret:dev-*"
  ]
}
```

Here's a more restrictive version of the IAM policy:

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:CreateSecret",
        "secretsmanager:PutSecretValue",
        "secretsmanager:TagResource"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-west-2:111122223333:secret:dev-*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:DeleteSecret"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-west-2:111122223333:secret:dev-*"
      ],
      "Condition": {
        "StringEquals": {
          "secretsmanager:ResourceTag/managed-by": "external-secrets"
        }
      }
    }
  ]
}
```

In this policy, the DeleteSecret action is restricted to secrets that have the specified tag, ensuring that deletion operations are more controlled and in line with the intended management of the secrets.

#### Additional Settings for PushSecret

Additional settings can be set at the `SecretStore` level to control the behavior of `PushSecret` when interacting with AWS Secrets Manager.

```yaml
{% include 'aws-sm-store-secretsmanager-config.yaml' %}
```

#### Additional Metadata for PushSecret

It's possible to configure AWS Secrets Manager to either push secrets in `binary` format or as plain `string`.

To control this behaviour set the following provider metadata:

```yaml
{% include 'aws-sm-push-secret-with-metadata.yaml' %}
```

`secretPushFormat` takes two options. `binary` and `string`, where `binary` is the _default_.

### JSON Secret Values

SecretsManager supports *simple* key/value pairs that are stored as json. If you use the API you can store more complex JSON objects. You can access nested values or arrays using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md):

Consider the following JSON object that is stored in the SecretsManager key `friendslist`:
``` json
{
  "name": {"first": "Tom", "last": "Anderson"},
  "friends": [
    {"first": "Dale", "last": "Murphy"},
    {"first": "Roger", "last": "Craig"},
    {"first": "Jane", "last": "Murphy"}
  ]
}
```

This is an example on how you would look up nested keys in the above json object:

``` yaml
{% include 'aws-sm-external-secret.yaml' %}
```

### Secret Versions

SecretsManager creates a new version of a secret every time it is updated. The secret version can be reference in two ways, the `VersionStage` and the `VersionId`. The `VersionId` is a unique uuid which is generated every time the secret changes. This id is immutable and will always refer to the same secret data. The `VersionStage` is an alias to a `VersionId`, and can refer to different secret data as the secret is updated. By default, SecretsManager will add the version stages `AWSCURRENT` and `AWSPREVIOUS` to every secret, but other stages can be created via the [update-secret-version-stage](https://docs.aws.amazon.com/cli/latest/reference/secretsmanager/update-secret-version-stage.html) api.

The `version` field on the `remoteRef` of the ExternalSecret will normally consider the version to be a `VersionStage`, but if the field is prefixed with `uuid/`, then the version will be considered a `VersionId`.

So in this example, the operator will request the same secret with different versions: `AWSCURRENT` and `AWSPREVIOUS`:

``` yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: versioned-api-key
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: SecretStore
  target:
    name: versioned-api-key
    creationPolicy: Owner
  data:
  - secretKey: previous-api-key
    remoteRef:
      key: "production/api-key"
      version: "AWSPREVIOUS"
  - secretKey: current-api-key
    remoteRef:
      key: "production/api-key"
      version: "AWSCURRENT"
```

While in this example, the operator will request the secret with `VersionId` as `abcd-1234`

``` yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: versioned-api-key
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: SecretStore
  target:
    name: versioned-api-key
    creationPolicy: Owner
  data:
  - secretKey: api-key
    remoteRef:
      key: "production/api-key"
      version: "uuid/123e4567-e89b-12d3-a456-426614174000"
```

--8<-- "snippets/provider-aws-access.md"
