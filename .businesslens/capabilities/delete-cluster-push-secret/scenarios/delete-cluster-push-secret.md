---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator deletes a ClusterPushSecret, and every PushSecret it provisioned goes with it
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-push-secret, effect: removes }
      - { entity: push-secret, effect: removes, with: cluster-push-secret }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
---

# Delete a ClusterPushSecret and what it provisioned

## Trigger

Pushing from many namespaces is no longer needed.

## Outcome

The ClusterPushSecret and its PushSecrets are gone; each PushSecret's deletion policy decides whether its provider secrets go too.
