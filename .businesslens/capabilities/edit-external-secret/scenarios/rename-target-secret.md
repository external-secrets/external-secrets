---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer changes the target name of an ExternalSecret that owns its target
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Target name] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the values again through the store
    kind: product
    actor: application-developer
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product creates the Secret under the new name and deletes the Secret it owned under the old name
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, as: new-secret, effect: creates, facts: [Data, Type, Labels and annotations, Owner] }
      - { entity: secret, as: previous-secret, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Rename the target Secret

## Trigger

The workload expects its Secret under another name.

## Outcome

Only the newly named Secret remains; the one left behind under the old name is cleaned up.
