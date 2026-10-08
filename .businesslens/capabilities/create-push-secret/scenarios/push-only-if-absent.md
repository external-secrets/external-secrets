---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a PushSecret with update policy IfNotExists
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: creates, facts: [Secret stores, Source, Data, Update policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The provider already holds the key the entry writes to
    kind: condition
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product leaves the provider value unchanged and marks the PushSecret ready, noting existing values were kept
    kind: product
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Leave existing provider secrets alone

## Trigger

A value must be seeded once at the provider and never overwritten from the cluster.

## Outcome

The provider keeps its existing value; keys that did not exist yet are written.
