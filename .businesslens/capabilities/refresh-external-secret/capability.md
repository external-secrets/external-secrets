---
domain: core-resources
availability:
  - { place: kubernetes-api::namespace }
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.Reconcile
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#shouldRefresh
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#isPeriodicRefreshAllowedByWindows
  - kind: doc
    role: intent
    target: docs/api/externalsecret.md
---

# Refresh an ExternalSecret

Keep a target in step with its provider. The Product refreshes each
ExternalSecret on its refresh interval within its sync windows, whenever the
target it manages is changed or deleted behind its back, and whenever someone
asks for it with the force-sync annotation, on the ExternalSecret itself or on
the ClusterExternalSecret that provisioned it. What happens when the provider no
longer holds the values follows the deletion policy.
