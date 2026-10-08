---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator applies a ClusterPushSecret with the declaration to provision and the namespaces to place it in
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-push-secret, effect: creates, facts: [Push secret spec, Push secret name, Push secret metadata, Namespace selection, Refresh interval] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product creates the PushSecret in every selected namespace, owned by the ClusterPushSecret
    kind: product
    actor: cluster-operator
    entities:
      - { entity: push-secret, effect: creates, facts: [Secret stores, Source, Data, Data to, Template, Update policy, Deletion policy, Refresh interval] }
      - { entity: cluster-push-secret, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product records the provisioned namespaces and marks the ClusterPushSecret ready
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-push-secret, effect: changes, facts: [Provisioned namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
---

# Provision a PushSecret into selected namespaces

## Trigger

A Cluster Operator wants Secrets from many namespaces pushed to a provider the same way.

## Outcome

Every selected namespace has the PushSecret, and the ClusterPushSecret lists them.

## Edge cases

- A namespace that already has a PushSecret of that name not created by the ClusterPushSecret is recorded as failed and left alone.
