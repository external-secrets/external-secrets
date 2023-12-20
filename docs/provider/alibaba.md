
## Alibaba Cloud Secrets Manager

External Secrets Operator integrates with [Alibaba Cloud Key Management Service](https://www.alibabacloud.com/help/en/key-management-service/latest/kms-what-is-key-management-service/) for secrets and Keys management.

### Authentication

We support Access key and RRSA authentication.

To use RRSA authentication, you should follow [Use RRSA to authorize pods to access different cloud services](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/use-rrsa-to-enforce-access-control/) to assign the RAM role to external-secrets operator.

#### Access Key authentication

To use `accessKeyID` and `accessKeySecrets`, simply create them as a regular `Kind: Secret` beforehand and associate it with the `SecretStore`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-sample
data:
  accessKeyID: bXlhd2Vzb21lYWNjZXNza2V5aWQ=
  accessKeySecret: bXlhd2Vzb21lYWNjZXNza2V5c2VjcmV0
```

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secretstore-sample
spec:
  provider:
    alibaba:
      regionID: ap-southeast-1
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: secret-sample
            key: accessKeyID
          accessKeySecretSecretRef:
            name: secret-sample
            key: accessKeySecret
```


#### RRSA authentication

When using RRSA authentication we manually project the OIDC token file to pod as volume

```yaml
extraVolumes:
  - name: oidc-token
    projected:
      sources:
      - serviceAccountToken:
          path: oidc-token
          expirationSeconds: 7200    # The validity period of the OIDC token in seconds.
          audience: "sts.aliyuncs.com"

extraVolumeMounts:
  - name: oidc-token
    mountPath: /var/run/secrets/tokens
```

and provide the RAM role ARN and OIDC volume path to the secret store
```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secretstore-sample
spec:
  provider:
    alibaba:
      regionID: ap-southeast-1
      auth:
        rrsa:
          oidcProviderArn: acs:ram::1234:oidc-provider/ack-rrsa-ce123456
          oidcTokenFilePath: /var/run/secrets/tokens/oidc-token
          roleArn: acs:ram::1234:role/test-role
          sessionName: secrets
```

### Creating external secret

To create a kubernetes secret from the Alibaba Cloud Key Management Service secret a `Kind=ExternalSecret` is needed.

```yaml
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
    name: example-secret
    creationPolicy: Owner
  data:
    - secretKey: secret-key
      remoteRef:
        key: ext-secret
```
