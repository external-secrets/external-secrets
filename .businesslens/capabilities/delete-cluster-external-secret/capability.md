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

# Delete a ClusterExternalSecret

Remove a ClusterExternalSecret together with every ExternalSecret it
provisioned.
