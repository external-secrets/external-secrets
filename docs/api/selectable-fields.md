# ExternalSecret Selectable Fields

As of Kubernetes 1.30, External Secrets Operator supports selectable fields for querying ExternalSecret resources based on spec field values. This feature enables efficient server-side filtering of ExternalSecret resources.

## Overview

Selectable fields allow you to use `kubectl` field selectors and Kubernetes API field selectors to filter ExternalSecret resources based on specific spec field values rather than just metadata fields like `metadata.name` and `metadata.namespace`.

## Supported Selectable Fields

The following spec fields are available for field selectors in ExternalSecret resources:

- `spec.secretStoreRef.name` - Name of the SecretStore or ClusterSecretStore
- `spec.secretStoreRef.kind` - Type of store (SecretStore or ClusterSecretStore)
- `spec.target.name` - Name of the target Kubernetes Secret
- `spec.refreshInterval` - Refresh interval for the external secret

## Usage Examples

### Using kubectl with field selectors

Query all ExternalSecrets that use a specific SecretStore:
```bash
kubectl get externalsecrets --field-selector spec.secretStoreRef.name=my-vault-store
```

Find all ExternalSecrets that use ClusterSecretStores:
```bash
kubectl get externalsecrets --field-selector spec.secretStoreRef.kind=ClusterSecretStore
```

Find ExternalSecrets with a specific refresh interval:
```bash
kubectl get externalsecrets --field-selector spec.refreshInterval=15m
```

Find ExternalSecrets that create a specific target secret:
```bash
kubectl get externalsecrets --field-selector spec.target.name=database-credentials
```

You can also combine multiple field selectors:
```bash
kubectl get externalsecrets --field-selector spec.secretStoreRef.kind=SecretStore,spec.refreshInterval=1h
```

### Using the Kubernetes API

When using the Kubernetes client libraries, you can use field selectors programmatically:

```go
import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

// List ExternalSecrets using a specific SecretStore
fieldSelector := fields.OneTermEqualSelector("spec.secretStoreRef.name", "my-vault-store")
listOptions := &client.ListOptions{
    FieldSelector: fieldSelector,
}

var externalSecrets esv1.ExternalSecretList
err := kubeClient.List(ctx, &externalSecrets, listOptions)
```

### Advanced Filtering

You can combine field selectors with label selectors for more complex queries:

```bash
# Find ExternalSecrets with specific store AND specific label
kubectl get externalsecrets \
  --field-selector spec.secretStoreRef.kind=ClusterSecretStore \
  --selector environment=production
```
