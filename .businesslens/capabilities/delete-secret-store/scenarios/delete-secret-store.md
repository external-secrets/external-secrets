---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes a SecretStore nothing pushes through
    kind: actor
    actor: application-developer
    entities:
      - { entity: secret-store, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# Delete an unused SecretStore

## Trigger

An Application Developer no longer needs a store.

## Outcome

The SecretStore is gone; ExternalSecrets still naming it fail to sync until
they name another store.
