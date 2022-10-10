```yaml
---
title: PushSecret
version: v1alpha1
authors:
creation-date: 2022-01-25
status: draft
---
```

# Secret Sink

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->


## Summary
The Secret Sink is a feature to allow Secrets from Kubernetes to be saved back into some providers. Where ExternalSecret is responsible to download a Secret from a Provider into Kubernetes (as a K8s Secret), PushSecret will upload a Kubernetes Secret to a Provider.

## Motivation
Secret Sink allows some inCluster generated secrets to also be available on a given secret provider. It also allows multiple Providers having the same secret (which means a way to perform failover in case a given secret provider is on downtime or compromised for whatever the reason).

### Goals
- CRD Design for the SecretSink
- Define the need for a SinkStore
-
### Non-Goals
Do not implement full compatibility mechanisms with each provider (we are not Terraform neither Crossplane)

### Terminology
- Sink object: any Secret (a part or the whole secret) from Kubernetes that is going to be uploaded to a Provider.
## Proposal

A controller that checks for Sink Objects, gets K8s Secrets and creates the equivalent secret on the SecretStore Provider.

### User Stories
1. As an ESO Operator I want to be able to Sync Secrets in my cluster with my External Provider
1. As an ESO Operator I want to be able to Sync Secrets even if they are not bound to a given ExternalSecret

### API
Proposed CRD changes:

```yaml
apiVersion: external-secrets.io/v1alpha1
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
      encryptionConfig: {} # Specific config for Creating Secrets on AWS
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
      encryptionConfig: {} # Specific config for Creating Secrets on Vault ()
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
      encryptionConfig: {} # Specific config for Creating Secrets GCP SM
      auth:
        secretRef:
          secretAccessKeySecretRef:
            name: gcpsm-secret
            key: secret-access-credentials
      projectID: myproject
```

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: "hello-world"
  namespace: my-ns # Same of the SecretStores
spec:
  secretStoreRefs:
  - name: secret-store
    kind: SecretStore
  - name: secret-store-2
    kind: SecretStore
  - name: cluster-secret-store
    kind: ClusterSecretStore
  refreshInterval: "1h"
  selector:
    secret:
      name: foobar
  data:
  - match:
      secretKey: foobar
      remoteRefs:
      - remoteKey: my/path/foobar
        property: my-property #optional. To allow coming back from a 'dataFrom'
      - remoteKey: secret/my-path-foobar
        property: another-property
    rewrite:
      secretKey: game-(.+).(.+)
      remoteRefs:
      - remoteKey: my/path/($1)
        property: prop-($2)
      - remoteKey: my-path-($1)-($2) #Applies this way to all other secretStores

status:
  refreshTime: "2019-08-12T12:33:02Z"
  conditions:
  - type: Ready
    status: "True"
    reason: "SecretSynced"
    message: "Secret was synced" #Fully synced
    lastTransitionTime: "2019-08-12T12:33:02Z"
  - type: Ready
    status: "True"
    reason: "SecretSyncError"
    message: "Secret sync failed to Sink SecretStore: abc"
    lastTransitionTime: "2019-08-12T12:33:02Z"
  - type: Ready
    status: "False"
    reason: "SecretSyncError"
    message: "Secret sync failed to Sink SecretStore: abc, def"
    lastTransitionTime: "2019-08-12T12:33:02Z"
```

### Behavior
When checking PushSecret for the Source Secret, check existing labels for SecretStore reference of that particular Secret. If this SecretStore reference is an object in PushSecret SecretStore lists, a SecretSyncError should be emited as we cannot sync the secret to the same SecretStore.

If the SecretStores are all fine or if the Secret has no labels (secret created by user / another tool), for Each SecretStore, get the SyncState of this store (New, SecretSynced, SecretSyncedErr).

If new Secret, or SecretSynced with refreshInterval expired, get the secret from the secretStore and see if it matches the content of the secrets. If it doesn't match, create a new secret (bumping the version, if possible) within the provider. On errors, emit SecretSyncedErr.

### Drawbacks

We had several discussions on how to implement this feature, and it turns out just by typing how many duplicate fields we would have defeated my original issue to have two separate CRDs. The biggest drawback of this solution is that it implies SecretStores to be able to write with no other mechanism available. Also, it might overload the reconciliation loop as we have 1xN secret Syncing, where most of them are actually outside the cluster.

### Acceptance Criteria
+ ExternalSecrets create appropriate labels on generated Secrets
+ PushSecrets can read labels on source Secrets
+ PushSecrets cannot have same references to SecretStores
+ PushSecrets respect refreshInterval
## Alternatives
Using some integration with Crossplane can allow to sync the secrets. Cons is this must be either manual or through some integration that would be an independent project on its own.

