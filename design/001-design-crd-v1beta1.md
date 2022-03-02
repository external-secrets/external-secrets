```yaml
---
title: External Secrets CRD promotion
version: v1beta1
authors: all of us
creation-date: 2022-feb-08
status: approved
---
```

# External Secrets Operator CRD

## Table of Contents

- [External Secrets Operator CRD](#external-secrets-operator-crd)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Terminology](#terminology)
    - [User Definitions](#user-definitions)
    - [User Stories](#user-stories)
  - [Proposal](#proposal)
    - [External Secret](#external-secret)
      - [Behavior](#behavior)
    - [Secret Store](#secret-store)

## Summary

This is a proposal to design the Promoted ExternalSecrets CRD. This proposal was approved in 16-feb-2022 during our Community Meeting.

## Motivation

The project came up to the point to have grown in users and maturity, hence we are starting to drive efforts to bring it to GA. The promotion of the ExternalSecrets CRD to beta is one of this efforts.
This design documentation aims to capture some final changes for ExternalSecrets CRD.

### Goals

- Define a beta CRD
- Define strucutre for getting all provider secrets
- Define structure for new templating engine
### Non-Goals

This KEP proposes the CRD Spec and documents the use-cases, not the choice of technology or migration path towards implementing the CRD.

## Terminology

* External Secrets Operator `ESO`: A Application that runs a control loop which syncs secrets
* ESO `instance`: A single entity that runs a control loop
* Provider: Is a **source** for secrets. The Provider is external to ESO. It can be a hosted service like Alibaba Cloud SecretsManager, AWS SystemsManager, Azure KeyVault etc
* SecretStore `ST`: A Custom Resource to authenticate and configure the connection between the ESO instance and the Provider
* ExternalSecret `ES`: A Custom Resource that declares which secrets should be synced
* Frontend: A **sink** for the synced secrets, usually a `Secret` resource
* Secret: Credentials that act as a key to sensitive information

### User Definitions
* `operator :=` I manage one or multiple `ESO` instances
* `user :=` I only create `ES`, ESO is managed by someone else

### User Stories
From that we can derive the following requirements or user stories:
1. As a ESO operator I want to get all the secrets of a given path from a given provider, if the provider supports it
2. As a ESO operator I want to handle templating like it is a natural language, not needing to worry about how it is actually implemented.

## Proposal

### External Secret

```yaml
#only changed fields are commented out.
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: "hello-world"
  labels:
    acme.org/owned-by: "q-team"
  annotations:
    acme.org/sha: 1234

spec:
  secretStoreRef:
    name: secret-store-name
    kind: SecretStore
  refreshInterval: "1h"
  target:
    name: my-secret
    creationPolicy: 'Merge'
    deletionPolicy: 'None' #Possible values are None, Merge, Delete - TBC during implementation.
    template:
      engineVersion: v2 #Defaults to v2 in v1beta1
      type: kubernetes.io/dockerconfigjson 
      metadata:
        annotations: {}
        labels: {}
      data:
        config.yml: |
          endpoints:
          - https://{{ .data.user }}:{{ .data.password }}@api.exmaple.com

      templateFrom:
      - configMap:
          name: alertmanager
          items:
          - key: alertmanager.yaml
  data:
    - secretKey: secret-key-to-be-managed
      remoteRef:
        key: provider-key
        version: provider-key-version
        property: provider-key-property
  dataFrom:
  - extract: #extract all the keys from one given secret
      key: provider-key
      version: provider-key-version
      property: provider-key-property
  - find:
      name:  #find secrets that match a particular pattern
        regexp: .*pattern.*
      tags:  #find secrets that match the following labels/tags
        provider-label: provider-value
status:
  refreshTime: "2019-08-12T12:33:02Z"
  conditions:
  - type: Ready
    status: "True"
    reason: "SecretSynced"
    message: "Secret was synced"
    lastTransitionTime: "2019-08-12T12:33:02Z"
```

#### Behavior

ExternalSecrets now will have a different structure for `dataFrom`, which will allow fetching several provider secrets with only one ExternalSecret definition. It should be possible to fetch secrets based on regular expressions or by a label/tag selector.

If the user desires to rename the secret keys (e.g. because the key name is not a valid secret key name `/foo/bar`) they should use `template` functions to produce a mapping. 
### Secret Store

SecretStore and ClusterSecretStore do not have any changes from v1alpha1.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: example
  namespace: example-ns
spec:
  controller: dev
  retrySettings:
    maxRetries: 5
    retryInterval: "10s"
  provider:
    aws:
      service: SecretsManager
      role: iam-role
      region: eu-central-1
      auth:
        secretRef:
          accessKeyID:
            name: awssm-secret
            key: access-key
          secretAccessKey:
            name: awssm-secret
            key: secret-access-key
    vault:
      server: "https://vault.acme.org"
      path: "secret"
      version: "v2"
      namespace: "a-team"
      caBundle: "..."
      caProvider:
        type: "Secret"
        name: "my-cert-secret"
        key: "cert-key"
      auth:
        tokenSecretRef:
          name: "my-secret"
          namespace: "secret-admin"
          key: "vault-token"
        appRole:
          path: "approle"
          roleId: "db02de05-fa39-4855-059b-67221c5c2f63"
          secretRef:
            name: "my-secret"
            namespace: "secret-admin"
            key: "vault-token"
        kubernetes:
          mountPath: "kubernetes"
          role: "demo"
          serviceAccountRef:
            name: "my-sa"
            namespace: "secret-admin"
          secretRef:
            name: "my-secret"
            namespace: "secret-admin"
            key: "vault"
    gcpsm:
      auth:
        secretRef:
          secretAccessKeySecretRef:
            name: gcpsm-secret
            key: secret-access-credentials
      projectID: myproject
status:
  conditions:
  - type: Ready
    status: "False"
    reason: "ConfigError"
    message: "SecretStore validation failed"
    lastTransitionTime: "2019-08-12T12:33:02Z"
```