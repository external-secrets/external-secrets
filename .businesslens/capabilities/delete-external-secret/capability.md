---
domain: core-resources
availability:
  - { place: kubernetes-api::namespace }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.Reconcile
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.cleanupManagedSecrets
---

# Delete an ExternalSecret

Remove an ExternalSecret from a namespace. A target the ExternalSecret owns goes
with it; any other target stays.
