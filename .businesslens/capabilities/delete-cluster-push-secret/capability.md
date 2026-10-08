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

# Delete a ClusterPushSecret

Remove a ClusterPushSecret together with every PushSecret it provisioned.
