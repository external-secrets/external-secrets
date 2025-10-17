# Volcengine Provider

## Quick start

This guide demonstrates how to use the Volcengine (BytePlus) provider.

### Step 1 

Create a secret in the [Volcengine KMS](https://console.volcengine.com/kms).

### Step 2

Create a `SecretStore`.

#### Case 1: IRSA is not enabled

You need to provide a Kubernetes `Secret` containing the credentials (Access Key ID, Secret Access Key and STS token) for accessing Volcengine KMS.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: volcengine-creds
type: Opaque
data:
  accessKeyID: YOUR_ACCESS_KEY_ID_IN_BASE64
  secretAccessKey: YOUR_SECRET_ACCESS_KEY_IN_BASE64
  sts-token: YOUR_STS_TOKEN_IN_BASE64 # Optional
---
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: volcengine-kms
spec:
  provider:
    volcengine:
      # Region (Required)
      region: "cn-beijing"
      auth:
        secretRef:
          accessKeyID:
            name: volcengine-creds
            key: accessKeyID
          secretAccessKey:
            name: volcengine-creds
            key: secretAccessKey
          # (Optional, provide the Secret reference for the STS token if you are using one)
          token:
            name: volcengine-creds
            key: sts-token
```

#### Case 2: IRSA is enabled

When the `auth` block is not specified or does not contain secretRef, IRSA is enabled by default. 

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: volcengine-kms
spec:
  provider:
    volcengine:
      # Region (Required)
      region: "cn-beijing"
```

Add service account and environment variables in helm `values.yaml` as below to enable IRSA.

```yaml
# Environment variables of external-secrets Pod
extraEnv:
  - name: VOLCENGINE_OIDC_ROLE_TRN
    value: "YOUR_ROLE_TRN"
  - name: VOLCENGINE_OIDC_TOKEN_FILE
    value: "/var/run/secrets/vke.volcengine.com/irsa-tokens/token"
# Volume mounts of external-secrets Pod
extraVolumeMounts:
- mountPath: /var/run/secrets/vke.volcengine.com/irsa-tokens
  name: irsa-oidc-token
  readOnly: true
extraVolumes:
- name: irsa-oidc-token
  projected:
    defaultMode: 420
    sources:
    - serviceAccountToken:
        audience: sts.volcengine.com
        expirationSeconds: 3600
        path: token
# Service account of external-secrets Pod
serviceAccount:
  name: "YOUR_SERVICE_ACCOUNT_NAME"
```

Note:

- Ensure that your role has the permission `KMSFullAccess` .

### Step 3

Create `ExternalSecret`.

#### Case 1: Get the entire Secret (JSON format) from the secret manager and extract a single property

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secret
spec:
  secretStoreRef:
    name: volcengine-kms
    kind: SecretStore
  target:
    name: db-credentials
  data:
  - secretKey: password
    remoteRef:
      key: "my-app/db/credentials" # The name of the secret in the secret manager
      property: "password" # The field name in the JSON
```

#### Case 2: Do not specify a property, get the entire Secret from the secret manager and sync all its key-value pairs

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secret
spec:
  secretStoreRef:
    name: volcengine-kms
    kind: SecretStore
  target:
    name: db-credentials
  data:
  - secretKey: password
    remoteRef:
      key: "my-app/db/credentials" # The name of the secret in the secret manager
```
