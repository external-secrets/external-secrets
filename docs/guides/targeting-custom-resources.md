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

## How Sync Works

For an existing target, the operator applies a **strategic merge patch** built from the templated fields only. Unmanaged sibling fields on the live object are left alone.

That means:

- Map-shaped paths (for example ConfigMap `data` keys or nested `spec.database`) merge into the live object.
- List-shaped paths use the resource schema’s merge metadata (commonly a merge key such as `name`). Templated list items must include that merge key so the apiserver can match the right element.
- Drift detection and refresh compare a hash of the **managed template target paths**, not the entire `spec` or `data` map.

When the target does **not** exist yet, the operator creates it from the same templated payload. Create still has to pass API validation, so the template must include every required field for that resource type — not only the secret values you want to inject. See [Creation Policy](#creation-policy) below.

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

The operator will create or patch the Application resource with the data from your external secret provider.

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

### Array Index Notation

Paths can navigate into arrays using `[n]` index notation, for example:

```yaml
target: spec.rules[0].from[0].source.notRemoteIpBlocks
```

This templates a resource like an Istio `AuthorizationPolicy`:

```yaml
spec:
  rules:
    - from:
        - source:
            notRemoteIpBlocks:
              - 10.0.0.0/8
```

Intermediate maps and arrays are created if they don't exist yet, and existing sibling keys in the same array element are preserved during templating. Paths must start with a key; a bare leading index like `[0].foo` is rejected.

!!! note "List merge keys when patching"
    For list fields that use a merge key in the resource schema (often `name`, for example containers or env), include that key in each templated list item. A partial item without the merge key can fail strategic merge or update the wrong element. Prefer emitting complete merge-keyed items for the fields you manage rather than relying on index-only replace semantics.

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

## Creation Policy

`spec.target.creationPolicy` works for generic targets the same way it does for Secrets. See [Ownership and Deletion Policy](ownership-deletion-policy.md) for the full matrix. The important generic-target nuances are:

| Policy | Missing target | Existing target |
|--------|----------------|-----------------|
| `Owner` / `Orphan` / `CreateOrMerge` | **Create** from the templated object | Strategic merge **patch** of managed fields |
| `Merge` | Do not create; wait until the resource exists | Strategic merge **patch** of managed fields |
| `None` | No-op | No-op |

For **create** (`Owner`, `Orphan`, or `CreateOrMerge` when the object is missing), the template must produce a valid complete object for the API. Required fields for that kind (for example a Deployment `selector`, `template`, and container `image`) must be present in the template along with the secret data you inject. A partial fragment that only sets an env var or nested field is enough for **patch** against an existing object, but it will fail create validation.

Practical guidance:

- Prefer `creationPolicy: Merge` (or pre-create the resource) when you only want to inject a subset of fields into a complex resource.
- Use `CreateOrMerge` / `Owner` / `Orphan` when the template is complete enough to create the resource from scratch.

## Drift Detection

The operator watches the target GVK and re-syncs when managed content drifts. Only the fields covered by the ExternalSecret template targets are re-applied via strategic merge patch. Unrelated fields on the live object are not wiped to match a full desired spec.

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
