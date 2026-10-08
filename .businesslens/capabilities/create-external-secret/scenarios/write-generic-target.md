---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret whose target is a ConfigMap or a custom resource, which the Product admits
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Target kind, Template, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The controller options allow ConfigMaps and custom resources as targets
    kind: condition
    actor: application-developer
    entities:
      - { entity: controller-options, effect: reads, facts: [Generic targets] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product fetches the requested values through the SecretStore
    kind: product
    actor: application-developer
    entities:
      - { entity: secret-store, effect: reads, facts: [Provider, Provider settings, Ready] }
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product renders the Generic target from the template and creates it, owned by the ExternalSecret
    kind: product
    actor: application-developer
    entities:
      - { entity: generic-target, effect: creates, facts: [Kind, Content, Owner] }
      - { entity: external-secret, effect: reads, facts: [] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret ready
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Write a ConfigMap or custom resource

## Trigger

Configuration that is not a Secret must carry provider values.

## Outcome

The ConfigMap or custom resource exists with the rendered content and the ExternalSecret is ready.
