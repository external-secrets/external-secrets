---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes a PushSecret whose deletion policy is None
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: removes }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Delete a PushSecret and keep what it pushed

## Trigger

The pushed values should stay in the provider after the cluster stops managing them.

## Outcome

The PushSecret is gone; the providers keep the last pushed values.
