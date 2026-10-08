---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret with creation policy Merge or CreateOrMerge for a target that already exists, which the Product admits
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
  - text: The Product writes the fetched keys into the existing Secret, keeping keys other writers own, without taking ownership
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, effect: changes, facts: [Data, Labels and annotations] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Merge provider values into an existing Secret

## Trigger

A Secret is managed partly by someone else and the provider supplies only some of its keys.

## Outcome

The existing Secret holds the provider values beside the keys others manage; it
has no owner, so deleting the ExternalSecret leaves it in place.

## Edge cases

- Several ExternalSecrets may merge into one Secret as long as their keys do not collide; colliding keys keep changing.
