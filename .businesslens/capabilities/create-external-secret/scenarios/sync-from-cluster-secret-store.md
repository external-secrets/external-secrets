---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret naming a cluster-wide store, which the Product admits
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The ClusterSecretStore admits the namespace it is used from
    kind: condition
    actor: application-developer
    entities:
      - { entity: cluster-secret-store, effect: reads, facts: [Namespace conditions] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the requested values through the ClusterSecretStore
    kind: product
    actor: application-developer
    entities:
      - { entity: cluster-secret-store, effect: reads, facts: [Provider, Provider settings, Ready] }
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product renders the target Secret and creates it, owned by the ExternalSecret
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, effect: creates, facts: [Data, Type, Labels and annotations, Owner] }
      - { entity: external-secret, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Sync through a ClusterSecretStore

## Trigger

An Application Developer uses the shared store a Cluster Operator published.

## Outcome

The Secret exists with the fetched values and the ExternalSecret is ready.

## Edge cases

- Credentials the ClusterSecretStore references without a namespace are read from the ExternalSecret's namespace.
- While the controller options do not have ClusterSecretStores processed, an ExternalSecret naming one is left alone and gets no status.
