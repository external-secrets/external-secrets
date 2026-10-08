---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: A provisioned namespace no longer matches the ClusterPushSecret's selection
    kind: condition
    unattended: true
    entities:
      - { entity: cluster-push-secret, effect: reads, facts: [Namespace selection, Provisioned namespaces] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product deletes the PushSecret it created there
    kind: product
    entities:
      - { entity: push-secret, effect: removes }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product removes the namespace from the provisioned namespaces
    kind: product
    entities:
      - { entity: cluster-push-secret, effect: changes, facts: [Provisioned namespaces] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
---

# Withdraw from a namespace that no longer matches

## Trigger

A namespace loses the labels the ClusterPushSecret selects.

## Outcome

The namespace no longer has the provisioned PushSecret; what happens to its provider secrets follows its deletion policy.
