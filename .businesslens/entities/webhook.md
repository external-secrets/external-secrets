---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_webhook.go
  - kind: code
    role: implementation
    target: generators/v1/webhook/webhook.go
  - kind: doc
    role: intent
    target: docs/api/generator/webhook.md
---

# Webhook

A generator that calls an HTTP endpoint and returns values taken from its response.

## Information kept

- **URL** — the endpoint to call
- **Method** — the HTTP method
- **Headers** — the request headers
- **Body** — the request body
- **Authentication** — how the call authenticates
- **Timeout** — how long to wait for a response
- **Result** — the path to the values in the response
- **Secrets** — Secrets whose values may be used in the request
- **CA** — the certificate authority to trust
