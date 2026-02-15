# Targeting Custom Resources

!!! warning "Maturity"
    At the time of this writing (1.11.2025) this feature is in heavy alpha status. Please consider the following documentation with the limitations and guardrails
    described below.

External Secrets Operator can create and manage resources beyond Kubernetes Secrets. When you need to populate ConfigMaps or Custom Resource Definitions with secret data from your external provider, you can use the manifest target feature.

!!! warning "Security Consideration"
    Custom resources are not encrypted at rest by Kubernetes. Only use this feature when you need to populate resources that do not contain sensitive credentials, or when the target resource is encrypted by other means.

This feature must be explicitly enabled in your deployment using the `--unsafe-allow-generic-targets` flag.

!!! note "Namespaced Resources Only"
    With this feature you can only target namespaced resources - and resources can only be managed by an ExternalSecret in the same namespace as the resource.

!!! note "Performance"
    Using generic targets or custom resources at the moment of this writing is ~20% slower than handling secrets due to certain missing features yet to be implemented.
    We recommend not overusing this feature without too many objects until further performance improvement are implemented.

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

When working with custom resources that have complex structures, you can use `target` to specify where template output should be placed. This is particularly useful for resources with nested specifications.

```yaml
{% include 'manifest-advanced-path.yaml' %}
```

The `target` field accepts dot-notation paths like `spec.database` or `spec.logging` to place the rendered template output at specific locations in the resource structure. When `target` is not specified it defaults to `Data` for backward compatibility with Secrets.

!!! note "Using `property` when templating `data`"
    The return of `data:` isn't an object on the template scope. If templated as a `string` it will fail in finding the right key. Therefore, something like this:
    ```yaml
      data:
        - secretKey: url
          remoteRef:
            key: slack-alerts/myalert-dev
    ```
    templated as a literal:
    ```yaml
    {% raw %}
    template:
      engineVersion: v2
      templateFrom:
        - literal: |
            api_url: {{ .url }}
          target: spec.slack
    {% endraw %}
    ```
    will not work. A property like `property: url` MUST be defined.

## Drift Detection

The operator automatically detects and corrects manual changes to managed custom resources. If you modify a ConfigMap or custom resource that is managed by an ExternalSecret, the operator will restore it to the desired state immediately.
This is achieved with informers watching the relevant GVK of the Resource.

## Metadata and Labels

You can add labels and annotations to your target resources using the template metadata:

```yaml
{% include 'manifest-labeled-configmap.yaml' %}
```

The operator automatically adds the `externalsecrets.external-secrets.io/managed: "true"` label to track which resources it manages.

## RBAC Requirements

When using custom resource targets, ensure the External Secrets Operator has appropriate RBAC permissions to create and manage those resources. The Helm chart provides configuration options to enable these permissions:

```yaml
genericTargets:
  enabled: true
  resources:
  - apiGroups: ["config.example.com"]
    resources: ["appconfigs"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

Without these permissions, the operator will not be able to create or update your target resources.
