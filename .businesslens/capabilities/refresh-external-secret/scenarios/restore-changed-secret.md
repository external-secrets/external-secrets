---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: Someone changed or deleted the data of a target Secret the Product manages
    kind: condition
    unattended: true
    entities:
      - { entity: secret, effect: reads, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the values again through the store
    kind: product
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product renders the Secret again from the provider values
    kind: product
    entities:
      - { entity: secret, effect: changes, facts: [Data, Labels and annotations] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product records the refresh on the ExternalSecret
    kind: product
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Restore a target changed behind the Product's back

## Trigger

A person or another controller edits the data of a managed target.

## Outcome

The target holds the provider's values again; manual changes are not kept.

## Edge cases

- A deleted target is recreated at once with creation policy Owner or CreateOrMerge, at the next periodic refresh with Orphan, and never with Merge.
- With creation policy Orphan, a changed target is overwritten only at the next refresh.
