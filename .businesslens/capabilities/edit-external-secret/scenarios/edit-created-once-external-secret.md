---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer changes what an ExternalSecret fetches
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Data] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The ExternalSecret has refresh policy CreatedOnce and has already synced
    kind: condition
    actor: application-developer
    entities:
      - { entity: external-secret, effect: reads, facts: [Refresh policy, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Edit an ExternalSecret that syncs only once

## Trigger

A developer edits an ExternalSecret that was declared to sync a single time.

## Outcome

Nothing is fetched and the target keeps its values; deleting and recreating the
ExternalSecret is what syncs it again.
