---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator changes the settings of a ClusterGenerator
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-generator, effect: changes, facts: [Generator settings] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-generator } }
---

# Change a ClusterGenerator's settings

## Trigger

Values shared across namespaces must be produced differently.

## Outcome

The ClusterGenerator keeps the new settings; every namespace's next refresh uses
them.
