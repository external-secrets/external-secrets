---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer removes an entry from a PushSecret whose deletion policy is None
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product pushes the remaining entries and records the keys it now manages
    kind: product
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Stop pushing an entry but keep it at the provider

## Trigger

A value should stop being updated from the cluster but stay in the provider.

## Outcome

The provider keeps the last value of the removed entry; the Product no longer updates it.
