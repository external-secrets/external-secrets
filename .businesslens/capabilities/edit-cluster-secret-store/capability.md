---
domain: core-resources
availability:
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/secretstore_validator.go
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/clustersecretstore_controller.go
---

# Edit a ClusterSecretStore

Change a ClusterSecretStore's provider settings or the namespaces it admits. The
Product checks the change as it is submitted and validates the store again
straight away; namespaces it no longer admits fail at their next sync.
