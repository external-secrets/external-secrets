---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The source named by the PushSecret no longer exists and its deletion policy is Delete
    kind: condition
    unattended: true
    entities:
      - { entity: push-secret, effect: reads, facts: [Source, Deletion policy, Synced push secrets] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product deletes every provider secret it pushed
    kind: product
    entities:
      - { entity: provider-secret, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product clears the pushed keys and marks the PushSecret not ready with reason SourceDeleted
    kind: product
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Clean up after the source is deleted

## Trigger

The Secret a PushSecret with deletion policy Delete pushes from is deleted.

## Outcome

The provider no longer holds the values; the PushSecret stays and reports that its source is gone.
