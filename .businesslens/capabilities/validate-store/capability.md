---
domain: core-resources
availability:
  - { place: kubernetes-api::namespace }
  - { place: kubernetes-api::cluster }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/common.go
  - kind: code
    role: implementation
    target: cmd/controller/root.go
---

# Validate a store

The Product checks every SecretStore and ClusterSecretStore again on its own
schedule — the store's refresh interval, or the controller's store requeue
interval of five minutes by default — and whenever the store changes, so its
readiness follows the provider.
