---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator applies a ClusterGenerator naming one generator kind and its settings
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-generator, effect: creates, facts: [Generator kind, Generator settings] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-generator } }
---

# Publish a ClusterGenerator

## Trigger

Many namespaces need values produced the same way, such as registry tokens for one shared registry.

## Outcome

The ClusterGenerator exists; ExternalSecrets and PushSecrets in every namespace can name it.

## Edge cases

- A ClusterGenerator whose kind and settings do not match is reported as invalid to the ExternalSecret or PushSecret that names it.
