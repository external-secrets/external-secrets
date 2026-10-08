---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a Webhook generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: webhook, effect: creates, facts: [URL, Method, Headers, Body, Authentication, Timeout, Result, Secrets, CA] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a Webhook generator

## Trigger

A namespace needs a generator that calls an HTTP endpoint and returns values taken from its response.

## Outcome

The Webhook generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
