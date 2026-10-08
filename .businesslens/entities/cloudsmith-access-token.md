---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_cloudsmith.go
  - kind: code
    role: implementation
    target: generators/v1/cloudsmith/cloudsmith.go
  - kind: doc
    role: intent
    target: docs/api/generator/cloudsmith.md
---

# CloudsmithAccessToken

A generator that exchanges a Kubernetes service account token for a short-lived Cloudsmith access token.

## Information kept

- **API URL** — the Cloudsmith API to call
- **Organisation** — the Cloudsmith organisation
- **Service** — the Cloudsmith service whose token is issued
- **Service account** — the Kubernetes service account whose token is exchanged
