---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a CloudsmithAccessToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: cloudsmith-access-token, effect: creates, facts: [API URL, Organisation, Service, Service account] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a CloudsmithAccessToken generator

## Trigger

A namespace needs a generator that exchanges a Kubernetes service account token for a short-lived Cloudsmith access token.

## Outcome

The CloudsmithAccessToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
