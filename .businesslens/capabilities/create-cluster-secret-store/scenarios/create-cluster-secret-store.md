---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator applies a ClusterSecretStore with its provider, credentials and the namespaces allowed to use it, which the Product admits after checking its settings
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-secret-store, effect: creates, facts: [Provider, Provider settings, Namespace conditions, Controller class, Refresh interval, Retry settings] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
  - text: The Product connects to the provider and validates the connection
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-secret-store, effect: changes, facts: [Ready, Capabilities] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
  - text: The ClusterSecretStore reports that it is ready and which namespaces may use it
    kind: condition
    actor: cluster-operator
    entities:
      - { entity: cluster-secret-store, effect: reads, facts: [Ready, Capabilities, Namespace conditions] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
---

# Publish a ready ClusterSecretStore

## Trigger

A Cluster Operator wants to give many namespaces one managed way into a
provider.

## Outcome

The ClusterSecretStore is ready; ExternalSecrets and PushSecrets in every
admitted namespace can name it.

## Edge cases

- With no namespace conditions, every namespace may use the store.
- A store the provider rejects is not ready with reason InvalidProviderConfig, as for a SecretStore.
