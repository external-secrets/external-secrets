---
appliesTo:
  - { type: capability, id: create-external-secret }
  - { type: capability, id: refresh-external-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_secret.go#Reconciler.GetProviderSecretData
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.Reconcile
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
---

# What happens when the provider no longer holds a value follows the deletion policy

With deletion policy Retain, a requested value the provider does not hold is an
error and the target keeps its last values. With Delete or Merge a missing
value is skipped; once the provider holds none of them, Delete deletes the
target the ExternalSecret owns and Merge removes only the keys the
ExternalSecret wrote. An unreachable provider or refused access is always an
error, never a missing value.
