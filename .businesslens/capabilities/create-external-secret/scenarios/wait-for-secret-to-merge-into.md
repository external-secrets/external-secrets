---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an ExternalSecret with creation policy Merge for a target that does not exist yet, which the Product admits
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Target name, Creation policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: Nothing with the target name exists in the namespace yet
    kind: condition
    actor: application-developer
    entities:
      - { entity: external-secret, effect: reads, facts: [Creation policy, Target name] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
  - text: The Product writes nothing and marks the ExternalSecret ready with reason SecretMissing
    kind: product
    actor: application-developer
    entities:
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Wait for a Secret to merge into

## Trigger

Merge is asked for before the Secret to merge into has been created.

## Outcome

No Secret is created; the Product tries again at the next refresh.
