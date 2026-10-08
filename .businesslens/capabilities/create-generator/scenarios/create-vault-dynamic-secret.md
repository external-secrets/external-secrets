---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a VaultDynamicSecret generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: vault-dynamic-secret, effect: creates, facts: [Provider settings, Path, Method, Parameters, Result type, Allow empty response, Retry settings, Controller class] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a VaultDynamicSecret generator

## Trigger

A namespace needs a generator that reads dynamic credentials from a HashiCorp Vault secrets engine.

## Outcome

The VaultDynamicSecret generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
