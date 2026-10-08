---
kind: validation
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer submits store settings that are incomplete or contradictory
    kind: actor
    actor: application-developer
    entities: []
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The Product refuses it, naming the problem
    kind: condition
    actor: application-developer
    entities: []
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# Refuse an invalid SecretStore

## Trigger

The submitted store names no provider, more than one, or one the Product does
not include; has a refresh interval that is not a duration; or carries settings
its provider rejects.

## Outcome

No SecretStore is created and the developer sees why.
