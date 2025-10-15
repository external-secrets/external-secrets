# Targeting Custom Resources

External Secrets Operator can create and manage resources beyond Kubernetes Secrets. When you need to populate ConfigMaps or Custom Resource Definitions with secret data from your external provider, you can use the manifest target feature.

!!! warning "Security Consideration"
    Custom resources are not encrypted at rest by Kubernetes. Only use this feature when you need to populate resources that do not contain sensitive credentials, or when the target resource is encrypted by other means.

This feature must be explicitly enabled in your deployment using the `--unsafe-allow-non-secret-targets` flag.
!!! note "Namespaced Resources Only"
    With this feature you can only target namespaced resources - and resources can only be managed by an ExternalSecret in the same namespace as the resource.
## Basic ConfigMap Example

The simplest use case is creating a ConfigMap from external secrets. This is useful when applications expect configuration in ConfigMaps rather than Secrets, or when the data is not sensitive.

```yaml
{% include 'manifest-basic-configmap.yaml' %}
```

This creates a ConfigMap named `app-config` with the data populated from your secret provider.

## Custom Resource Definitions

You can target any custom resource that exists in your cluster. This example creates an Argo CD Application resource:

```yaml
{% include 'manifest-argocd-app.yaml' %}
```

The operator will create or update the Application resource with the data from your external secret provider.

## Templating with Custom Resources

Templates work with custom resources just as they do with Secrets. You can use the `template.data` field to create structured configuration:

```yaml
{% include 'manifest-templated-configmap.yaml' %}
```

## Advanced Path Targeting

When working with custom resources that have complex structures, you can use `manifestTarget` to specify where template output should be placed. This is particularly useful for resources with nested specifications.

```yaml
{% include 'manifest-advanced-path.yaml' %}
```

The `manifestTarget` field accepts dot-notation paths like `spec.database` or `spec.logging` to place the rendered template output at specific locations in the resource structure. When `manifestTarget` is not specified, the template system uses the `target` field which defaults to `Data` for backward compatibility with Secrets.

## Drift Detection

The operator automatically detects and corrects manual changes to managed custom resources. If you modify a ConfigMap or custom resource that is managed by an ExternalSecret, the operator will restore it to the desired state during the next reconciliation cycle.

This ensures that your configuration remains consistent with what is defined in your external secret provider, preventing configuration drift.

## Metadata and Labels

You can add labels and annotations to your target resources using the template metadata:

```yaml
{% include 'manifest-labeled-configmap.yaml' %}
```

The operator automatically adds the `externalsecrets.external-secrets.io/managed: "true"` label to track which resources it manages.

## RBAC Requirements

When using custom resource targets, ensure the External Secrets Operator has appropriate RBAC permissions to create and manage those resources. The Helm chart provides configuration options to enable these permissions:

```yaml
nonSecretTargets:
  enabled: true
  rbac:
    configMaps: true
    customResources:
    - apiGroups: ["config.example.com"]
      resources: ["appconfigs"]
```

Without these permissions, the operator will not be able to create or update your target resources.
