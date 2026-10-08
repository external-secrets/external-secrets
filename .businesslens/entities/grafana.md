---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_grafana.go
  - kind: code
    role: implementation
    target: generators/v1/grafana/grafana.go
  - kind: doc
    role: intent
    target: docs/api/generator/grafana.md
---

# Grafana

A generator that creates a Grafana service account token.

## Information kept

- **URL** — the Grafana instance
- **Authentication** — the credentials used to manage service accounts
- **Service account** — the service account, and its role, the token belongs to
