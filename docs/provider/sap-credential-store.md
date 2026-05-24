# SAP Credential Store Provider

The External Secrets Operator supports [SAP Credential Store](https://help.sap.com/docs/credential-store), the native secrets service on SAP Business Technology Platform (BTP).

## Features

| Feature | Supported |
|---------|-----------|
| `ExternalSecret` (read) | ✅ |
| `PushSecret` (write) | ✅ |
| `dataFrom` bulk sync | ✅ |
| OAuth2 authentication | ✅ |
| Mutual TLS (mTLS) authentication | ✅ |

## Authentication

The provider supports two authentication modes that correspond to the service binding formats issued by BTP.

### OAuth2 Client Credentials (recommended)

Store the `clientid` and `clientsecret` from the BTP service binding in a Kubernetes Secret:

```bash
kubectl create secret generic sap-cs-oauth2 \
  --from-literal=clientid='<your-client-id>' \
  --from-literal=clientsecret='<your-client-secret>'
```

Reference them in the `SecretStore`:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: sap-credential-store
spec:
  provider:
    sapCredentialStore:
      serviceURL: https://<instance>.credstore.cfapps.<region>.hana.ondemand.com
      namespace: <credential-store-namespace>
      auth:
        oauth2:
          tokenURL: https://<subaccount>.authentication.<region>.hana.ondemand.com/oauth/token
          clientId:
            name: sap-cs-oauth2
            key: clientid
          clientSecret:
            name: sap-cs-oauth2
            key: clientsecret
```

### Mutual TLS (mTLS)

Store the PEM-encoded certificate and private key from the BTP service binding:

```bash
kubectl create secret generic sap-cs-mtls \
  --from-file=tls.crt=client.crt \
  --from-file=tls.key=client.key
```

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: sap-credential-store
spec:
  provider:
    sapCredentialStore:
      serviceURL: https://<instance>.credstore.cfapps.<region>.hana.ondemand.com
      namespace: <credential-store-namespace>
      auth:
        mtls:
          certificate:
            name: sap-cs-mtls
            key: tls.crt
          privateKey:
            name: sap-cs-mtls
            key: tls.key
```

> Use a `ClusterSecretStore` when the ESO controller runs in a different namespace than the auth secrets. Set the `namespace` field on each `SecretKeySelector` in that case.

## Reading Credentials (ExternalSecret)

SAP Credential Store has three credential types: `password`, `key`, and `certificate`.

The `remoteRef.key` field is the credential **name**. The `remoteRef.property` field is the **type** (defaults to `password` when omitted).

### Password credential

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: db-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: sap-credential-store
    kind: SecretStore
  target:
    name: db-secret
  data:
    - secretKey: password
      remoteRef:
        key: db-password        # credential name
        property: password      # credential type (default when omitted)
```

### Key credential

```yaml
  data:
    - secretKey: api-token
      remoteRef:
        key: my-api-key
        property: key
```

### Certificate credential

Accessing the certificate PEM:

```yaml
  data:
    - secretKey: tls.crt
      remoteRef:
        key: my-service-cert
        property: certificate
```

Accessing the private key PEM (use the special `certificate/key` property):

```yaml
  data:
    - secretKey: tls.key
      remoteRef:
        key: my-service-cert
        property: certificate/key
```

### Fetching all fields of a credential (GetSecretMap)

Use `dataFrom` with `extract` to get all fields of a credential as separate keys:

```yaml
spec:
  dataFrom:
    - extract:
        key: db-password
        property: password
```

This produces a Kubernetes Secret with keys `name`, `value`, and `username` (for password credentials).

## Bulk Sync (dataFrom find)

Use `dataFrom` with `find` to sync **all credentials** in the SAP CS namespace into a single Kubernetes Secret. Keys are formatted as `<type>/<name>` (for example, `password/db-pass`, `key/api-key`, `certificate/tls-cert`).

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: all-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: sap-credential-store
    kind: SecretStore
  target:
    name: all-creds-secret
  dataFrom:
    - find: {}
```

> Each key in the resulting Kubernetes Secret maps to the primary `value` field of the credential. For certificate private keys, use individual `data` entries with `property: certificate/key`.

## Writing Credentials (PushSecret)

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-db-password
spec:
  secretStoreRefs:
    - name: sap-credential-store
      kind: SecretStore
  selector:
    secret:
      name: my-k8s-secret
  data:
    - match:
        secretKey: password        # key within the Kubernetes Secret
        remoteRef:
          remoteKey: db-password   # credential name in SAP CS
          property: password       # credential type (defaults to "password")
```

### Credential type mapping for PushSecret

| `property` value | SAP CS type |
|------------------|-------------|
| `password` (or empty) | password |
| `key` | key |
| `certificate` | certificate |

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `ExternalSecret` stuck in `SecretSyncedError` with "credential not found" | Wrong credential name or type | Verify `remoteRef.key` and `remoteRef.property` match the SAP CS credential |
| `SecretStore` not ready: "resolving clientId" | Kubernetes Secret or key not found | Check `auth.oauth2.clientId.name` and `key` match the actual Kubernetes Secret |
| `SecretStore` not ready: "failed to parse mTLS key pair" | Invalid certificate/key PEM | Verify the PEM data from the BTP service binding is complete and unmodified |
| `PushSecret` fails with unexpected status 403 | OAuth2 credentials lack write permission | Ensure the BTP service plan includes write access to Credential Store |
