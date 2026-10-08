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

# Edit a generator

Change how a generator in a namespace produces values. Nothing is generated on
the edit itself: the ExternalSecrets and PushSecrets that name the generator
use the new settings the next time they refresh.
