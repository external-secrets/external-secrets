---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret whose data comes from a generator, which the Product admits
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data from, Target name, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product generates new values with the Password generator's settings
    kind: product
    actor: application-developer
    entities:
      - { entity: password, effect: reads, facts: [Length, Digits, Symbols, Symbol characters, No uppercase, Allow repeat, Secret keys, Encoding, Prefix, Suffix] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product creates the target Secret from the generated values, owned by the ExternalSecret
    kind: product
    actor: application-developer
    entities:
      - { entity: secret, effect: creates, facts: [Data, Type, Labels and annotations, Owner] }
      - { entity: external-secret, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Fill a Secret from a generator

## Trigger

An application needs a value no provider holds, such as a fresh database password.

## Outcome

The Secret holds newly generated values; each later refresh generates new
ones.

## Edge cases

- Any generator kind, or a ClusterGenerator, can be named the same way.
- A generator naming another controller class is left for that controller.
- A generator that does not exist, or a ClusterGenerator whose kind has no matching settings, fails the sync instead of counting as missing provider values.
