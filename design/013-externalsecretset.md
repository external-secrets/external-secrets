```yaml
---
title: ExternalSecretSet CRD
version: v1alpha1
authors: Netanel Kadosh
creation-date: 2024-05-03
status: draft
---
```

# 013 - ExternalSecretSet: Generate Multiple ExternalSecrets from a Wildcard Path

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->


## Summary

This proposal introduces a new Custom Resource Definition (CRD) called `ExternalSecretSet`.
The purpose of this resource is to dynamically generate multiple `ExternalSecret` objects discovered in the external secret store.

This feature aims to simplify managing large numbers of secrets that follow a predictable naming structure, without modifying the existing `ExternalSecret` CRD semantics.

## Motivation

In many environments, secrets are organized hierarchically and share a common prefix.
For example, a team might store application secrets in paths like:

```
/teams/teamone/argocd/repositories/repo1
/teams/teamone/argocd/repositories/repo2
/teams/teamtwo/argocd/repositories/repoA
/teams/teamtwo/argocd/repositories/repoB
```

Each repository has multiple keys under its own subpath (`repoURL`, `username`, `password`, etc.).

Currently, a user must define a separate `ExternalSecret` resource for every repository.
This leads to:

* Dozens or hundreds of repetitive manifests
* High maintenance overhead when repositories change
* Reduced clarity in large GitOps setups

The proposed `ExternalSecretSet` will automatically discover all secrets under a given path and create individual `ExternalSecret` resources based on a user-defined template.

## Goals

* Introduce a new CRD (`ExternalSecretSet`) to support path-based secret discovery.
* Automatically generate and manage multiple `ExternalSecret` resources under a given namespace.
* Ensure consistent templating, naming, and label conventions for generated `ExternalSecret`s.
* Maintain clear ownership and state tracking between the `ExternalSecretSet` and its generated `ExternalSecret`s.

## Non-Goals

* Modify the existing `ExternalSecret` behavior or introduce one-to-many relationships in it.
* Replace Helm, Kustomize, or other manifest generators.
* Handle cross-namespace secret generation.

## Proposal

### CRD Definition (Simplified)

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecretSet
metadata:
  name: argocd-repositories
  namespace: argocd
spec:
  secretStoreRef:
    name: vault-secret-store
  discovery:
    # path based discovery
    - prefix:
        - /teams/teamone/argocd/repositories/
        - /teams/teamtow/argocd/repositories/
    # tag based diiscovery
    - tags:
        type: argocd-repo
        team: 
          - teamone
          - teamtwo
  # ExternalSecret template
  template:
    metadata:
      labels:
        app: argocd
        team: "{{ .tags.team }}"   # optional: template values sourced from discovery
    spec:
      refreshInterval: 1h
      data:
        - secretKey: repoURL
          remoteRef:
            key: "{{ .path }}/repoURL"
        - secretKey: username
          remoteRef:
            key: "{{ .path }}/username"
        - secretKey: password
          remoteRef:
            key: "{{ .path }}/password"
```

### Controller Flow

1. **Discovery Phase**
   * Prefix based discovery:
      *  Lists secrets directly under the configured prefix (no recursive traversal into deeper nested folders) using the configured `SecretStoreRef`.
      * Apply the wildcard pattern to filter relevant subpaths.
   * Tag based discovery:
      * Uses provider tags/labels to discover secrets matching a set of key:value pairs.
      * Only available for providers that expose tag/label metadata and allow listing/filtering by those metadata keys.
      * If the provider doesn't support tag filtering server-side, the controller can optionally fall back to listing all secrets for that store then filtering client-side — but this must be an opt-in behavior due to performance and API rate limits.
  * Multiple discovery entries
      * discovery accepts multiple entries; the controller independently resolves each and merges results.
      * Duplicate `sourceRef` across discovery entries must deduplicate so each external source maps to a single generated `ExternalSecret`.

2. **Generation Phase**
   * For each matched subpath, render a new `ExternalSecret` manifest from `spec.template`.
   * Replace template variables (e.g., `{{ .path }}` or derived names).
   * Apply a deterministic name (e.g., `<ess.Name>-<sanitized-source>` or user-provided `nameTemplate`).

3. **Reconciliation Phase**
   * Ensure all generated `ExternalSecret`s exist and are up to date.
   * Remove any generated `ExternalSecret`s that no longer match the discovery results.

4. **State Tracking**
   * The controller maintains a `status.generatedSecrets` list to reflect current child resources.

```yaml
status:
  observedGeneration: 2
  generatedExternalSecrets:
    - name: argocd-repo1
      sourceRef: /teams/teamone/argocd/repositories/repo1
      status: Ready
      lastReconciledTime: "2025-10-15T12:34:56Z"
    - name: argocd-repoB
      sourceRef: /teams/teamtwo/argocd/repositories/repoB
      status: Pending
      lastReconciledTime: "2025-10-15T12:35:20Z"
```

### Ownership and Cleanup

* Each generated `ExternalSecret` will have an `ownerReference` pointing back to the `ExternalSecretSet`.
* Deleting the `ExternalSecretSet` will automatically clean up all child `ExternalSecret`s.

## Alternatives

### Extending `ExternalSecret` to Support Wildcards

Another option would be to add wildcard support directly in `ExternalSecret.remoteRef.key`.
While this simplifies the API surface, it blurs the one-to-one design principle of `ExternalSecret`, making reconciliation and status management complex.

Given that `ExternalSecret` was intentionally designed to map a single resource to a single external secret, a new CRD provides a cleaner separation of concerns.

## Open Questions

* Should `ExternalSecretSet` support templated naming patterns for generated resources (`nameTemplate`)?
* Should we allow field-level overrides on the generated `ExternalSecretSpec`?
* Tag discovery fallback: allow client-side filtering (expensive) or require server-side support only?

---

## Code (sketch)

High-level pseudocode for Reconcile:
```go

func (r *Reconciler) Reconcile(ctx context.context, req ctrl.Request) (result ctrl.Result, err Error) {
  ess := &esv1alpha1.ExternalSecretSet{}
  if err = r.Get(ctx, req.NamespacedName, ess); err != nil { ... }
  discovered, err := r.ListProviderSecrets(ctx, ess.discovery)
  for _, src := range discovered {
    genName := renderName(ess, src) // deterministic
    externalSecret = &esv1.ExternalSecret{
      ObjectMeta: &metav1.ObjectMeta{
        Name: genName,
        Namespace: ess.namespace,
        OwnerReferences: &metav1.OwnerReferences{
          APIVersion: "external-secrets.io/v1alpha1",
          Kind: "ExternalSecretSet",
          Name: ess.name,
          UID: ess.UID
        }
      },
      Spec: {
        SecretStoreRef: &esv1.SecretStoreRef{
          Name: ess.secretStoreRef.name,
          Kind: ess.secretStoreRef.kind
        }
        Data: [{
          RemoteRef: {
            Key: src.key
          } 
        }]
      }
    }

    if exists := r.getExternalSecret(ctx, req.Namespace, genName); !exists {
      r.Create(ctx, es)
    } else {
      r.UpdateIfNeeded(ctx, es)
    }
    // update status.generatedExternalSecrets entry for src
  }

  // cleanup removed sources: compare status -> discovered
  // update status and requeue if needed
}
```


## Consequences

* **First Class support for GitOps tools**: fewer repetitive manifests and simpler scaling for many secrets.

* **Increased Complexity**: Controller complexity: generator logic + discovery + templating increases controller responsibilities; keep it isolated in a dedicated controller to avoid impacting single-`ExternalSecret` reconciliation.

* **RBAC and Documentation Update**: extra RBAC for listing provider secrets may be required and must be documented for each provider.

* **Extensibility**: the design allows future discovery strategies (regex, manifest-driven, etc.).

* **Backwards Compatible**: `ExternalSecret` remains unchanged.

## Acceptance Criteria

* behavior:
  * Reconciliation of standard `ExternalSecret` should be unchanged.
  * `creationPolicy`, `updatePolicy`, `deletionPolicy` must be supported for generated resources (and behave consistently).
  * `target` default (`Secret`) behavior must remain safe.
* deployment:
  * Extra RBAC options must be available on helm values (to allow the usage of this feature)
  * Helm values must allow the installation of this new feature (setting up the appropriate feature flags, etc)
* tests: 
  * Unit tests for discovery (prefix and tag) behavior.
  * Unit/regression tests for generated resource lifecycle.
  * E2E tests:
     * generated `ExternalSecret` -> Kubernetes Secret
* the API changes need to be documented
    * API/CRD spec inline documentation
    * ExternalSecrets API documentation
    * Guides section for `ExternalSecret` 'Creating Non-Secret Resources'.
    * Warnings on the feature as non-Secret manifests are not meant to contain sensitive information.

## Provider support (tag discovery)

Initial implementation will fully support tag-based discovery for providers that expose server-side tag/label filtering. Confirmed/priority for initial implementation:

* **AWS Secrets Manager** - supports tags and server-side filtering via ListSecrets with filters.
* **AWS Systems Manager Parameter Store** - supports tagging and listing by tag (needs provider verification).
* **GCP Secret Manager** - supports labels and filtering by labels.

Conditional / experimental support:

* **HashiCorp Vault (KV v2)** - Vault does not have first-class "tags", but KV v2 supports metadata fields (e.g., custom_metadata). Supporting tag-like discovery is possible but requires scanning metadata per path (more expensive) and careful provider-specific handling.
* **Alibaba Cloud** - Some Alibaba secret services support tags; provider implementation must be verified and tested.
* **Azure Key Vault and other providers** - most do not support efficient server-side tag filtering; client-side filtering (listing all secrets then filtering) is potentially possible but not recommended by default due to scale and rate-limit concerns.
