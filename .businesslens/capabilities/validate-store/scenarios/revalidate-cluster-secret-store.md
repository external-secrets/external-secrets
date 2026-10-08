---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The ClusterSecretStore's refresh interval has passed
    kind: condition
    unattended: true
    entities:
      - { entity: cluster-secret-store, effect: reads, facts: [Refresh interval] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
  - text: The Product connects to the provider again and records whether the ClusterSecretStore is ready
    kind: product
    entities:
      - { entity: cluster-secret-store, effect: changes, facts: [Ready, Capabilities] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-secret-store } }
---

# Revalidate a ClusterSecretStore

## Trigger

The ClusterSecretStore's refresh interval passes.

## Outcome

The ClusterSecretStore's readiness reflects whether the provider accepts its
credentials now.
