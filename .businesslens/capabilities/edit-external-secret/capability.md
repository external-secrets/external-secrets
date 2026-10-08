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
    target: apis/externalsecrets/v1/externalsecret_validator.go
---

# Edit an ExternalSecret

Change what an ExternalSecret fetches, how it renders its target or where it
writes it. The Product checks the change as it is submitted and syncs the
target again straight away, unless the ExternalSecret syncs only once.
