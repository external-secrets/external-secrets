# SAP Credential Store

External Secrets Operator integrates with [SAP Credential Store](https://help.sap.com/docs/credential-store), a BTP service for storing and managing credentials (passwords, keys, and certificates) in a secure, encrypted manner.

## Prerequisites

- An SAP BTP subaccount with a Credential Store service instance.
- A service binding or service key with mTLS credentials (the default authentication type for standard, trial, and free plans).
- If payload encryption is enabled on the instance, you will also need the encryption keys from the service binding.

## Authentication

The provider supports two authentication methods:

### Service Binding Secret (Recommended)

The simplest approach is to store the Credential Store service binding JSON in a Kubernetes secret. The provider automatically extracts the service URL, mTLS credentials (certificate and private key), and encryption keys from the binding JSON.

```sh
kubectl create secret generic sap-credstore-binding \
  --from-literal=credentials='<your-service-binding-json>'
```

The binding JSON is the full JSON object from the SAP BTP service binding. It contains fields like `url`, `certificate`, `key`, and `encryption`. The provider detects the authentication type from the `parameters.authentication.type` field in the binding.

Currently, only the `mtls` authentication type is supported for service bindings. Bindings with `oauth:mtls`, `oauth:key`, or `basic` authentication types will be rejected with a clear error message.

### Inline mTLS

If you prefer to configure credentials explicitly (e.g., when not using the BTP Service Operator), create Kubernetes secrets with the client certificate and private key:

```sh
kubectl create secret generic sap-credstore-mtls \
  --from-file=certificate=./client-cert.pem \
  --from-file=private-key=./client-key.pem
```

## Payload Encryption (JWE)

SAP Credential Store uses payload encryption by default, based on JWE compact serialization (RSA-OAEP-256 + A256GCM). The encryption keys are included in the service binding.

When using `serviceBindingSecretRef`, encryption is **automatically enabled** if the binding JSON contains `encryption.client_private_key` and `encryption.server_public_key` fields — no additional configuration is needed.

For inline mTLS authentication, create Kubernetes secrets for the encryption keys:

```sh
kubectl create secret generic sap-credstore-encryption \
  --from-literal=client-private-key="<base64-encoded-private-key>" \
  --from-literal=server-public-key="<base64-encoded-public-key>"
```

The keys are base64-encoded DER format (PKCS8 for private, SPKI for public) as provided by the service binding.

## Creating a SecretStore

### With Service Binding (Recommended)

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: sap-credential-store-binding
spec:
  provider:
    sapCredentialStore:
      namespace: my-namespace
      serviceBindingSecretRef:
        name: sap-credstore-binding
```

When using `serviceBindingSecretRef`, the `serviceURL` and `auth` fields are not needed — they are derived from the binding. The `credentialsKey` field defaults to `credentials` and specifies which key in the Kubernetes secret contains the binding JSON.

### With Inline mTLS

```yaml
{% include 'sap-credential-store-secret-store.yaml' %}
```

### With Inline mTLS and Encryption

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: sap-credential-store-encrypted
spec:
  provider:
    sapCredentialStore:
      serviceURL: https://credstore.cfapps.eu10.hana.ondemand.com
      namespace: my-namespace
      auth:
        mtls:
          certificate:
            secretRef:
              name: sap-credstore-mtls
              key: certificate
          privateKey:
            secretRef:
              name: sap-credstore-mtls
              key: private-key
      encryption:
        clientPrivateKey:
          secretRef:
            name: sap-credstore-encryption
            key: client-private-key
        serverPublicKey:
          secretRef:
            name: sap-credstore-encryption
            key: server-public-key
```

## Creating an ExternalSecret

SAP Credential Store has three credential types: `password`, `key`, and `certificate`. Use the `property` field on `remoteRef` to specify the type. If omitted, `password` is assumed.

```yaml
{% include 'sap-credential-store-external-secret.yaml' %}
```

### Credential Type Mapping

| `remoteRef.property` | Credential Type | Value Returned |
|---|---|---|
| _(empty)_ or `password` | password | The credential value |
| `key` | key | The credential value (base64-decoded) |
| `certificate` | certificate | The certificate PEM |
| `certificate/key` | certificate | The private key PEM only |

## Get the K8s Secret

```shell
kubectl get secret my-sap-secret -o jsonpath="{.data.password}" | base64 --decode && echo
```

## Creating a Secret

The following example shows how to create a Kubernetes `Secret` that will later be pushed to SAP Credential Store.

```yaml
{% include 'sap-credential-store-secret.yaml' %}
```

## Creating a ClusterSecretStore

When using a `ClusterSecretStore`, secret references must include the `namespace` field:

```yaml
{% include 'sap-credential-store-cluster-secret-store.yaml' %}
```

## Creating a PushSecret

Push secrets from Kubernetes to SAP Credential Store. The `property` field determines the credential type (defaults to `password`):

```yaml
{% include 'sap-credential-store-push-secret.yaml' %}
```

## Bulk Sync with dataFrom

The provider supports `dataFrom.find` for bulk synchronization. Secrets are keyed as `{type}/{name}` (e.g., `password/my-secret`). You can filter by name using a regular expression:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: sap-credential-store-bulk
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: SecretStore
    name: sap-credential-store
  target:
    name: all-sap-secrets
  dataFrom:
    - find:
        name:
          regexp: ".*"
```

## Limitations

- Each SecretStore targets a single Credential Store namespace. To access multiple namespaces, create separate SecretStore resources.
- `dataFrom.extract` (`GetSecretMap`) returns all fields of a single credential (name, value, username, key). Use `dataFrom.find` for listing multiple credentials.
- The `version` field on `remoteRef` is not used by this provider.
- Only `mtls` service binding authentication is currently supported. The `oauth:mtls`, `oauth:key`, and `basic` binding types will be added in future releases.
