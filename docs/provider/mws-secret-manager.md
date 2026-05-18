## MWS Secret Manager

External Secrets Operator integrates with [MWS Secret Manager](https://mws.ru/docs/cloud-platform/secret-manager/general/whatis-secret-manager.html) for secure secret management.

### Authentication

The operator supports authentication using an authorized key for a service account. You can create the service account and authorized key in the MWS Console. See the [documentation](https://mws.ru/docs/cloud-platform/iam/keys.html).

The service account must have the following role to download the secret content:

```
secretmanager.secret.user
```

The authorized key has the following format:

```json
{
  "keyId" : "projects/{project}/serviceAccounts/{service-account}/authorizedKeys/{key-name}",
  "privateKey" : "MEECAQ...6w==",
  "publicKey" : "MFkwEw...JA==",
  "algorithm" : "ES256"
}
```

To use the authorized key, create a Kubernetes Secret from the key file:

```bash
kubectl create secret generic mws-auth --from-file=./authorized-key
```

Alternatively, you can declare the Secret as a Kubernetes resource:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mws-auth
stringData:
  authorized-key: |
    {
      "keyId" : "projects/{project}/serviceAccounts/{service-account}/authorizedKeys/{key-name}",
      "privateKey" : "MEECAQ...6w==",
      "publicKey" : "MFkwEw...JA==",
      "algorithm" : "ES256"
    }
```

The resulting Secret must be accessible as a SecretKeyRef.

### Provider definition

To define a provider for MWS Secret Manager, create a SecretStore and pass a reference to the authorized key in the `auth` field:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: mws-secret-manager-store
spec:
  provider:
    mwssecretmanager:
      auth:
        authorizedKeySecretRef:
          name: mws-auth
          key: authorized-key
```

### Creating an external secret

To create a Kubernetes Secret from a secret stored in MWS Secret Manager, create an ExternalSecret that references the SecretStore:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: mws-external-secret
spec:
  secretStoreRef:
    name: mws-secret-manager-store
    kind: SecretStore
  data:
  - secretKey: my-secret
    remoteRef:
      key: my-secret
      version: "1"
      property: password
```

The resulting Secret contains the value of the specified property.

If the property is not defined, the Secret contains a JSON object with all existing properties:

```json
{"username":"...","password":"..."}
```

If the version is not defined, the Secret contains the current version of the secret. Alternatively, you can use the word `current` for the version.
