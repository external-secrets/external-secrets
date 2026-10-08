---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a PushSecret whose store cannot write the entry
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: creates, facts: [Secret stores, Source, Data, Update policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product reads the source Secret
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, effect: reads, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product cannot write the entry and marks the PushSecret not ready with reason Errored, keeping the keys it did write
    kind: product
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Provider rejects the push

## Trigger

The store's provider is read-only, refuses the credentials' permissions, or the source lacks the named key.

## Outcome

The provider keeps what it had; the PushSecret reports the failure and the Product retries.

## Edge cases

- A missing source is reported the same way.
