
![aws sm](../pictures/diagrams-provider-aws-ssm-parameter-store.png)

## Parameter Store

A `ParameterStore` points to AWS SSM Parameter Store in a certain account within a
defined region. You should define Roles that define fine-grained access to
individual secrets and pass them to ESO using `spec.provider.aws.role`. This
way users of the `SecretStore` can only access the secrets necessary.

``` yaml
{% include 'aws-parameter-store.yaml' %}
```
**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `accessKeyIDSecretRef` and `secretAccessKeySecretRef`  with the namespaces where the secrets reside.

!!! warning "API Pricing & Throttling"
    The SSM Parameter Store API is charged by throughput and
    is available in different tiers, [see pricing](https://aws.amazon.com/systems-manager/pricing/#Parameter_Store).
    Please estimate your costs before using ESO. Cost depends on the RefreshInterval of your ExternalSecrets.

### IAM Policy

Create a IAM Policy to pin down access to secrets matching `dev-*`, for further information see [AWS Documentation](https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-paramstore-access.html):

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameter*"
      ],
      "Resource": "arn:aws:ssm:us-east-2:123456789012:parameter/dev-*"
    }
  ]
}
```
### JSON Secret Values

You can store JSON objects in a parameter. You can access nested values or arrays using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md):

Consider the following JSON object that is stored in the Parameter Store key `my-json-secret`:
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
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: example
spec:
  # [omitted for brevity]
  data:
  - secretKey: firstname
    remoteRef:
      key: my-json-secret
      property: name.first # Tom
  - secretKey: first_friend
    remoteRef:
      key: my-json-secret
      property: friends.1.first # Roger

```

## Push Secret

### Creating a Push Secret

#### Add push secret

#### Check successful secret sync

#### Test new secret using AWS CLI


--8<-- "snippets/provider-aws-access.md"
