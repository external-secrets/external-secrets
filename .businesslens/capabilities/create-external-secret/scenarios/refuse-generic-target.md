---
kind: validation
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret whose target is a ConfigMap or a custom resource, which the Product admits
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Target kind, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The controller options do not allow ConfigMaps and custom resources as targets
    kind: condition
    actor: application-developer
    entities:
      - { entity: controller-options, effect: reads, facts: [Generic targets] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product marks the ExternalSecret not ready, saying such targets are disabled
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Refuse a generic target that is not allowed

## Trigger

A target other than a Secret is asked for in a cluster where generic targets are off.

## Outcome

Nothing is written and the ExternalSecret explains that generic targets must be enabled first.
