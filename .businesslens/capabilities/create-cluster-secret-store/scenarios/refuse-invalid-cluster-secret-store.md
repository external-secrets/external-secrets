---
kind: validation
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator submits a cluster-wide store with a namespace pattern that is not a valid regular expression
    kind: actor
    actor: cluster-operator
    entities: []
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
  - text: The Product refuses it, naming the condition that cannot be read
    kind: condition
    actor: cluster-operator
    entities: []
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
---

# Refuse an invalid ClusterSecretStore

## Trigger

The submitted store has an unreadable namespace pattern, or any of the problems
that make a SecretStore invalid.

## Outcome

No ClusterSecretStore is created and the operator sees why.
