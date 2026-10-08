---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret naming a store, the provider keys to fetch and the target to write, which the Product admits after checking it
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product cannot fetch the requested values from the provider
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: reads, facts: [Provider, Provider settings, Ready] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret not ready with reason SecretSyncedError and records a warning event
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Provider cannot supply the values

## Trigger

The provider is unreachable, refuses access, or does not hold a requested key while the deletion policy is Retain.

## Outcome

No Secret is written; the ExternalSecret's status says the data could not be
fetched without repeating the provider's error, and the Product retries.
