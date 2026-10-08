---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: A namespace now matches the ClusterExternalSecret's selection
    kind: condition
    unattended: true
    entities:
      - { entity: cluster-external-secret, effect: reads, facts: [Namespace selection] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product creates the ExternalSecret in that namespace
    kind: product
    entities:
      - { entity: external-secret, effect: creates, facts: [Secret store, Data, Data from, Target name, Creation policy, Deletion policy, Template, Immutable, Target kind, Refresh policy, Refresh interval, Sync windows] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product adds the namespace to the provisioned namespaces
    kind: product
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [Provisioned namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
---

# Provision a newly matching namespace

## Trigger

A namespace is created with, or given, labels the ClusterExternalSecret selects.

## Outcome

The new namespace has the ExternalSecret without anyone applying it there.
