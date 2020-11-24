```yaml
---
title: External Secrets Kubernetes Operator CRD
version: v1alpha1
authors: all of us
creation-date: 2020-09-01
status: draft
---
```

# External Secrets Kubernetes Operator CRD

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Terminology](#terminology)
- [Use-Cases](#use-cases)
- [Proposal](#proposal)
  - [API](#api)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This is a proposal to standardize the External Secret Kubernetes operator CRDs in an combined effort through all projects that deal with syncing external secrets. This proposal aims to do find the _common denominator_ for all users of an External Secrets project.

## Motivation

There are a lot of different projects in the wild that essentially do the same thing: sync secrets with Kubernetes. The idea is to unify efforts into a single project that serves the needs of all users in this problem space.

As a starting point I (@moolen) would like to define a **common denominator** for a Custom Resource Definition that serves all known use-cases. This CRD should follow the standard alpha -> beta -> GA feature process.

Once the CRD API is defined we can move on with more delicate discussions about technology, organization and processes.

List of Projects known so far or related:
* https://github.com/godaddy/kubernetes-external-secrets
* https://github.com/itscontained/secret-manager
* https://github.com/ContainerSolutions/externalsecret-operator
* https://github.com/mumoshu/aws-secret-operator
* https://github.com/cmattoon/aws-ssm
* https://github.com/tuenti/secrets-manager
* https://github.com/kubernetes-sigs/k8s-gsm-tools

### Goals

- Define an alpha CRD
- Fully document the Spec and use-cases

### Non-Goals

This KEP proposes the CRD Spec and documents the use-cases, not the choice of technology or migration path towards implementing the CRD.

We do not want to sync secrets into a `ConfigMap`.

## Terminology

* Kubernetes External Secrets `KES`: A Application that runs a control loop which syncs secrets
* KES `instance`: A single entity that runs a control loop
* Provider: Is a **source** for secrets. The Provider is external to KES. It can be a hosted service like Alibaba Cloud SecretsManager, AWS SystemsManager, Azure KeyVault etc
* SecretStore `ST`: A Custom Resource to authenticate and configure the connection between the KES instance and the Provider
* ExternalSecret `ES`: A Custom Resource that declares which secrets should be synced
* Frontend: A **sink** for the synced secrets, usually a `Secret` resource
* Secret: Credentials that act as a key to sensitive information

## Use-Cases
* One global KES instance that manages ES in **all namespaces**, which gives access to **all providers**, with ACL
* Multiple global KES instances, each manages access to a single or multiple providers (e.g.: shard by stage or team...)
* One KES per namespace (a user manages their own KES instance)

### User Definitions
* `operator :=` I manage one or multiple `KES` instances
* `user :=` I only create `ES`, KES is managed by someone else

### User Stories
From that we can derive the following requirements or user stories:
1. As a KES operator I want to run multiple KES instances per cluster (e.g. one KES instance per DEV/PROD)
1. As a KES operator or user I want to integrate **multiple SecretStores** with a **single KES instance** (e.g. dev namespace has access only to dev secrets)
1. As a KES user I want to control the Frontend for the secrets, usually a `Secret` resource
1. As a KES user I want to fetch **from multiple** Providers and store the secrets **in a single** Frontend
1. As a KES operator I want to limit the access to certain stores or sub resources (e.g. having one central KES instance that handles all ES - similar to `iam.amazonaws.com/permitted` annotation per namespace)
1. As a KES user I want to provide an application with a configuration that contains a secret

### Providers

These providers are relevant for the project:
* AWS Secure Systems Manager Parameter Store
* AWS Secrets Manager
* Hashicorp Vault
* Azure Key Vault
* Alibaba Cloud KMS Secret Manager
* Google Cloud Platform Secret Manager
* Kubernetes (see [#422](https://github.com/external-secrets/kubernetes-external-secrets/issues/422))
* noop (see [#476](https://github.com/external-secrets/kubernetes-external-secrets/issues/476))

### Frontends

* A Secret Kubernetes resource
* *potentially* we could sync Provider to Provider

## Proposal

### API

### External Secret

The `ExternalSecret` Custom Resource Definition is **namespaced**. It defines the following:
1. Source for the secret (`SecretStore`)
2. Sink for the secret (Fronted)
3. A mapping to translate the keys

```yaml
apiVersion: external-secrets.x-k8s.io/v1alpha1
kind: ExternalSecret
metadata: {...}

spec:
  # SecretStoreRef defines which SecretStore to fetch the ExternalSecret data
  secretStoreRef:
    name: secret-store-name
    kind: SecretStore  # or ClusterSecretStore

	# RefreshInterval is the amount of time before the values reading again from the SecretStore provider
	# Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h" (from time.ParseDuration)
	# May be set to zero to fetch and create it once
  refreshInterval: "1h"

  # There can only be one target per ES
  target:

    # The secret name of the resource
    # Defaults to .metadata.name of the ExternalSecret
    # It is immutable
    name: my-secret

    # Enum with values: 'Owner', 'Merge', or 'None'
    # Default value of 'Owner'
    # Owner creates the secret and sets .metadata.ownerReferences of the resource
    # Merge does not create the secret, but merges in the data fields to the secret
    # None does not create a secret (future use with injector)
    creationPolicy: 'Merge'

    # Specify a blueprint for the resulting Kind=Secret
    template:
      type: kubernetes.io/dockerconfigjson # or TLS...

      metadata:
        annotations: {}
        labels: {}

      # Use inline templates to construct your desired config file that contains your secret
      data:
        config.yml: |
          endpoints:
          - https://{{ .data.user }}:{{ .data.password }}@api.exmaple.com

      # Uses an existing template from configmap
      # Secret is fetched, merged and templated within the referenced configMap data
      # It does not update the configmap, it creates a secret with: data["alertmanager.yml"] = ...result...
      templateFrom:
      - configMap:
          name: alertmanager
          items:
          - key: alertmanager.yaml

  # Data defines the connection between the Kubernetes Secret keys and the Provider data
  data:
    - secretKey: secret-key-to-be-managed
      remoteRef:
        key: provider-key
        version: provider-key-version
        property: provider-key-property

  # Used to fetch all properties from the Provider key
  # If multiple dataFrom are specified, secrets are merged in the specified order
  dataFrom:
  - remoteRef:
      key: provider-key
      version: provider-key-version
      property: provider-key-property

status:
  # Represents the current phase of the secret sync:
  # * Pending | ES created, controller did not yet sync the ES or other dependencies are missing (e.g. secret store or configmap template)
  # * Syncing | ES is being actively synced according to spec
  # * Failing | Secret can not be synced, this might require user intervention
  # * Failed  | ES can not be synced right now and will not able to
  # * Completed | ES was synced successfully (one-time use only)
  phase: Syncing
  conditions:
  - type: InSync
    status: "True" # False if last sync was not successful
    reason: "SecretSynced"
    message: "Secret was synced"
    lastTransitionTime: "2019-08-12T12:33:02Z"
    lastSyncTime: "2020-09-23T16:27:53Z"

```

#### Behavior

The ExternalSecret control loop **ensures** that the target resource exists and stays up to date with the upstream provider. Because most upstream APIs are limited in throughput the control loop must implement some sort of jitter and retry/backoff mechanic.

### Secret Store

The store configuration in an `ExternalSecret` may contain a lot of redundancy, this can be factored out into its own CRD.
These stores are defined in a particular namespace using `SecretStore` **or** globally with `ClusterSecretStore`.

```yaml
apiVerson: external-secrets.x-k8s.io/v1alpha1
kind: SecretStore # or ClusterSecretStore
metadata:
  name: vault
  namespace: example-ns
spec:

  # Used to select the correct KES controller (think: ingress.ingressClassName)
  # The KES controller is instantiated with a specific controller name and filters ES based on this property
  # Optional
  controller: dev

  # AWSSM configures this store to sync secrets using AWS Secret Manager provider
  awssm:
    # Auth defines the information necessary to authenticate against AWS by 
    # getting the accessKeyID and secretAccessKey from an already created Kubernetes Secret
    auth:
      secretRef:
        accessKeyID:
          name: awssm-secret
          key: access-key

        secretAccessKey:
          name: awssm-secret
          key: secret-access-key

    # Role is a Role ARN which the SecretManager provider will assume
    role: iam-role

    # AWS Region to be used for the provider
    region: eu-central-1

status:
  # * Pending: e.g. referenced secret containing credentials is missing
  # * Running: all dependencies are met, sync
  phase: Running
  conditions:
  - type: Ready
    status: "False"
    reason: "ErrorConfig"
    message: "Unable to assume role arn:xxxx"
    lastTransitionTime: "2019-08-12T12:33:02Z"
```

## Workflow in a KES instance

1. A user creates a `SecretStore` with a certain `spec.controller`
2. A controller picks up the `ExternalSecret` if it matches the `controller` field
3. The controller fetches the secret from the Provider and stores it as Secret Kubernetes resource in the same namespace as ES

## Backlog

We have a bunch of features which are not relevant for the MVP implementation. We keep the features here in this backlog. Order is not specific:

1. Secret injection with a mutating Webhook [#81](https://github.com/godaddy/kubernetes-external-secrets/issues/81)
