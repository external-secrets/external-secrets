## MWS Certificate Manager

External Secrets Operator integrates with [MWS Certificate Manager](https://mws.ru/docs/cloud-platform/certmanager/general/whatis-cert-manager.html) for secure secret management.

### Authentication

The operator supports authentication using an authorized key for a service account. You can create the service account and authorized key in the MWS Console. See the [documentation](https://mws.ru/docs/cloud-platform/iam/keys.html).

The service account must have the following role to download the certificate content:

```text
certmanager.certificate.downloader
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

To define a provider for MWS Certificate Manager, create a SecretStore and pass a reference to the authorized key in the `auth` field:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: mws-certificate-manager-store
spec:
  provider:
    mwscertificatemanager:
      auth:
        authorizedKeySecretRef:
          name: mws-auth
          key: authorized-key
```

### Creating an external secret

To create a Kubernetes Secret from a certificate stored in MWS Certificate Manager, create an ExternalSecret that references the SecretStore:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: mws-external-certificate
spec:
  secretStoreRef:
    name: mws-certificate-manager-store
    kind: SecretStore
  data:
  - secretKey: my-certificate
    remoteRef:
      key: my-certificate
```

By default, the Secret contains a JSON object with the certificate data. For example:

```json
{"certificate":"...","privateKey":"...","chainedCert":"..."}
```

The MWS Certificate Manager provider supports the following properties:

- `certificate` — the certificate content
- `privateKey` — the private key of the certificate
- `chainedCert` — the fullchain certificate

You can select a specific property in the external secret definition:

```yaml
    remoteRef:
      key: my-certificate
      property: privateKey
```

When a property is specified, the Secret contains only the value of that property, without JSON serialization.
