---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer changes the length and symbols of a Password generator
    kind: actor
    actor: application-developer
    entities:
      - { entity: password, effect: changes, facts: [Length, Symbols] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Change a generator's settings

## Trigger

Generated values must follow a new policy, such as longer passwords.

## Outcome

The generator keeps the new settings; values produced from now on follow them,
while values already written stay as they are until the next refresh.

## Edge cases

- Every generator kind is edited the same way.
