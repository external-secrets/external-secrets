
![aws sm](./pictures/eso-az-kv-aws-sm.png)

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
### JSON Secret Values

SecretsManager supports *simple* key/value pairs that are stored as json. If you use the API you can store more complex JSON objects. You can access nested values or arrays using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md):

Consider the following JSON object that is stored in the SecretsManager key `my-json-secret`:
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

So in this example, the operator will request the secret with `VersionStage` as `AWSPREVIOUS`:

``` yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: secretstore-sample
    kind: SecretStore
  target:
    name: secret-to-be-created
    creationPolicy: Owner
  data:
  - secretKey: secret-key-to-be-managed
    remoteRef:
      key: "example/secret"
      version: "AWSPREVIOUS"
```

While in this example, the operator will request the secret with `VersionId` as `abcd-1234`

``` yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: secretstore-sample
    kind: SecretStore
  target:
    name: secret-to-be-created
    creationPolicy: Owner
  data:
  - secretKey: secret-key-to-be-managed
    remoteRef:
      key: "example/secret"
      version: "uuid/abcd-1234"
```

--8<-- "snippets/provider-aws-access.md"
