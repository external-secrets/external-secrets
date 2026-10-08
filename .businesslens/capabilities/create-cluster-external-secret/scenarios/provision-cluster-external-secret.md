---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator applies a ClusterExternalSecret with the declaration to provision and the namespaces to place it in
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: creates, facts: [External secret spec, External secret name, External secret metadata, Namespace selection, Refresh interval] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product creates the ExternalSecret in every selected namespace, owned by the ClusterExternalSecret
    kind: product
    actor: cluster-operator
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Data from, Target name, Creation policy, Deletion policy, Template, Immutable, Target kind, Refresh policy, Refresh interval, Sync windows] }
      - { entity: cluster-external-secret, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product records the provisioned namespaces and marks the ClusterExternalSecret ready
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [Provisioned namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
---

# Provision an ExternalSecret into selected namespaces

## Trigger

A Cluster Operator wants the same secret available in many namespaces.

## Outcome

Every selected namespace has the ExternalSecret, and the ClusterExternalSecret lists them.

## Edge cases

- A namespace being deleted is skipped.
