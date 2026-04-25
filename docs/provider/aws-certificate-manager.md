
## AWS Certificate Manager

External Secrets Operator integrates with [AWS Certificate Manager (ACM)](https://docs.aws.amazon.com/acm/latest/userguide/acm-overview.html) to support:

- **Exporting** certificates from ACM into Kubernetes via `ExternalSecret`.
- **Importing** TLS certificates into ACM via `PushSecret` (e.g., certificates issued by [cert-manager](https://cert-manager.io/)).

Both public and private ACM certificates can be exported, provided the export option was enabled when the certificate was requested. See [Exportable Certificates](#exportable-certificates) below for details.

### SecretStore

``` yaml
{% include 'aws-acm-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` in `accessKeyIDSecretRef` and `secretAccessKeySecretRef` with the namespaces where the secrets reside.

### IAM Policy

The required IAM permissions depend on whether you use `ExternalSecret` (export), `PushSecret` (import), or both.

#### Export only (`ExternalSecret`)

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "acm:DescribeCertificate",
        "acm:ExportCertificate"
      ],
      "Resource": "*"
    }
  ]
}
```

#### Import only (`PushSecret`)

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "acm:ImportCertificate",
        "acm:DeleteCertificate",
        "acm:AddTagsToCertificate",
        "acm:ListTagsForCertificate",
        "acm:RemoveTagsFromCertificate",
        "tag:GetResources"
      ],
      "Resource": "*"
    }
  ]
}
```

The `tag:GetResources` permission (Resource Groups Tagging API) is used to locate certificates by their ESO management tags with server-side filtering. `tag:GetResources` does not support resource-level ARNs and must be granted on `"Resource": "*"`.

#### Both export and import

``` json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "acm:ImportCertificate",
        "acm:DeleteCertificate",
        "acm:DescribeCertificate",
        "acm:ExportCertificate",
        "acm:AddTagsToCertificate",
        "acm:ListTagsForCertificate",
        "acm:RemoveTagsFromCertificate",
        "tag:GetResources"
      ],
      "Resource": "*"
    }
  ]
}
```

You can scope the `acm:*` actions to specific certificate ARNs or use conditions to restrict access to certain regions or accounts. `tag:GetResources` must remain on `"Resource": "*"`.

### ExternalSecret

#### Exportable Certificates

Both **public** and **private** ACM certificates can be exported, but the certificate must have been requested with the export option **enabled**. This option cannot be changed after the certificate is created — if it was not enabled at request time, the certificate is not exportable.

Requesting a public certificate with the export option enabled is a **paid** feature — AWS charges a fee upon issuance and again on each renewal. Separately, the `ExportCertificate` API is free for the first 10,000 calls per month per account; exceeding this threshold incurs additional per-call charges. See the [ACM pricing page](https://aws.amazon.com/certificate-manager/pricing/) for current details.

#### Export Caching

To minimize calls to the paid `ExportCertificate` API, the provider caches exported PEM bundles in memory, keyed by certificate ARN. On each access the provider calls the free `DescribeCertificate` and compares the certificate's serial number against the cached entry, which prevents serving stale data; `ExportCertificate` is only called on the first access, when the serial changes (certificate renewal), or when the cached entry has been evicted. The cache is bounded to 128 entries with a 24-hour TTL per entry (LRU eviction above that). It is held on the long-lived AWS provider instance and survives across reconciles, but is in-memory only and is reset when the operator pod restarts or when the underlying `SecretStore` is updated.

#### Usage

Use `remoteRef.key` to specify the certificate ARN. The provider returns the certificate, chain, and decrypted private key as a single PEM bundle. Use [template functions](../guides/templating.md) such as `filterPEM` and `filterCertChain` to extract individual components:

``` yaml
{% include 'aws-acm-external-secret.yaml' %}
```

You can also extract just the leaf or intermediate certificates using `filterCertChain`:

``` yaml
{% include 'aws-acm-external-secret-chain.yaml' %}
```

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
- Any subsequent PEM blocks are passed as-is in the `CertificateChain` field.

This matches the output format of cert-manager, which places the leaf certificate first, followed by intermediates.

#### Custom Tags (Metadata)

You can apply custom AWS resource tags to the imported certificate using PushSecret metadata:

``` yaml
{% include 'aws-acm-push-secret-with-metadata.yaml' %}
```

The tags `managed-by`, `external-secrets-remote-key`, and `external-secrets-content-hash` are reserved for ESO internal tracking and cannot be overridden via metadata. The content hash tag stores a SHA-256 digest of the certificate and private key, allowing the provider to skip redundant re-imports when the secret content has not changed.

### Usage with cert-manager

A common pattern is to use [cert-manager](https://cert-manager.io/) to provision TLS certificates and then push them to ACM via `PushSecret`. cert-manager stores issued certificates as `kubernetes.io/tls` secrets, which is exactly the format expected by the ACM provider.

**1. Create a `Certificate` resource:**

``` yaml
{% include 'aws-acm-cert-manager-certificate.yaml' %}
```

**2. Create a `PushSecret` to import the certificate into ACM:**

``` yaml
{% include 'aws-acm-cert-manager-push-secret.yaml' %}
```

When cert-manager renews the certificate, the `PushSecret` controller will detect the change and re-import the updated certificate into ACM using the same ARN (identified by the `external-secrets-remote-key` tag). This keeps the ACM certificate in sync without creating duplicates.

If the `PushSecret` is deleted and `deletionPolicy` is set to `Delete`, the imported certificate will be removed from ACM (provided it is not in use).
