---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator changes the namespace conditions of a ClusterSecretStore, which the Product admits after checking them
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-secret-store, effect: changes, facts: [Namespace conditions] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
  - text: The Product connects to the provider again and records whether the ClusterSecretStore is ready
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-secret-store, effect: changes, facts: [Ready, Capabilities] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
---

# Change which namespaces a ClusterSecretStore serves

## Trigger

A Cluster Operator opens a shared store to more teams, or closes it to some.

## Outcome

The ClusterSecretStore serves exactly the namespaces its new conditions admit;
ExternalSecrets elsewhere fail to sync from it at their next refresh.

## Edge cases

- A namespace pattern that is not a valid regular expression is refused.
