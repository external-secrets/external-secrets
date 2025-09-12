# KEP-NNNN: SecretStores and Generators v2 (Out-of-Tree Providers)

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Overview](#overview)
  - [API Resources](#api-resources)
    - [SecretStore](#secretstore)
    - [ClusterSecretStore](#clustersecretstore)
    - [Generator](#generator)
    - [ClusterGenerator](#clustergenerator)
  - [Changes to ExternalSecret and PushSecret](#changes-to-externalsecret-and-pushsecret)
  - [New Provider Interfaces](#new-provider-interfaces)
  - [Out-of-Tree Providers Maintenance](#out-of-tree-providers-maintenance)
    - [Deployment](#deployment)
    - [Governance](#governance)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist


## Summary

This KEP proposes a v2 architecture and API for SecretStores and Generators in External Secrets Operator (ESO). The primary goals are to:

- Support out-of-tree providers as first-class citizens, allowing independent versioning and distribution.
- Unify feature sets of SecretStores and Generators (e.g., refresh, gating, generator state) under consistent CRDs and controllers.
- Make referent authentication modes explicit and easier to use.
- Allow users to install only the providers they need.

## Motivation

There are several limitations in the current (v1) SecretStore and Generator architectures that hinder flexibility and maintainability:

- Different provider versioning is difficult.
- Out-of-tree providers are not first-class.
- Users cannot easily install/uninstall only the desired providers.
- Referent authentication modes are implicit and hard to learn/use.

SecretStores and Generators also diverge in core features:

- SecretStores support ClusterSecretStores with referent authentication and cross-namespace generation; Generators do not.
- Generators support GeneratorState for caching; SecretStores do not.
- SecretStores are periodically refreshed; Generators are not.
- Badly configured SecretStores gate jobs; Generators do not.

### Goals

- Implement new CRDs: SecretStore/v2alpha1, ClusterSecretStore/v2alpha1, Generator/v2alpha1, ClusterGenerator/v2alpha1.
- Enable ESO to run without in-tree providers; users install providers separately.
- Provide a provider configuration model that connects to out-of-tree providers via gRPC/TLS.
- Make referent authentication explicit (e.g., authentication scope for cluster-scoped resources).
- Add unified behaviors: refresh intervals, controller classes, retry settings, and gating policies.
- Introduce explicit state tracking policy for Generators.
- Maintain ExternalSecret/PushSecret compatibility via apiVersion on storeRef.
- Provide a migration path from v1 to v2, including a v1 plugin provider bridge and dedicated builds without v1 code.
- Deliver at least one test provider (fake) and one functional reference provider (e.g., AWS/GCP/Vault) as out-of-tree projects.
- Add end-to-end tests and documentation to validate and explain the new model.

### Non-Goals


## Proposal

### Overview

ESO will run without bundled providers. Users deploy desired providers independently as separate services. ESO connects to providers over the network using secure gRPC/TLS.

![Out of Tree](../assets/eso-out-of-tree.png)

### API Resources

#### SecretStore

Remove `spec.provider` and introduce `spec.providerConfig`, which contains the endpoint and authentication required to reach an out-of-tree provider, plus a provider-owned reference forwarded on requests.

```yaml
apiVersion: secretstore.external-secrets.io/v2alpha1
kind: SecretStore
metadata:
  name: my-aws-store
  namespace: default
spec:
  refreshInterval: 1m
  controller: dev
  # ESO reconciles only if the store is healthy or unknown
  gatingPolicy: Enabled # or Disabled
  retrySettings:
    maxRetries: 3
    retryInterval: 10s
  providerConfig:
    address: http+unix:///path/to/aws.sock
    auth:
      clientCertificate: {}
      serviceAccountRef: {}
    providerRef:
      name: my-aws-store
      namespace: default
      kind: AWSSecretManager
---
apiVersion: provider.secretstore.external-secrets.io/v2alpha1
kind: AWSSecretManager
metadata:
  name: my-aws-store
  namespace: default
spec:
  role: arn:aws:iam::123456789012:role/external-secrets
  region: eu-central-1
  auth:
    secretRef:
      accessKeyIDSecretRef:
        name: awssm-secret
        key: access-key
      secretAccessKeySecretRef:
        name: awssm-secret
        key: secret-access-key
status: {}
```

Notes:
- `providerRef` is owned and managed by the provider and lives in a separate API group (`provider.secretstore.external-secrets.io`).

#### ClusterSecretStore

`ClusterSecretStore` makes referent authentication explicit via `authenticationScope`, selecting provider namespace or the manifest namespace for credentials. Cluster-scoped resources delegate to namespaced providers.

```yaml
apiVersion: secretstore.external-secrets.io/v2alpha1
kind: ClusterSecretStore
metadata:
  name: my-cluster-store
spec:
  refreshInterval: 1m
  controller: dev
  retrySettings:
    maxRetries: 3
    retryInterval: 10s
  providerConfig:
    address: http+unix:///path/to/socket.sock
    providerRef:
      name: my-aws-store
      namespace: default
      kind: AWSSecretManager
    auth: {}
  gatingPolicy: Enabled
  authenticationScope: ProviderNamespace # or ManifestNamespace
  conditions:
  - namespaceSelector: {}
    namespaces: []
    namespaceRegexes: []
```

#### Generator

Generators adopt `providerConfig` to delegate generation to out-of-tree providers and gain parity features.

- `refreshInterval` to periodically reconcile.
- `controller` to support controller classes.
- `providerConfig` to delegate to an out-of-tree provider.
- `statePolicy` and `stateSpec` to control GeneratorState creation/patching.
- `gatingPolicy` to enable/disable floodgating for generators.

```yaml
apiVersion: generator.external-secrets.io/v2alpha1
kind: Generator
metadata:
  name: my-password
  namespace: default
spec:
  refreshInterval: 1m
  controller: dev
  providerConfig:
    address: http+unix:///path/to/socket.sock
    providerRef:
      name: password-gen
      namespace: default
      kind: Password
  statePolicy: Track # or Ignore
  stateSpec:
    garbageCollectionDeadline: "5m"
    statePatch: {}
  gatingPolicy: Enabled
---
apiVersion: provider.generator.external-secrets.io/v2alpha1
kind: Password
metadata:
  name: password-gen
  namespace: default
spec:
  digits: 5
  symbols: 5
  symbolCharacters: "-_$@"
  noUpper: false
  allowRepeat: true
```

#### ClusterGenerator

ClusterGenerators mirror ClusterSecretStores and extend namespaced Generators cluster-wide.

```yaml
apiVersion: generator.external-secrets.io/v2alpha1
kind: ClusterGenerator
metadata:
  name: my-cluster-generator
spec:
  refreshInterval: 1m
  controller: dev
  providerConfig:
    address: http+unix:///path/to/socket.sock
    providerRef:
      name: password-gen
      namespace: default
      kind: Password
  statePolicy: Track # or Ignore
  stateSpec:
    garbageCollectionDeadline: "5m"
    statePatch: {}
  gatingPolicy: Enabled
  authenticationNamespace: ProviderReference # or ManifestReference
  conditions:
  - namespaceSelector: {}
    namespaces: []
    namespaceRegexes: []
```

### Changes to ExternalSecret and PushSecret

To maintain compatibility, `ExternalSecret` and `PushSecret` add `secretStoreRef.apiVersion`. Controllers use this field to decide whether to call v1 providers or v2 out-of-tree providers. No other changes are required.

### New Provider Interfaces

Provider and Generator interfaces are updated to pass full specs and enable provider-side processing.

```
ProviderV2
----------
GetSecret(SecretStoreSpec, ExternalSecretDataRemoteRef) ([]byte, error)
PushSecret(SecretStoreSpec, *corev1.Secret, PushSecretData) error
DeleteSecret(SecretStoreSpec, PushSecretRemoteRef) error
SecretExists(SecretStoreSpec, PushSecretRemoteRef) (bool, error)
GetAllSecrets(SecretStoreSpec, ExternalSecretFind) (map[string][]byte, error)
Validate(SecretStoreSpec) (admission.Warnings, error)
Capabilities(SecretStoreSpec) SecretStoreCapabilities

GeneratorV2
-----------
Generate(GeneratorSpec) (map[string][]byte, GeneratorProviderState, error)
Cleanup(GeneratorSpec, GeneratorProviderState) error
```

### Out-of-Tree Providers Maintenance

#### Deployment

Out-of-tree providers are separate projects with their own repos, images, and Helm charts. Users deploy ESO and the providers they need. ESO connects to providers via a Kubernetes Service indicated by `providerConfig.address`. Co-locating providers as sidecars is discouraged to preserve isolation and scalability.

#### Governance

- One repo per provider (e.g., `external-secrets-provider-aws`).
- Promotion lifecycle (experimental → stable).
- CODEOWNERS and standard PR workflows per provider.
- Optionally a collective community repo for community-maintained providers.

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

- Do not implement Unix domain sockets for sidecars; providers should run as independent deployments to ensure horizontal scalability, separate network policies, and stronger isolation.
- Cluster-scoped resources delegate to namespaced providers; referent authentication is explicit via `authenticationScope`.
- Generators should not introduce caching policy that stores sensitive data in ESO; state tracking controls metadata/state only.

### Risks and Mitigations

- Operational overhead: Users manage separate provider deployments → mitigated by dedicated Helm charts and independent versioning per provider.
- Misconfiguration and noisy retries: Introduce `gatingPolicy`, `retrySettings`, and periodic reconciliation to surface and control failures.
- Network boundary and TLS management: Use gRPC/TLS with explicit `providerConfig.auth`; document rotation and certificate management practices.

## Design Details

### Test Plan

#### Prerequisite testing updates

#### Unit tests

#### Integration tests

#### e2e tests

### Graduation Criteria


### Upgrade / Downgrade Strategy

A phased migration enables safe adoption from v1 to v2:

1) Early adoption via a v1 plugin provider
- Introduce a special `plugin` provider within `SecretStore/v1` that forwards requests to v2 out-of-tree providers. This allows testing v2 providers without changing existing v1 resources.

2) Full migration
- Define v2 SecretStore and Generator CRDs pointing to out-of-tree provider deployments/CRs.
- Update `ExternalSecret` manifests to use `secretStoreRef.apiVersion: secretstore.external-secrets.io/v2alpha1` and reference v2 stores.
- Decommission v1 stores after all `ExternalSecret` resources are migrated.

3) Dedicated builds
- Provide ESO builds without v1 CRDs and in-tree provider code to reduce footprint for fully migrated users.

### Version Skew Strategy


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback


### Rollout, Upgrade and Rollback Planning


### Monitoring Requirements


### Dependencies


### Scalability


### Troubleshooting


## Implementation History


## Drawbacks

- Increased operational responsibility for users to deploy and manage provider lifecycles in addition to ESO.
- New CRDs introduce a learning curve and require updated documentation.
- Separate repositories, issue trackers, and release pipelines for each provider increase maintenance overhead.
- Distributed maintenance across community providers can fragment ownership.

## Alternatives


## Infrastructure Needed (Optional)
