---
appliesTo:
  - { type: entity, id: external-secret, facts: [Ready] }
  - { type: capability, id: create-external-secret }
  - { type: capability, id: refresh-external-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.markAsFailed
  - kind: code
    role: implementation
    target: pkg/controllers/util/statuserr.go
---

# An ExternalSecret's status never repeats provider error details

When a sync fails, the ExternalSecret's Ready condition says what failed in general terms and adds detail only from errors the Product knows to be free of secret values.

## Rationale

Provider errors can echo the values they were handling, and an ExternalSecret's status is readable by more people than its target.
