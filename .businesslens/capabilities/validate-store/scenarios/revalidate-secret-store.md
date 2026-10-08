---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The SecretStore's refresh interval has passed
    kind: condition
    unattended: true
    entities:
      - { entity: secret-store, effect: reads, facts: [Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The Product connects to the provider again and records whether the SecretStore is ready
    kind: product
    entities:
      - { entity: secret-store, effect: changes, facts: [Ready, Capabilities] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# Revalidate a SecretStore

## Trigger

The SecretStore's refresh interval passes.

## Outcome

The SecretStore's readiness reflects whether the provider accepts its
credentials now.
