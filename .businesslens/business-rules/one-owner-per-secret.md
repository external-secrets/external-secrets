---
appliesTo:
  - { type: capability, id: create-external-secret }
  - { type: capability, id: refresh-external-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.applyOwnership
---

# A Secret belongs to at most one ExternalSecret

When a target Secret is already owned by another ExternalSecret, the Product leaves it unchanged and reports the conflict instead of writing it.

## Rationale

Two owners would overwrite each other's values on every sync.
