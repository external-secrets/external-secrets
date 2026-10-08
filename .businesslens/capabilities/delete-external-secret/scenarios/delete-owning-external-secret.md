---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer deletes an ExternalSecret whose creation policy is Owner, and its target Secret goes with it
    kind: actor
    actor: application-developer
    entities:
      - { entity: external-secret, effect: removes }
      - { entity: secret, effect: removes, with: external-secret }
    contexts: { api: { place: kubernetes-api::namespace::external-secret } }
---

# Delete an ExternalSecret and the Secret it owns

## Trigger

An application no longer needs a Secret the Product created for it.

## Outcome

Both the ExternalSecret and the Secret it owned are gone.
