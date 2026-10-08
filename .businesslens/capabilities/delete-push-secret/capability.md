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

# Delete a PushSecret

Remove a PushSecret from a namespace. With deletion policy Delete the Product
first deletes every provider secret it pushed; with None they stay.
