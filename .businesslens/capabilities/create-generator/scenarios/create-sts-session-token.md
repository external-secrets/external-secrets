---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an STSSessionToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: sts-session-token, effect: creates, facts: [Region, Authentication, Role, Request parameters] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create an STSSessionToken generator

## Trigger

A namespace needs a generator that issues temporary AWS credentials from AWS STS.

## Outcome

The STSSessionToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
