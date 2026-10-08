---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a PushSecret whose source is a generator
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: creates, facts: [Secret stores, Source, Data, Update policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product generates a value with the Password generator's settings
    kind: product
    actor: application-developer
    entities:
      - { entity: password, effect: reads, facts: [Length, Digits, Symbols, Symbol characters, No uppercase, Allow repeat, Secret keys, Encoding, Prefix, Suffix] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product writes the generated value to the provider through each store
    kind: product
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: creates, facts: [Key, Value] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
  - text: The Product records the pushed keys and marks the PushSecret ready
    kind: product
    actor: application-developer
    entities:
      - { entity: push-secret, effect: changes, facts: [Synced push secrets, Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Push a generated value

## Trigger

A credential should be created by the cluster and kept in the provider.

## Outcome

The provider holds the generated value; any generator kind or ClusterGenerator can be the source.
