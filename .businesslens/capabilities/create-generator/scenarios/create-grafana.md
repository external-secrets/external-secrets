---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a Grafana generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: grafana, effect: creates, facts: [URL, Authentication, Service account] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a Grafana generator

## Trigger

A namespace needs a generator that creates a Grafana service account token.

## Outcome

The Grafana generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
