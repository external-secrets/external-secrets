---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator changes the spec and metadata a ClusterExternalSecret provisions
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [External secret spec, External secret metadata] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product updates the ExternalSecret it provisioned in every selected namespace to match
    kind: product
    actor: cluster-operator
    entities:
      - { entity: external-secret, effect: changes, facts: [Secret store, Data, Data from, Target name, Creation policy, Deletion policy, Template, Immutable, Target kind, Refresh policy, Refresh interval, Sync windows] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product records the provisioned namespaces on the ClusterExternalSecret
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [Provisioned namespaces, Failed namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
---

# Update every provisioned ExternalSecret

## Trigger

A Cluster Operator needs another key, store or template in every namespace.

## Outcome

Each provisioned ExternalSecret carries the new declaration and syncs its
target again, as an edit of that ExternalSecret would.

## Edge cases

- Labels and annotations set directly on a provisioned ExternalSecret are replaced by the ClusterExternalSecret's.
