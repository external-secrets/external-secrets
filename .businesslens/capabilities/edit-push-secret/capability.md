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

# Edit a PushSecret

Change what a PushSecret pushes. The Product pushes again at once; with deletion
policy Delete it also deletes the provider secrets the PushSecret no longer
pushes.
