---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a Fake generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: fake, effect: creates, facts: [Data, Controller class] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a Fake generator

## Trigger

A namespace needs a generator that returns fixed values, for trying out and testing generators.

## Outcome

The Fake generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
