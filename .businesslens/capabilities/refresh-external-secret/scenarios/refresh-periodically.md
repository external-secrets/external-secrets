---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The ExternalSecret's refresh interval has passed and no sync window holds it back
    kind: condition
    unattended: true
    entities:
      - { entity: external-secret, effect: reads, facts: [Refresh policy, Refresh interval, Sync windows, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the values again through the store
    kind: product
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product updates the target Secret when the values changed
    kind: product
    entities:
      - { entity: secret, effect: changes, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product records the refresh on the ExternalSecret
    kind: product
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Refresh on the refresh interval

## Trigger

The refresh interval of an ExternalSecret with refresh policy Periodic passes.

## Outcome

The target holds the provider's current values and the ExternalSecret records when it last synced.

## Edge cases

- A refresh interval of zero syncs only once, like CreatedOnce.
- An immutable target keeps its data once it exists.
