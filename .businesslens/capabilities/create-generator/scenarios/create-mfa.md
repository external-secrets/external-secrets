---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an MFA generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: mfa, effect: creates, facts: [Seed, Length, Time period, Algorithm, When] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create an MFA generator

## Trigger

A namespace needs a generator that produces a time-based one-time password from a stored seed.

## Outcome

The MFA generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
