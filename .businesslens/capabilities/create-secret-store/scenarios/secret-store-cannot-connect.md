---
kind: edge
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a SecretStore whose credentials the provider does not accept, and the Product admits it
    kind: actor
    actor: application-developer
    entities:
      - { entity: secret-store, effect: creates, facts: [Provider, Provider settings, Controller class, Refresh interval, Retry settings] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The Product cannot build a working connection and marks the SecretStore not ready
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# SecretStore that cannot connect

## Trigger

The provider rejects the store's credentials, or — where no admission webhook
checked it — the store names a provider the running Product does not include.

## Outcome

The SecretStore exists but is not ready, with reason InvalidProviderConfig or
ProviderNotFound and a warning event; the Product tries again on its next
validation.
