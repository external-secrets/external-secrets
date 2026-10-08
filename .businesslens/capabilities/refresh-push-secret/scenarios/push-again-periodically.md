---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The PushSecret's refresh interval has passed
    kind: condition
    unattended: true
    entities:
      - { entity: push-secret, effect: reads, facts: [Refresh interval, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product reads the source Secret again
    kind: product
    entities:
      - { entity: secret, effect: reads, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product writes the current values to the provider
    kind: product
    entities:
      - { entity: provider-secret, effect: changes, facts: [Value] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product records the push on the PushSecret
    kind: product
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Push again on the refresh interval

## Trigger

A PushSecret's refresh interval passes.

## Outcome

The provider holds the source's current values; changes made directly at the provider are overwritten unless the update policy is IfNotExists.
