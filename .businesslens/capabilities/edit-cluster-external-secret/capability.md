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

# Edit a ClusterExternalSecret

Change what a ClusterExternalSecret provisions, under which name, or where. The
Product applies the change to every ExternalSecret it provisioned straight away.
