---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes an ExternalSecret whose creation policy is Orphan, Merge, CreateOrMerge or None
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The target Secret stays with its last values
    kind: condition
    actor: application-developer
    entities:
      - { entity: secret, effect: reads, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Delete an ExternalSecret and keep its target

## Trigger

The Secret was created to outlive its ExternalSecret, or is managed by someone else.

## Outcome

The ExternalSecret is gone and its target keeps its last values; the Product no longer refreshes it.
