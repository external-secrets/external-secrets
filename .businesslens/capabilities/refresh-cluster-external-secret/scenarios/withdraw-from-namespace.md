---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: A provisioned namespace no longer matches the ClusterExternalSecret's selection
    kind: condition
    unattended: true
    entities:
      - { entity: cluster-external-secret, effect: reads, facts: [Namespace selection, Provisioned namespaces] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product deletes the ExternalSecret it created there
    kind: product
    entities:
      - { entity: external-secret, effect: removes }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product removes the namespace from the provisioned namespaces
    kind: product
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [Provisioned namespaces] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
---

# Withdraw from a namespace that no longer matches

## Trigger

A namespace loses the labels the ClusterExternalSecret selects, or the selection changes.

## Outcome

The namespace no longer has the provisioned ExternalSecret.

## Edge cases

- Changing the ExternalSecret name deletes the ExternalSecrets provisioned under the old name.
