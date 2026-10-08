---
kind: alternative
routes:
  cli: esoctl
steps:
  - text: The Application Developer runs the template command with a PushSecret file and a file of sample data
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: reads, facts: [Template] }
    contexts: { cli: { place: esoctl } }
  - text: The Product renders the template with the sample data and prints the resulting object
    kind: product
    entities: []
    contexts: { cli: { place: esoctl } }
---

# Render a PushSecret template

## Trigger

A developer wants to check what a PushSecret would write to a provider.

## Outcome

The developer sees the values the template would push.
