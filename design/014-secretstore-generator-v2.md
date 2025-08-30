```yaml
---
title: SecretStores and Generators v2
version: v1alpha1
authors: Gustavo Carvalho
creation-date: 2025-05-18
status: approved
---
```

# SecretStores and Generators v2

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->

## Summary

This Document describes a design proposal for SecretStores and Generators on its v2 version.

## Motivation

Currently, there are a number of features that cannot be easily done with the codebase / CRD structure for SecretStores and Generators, including:  

* Different Provider Versioning;
* Allowing first-class support for out-of-tree providers;
* Allowing to install/uninstall SecretStores and Generators wanted / not wanted by the user.
* Referent authentication modes are implicit and hard to learn / use.

Furthermore, Generators and SecretStores themselves have different core features, while in reality, we should make them have the same feature set, if not sharing the same codebase:

* SecretStores support ClusterSecretStores with Referrent Authentication and cross namespace generation; Generators dont;
* Generators support GeneratorState to track down eventual cache information on the CRD; SecretStores dont;
* SecretStores Are periodically refreshed; Generators arent;
* Badly configured SecretStores prevent jobs to happen via a gating mechanism; Generators dont;

## Proposal

The proposal is a new CRD for both SecretStores and Generators that unifies all of their feature sets, bringing the best of the worlds. Since this is a significant change, the idea is to also leverage a new controller and client manager for this CRD, in order to make it easier to maintain and scale.

While changing the provider interface for both SecretStores and Generators itself is a non goal, it might be needed to accomodate this proposal.

### SecretStore
SecretStore manifests looks very much like our original implementation. Provider configuration goes to their specific CRD manifests. 
```yaml
apiVersion: secretstore.external-secrets.io/v2alpha1
kind: SecretStore
metadata:
  name: my-aws-store
  namespace: default
spec:
  refreshInterval: 1m
  controller: dev
  retrySettings:
    maxRetries: 3
    retryInterval: 10s
  providerConfig:
    address: http+unix:///path/to/aws.sock
    providerRef:
      name: my-aws-store
      namespace: default
      kind: AWSSecretManager
    auth:
      clientCertificate: {}
      serviceAccountRef: {}
      spiffeRef: {}
  gatingPolicy: Enabled/Disabled
---
apiVersion: provider.secretstore.external-secrets.io/v2alpha1
kind: AWSSecretManager
metadata:
  name: my-aws-store
  namespace: default
status:
  #... Same status as we already have
spec:
    role: arn:aws:iam::123456789012:role/external-secrets
    region: eu-central-1
    auth:
      secretRef:
        accessKeyIDSecretRef:
          name: awssm-secret
          key: access-key
          # note: namespace key in here is always blank. and removed from manifests
        secretAccessKeySecretRef:
          name: awssm-secret
          key: secret-access-key    
```
New fields would be:
* gatingPolicy - per SecretStore, explicit behavior for `enablefloodGate` flag (currently at controller level only)

Notes:
* Provider groups are not reconciled by External-Secrets. They are reconciled and managed by external controllers representing each provider;
* Each Provider must implement the `ProviderV2` interface (see below)

### ClusterSecretStore
As a part of this design, the ClusterSecretStore will fundamentally change to be a providerless-type of store:
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
    providerRef: # Forwarded to the provider at the Address above
      name: my-aws-store
      namespace: default
      kind: AWSSecretManager
    auth: {} # same as SecretStore
  gatingPolicy: Enabled/Disabled
  authenticationScope: ProviderNamespace/ManifestNamespace
  conditions:
  - namespaceSelector: {}
    namespaces: []
    namespaceRegexes: []
```
This CRD introduces the `providerRef` concept - as this is where the `provider` session of the store is going to be obtained.
We make the `authenticationScope` also as a new field to explicitly configure the behavior of the SecretStore (use the provider namespace, or use the ManifestNamespace), to provide the referrent authentication feature.

With this change, we only need maintain providers by different CRDs in one layer - the Namespaced one. Anything else that wants to be exported to the cluster will need to have a backing SecretStore to it.

Note: Common field on the ClusterSecretStore with the SecretStore ones could be possibly removed from the spec :thinking: I am not sure if there is any benefit in overwriting them or keeping them as of yet.

### Generator
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
    providerRef: # Forwarded to the provider at the Address above
      name: password-gen
      namespace: default
      kind: Password
  statePolicy: Track/Ignore # track or ignore GeneratorStates
  stateSpec: # field to inject custom annotations/labels, configuration to the GeneratorState Spec
    garbageCollectionDeadline: "5m" # value to use for this specific generator:
    statePatch:
      #a patch to the state after the provider created a generator State.
      #anything to be added by the user to the state - as to control state creation/deletion with custom logic.
      #this is an extension point for custom logic the user wants to provide.
    gatingPolicy: Enabled/Disabled #introducing floodgating for generators
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
New fields would be:
* refreshInterval - allowing Generators to benefit reconciliation for configuration mistakes;
* controller - introducing Controller Classes to Generators
* providerConfig - whether this is an InTree or OutOfTree provider. if OutOfTree, we don't expect any of the `Provider` Interface to be implemented, and instead just delegate to the outOfTree provider that a new job needs to be executed to fetch a given secret. 
* statePolicy - whether or not to track generator states
* stateSpec - field to inject custom annotations/labels, configuration to the GeneratorState Spec
* gatingPolicy - per SecretStore, explicit behavior for `enablefloodGate` flag (currently at controller level only)
Note: IMO, caching policy makes no sense here, since we do not store the generator sensitive data ourselves anywhere.

### ClusterGenerator
IMO, the beauty of this design is how clusterGenerators are just a natural extension of ClusterSecretStore:
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
    providerRef: # Forwarded to the provider at the Address above
      name: password-gen
      namespace: default
      kind: Password
  statePolicy: Track/Ignore # track or ignore GeneratorStates
  stateSpec: # field to inject custom annotations/labels, configuration to the GeneratorState Spec
    garbageCollectionDeadline: "5m" # value to use for this specific generator:
    statePatch:
      #a patch to the state after the provider created a generator State.
      #anything to be added by the user to the state - as to control state creation/deletion with custom logic.
      #this is an extension point for custom logic the user wants to provide.
  gatingPolicy: Enabled/Disabled #introducing floodgating for generators
  authenticationNamespace: ProviderReference/ManifestReference
  conditions: # just like we have for ClusterSecretStores
  - namespaceSelector: {}
    namespaces: []
    namespaceRegexes: []
```


### Changes to `ExternalSecret`/ `PushSecret`
In order to make this change compatible with the existing implementations of `ExternalSecret`/`PushSecret`, the proposed changes to them are limited to `SecretStoreRef`, which should now accomodate a `apiVersion` field. This field will allow us to add conditional logic to use the new interfaces described below or to keep the current behavior leveraging `SecretStore/v1` and `Generators/v1alpha1`.

There will be no need to change the support for `Generators`, only to add conditionals on the controller runtime.

(While I know conditionals are bad, I don't think we can pull this off without them - these interfaces are not compatible from what I could think of)

### New Interfaces

In order to achieve this proposed design, we need to add a new interface for which providers must implement.

The signature is basically changed to include the SecretStoreSpec/GeneratorSpec, as some combination of it will be needed to be processed at the provider level and at ESO level (like e.g. caching olicy or Referent Authentication)


#### ProviderV2
```
GetSecret(SecretStoreSpec, ExternalSecretDataRemoteRef) ([]byte, error)

PushSecret(SecretStoreSpec, *corev1.Secret, PushSecretData) error
DeleteSecret(SecretStoreSpec, PushSecretRemoteRef) error
SecretExists(SecretStoreSpec, PushSecretRemoteRef) (bool, error)
GetAllSecrets(SecretStoreSpec, ExternalSecretFind) (map[string][]byte, error)

Validate(SecretStoreSpec) (admission.Warnings, error)

Capabilities(SecretStoreSpec) SecretStoreCapabilities
```

#### GeneratorV2

```
Generate(GeneratorSpec) (map[string][]byte, GeneratorProviderState, error)
Cleanup(GeneratorSpec, GeneratorProviderState) error
```

## Out-of-Tree Providers Maintenance

### Deployment

Out-of-tree providers would be self-contained projects, each with its own dedicated Git repository, container images, issue trackers, and Helm chart. We might decide that some providers will still be released under `external-secrets` organization, or not (we can decide this afterwards). This model simplifies distribution and versioning, allowing providers to evolve independently of the core External Secrets Operator (ESO).

The user's responsibility is to deploy two main components:
1.  The core External Secrets Operator.
2.  The Helm chart for each out-of-tree provider they intend to use.

Communication between ESO and the provider occurs over a standard Kubernetes `Service`. The `providerConfig.address` field in the `SecretStore` manifest will point to this service endpoint. We explicitly discourage co-locating providers in the same pod as ESO to maintain strict security and resource isolation boundaries.

### Governance

To foster a healthy ecosystem of community-maintained providers, a clear governance model is essential. We propose:

*   **Repository Structure**: A new GitHub repository for each provider (e.g., `external-secrets-provider-aws`) will house the code for each provider.
*   **Provider Lifecycle**: A promotion-based lifecycle (e.g., `experimental` -> `stable`) will be used to signal the maturity and stability of each provider.
*   **Ownership and Contribution**: Each provider will have dedicated `CODEOWNERS`. Contributions will follow a standard pull-request workflow, ensuring quality and consistency.
*  **Community Providers on specific repository**: Community providers could all exist on a `external-secrets-provider-community` 

This structure provides a clear path for community contributions while ensuring a baseline of quality and security for all official out-of-tree providers.

## Migration Strategy

A phased migration is recommended to ensure a smooth transition from `v1` to `v2` SecretStores without disrupting existing workflows.

### Phase 1: Early Adoption via a Plugin Provider

To facilitate early adoption and gather feedback, we can introduce a special `plugin` provider within the existing `SecretStore/v1` definition. This `plugin` provider acts as a client, forwarding requests to a `v2` provider running out-of-tree. This allows users to test and use `v2` providers with their existing `v1` resources, gaining the benefits of the new model while preparing for a full migration. There is already a draft PR in place that addresses the plugin consumer (though using a different interface than the one proposed here)

### Phase 2: Full Migration

Once users are comfortable with the `v2` providers, they can perform a full migration by following these steps:

1.  **Define `v2` Resources**: Create the new `v2` `SecretStore` and `Generator` CRDs, pointing them to the out-of-tree provider deployments / CRs.
2.  **Update `ExternalSecret` Manifests**: Modify existing `ExternalSecret` resources to use the new API version (`secretStoreRef.apiVersion: ecretstore.external-secrets.io/v2alpha1`) and reference the `v2` stores.
3.  **Decommission `v1` Stores**: After all `ExternalSecret` resources have been migrated, the old `v1` `SecretStores` can be safely removed - potentially removing `SecretStore/v1` CRDs.

This approach provides a clear and safe migration path, allowing users to adopt the more secure, flexible, and maintainable `v2` architecture at their own pace.

#### Phase3 : Use edicated builds
We provide dedicated eso builds that already reduces the code footprint for early adopters by not having `SecretStore/v1` and `pkg/provider` code registered. This allows users to fully migrate by not having the package dependencies of our in-tree system 

## Consequences

*   **Increased Operational Responsibility for Users**: Users will be responsible for deploying and managing the lifecycle of each out-of-tree provider they use, in addition to the core ESO operator. This adds an operational step but provides greater control over provider versions.
*   **New CRDs and Learning Curve**: The introduction of `v2` CRDs requires users to learn a new API. Clear documentation and a seamless migration path are critical to mitigate this.
*   **Development and Release Overhead**: Maintaining separate repositories, issue trackers, and release pipelines for each provider introduces complexity for maintainers compared to a monolithic release.
*   **Improved Security and Isolation**: Decoupling providers into separate processes enhances security. A vulnerability in one provider is less likely to impact the core operator or other providers.
*   **Distributed Maintenance Overhead**: Community can now decide if they want to maintain providers themselves or make them available as part of External Secrets providers (today more than half of our open feature requests are provider-specific)
*   **Independent Versioning and Flexibility**: Providers can be updated, patched, or rolled back independently of the core operator, allowing users to adopt new provider features or fixes more quickly.

## Acceptance Criteria

*   **CRDs Implemented**: The new `SecretStore/v2alpha1`, `ClusterSecretStore/v2alpha1`, `Generator/v2alpha1`, and `ClusterGenerator/v2alpha1` CRDs are implemented and functional.
*   **Test Out-of-Tree Provider**: At least one test out-of-tree provider (e.g., Fake) is created, can be deployed via its own Helm chart, and successfully serves secrets to ESO using the `v2` interface.
*   **Functional Out-of-Tree Provider**: At least one reference out-of-tree provider (e.g., for AWS, GCP, or Vault) is created, can be deployed via its own Helm chart, and successfully serves secrets to ESO using the `v2` interface.
*   **`ExternalSecret` Compatibility**: The `ExternalSecret` controller can reconcile secrets using a `storeRef` that points to a `v2` `SecretStore` by specifying the new `apiVersion`.
*   **Validated Migration Path**: The `v1` `plugin` provider is implemented and allows an existing `SecretStore/v1` to successfully fetch secrets from a `v2` out-of-tree provider.
*   **Cluster Scoped Resources**: `ClusterSecretStore/v2alpha1` and `ClusterGenerator/v2alpha1` are fully functional and can delegate authentication and secret operations to their namespaced `v2` counterparts.
*   **Dedicated Builds**: A dedicated, slimmed-down build of ESO is available that excludes all `v1` CRDs and in-tree provider code, offering a reduced footprint for new deployments or fully migrated users.
*   **End-to-End Testing**: Comprehensive end-to-end tests are written to validate the functionality of `v2` stores, generators, the migration path, and cluster-scoped resources.
*   **Documentation**: The official documentation is updated to include guides for deploying out-of-tree providers, using the new `v2` CRDs, and following the migration strategy.
