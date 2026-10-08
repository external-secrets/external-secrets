---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a GitlabDeployToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: gitlab-deploy-token, effect: creates, facts: [URL, Project or group, Token name, Scopes, Expiry, Username, Authentication] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a GitlabDeployToken generator

## Trigger

A namespace needs a generator that creates a GitLab deploy token for a project or group.

## Outcome

The GitlabDeployToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
