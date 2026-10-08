---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The provider no longer holds any requested value and the deletion policy is Merge
    kind: condition
    unattended: true
    entities:
      - { entity: external-secret, effect: reads, facts: [Deletion policy] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product removes the keys it managed from the target Secret and keeps the Secret
    kind: product
    entities:
      - { entity: secret, effect: changes, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product records the refresh on the ExternalSecret
    kind: product
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Remove keys when the provider values are gone

## Trigger

Every value an ExternalSecret with deletion policy Merge fetches was deleted at the provider.

## Outcome

The target stays without the keys the ExternalSecret managed.
