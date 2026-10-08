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

# Create a ClusterPushSecret

Provision the same PushSecret into every namespace a Cluster Operator selects by
labels. The Product provisions only while its controller options have
ClusterPushSecrets processed.
