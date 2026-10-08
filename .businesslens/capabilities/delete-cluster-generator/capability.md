---
domain: generators
availability:
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_cluster.go
---

# Delete a ClusterGenerator

Remove a ClusterGenerator from the cluster. Values it already produced stay
where they were written.
