---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: A namespace now matches the ClusterPushSecret's selection
    kind: condition
    unattended: true
    entities:
      - { entity: cluster-push-secret, effect: reads, facts: [Namespace selection] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product creates the PushSecret in that namespace
    kind: product
    entities:
      - { entity: push-secret, effect: creates, facts: [Secret stores, Source, Data, Data to, Template, Update policy, Deletion policy, Refresh interval] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
  - text: The Product adds the namespace to the provisioned namespaces
    kind: product
    entities:
      - { entity: cluster-push-secret, effect: changes, facts: [Provisioned namespaces, Ready] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-push-secret } }
---

# Provision a newly matching namespace

## Trigger

A namespace is created with, or given, labels the ClusterPushSecret selects.

## Outcome

The new namespace has the PushSecret.
