---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: A requested value no longer exists at the provider and the deletion policy is Retain
    kind: condition
    unattended: true
    entities:
      - { entity: external-secret, effect: reads, facts: [Deletion policy] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product keeps the target as it is and marks the ExternalSecret not ready with reason SecretSyncedError
    kind: product
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Keep the target when the provider values are gone

## Trigger

A value an ExternalSecret with deletion policy Retain fetches was deleted at the provider.

## Outcome

The target keeps its last values and the ExternalSecret reports the error until the value returns or the declaration changes.
