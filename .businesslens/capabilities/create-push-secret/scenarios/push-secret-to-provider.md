---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a PushSecret naming its source, the stores to write to and the provider key for each entry
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
  - text: The Product writes each entry to the provider through each store
    kind: product
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: creates, facts: [Key, Value, Metadata] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product records the pushed keys and marks the PushSecret ready with reason Synced
    kind: product
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Push values to a provider

## Trigger

A value created in the cluster, such as a certificate or a generated credential, must be available in a provider.

## Outcome

Each provider holds the pushed values under the given keys, and the PushSecret lists which keys it wrote to which store.

## Edge cases

- Stores can be named one by one or selected by labels; stores being deleted, or belonging to another controller class, are skipped.
- Bulk entries push every matching source key at once, renamed by rewrites, to the stores they select; a single entry for the same source key wins over a bulk one.
- A template renders the pushed values from the source before they are written.
- All Secrets matching a label selector can be the source, each pushed in turn.
