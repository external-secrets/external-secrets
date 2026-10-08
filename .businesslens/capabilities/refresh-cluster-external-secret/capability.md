---
domain: core-resources
availability:
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/clusterexternalsecret/clusterexternalsecret_controller.go
  - kind: doc
    role: intent
    target: docs/api/clusterexternalsecret.md
---

# Refresh a ClusterExternalSecret

Keep a ClusterExternalSecret's namespaces current. Whenever a namespace is
created, deleted or relabelled, and on the ClusterExternalSecret's refresh
interval, the Product provisions it where it now applies and removes it where it
no longer does.
