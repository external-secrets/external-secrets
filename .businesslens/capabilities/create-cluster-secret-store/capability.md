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
  - kind: doc
    role: intent
    target: docs/api/clustersecretstore.md
---

# Create a ClusterSecretStore

Publish a connection to one secret provider that namespaces across the cluster
can share, optionally limited to the namespaces its conditions admit. The
Product checks it as it is submitted, then connects to the provider and reports
whether it is ready. The Product validates ClusterSecretStores, and syncs the
ExternalSecrets that name one, only while its controller options have
ClusterSecretStores processed.
