---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret with creation policy Orphan or CreateOrMerge, which the Product admits
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
  - text: The Product renders the target Secret and creates it without an owner
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, effect: creates, facts: [Data, Type, Labels and annotations] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Create a Secret the ExternalSecret does not own

## Trigger

The Secret must outlive the ExternalSecret that creates it.

## Outcome

The Secret exists with the fetched values and no owner; deleting the ExternalSecret leaves it in place.

## Edge cases

- With Orphan, a Secret changed by hand is overwritten only at the next refresh, never as soon as it changes.
