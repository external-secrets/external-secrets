---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes a PushSecret whose deletion policy is Delete
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product cannot delete a provider secret, keeps the PushSecret and marks it not ready
    kind: product
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key] }
      - { entity: push-secret, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Provider refuses to delete what was pushed

## Trigger

A provider refuses or fails a deletion.

## Outcome

The PushSecret stays, marked for deletion, and the Product retries until every pushed provider secret is deleted.
