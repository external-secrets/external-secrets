---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a QuayAccessToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: quay-access-token, effect: creates, facts: [URL, Robot account, Service account] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a QuayAccessToken generator

## Trigger

A namespace needs a generator that exchanges a Kubernetes service account token for a Quay robot account token.

## Outcome

The QuayAccessToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
