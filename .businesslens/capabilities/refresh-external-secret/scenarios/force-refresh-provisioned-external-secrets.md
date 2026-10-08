---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Cluster Operator sets a new force-sync annotation on a ClusterExternalSecret
    kind: actor
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: changes, facts: [Force sync] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product copies the annotation to every ExternalSecret the ClusterExternalSecret provisioned
    kind: product
    actor: cluster-operator
    entities:
      - { entity: cluster-external-secret, effect: reads, facts: [Force sync] }
      - { entity: external-secret, effect: changes, facts: [Force sync] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product fetches the provisioned values again through each store
    kind: product
    actor: cluster-operator
    entities:
      - { entity: provider-secret, effect: reads, facts: [Key, Value, Version] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
  - text: The Product updates each target Secret whose values changed and records the refresh on its ExternalSecret
    kind: product
    actor: cluster-operator
    entities:
      - { entity: secret, effect: changes, facts: [Data] }
      - { entity: external-secret, effect: changes, facts: [Ready, Refresh time] }
    contexts: { api: { place: kubernetes-api::cluster::cluster-external-secret } }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/clusterexternalsecret/clusterexternalsecret_controller.go
  - kind: code
    role: implementation
    target: pkg/controllers/util/util.go#GetResourceVersion
---

# Force a refresh of every provisioned ExternalSecret

## Trigger

A Cluster Operator rotated a shared value at the provider and every namespace
needs it now.

## Outcome

Every ExternalSecret the ClusterExternalSecret provisioned syncs straight away
and its target holds the provider's current values.

## Edge cases

- Removing the annotation from the ClusterExternalSecret removes it from the provisioned ExternalSecrets too.
- A provisioned ExternalSecret with refresh policy CreatedOnce syncs nothing.
