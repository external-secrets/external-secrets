---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a UUID generator
    kind: actor
    actor: application-developer
    entities:
      - { entity: uuid, effect: creates, facts: [Name] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a UUID generator

## Trigger

A namespace needs a generator that produces a random UUID. It takes no settings.

## Outcome

The UUID generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
