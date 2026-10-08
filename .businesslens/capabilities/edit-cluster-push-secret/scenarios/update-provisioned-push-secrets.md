---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator changes the spec and metadata a ClusterPushSecret provisions
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-push-secret, effect: changes, facts: [Push secret spec, Push secret metadata] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product updates the PushSecret it provisioned in every selected namespace to match
    kind: product
    actor: cluster-operator
    entities:
      - { entity: push-secret, effect: changes, facts: [Secret stores, Source, Data, Data to, Template, Update policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product records the provisioned namespaces on the ClusterPushSecret
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-push-secret, effect: changes, facts: [Provisioned namespaces, Failed namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
---

# Update every provisioned PushSecret

## Trigger

A Cluster Operator changes what every namespace pushes, or where.

## Outcome

Each provisioned PushSecret carries the new declaration and pushes again, as an
edit of that PushSecret would.

## Edge cases

- Changing the PushSecret name deletes the PushSecrets provisioned under the old name and provisions them under the new one.
