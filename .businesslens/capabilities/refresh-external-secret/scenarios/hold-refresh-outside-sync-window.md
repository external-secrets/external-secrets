---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The refresh interval has passed but a deny window is open, or no allow window is
    kind: condition
    unattended: true
    entities:
      - { entity: external-secret, effect: reads, facts: [Sync windows, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product leaves the target as it is until a window permits the refresh
    kind: condition
    entities: []
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Hold a refresh outside the sync windows

## Trigger

A periodic refresh falls due while the ExternalSecret's sync windows forbid it.

## Outcome

The target keeps its values; the refresh happens once the windows allow it.
Edits and the force-sync annotation are not held back by sync windows.
