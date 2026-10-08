---
kind: primary
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator deletes a ClusterExternalSecret, and every ExternalSecret it provisioned goes with it
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: removes }
      - { entity: external-secret, effect: removes, with: cluster-external-secret }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
---

# Delete a ClusterExternalSecret and what it provisioned

## Trigger

A secret no longer needs to be spread across namespaces.

## Outcome

The ClusterExternalSecret and all ExternalSecrets it provisioned are gone; their
targets follow each ExternalSecret's own creation policy.
