---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes a SecretStore
    kind: actor
    actor: application-developer
    entities:
      - { entity: secret-store, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: PushSecrets with deletion policy Delete still push through the store
    kind: condition
    actor: application-developer
    entities:
      - { entity: push-secret, effect: reads, facts: [Deletion policy, Synced push secrets] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The Product keeps the SecretStore until none of them remains, then lets it go
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# SecretStore deletion waits for PushSecrets

## Trigger

A store is deleted while PushSecrets that remove their provider secrets on
deletion still push through it.

## Outcome

The SecretStore stays, marked for deletion, until those PushSecrets are deleted
and have cleaned up; then it is removed.
