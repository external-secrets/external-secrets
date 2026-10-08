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
  - kind: doc
    role: intent
    target: docs/guides/pushsecret-datato.md
---

# Create a PushSecret

Declare which values of a namespace's Secrets — or of a generator — should be
written into which providers. The Product writes them through the named stores
and keeps track of every provider key it wrote. The Product pushes only while
its controller options have PushSecrets processed.
