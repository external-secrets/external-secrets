---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator changes the name a ClusterExternalSecret provisions under
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [External secret name] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product deletes each ExternalSecret it provisioned under the old name and creates one under the new name
    kind: product
    actor: cluster-operator
    entities:
      - { entity: external-secret, as: old, effect: removes }
      - { entity: external-secret, as: new, effect: creates, facts: [Secret store, Data, Data from, Target name, Creation policy, Deletion policy, Template, Immutable, Target kind, Refresh policy, Refresh interval, Sync windows] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product records the provisioned namespaces on the ClusterExternalSecret
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [Provisioned namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
---

# Rename the provisioned ExternalSecrets

## Trigger

The provisioned ExternalSecrets must be known under another name.

## Outcome

Every selected namespace has the ExternalSecret under the new name only; what
happens to the old ones' targets follows their creation and deletion policies.
