## AWS Authentication

### Controller's Pod Identity

![Pod Identity Authentication](../pictures/diagrams-provider-aws-auth-pod-identity.png)

Note: If you are using Parameter Store replace `service: SecretsManager` with `service: ParameterStore` in all examples below.

This is basically a zero-configuration authentication method that inherits the credentials from the runtime environment using the [aws sdk default credential chain](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default).

You can attach a role to the pod using [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html), [kiam](https://github.com/uswitch/kiam) or [kube2iam](https://github.com/jtblin/kube2iam). When no other authentication method is configured in the `Kind=Secretstore` this role is used to make all API calls against AWS Secrets Manager or SSM Parameter Store.

Based on the Pod's identity you can do a `sts:assumeRole` before fetching the secrets to limit access to certain keys in your provider. This is optional.

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: team-b-store
spec:
  provider:
    aws:
      service: SecretsManager
      region: eu-central-1
      # optional: do a sts:assumeRole before fetching secrets
      role: team-b
```

### Access Key ID & Secret Access Key

![SecretRef](../pictures/diagrams-provider-aws-auth-secret-ref.png)

You can store Access Key ID & Secret Access Key in a `Kind=Secret` and reference it from a SecretStore.

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: team-b-store
spec:
  provider:
    aws:
      service: SecretsManager
      region: eu-central-1
      # optional: assume role before fetching secrets
      role: team-b
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: awssm-secret
            key: access-key
          secretAccessKeySecretRef:
            name: awssm-secret
            key: secret-access-key
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `accessKeyIDSecretRef`, `secretAccessKeySecretRef` with the namespaces where the secrets reside.

### EKS Service Account credentials

![Service Account](../pictures/diagrams-provider-aws-auth-service-account.png)

This feature lets you use short-lived service account tokens to authenticate with AWS.
You must have [Service Account Volume Projection](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection) enabled - it is by default on EKS. See [EKS guide](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts-technical-overview.html) on how to set up IAM roles for service accounts.

The big advantage of this approach is that ESO runs without any credentials.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/team-a
  name: my-serviceaccount
  namespace: default
```

Reference the service account from above in the Secret Store:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secretstore-sample
spec:
  provider:
    aws:
      service: SecretsManager
      region: eu-central-1
      auth:
        jwt:
          serviceAccountRef:
            name: my-serviceaccount
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `serviceAccountRef` with the namespace where the service account resides.

#### Setting up the IAM Role for IRSA

First, ensure your cluster has an IAM OIDC provider associated with it:

```bash
# Get your cluster's OIDC issuer URL
aws eks describe-cluster --name <cluster-name> \
  --query "cluster.identity.oidc.issuer" \
  --output text

# Associate the OIDC provider (if not already done)
eksctl utils associate-iam-oidc-provider \
  --cluster <cluster-name> \
  --approve
```

Create an IAM role with the following trust policy. Replace the OIDC issuer URL, account ID, namespace, and service account name with your values:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.eu-central-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.eu-central-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:sub": "system:serviceaccount:default:my-serviceaccount",
          "oidc.eks.eu-central-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

The `sub` condition must match the `namespace:serviceaccount-name` of the service account referenced in your `SecretStore`. Scope this as narrowly as possible — granting access to a specific service account in a specific namespace is more secure than using a wildcard.

Attach an IAM permissions policy to this role granting `secretsmanager:GetSecretValue` (and `secretsmanager:DescribeSecret`) on the secrets ESO needs to access.

**NOTE:** AWS Secrets Manager appends a 6-character random suffix to secret ARNs (e.g. `arn:aws:secretsmanager:region:account:secret:my-secret-AbCdEf`). Use a trailing wildcard in your IAM policy resource to avoid access denied errors when the suffix is not known in advance: `arn:aws:secretsmanager:eu-central-1:123456789012:secret:my-secret-*`.

## EKS Pod Identity Setup

In order to use EKS Pod Identity Agent, create a role like this:

```json
{
    "Statement": [
        {
            "Action": [
                "secretsmanager:GetResourcePolicy",
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret",
                "secretsmanager:ListSecretVersionIds"
            ],
            "Effect": "Allow",
            "Resource": [
                "*"
            ]
        }
    ],
    "Version": "2012-10-17"
}
```

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowEksAuthToAssumeRoleForPodIdentity",
            "Effect": "Allow",
            "Principal": {
                "Service": "pods.eks.amazonaws.com"
            },
            "Action": [
                "sts:AssumeRole",
                "sts:TagSession"
            ]
        }
    ]
}

```


Install ESO using helm and define these values:

```yaml
serviceAccount:
  annotations:
  name: external-secrets
```

Create a pod association:

```
aws eks create-pod-identity-association --cluster-name my-cluster --role-arn arn:aws:iam::111122223333:role/my-role --namespace external-secrets --service-account external-secrets
```

Then create a secret store like this:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: store
spec:
  provider:
    aws:
      service: SecretsManager
      region: eu-central-1
```


_Note_: `serviceAccountRef` _cannot_ be used together with EKS Pod Identity. That's because ESO can not impersonate
service accounts which have iam roles bound using pod identity. Doing so will result in an error like this:
```
unable to create session: an IAM role must be associated with service account ...
```

_Note:_ No `auth` section is defined for the SecretStore.

_Note:_ For even more details you can follow this post for more setup and information using Terraform [here](https://containscloud.com/2024/03/24/integrating-aws-secrets-manager-to-eks-using-external-secrets/).


## Custom Endpoints

You can define custom AWS endpoints if you want to use regional, vpc or custom endpoints. See List of endpoints for [Secrets Manager](https://docs.aws.amazon.com/general/latest/gr/asm.html), [Secure Systems Manager](https://docs.aws.amazon.com/general/latest/gr/ssm.html) and [Security Token Service](https://docs.aws.amazon.com/general/latest/gr/sts.html).

Use the following environment variables to point the controller to your custom endpoints. Note: All resources managed by this controller are affected.

| ENV VAR                     | DESCRIPTION                                                                                                                                                          |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| AWS_SECRETSMANAGER_ENDPOINT | Endpoint for the Secrets Manager Service. The controller uses this endpoint to fetch secrets from AWS Secrets Manager.                                               |
| AWS_SSM_ENDPOINT            | Endpoint for the AWS Secure Systems Manager. The controller uses this endpoint to fetch secrets from SSM Parameter Store.                                            |
| AWS_STS_ENDPOINT            | Endpoint for the Security Token Service. The controller uses this endpoint when creating a session and when doing `assumeRole` or `assumeRoleWithWebIdentity` calls. |
| AWS_ECR_ENDPOINT            | Endpoint for the ECR Service. The controller uses this endpoint to fetch authorization tokens from ECR.                                                              |
| AWS_ECR_PUBLIC_ENDPOINT     | Endpoint for the Public ECR Service. The controller uses this endpoint to fetch authorization tokens from ECR.                                                       |

## STS Session Tags

You can have ESO automatically include Kubernetes context data into [STS session tags](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_session-tags.html) when assuming an IAM role. These tags can be used in IAM policy conditions to implement attribute-based access control (ABAC).

The behavior is controlled by setting the optional `spec.provider.aws.sessionTagsPolicy` field on a SecretStore, which can be set to one of the following values:

| Policy   | Description |
| -------- | ----------- |
| `None`   | Default. No session tags are added. |
| `Simple` | Automatically adds `esoNamespace`, `esoStoreName`, and `esoStoreKind` tags. |
| `Custom` | Adds the same three built-in tags plus any additional tags defined in `customSessionTags`. |

The automatically added tags are derived from the store configuration and the namespace of the ExternalSecret:

| Tag            | Value |
| -------------- | ----- |
| `esoNamespace` | The namespace of the `ExternalSecret` making the request. |
| `esoStoreName` | The name of the `SecretStore` or `ClusterSecretStore`. |
| `esoStoreKind` | The kind of the store (`SecretStore` or `ClusterSecretStore`). |

Session tags are configured per secret store. If using `spec.dataFrom[].sourceRef.storeRef` to reference secrets from multiple different stores, each store must be configured with the desired `sessionTagsPolicy` independently. Although the session tags for each secret will have the name and kind of the specified secret store, they'll all share the same namespace which comes from the ExternalSecret.

### Simple Policy

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: team-b-store
  namespace: team-b
spec:
  provider:
    aws:
      service: SecretsManager
      region: eu-central-1
      role: team-b
      sessionTagsPolicy: Simple
```

Session tags will include `esoNamespace=team-b`, `esoStoreName=team-b-store`, and `esoStoreKind=SecretStore`.

### Custom Policy

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: team-b-store
  namespace: team-b
spec:
  provider:
    aws:
      service: SecretsManager
      region: eu-central-1
      role: team-b
      sessionTagsPolicy: Custom
      customSessionTags:
        env: production
        team: platform
```

Session tags will include the three automatically added tags, plus `env=production` and `team=platform`.

**NOTE:** Custom tags with empty keys or empty values are silently ignored. Built-in tags (`esoNamespace`, `esoStoreName`, `esoStoreKind`) will always be included even when the sessionTagsPolicy is `Custom`. They cannot be overridden via `customSessionTags`.

### Required IAM Permissions

When session tags are enabled, the role trust policy must allow `sts:TagSession`:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "AWS": "arn:aws:iam::111122223333:role/eso-controller" },
      "Action": ["sts:AssumeRole", "sts:TagSession"]
    }
  ]
}
```

## Troubleshooting

### Checking sync status

Use `kubectl describe` to inspect the status conditions and recent events on your resources:

```bash
# Check ExternalSecret status and events
kubectl describe externalsecret <name> -n <namespace>

# Check SecretStore status
kubectl describe secretstore <name> -n <namespace>

# Check ClusterSecretStore status
kubectl describe clustersecretstore <name>
```

A healthy `ExternalSecret` shows `Ready=True` in its status conditions. When sync fails, the `message` field contains the error returned by the provider.

### Common errors

#### `AccessDeniedException: Not authorized to perform sts:AssumeRoleWithWebIdentity`

The IRSA trust policy is missing or misconfigured. Check:

1. Your cluster has an IAM OIDC provider: `aws iam list-open-id-connect-providers`
2. The `Federated` principal in the trust policy exactly matches your cluster's OIDC provider ARN
3. The `sub` condition matches `system:serviceaccount:<namespace>:<serviceaccount-name>` for the service account referenced in your `SecretStore`
4. The service account has the `eks.amazonaws.com/role-arn` annotation set to the correct role ARN

#### `AccessDeniedException: is not authorized to perform: secretsmanager:GetSecretValue`

The role can be assumed but lacks permissions on the secret. Check:

1. The IAM permissions policy is attached to the role
2. The resource ARN in the policy covers your secret — AWS Secrets Manager appends a 6-character random suffix to ARNs. Use a trailing wildcard: `arn:aws:secretsmanager:region:account:secret:my-secret-*`

#### `ResourceNotFoundException`

The secret cannot be found. Verify:

1. The `region` in your `SecretStore` or `ClusterSecretStore` matches the AWS region where the secret is stored
2. The `remoteRef.key` in your `ExternalSecret` is the correct secret name or ARN
3. The secret exists and is not scheduled for deletion (`aws secretsmanager describe-secret --secret-id <name>`)

#### `ClusterSecretStore` not syncing — service account not found

When using `auth.jwt.serviceAccountRef` in a `ClusterSecretStore`, the `namespace` field is required:

```yaml
auth:
  jwt:
    serviceAccountRef:
      name: my-serviceaccount
      namespace: external-secrets  # required for ClusterSecretStore
```

Without `namespace`, ESO cannot locate the service account and the store will fail to initialize.

#### Secret syncs once but never updates

Check the `refreshInterval` on your `ExternalSecret`. A value of `0` disables periodic re-sync — the secret is only fetched on creation:

```yaml
spec:
  refreshInterval: 1h  # re-sync every hour; use 0 to disable
```

To trigger an immediate re-sync without waiting for the interval, annotate the `ExternalSecret`:

```bash
kubectl annotate externalsecret <name> \
  force-sync=$(date +%s) --overwrite
```
