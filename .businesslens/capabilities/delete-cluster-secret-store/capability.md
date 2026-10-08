---
domain: core-resources
availability:
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/common.go#handleFinalizer
---

# Delete a ClusterSecretStore

Remove a ClusterSecretStore from the cluster. As for a SecretStore, the Product
holds the removal while PushSecrets that delete their provider secrets on
removal still use it.
