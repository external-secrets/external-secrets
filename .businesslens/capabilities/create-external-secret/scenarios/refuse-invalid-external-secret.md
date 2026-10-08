---
kind: validation
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer submits a declaration that breaks one of the admission rules
    kind: actor
    actor: application-developer
    entities: []
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product refuses it, naming every rule it breaks
    kind: condition
    actor: application-developer
    entities: []
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Refuse an invalid ExternalSecret

## Trigger

The declaration fetches nothing; mixes extract, find and generator in one
entry; has a source reference naming neither a store nor a generator; asks to delete a Secret it would not own; uses deletion policy Merge with
creation policy None; renders a bootstrap token or a service-account token bound
to an account; templates into fields other than data, labels and annotations of
a Secret; or repeats a target key while retaining.

## Outcome

Nothing is created and the developer sees every reason.
