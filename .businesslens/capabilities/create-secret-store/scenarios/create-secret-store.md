---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a SecretStore naming one provider and how to authenticate to it, which the Product admits after checking its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: secret-store, effect: creates, facts: [Provider, Provider settings, Controller class, Refresh interval, Retry settings] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The Product connects to the provider with the store's credentials and validates the connection
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: changes, facts: [Ready, Capabilities] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
  - text: The SecretStore reports that it is ready and what the provider allows
    kind: condition
    actor: application-developer
    entities:
      - { entity: secret-store, effect: reads, facts: [Ready, Capabilities] }
    contexts: { api: { place: kubernetes-api::namespace::secret-store } }
---

# Create a ready SecretStore

## Trigger

An Application Developer wants their namespace to read secrets from, or write
secrets to, a provider.

## Outcome

The SecretStore is ready with reason Valid and shows whether the provider is
read-only, write-only or both; ExternalSecrets and PushSecrets in the namespace
can now name it.

## Edge cases

- A provider marked unmaintained or deprecated is admitted with a warning that the store may stop working in a later version, and the store records a warning event on every validation unless it carries the ignore-maintenance-checks annotation.
- When the provider cannot say whether the credentials are valid, the store is ready with reason ValidationUnknown.
