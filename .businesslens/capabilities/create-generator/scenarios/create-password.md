---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a Password generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: password, effect: creates, facts: [Length, Digits, Symbols, Symbol characters, No uppercase, Allow repeat, Secret keys, Encoding, Prefix, Suffix] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a Password generator

## Trigger

A namespace needs a generator that produces random passwords.

## Outcome

The Password generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
