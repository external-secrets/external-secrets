---
domain: generators
availability:
  - { place: kubernetes-api::cluster }
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

# Create a ClusterGenerator

Publish one generator, of any kind, that every namespace can use without a copy
of its own. When an ExternalSecret or PushSecret names it, the Product uses its
settings as if the generator existed in that namespace. ClusterGenerators can be
used only where the installation includes and processes them.
