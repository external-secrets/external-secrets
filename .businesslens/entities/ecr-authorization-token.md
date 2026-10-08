---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_ecr.go
  - kind: code
    role: implementation
    target: generators/v1/ecr/ecr.go
  - kind: doc
    role: intent
    target: docs/api/generator/ecr.md
---

# ECRAuthorizationToken

A generator that issues an AWS Elastic Container Registry authorization token for pulling and pushing images.

## Information kept

- **Region** — the AWS region of the registry
- **Authentication** — how the Product authenticates to AWS
- **Role** — an IAM role to assume first
- **Scope** — private or public registry
