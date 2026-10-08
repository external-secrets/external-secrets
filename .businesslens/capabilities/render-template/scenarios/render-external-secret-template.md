---
kind: primary
routes:
  cli: esoctl
steps:
  - text: The Application Developer runs the template command with an ExternalSecret file and a file of sample data
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: reads, facts: [Template] }
    contexts: { cli: { place: esoctl } }
  - text: The Product renders the template with the sample data and prints the resulting object, or writes it to the chosen file
    kind: product
    entities: []
    contexts: { cli: { place: esoctl } }
---

# Render an ExternalSecret template

## Trigger

A developer wants to check a template without a cluster or a provider.

## Outcome

The developer sees exactly what the template would produce from that data.

## Edge cases

- Templates kept in ConfigMaps or Secrets are supplied as further files.
- A template that fails to render prints the error instead.
