---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes a Password generator
    kind: actor
    actor: application-developer
    entities:
      - { entity: password, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Delete a Password generator

## Trigger

A namespace no longer needs values produced this way.

## Outcome

The generator is gone; ExternalSecrets and PushSecrets still naming it fail at
their next refresh until they name another source.

## Edge cases

- Every generator kind is deleted the same way.
