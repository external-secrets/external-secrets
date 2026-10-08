---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_gitlab.go
  - kind: code
    role: implementation
    target: generators/v1/gitlab/gitlab.go
  - kind: doc
    role: intent
    target: docs/api/generator/gitlab.md
---

# GitlabDeployToken

A generator that creates a GitLab deploy token for a project or group.

## Information kept

- **URL** — the GitLab instance
- **Project or group** — where the deploy token is created
- **Token name** — the deploy token's name
- **Scopes** — what the token allows
- **Expiry** — when the token expires
- **Username** — the token's username
- **Authentication** — the GitLab access token used to create it
