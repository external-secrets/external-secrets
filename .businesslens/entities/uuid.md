---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_uuid.go
  - kind: code
    role: implementation
    target: generators/v1/uuid/uuid.go
  - kind: doc
    role: intent
    target: docs/api/generator/uuid.md
---

# UUID

A generator that produces a random UUID. It takes no settings.

## Information kept

- **Name** — the name ExternalSecrets and PushSecrets use to refer to it
