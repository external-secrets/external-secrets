---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret naming a store, the provider keys to fetch and the target to write, which the Product admits after checking it
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the requested values through the SecretStore
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: reads, facts: [Provider, Provider settings, Ready] }
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product renders the target Secret from the values and creates it, owned by the ExternalSecret
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, effect: creates, facts: [Data, Type, Labels and annotations, Owner] }
      - { entity: external-secret, effect: reads, facts: [Template, Target name] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready with reason SecretSynced
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Sync provider values into a new Secret

## Trigger

An Application Developer needs provider secrets available to a workload as a Kubernetes Secret.

## Outcome

The Secret exists with the fetched values, is owned by the ExternalSecret and
carries the Product's managed marker; the ExternalSecret is ready and records
when it synced.

## Decision points

### Where do the values come from?

Each entry of the ExternalSecret says how it brings values in.

- A single-value entry → its provider key, property or version becomes one key of the target.
- An extract entry → every property of one structured provider secret becomes a key.
- A find entry → every provider secret matching the name pattern, path or tags becomes a key, optionally rewritten.

### How is the target rendered?

The ExternalSecret may carry a template.

- No template → the fetched keys and values are written as they are.
- A template → type, labels, annotations and data are rendered from the values, merged with the fetched keys when the template asks for it.

## Edge cases

- Values are decoded as asked — Base64, Base64URL, automatically or not at all — before they are written.
- Keys that are not valid Secret keys are refused with a hint to rewrite or convert them.
- A value containing null bytes is kept or refused according to the entry's null-byte policy.
- With deletion policy Delete or Merge, a provider secret that does not exist yet is not an error.
- An ExternalSecret whose store names another controller class is left for that controller and gets no status from this one.
