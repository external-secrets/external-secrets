---
domain: core-resources
availability:
  - { place: kubernetes-api::namespace }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/common.go#handleFinalizer
---

# Delete a SecretStore

Remove a SecretStore from a namespace. The Product holds the removal while
PushSecrets that delete their provider secrets on removal still use the store,
so they can still clean up.
