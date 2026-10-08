---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The provider no longer holds any requested value and the deletion policy is Delete
    kind: condition
    unattended: true
    entities:
      - { entity: external-secret, effect: reads, facts: [Deletion policy, Creation policy] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product deletes the target Secret it owns
    kind: product
    entities:
      - { entity: secret, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready with reason SecretDeleted
    kind: product
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Delete the target when the provider values are gone

## Trigger

Every value an ExternalSecret with deletion policy Delete fetches was deleted at the provider.

## Outcome

The target is gone and the ExternalSecret reports it deleted it; it recreates the target if the values return.
