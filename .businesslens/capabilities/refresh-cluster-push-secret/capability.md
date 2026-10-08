---
domain: core-resources
availability:
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/clusterpushsecret/clusterpushsecret_controller.go
  - kind: doc
    role: intent
    target: docs/api/clusterpushsecret.md
---

# Refresh a ClusterPushSecret

Keep a ClusterPushSecret's namespaces current as namespaces change and on its
refresh interval.
