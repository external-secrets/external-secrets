```yaml
---
title: SecretSink
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
The Secret Sink is a feature to allow Secrets from Kubernetes to be saved back into some providers. Where ExternalSecret is responsible to download a Secret from a Provider into Kubernetes (as a K8s Secret), SecretSink will upload a Kubernetes Secret to a Provider.

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
kind: SecretSink
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

  data:
    selector:
      secret:
        name: foobar
    match:
    - secretKey: foobar
      remoteRefs:
      - remoteKey: my/path/foobar 
        property: my-property #optional. To allow coming back from a 'dataFrom'
        secretStores:  # To allow changing patterns between secretstores. If not provided, will write to all SecretStores
        - name: secret-store
        - name: secret-store-2
      - remoteKey: secret/my-path-foobar
        property: another-property
        secretStores:  # To allow changing patterns between secretstores
        - name: cluster-secret-store
    rewrite:
    - secretKey: game-(.+).(.+)
      remoteRefs:
      - remoteKey: my/path/($1) 
        property: prop-($2)
        secretStores:
        - name: secret-store #Applies this way only to 'secret-store' secretStore
      - remoteKey: my-path-($1)-($2) #Applies this way to all other secretStores

status:
  refreshTime: "2019-08-12T12:33:02Z"
  conditions:
  - type: Ready
    status: "True" 
    reason: "SecretSynced"
    message: "Secret was synced" #Fully synced (to and from)
    lastTransitionTime: "2019-08-12T12:33:02Z"
  - type: Ready
    status: "True"
    reason: "SecretSynced"
    message: "Secret sync failed from Source SecretStore abc"
    lastTransitionTime: "2019-08-12T12:33:02Z"
  - type: Ready
    status: "False"
    reason: "SecretSyncError"
    message: "Secret sync failed to Sink SecretStore def"
    lastTransitionTime: "2019-08-12T12:33:02Z"
```

### Behavior
When checking the the ExternalSecrets for a change, after it updates from the Source Secret Store, it proceeds to update Each Sink Secret Store as well. We probably need to check the reconciliation loop if we have a case no Source Secret Store is defined. We also need to check if a single config

### Drawbacks

We had several discussions on how to implement this feature, and it turns out just by typing how many duplicate fields we would have defeated my original issue to have two separate CRDs. The biggest drawback of this solution is that it implies SecretStores to be able to write with no other mechanism available. Also, it might overload the reconciliation loop as we have 1xN secret Syncing, where most of them are actually outside the cluster.

### Acceptance Criteria
TODO

## Alternatives
Using some integration with Crossplane can allow to sync the secrets. Cons is this must be either manual or through some integration that would be an independent project on its own.

