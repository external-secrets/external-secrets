---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_fake.go
  - kind: code
    role: implementation
    target: generators/v1/fake/fake.go
  - kind: doc
    role: intent
    target: docs/api/generator/fake.md
---

# Fake

A generator that returns fixed values, for trying out and testing generators.

## Information kept

- **Data** — the keys and values it returns
- **Controller class** — which controller instance may use the generator
