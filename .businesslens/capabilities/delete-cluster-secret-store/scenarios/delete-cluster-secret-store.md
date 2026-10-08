---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator deletes a ClusterSecretStore nothing pushes through
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-secret-store, effect: removes }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
---

# Delete an unused ClusterSecretStore

## Trigger

A Cluster Operator retires a shared store.

## Outcome

The ClusterSecretStore is gone; ExternalSecrets still naming it fail to sync
until they name another store.

## Edge cases

- While PushSecrets with deletion policy Delete still use it, the ClusterSecretStore stays marked for deletion until they are gone.
