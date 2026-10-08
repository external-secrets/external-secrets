---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The SecretStore's refresh interval has passed and the provider no longer accepts its credentials
    kind: condition
    unattended: true
    entities:
      - { entity: secret-store, effect: reads, facts: [Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The Product marks the SecretStore not ready and records a warning event
    kind: product
    entities:
      - { entity: secret-store, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# Store loses access to its provider

## Trigger

Credentials that once worked are revoked or expire at the provider.

## Outcome

The SecretStore is not ready with reason InvalidProviderConfig; while the flood
gate is on, ExternalSecrets using it stop syncing until it is ready again.
