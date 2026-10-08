---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator deletes a ClusterGenerator
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-generator, effect: removes }
    contexts: { api: { place: kubernetes-api::cluster::cluster-generator } }
---

# Delete a ClusterGenerator

## Trigger

Namespaces no longer share values produced this way.

## Outcome

The ClusterGenerator is gone; ExternalSecrets and PushSecrets in any namespace
that still name it fail at their next refresh.
