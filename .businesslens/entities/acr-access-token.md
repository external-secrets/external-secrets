---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_acr.go
  - kind: code
    role: implementation
    target: generators/v1/acr/acr.go
  - kind: doc
    role: intent
    target: docs/api/generator/acr.md
---

# ACRAccessToken

A generator that issues a short-lived access token for an Azure Container Registry.

## Information kept

- **Registry** — the registry the token is for
- **Tenant** — the Azure tenant to authenticate in
- **Authentication** — a service principal, managed identity or workload identity
- **Scope** — the repositories and actions the token allows; a refresh token when empty
- **Environment** — the Azure cloud the registry lives in
