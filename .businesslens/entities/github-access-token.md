---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_github.go
  - kind: code
    role: implementation
    target: generators/v1/github/github.go
  - kind: doc
    role: intent
    target: docs/api/generator/github.md
---

# GithubAccessToken

A generator that issues a GitHub App installation access token.

## Information kept

- **URL** — the GitHub API to call
- **App** — the GitHub App
- **Installation** — the App's installation
- **Repositories** — the repositories the token is limited to
- **Permissions** — the permissions the token is limited to
- **Authentication** — the App's private key
