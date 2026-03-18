
## AWS Certificate Manager

External Secrets Operator can manage certificates in [AWS Certificate Manager (ACM)](https://docs.aws.amazon.com/acm/latest/userguide/acm-overview.html) using `PushSecret`. This provider operates in **write-only mode** — it imports TLS certificates into ACM but does not support reading them back (ACM does not expose the private key or full certificate contents of imported certificates).

This is useful when you want to manage your TLS certificates in Kubernetes (for example, using [cert-manager](https://cert-manager.io/)) and automatically import them into ACM for use with AWS services like ALB, CloudFront, or API Gateway.

### Authentication

The ACM provider uses the same authentication mechanisms as the other AWS providers. See [AWS Access](aws-access.md) for details on configuring authentication via `secretRef`, IRSA, Pod Identity, and other supported methods.

### SecretStore

To use ACM, create a `SecretStore` (or `ClusterSecretStore`) with `service: CertificateManager`:

``` yaml
{% include 'aws-acm-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` in `accessKeyIDSecretRef` and `secretAccessKeySecretRef` with the namespaces where the secrets reside.

### IAM Policy

The following IAM policy grants the permissions required by the ACM provider:

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "acm:ImportCertificate",
        "acm:DeleteCertificate",
        "acm:ListCertificates",
        "acm:AddTagsToCertificate",
        "acm:ListTagsForCertificate",
        "acm:RemoveTagsFromCertificate"
      ],
      "Resource": "*"
    }
  ]
}
```

You can scope the `Resource` to specific certificate ARNs or use conditions to restrict imports to certain regions or accounts.

### PushSecret

The ACM provider expects the source Kubernetes secret to be a standard `kubernetes.io/tls` secret containing:

- `tls.crt` — PEM-encoded leaf certificate, optionally followed by intermediate CA certificates.
- `tls.key` — PEM-encoded private key.

The `spec.data` entry must **not** set `secretKey` — the provider always reads the full secret in whole-secret mode because ACM's `ImportCertificate` API requires both the certificate and private key in a single call.

The `remoteKey` field is used as a tag value (`external-secrets-remote-key`) to identify and track the imported certificate in ACM.

``` yaml
{% include 'aws-acm-push-secret.yaml' %}
```

#### Certificate Chain Handling

The provider automatically splits the `tls.crt` PEM bundle:

- The **first** PEM block is treated as the leaf certificate and passed as the `Certificate` field.
- Any subsequent PEM blocks are treated as intermediate CA certificates and passed as the `CertificateChain` field.
- **Root CA certificates** (self-signed) are automatically excluded from the chain, as ACM manages its own trust store and does not accept root certificates via `ImportCertificate`.

This matches the output format of cert-manager, which places the leaf certificate first, followed by intermediates.

#### Custom Tags (Metadata)

You can apply custom AWS resource tags to the imported certificate using PushSecret metadata:

``` yaml
{% include 'aws-acm-push-secret-with-metadata.yaml' %}
```

The tags `managed-by` and `external-secrets-remote-key` are reserved for ESO internal tracking and cannot be overridden via metadata.

### Usage with cert-manager

A common pattern is to use [cert-manager](https://cert-manager.io/) to provision TLS certificates and then push them to ACM via `PushSecret`. cert-manager stores issued certificates as `kubernetes.io/tls` secrets, which is exactly the format expected by the ACM provider.

**1. Create a `Certificate` resource:**

``` yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: my-app-cert
spec:
  secretName: my-tls-cert
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - myapp.example.com
```

**2. Create a `PushSecret` to import the certificate into ACM:**

``` yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: acm-my-app-cert
spec:
  deletionPolicy: Delete
  refreshInterval: 1h0m0s
  secretStoreRefs:
    - name: aws-certificate-manager
      kind: SecretStore
  selector:
    secret:
      name: my-tls-cert
  data:
    - match:
        remoteRef:
          remoteKey: my-app-certificate
```

When cert-manager renews the certificate, the `PushSecret` controller will detect the change and re-import the updated certificate into ACM using the same ARN (identified by the `external-secrets-remote-key` tag). This keeps the ACM certificate in sync without creating duplicates.

If the `PushSecret` is deleted and `deletionPolicy` is set to `Delete`, the imported certificate will be removed from ACM.
