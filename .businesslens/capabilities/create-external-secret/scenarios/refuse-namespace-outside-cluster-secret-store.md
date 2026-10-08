---
kind: validation
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret naming a cluster-wide store, which the Product admits
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The ClusterSecretStore's namespace conditions do not admit the namespace it is used from
    kind: condition
    actor: application-developer
    entities:
      - { entity: cluster-secret-store, effect: reads, facts: [Namespace conditions] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret not ready with reason SecretSyncedError
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Refuse a namespace a ClusterSecretStore does not admit

## Trigger

An ExternalSecret names a ClusterSecretStore from a namespace the store does not serve.

## Outcome

No Secret is written and the ExternalSecret reports that it could not get data from the provider.
