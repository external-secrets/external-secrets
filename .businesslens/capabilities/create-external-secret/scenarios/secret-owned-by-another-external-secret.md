---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret whose target name another one already uses, which the Product admits
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the requested values through the SecretStore
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: reads, facts: [Provider, Provider settings, Ready] }
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The target Secret already has a different owner
    kind: condition
    actor: application-developer
    entities:
      - { entity: secret, effect: reads, facts: [Owner] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product leaves the Secret unchanged and marks the ExternalSecret not ready with reason SecretOwnedByOther
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready] }
      - { entity: secret, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Target Secret belongs to another ExternalSecret

## Trigger

Two ExternalSecrets in one namespace name the same target.

## Outcome

The Secret keeps its owner and values; the second ExternalSecret reports the conflict and is not retried until it changes.
