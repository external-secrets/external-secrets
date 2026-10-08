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

# Edit a ClusterPushSecret

Change what a ClusterPushSecret provisions, under which name, or where. The
Product applies the change to every PushSecret it provisioned straight away.
