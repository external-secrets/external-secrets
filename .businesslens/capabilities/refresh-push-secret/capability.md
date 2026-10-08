---
domain: core-resources
availability:
  - { place: kubernetes-api::namespace }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/pushsecret/pushsecret_controller.go#Reconciler.Reconcile
  - kind: doc
    role: intent
    target: docs/api/pushsecret.md
  - kind: doc
    role: intent
    target: docs/guides/pushsecrets.md
---

# Refresh a PushSecret

Keep provider secrets in step with their source. The Product pushes each
PushSecret again on its refresh interval and whenever it changes, and cleans up
after a source that is gone.
