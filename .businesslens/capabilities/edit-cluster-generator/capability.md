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
---

# Edit a ClusterGenerator

Change the settings of a ClusterGenerator. ExternalSecrets and PushSecrets in
every namespace that name it use the new settings the next time they refresh.
