---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ECRAuthorizationToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: ecr-authorization-token, effect: creates, facts: [Region, Authentication, Role, Scope] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create an ECRAuthorizationToken generator

## Trigger

A namespace needs a generator that issues an AWS Elastic Container Registry authorization token for pulling and pushing images.

## Outcome

The ECRAuthorizationToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
