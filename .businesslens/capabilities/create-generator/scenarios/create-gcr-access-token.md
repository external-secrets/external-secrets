---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a GCRAccessToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: gcr-access-token, effect: creates, facts: [Project, Authentication] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a GCRAccessToken generator

## Trigger

A namespace needs a generator that issues a short-lived Google Container Registry access token.

## Outcome

The GCRAccessToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
