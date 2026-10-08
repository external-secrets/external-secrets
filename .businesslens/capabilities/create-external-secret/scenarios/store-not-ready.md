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
  - text: The SecretStore is not ready and the flood gate is on
    kind: condition
    actor: application-developer
    entities:
      - { entity: secret-store, effect: reads, facts: [Ready] }
      - { entity: controller-options, effect: reads, facts: [Flood gate] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product does not call the provider and marks the ExternalSecret not ready with reason SecretSyncedError
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Store is not ready

## Trigger

The ExternalSecret names a store whose last validation failed.

## Outcome

No Secret is written; the ExternalSecret syncs once the store is ready again.
