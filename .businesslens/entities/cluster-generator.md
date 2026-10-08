---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_cluster.go
  - kind: code
    role: implementation
    target: runtime/esutils/resolvers/generator.go
  - kind: doc
    role: intent
    target: docs/api/generator/cluster.md
---

# ClusterGenerator

A cluster-wide generator any namespace can use. It holds the settings of exactly
one generator kind and acts, for the namespace using it, as if that generator
existed there. ClusterGenerators can be used only where the installation
includes them and processes them.

## Information kept

- **Generator kind** — which kind of generator it is, such as Password or ECRAuthorizationToken
- **Generator settings** — the settings of that kind of generator
