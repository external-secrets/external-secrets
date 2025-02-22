```yaml
---
title: Sync to Custom Resources
version: v1alpha1
authors: Gustavo Carvalho
creation-date: 2024-05-03
status: approved
---
```

# Sync to Custom Resources Design

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->

## Summary

This design document describes how `ExternalSecrets` can leverage templates to generate non-kubernetes `Secret` resource as the target. This allows to push Sensitive information to specific CRs, `ConfiMaps`, etc.

## Motivation

Currently, several "semi-sensitive" information needs to be provisioned directly into CRs and ConfigMaps (such as OIDC Client IDs). While these information are not strictly speaking a secret, several regulated environments must treat them as such, causing several operational overhead to deal with this information - specially on a GitOps setup.

## Proposal

To simplify the workflow and enhance user experience, the proposal is to integrate a functionality to template the whole manifest. this would be additional logic to the existing `target: Manifest` directly into the `templateFrom.[].target` resource. This will allow users to specify a template to render any type of manifest, instead of the original Secret object.

Problems with this proposal is that the whole reconciliation logic reads from a secret - this would need to be updated if templates using Manifests are set. In that case, we should query for that specific resource as specified on the target.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
spec:
  target:
    name: target-custom-resource
    manifest: # Information as to find it afterwards during reconcile cycle
      APIVersion: my.custom.api/v1alpha1
      Kind: CustomResource
    template:
      templateFrom:
      - target: Spec #Additional target at root level (instead of data, metadata and annotations level).
      # Other option would be to allow a gjson path such as target:'.spec' where . is the indicator of this type of expression for backwards compatibility.
        literal: |
            customSpecField1: {{ .fromSecretStore }}
            field2:
              couldBeNested:
              - {{ .fromSecretStore }}
          ## Name: is obtained from spec.target.name
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
spec:
  target:
    name: target-configmap
    manifest: # Information as to find it afterwards during reconcile cycle
      APIVersion: v1
      Kind: ConfigMap
    template:
      templateFrom:
      - target: Data
        literal: |
          {{ .field1 }}: {{ .value1 }}
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
spec:
  target:
    name: target-configmap
    manifest: # Information as to find it afterwards during reconcile cycle
      APIVersion: v1
      Kind: ConfigMap # should render with no need of templates - as `.data` is the default Secret target.
```

Only one `templateFrom` entry can be set if its type is `target: Manifest`.

## Consequences

* **First Class support for GitOps tools**: GitOps tools often need configmap information considered as sensitive. This allows better integration with such tools

* **Increased Complexity**: Instead of getting a single secret to start reconciling process, we would need to first verify if `target.manifest` is set - and use the manifest information to get the appropriate resource (defaulting to a kubernetes secret). Templateing Logic would only be increased to some extend, as we would need to change the `Secret` type to a runtime.Object

* **Better Extensibility**: This feature allows ESO to not only be used by operators but also better integrated to systems.

* **Backwards Compatible**: This feature would not change how `target` and `template` are currently used by existing installations.

* **API and Documentation Update**: The API changes need to be well-documented to ensure users understand how to utilize the new feature effectively.


## Acceptance Criteria

* behavior:
  * Reconciliation logic should not change when `target!= Secret`
  * `creationPolicy`, `updatePolicy`, and `deletionPolicy` must be compatible when `target != Secret`
  * One of the two must be implemented:
    * a feature flag `--unsafe-allow-non-secret-targets` must be set to allow this feature. If not set, `template.manifest` should cause error to the reconciliation.
    * Feature is always eniabled - but Warnings must be emited whenever `target.manifest` is used pointing to the use of sensitive information on open manifests.
      * Warnings should be disabled with feature flags
* deployment:
  * Extra RBAC options must be available on helm values (to allow the usage of this feature)
  * Helm values must allow the installation of this new feature (setting up the appropriate feature flags, etc)
* tests: 
  * controller unit tests for `target.manifest` behavior and `target.template.target` behavior
  * controller regression tests for `target.manifest` and `target.template.target` focused on different `creationPolicy` and `deletionPolicy`
  * e2e test for `target.manifest` targeting a ConfigMap (first class support for ArgoCD and Flux)
  * e2e test for `target.manifest` targeting a custom resource

* the API changes need to be documented
    * API/CRD spec inline documentation
    * ExternalSecrets API documentation
    * Guides section for `ExternalSecret` 'Creating Non-Secret Resources'.
    * Warnings on the feature as non-Secret manifests are not meant to contain sensitive information.
