---
domain: core-resources
availability:
  - { place: kubernetes-api::namespace }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.Reconcile
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_secret.go
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_template.go
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/externalsecret_validator.go
  - kind: doc
    role: intent
    target: docs/api/externalsecret.md
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
  - kind: doc
    role: context
    target: docs/guides/templating.md
---

# Create an ExternalSecret

Declare which provider values a namespace needs and the Secret to write them
into. The Product checks the declaration as it is submitted, fetches the values
through the named store — or produces them with a generator — renders the
target from them and reports the outcome on the ExternalSecret.
