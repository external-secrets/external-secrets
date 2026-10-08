---
domain: generators
availability:
  - { place: kubernetes-api::namespace }
references:
  - kind: code
    role: implementation
    target: runtime/esutils/resolvers/generator.go
  - kind: doc
    role: intent
    target: docs/guides/generator.md
---

# Delete a generator

Remove a generator from a namespace. Values it already produced stay where they
were written.
