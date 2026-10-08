---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes a PushSecret whose deletion policy is Delete, and every provider secret it pushed goes with it
    kind: actor
    actor: application-developer
    entities:
      - { entity: push-secret, effect: removes }
      - { entity: provider-secret, effect: removes, with: push-secret }
    contexts: { api: { place: kubernetes-api::namespace::push-secret } }
---

# Delete a PushSecret and what it pushed

## Trigger

Values pushed from the cluster must not outlive the PushSecret.

## Outcome

The PushSecret is gone and the providers no longer hold what it pushed.
