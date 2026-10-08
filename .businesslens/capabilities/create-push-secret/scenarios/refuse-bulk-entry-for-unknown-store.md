---
kind: validation
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a PushSecret whose bulk entry names a store missing from its store list
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: creates, facts: [Secret stores, Source, Data to, Update policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product writes nothing and marks the PushSecret not ready, naming the entry
    kind: product
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Refuse a bulk entry for a store it does not push to

## Trigger

A bulk entry names, or selects by labels, a store the PushSecret does not list.

## Outcome

Nothing is pushed until the entry or the store list is corrected.
