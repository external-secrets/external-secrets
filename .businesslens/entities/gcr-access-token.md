---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_gcr.go
  - kind: code
    role: implementation
    target: generators/v1/gcr/gcr.go
  - kind: doc
    role: intent
    target: docs/api/generator/gcr.md
---

# GCRAccessToken

A generator that issues a short-lived Google Container Registry access token.

## Information kept

- **Project** — the Google Cloud project
- **Authentication** — a service account key, workload identity or workload identity federation
