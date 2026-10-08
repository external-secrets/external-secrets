---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator applies a ClusterExternalSecret whose provisioned name is already taken in a selected namespace
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: creates, facts: [External secret spec, External secret name, External secret metadata, Namespace selection, Refresh interval] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product leaves the existing ExternalSecret alone and provisions the other namespaces
    kind: product
    actor: cluster-operator
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Data from, Target name, Creation policy, Deletion policy, Template, Immutable, Target kind, Refresh policy, Refresh interval, Sync windows] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product records the namespace as failed, saying one already exists there, and marks the ClusterExternalSecret not ready
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [Provisioned namespaces, Failed namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
---

# A namespace already has an ExternalSecret of that name

## Trigger

A team already created an ExternalSecret with the name the ClusterExternalSecret uses.

## Outcome

The team's ExternalSecret is untouched; the ClusterExternalSecret reports the namespace among its failures.
