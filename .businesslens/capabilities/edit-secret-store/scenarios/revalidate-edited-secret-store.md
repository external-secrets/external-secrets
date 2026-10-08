---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer changes the provider settings of a SecretStore, which the Product admits after checking them
    kind: actor
    actor: application-developer
    entities:
      - { entity: secret-store, effect: changes, facts: [Provider settings] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The Product connects to the provider with the new settings and records whether the SecretStore is ready
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: changes, facts: [Ready, Capabilities] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# Validate a SecretStore again after an edit

## Trigger

An Application Developer rotates the credentials a store uses or points it
somewhere else.

## Outcome

The SecretStore's readiness reflects the new settings at once.

## Edge cases

- A change that breaks an admission rule is refused, as on creation.
- Settings the provider rejects leave the store not ready with reason InvalidProviderConfig.
