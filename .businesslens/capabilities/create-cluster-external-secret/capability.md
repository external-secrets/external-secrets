---
domain: core-resources
availability:
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/clusterexternalsecret/clusterexternalsecret_controller.go
  - kind: doc
    role: intent
    target: docs/api/clusterexternalsecret.md
---

# Create a ClusterExternalSecret

Provision the same ExternalSecret into every namespace a Cluster Operator
selects, by name or by labels. Each provisioned ExternalSecret then syncs its
own target as any other does. The Product provisions only while its controller
options have ClusterExternalSecrets processed.
