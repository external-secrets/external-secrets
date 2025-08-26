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

Currently, There are a number of features that cannot be easily done with the current codebase / CRD structure for SecretStores and Generators, including:

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
  cachingPolicy: ProviderDefined/Secrets/Auth/All/None
  cacheConfig: # if Caching Policy is Secrets/Auth/All 
      ttl: 1m
      maxEntries: 100
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
  provider: ## same as current secretstore.Provider field
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
* cachingPolicy - as to document per SecretStore what's the caching strategy (cachingScope, ttl, maxEntries). As some secretStores do not have this control in place. `ProviderDefined` policy means whatever the provider defines as the best practice.

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
  cachingPolicy: ProviderDefined/Secrets/Auth/All/None
  cacheConfig: # if Caching Policy is Secrets/Auth/All 
      ttl: 1m
      maxEntries: 100
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
kind: Password
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

## Consequences

TO DO

## Acceptance Criteria

TO DO
