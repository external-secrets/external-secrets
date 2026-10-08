---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_sts.go
  - kind: code
    role: implementation
    target: generators/v1/sts/sts.go
  - kind: doc
    role: intent
    target: docs/api/generator/sts.md
---

# STSSessionToken

A generator that issues temporary AWS credentials from AWS STS.

## Information kept

- **Region** — the AWS region
- **Authentication** — how the Product authenticates to AWS
- **Role** — an IAM role to assume first
- **Request parameters** — session duration, serial number and token code for the request
