---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ACRAccessToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: acr-access-token, effect: creates, facts: [Registry, Tenant, Authentication, Scope, Environment] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create an ACRAccessToken generator

## Trigger

A namespace needs a generator that issues a short-lived access token for an Azure Container Registry.

## Outcome

The ACRAccessToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
