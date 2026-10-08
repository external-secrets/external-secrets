---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer sets a new force-sync annotation on the ExternalSecret
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Force sync] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the values again through the store
    kind: product
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product updates the target Secret when the values changed
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, effect: changes, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product records the refresh on the ExternalSecret
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Force a refresh

## Trigger

A developer rotated a value at the provider and cannot wait for the next interval.

## Outcome

The target holds the provider's current values straight away.

## Edge cases

- With refresh policy CreatedOnce an annotation change syncs nothing.
