---
appliesTo:
  - { type: capability, id: delete-secret-store }
  - { type: capability, id: delete-cluster-secret-store }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/common.go#handleFinalizer
---

# A store is kept while PushSecrets that clean up still need it

While the Product processes PushSecrets, a SecretStore or ClusterSecretStore is not removed while a PushSecret with deletion policy Delete pushes through it.

## Rationale

Those PushSecrets need the store to delete what they pushed when they are deleted.
