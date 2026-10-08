---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a GithubAccessToken generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: github-access-token, effect: creates, facts: [URL, App, Installation, Repositories, Permissions, Authentication] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a GithubAccessToken generator

## Trigger

A namespace needs a generator that issues a GitHub App installation access token.

## Outcome

The GithubAccessToken generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
