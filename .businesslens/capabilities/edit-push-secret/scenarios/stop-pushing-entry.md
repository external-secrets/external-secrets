---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer removes an entry from a PushSecret whose deletion policy is Delete
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product deletes the provider secret the entry used to write
    kind: product
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product pushes the remaining entries and records the keys it now manages
    kind: product
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Stop pushing an entry

## Trigger

A value no longer belongs in the provider.

## Outcome

The provider no longer holds the removed entry's key, and the PushSecret no longer lists it.
