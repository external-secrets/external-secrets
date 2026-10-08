---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_quay.go
  - kind: code
    role: implementation
    target: generators/v1/quay/quay.go
  - kind: doc
    role: intent
    target: docs/api/generator/quay.md
---

# QuayAccessToken

A generator that exchanges a Kubernetes service account token for a Quay robot account token.

## Information kept

- **URL** — the Quay instance
- **Robot account** — the robot account the token belongs to
- **Service account** — the Kubernetes service account whose token is exchanged
