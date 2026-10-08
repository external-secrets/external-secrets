---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer changes what an ExternalSecret fetches or how it renders its target, and the Product admits the change
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Data, Data from, Template] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the values again through the store
    kind: product
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product rewrites the keys it manages in the target Secret, dropping keys no longer produced
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

# Sync again after an edit

## Trigger

An Application Developer needs another key, a different template or another source.

## Outcome

The target reflects the new declaration at once, without waiting for the refresh interval.

## Edge cases

- An edit that breaks an admission rule is refused, as on creation.
- With refresh policy OnChange, provider changes reach the target only when the ExternalSecret itself changes, its labels and annotations included.
