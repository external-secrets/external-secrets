---
appliesTo:
  - { type: capability, id: create-external-secret }
  - { type: capability, id: refresh-external-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/client_manager.go#assertStoreIsUsable
  - kind: code
    role: implementation
    target: cmd/controller/root.go
---

# With the flood gate on, ExternalSecrets sync only through ready stores

While the flood gate is on — the default — the Product does not call a provider for an ExternalSecret whose store is not ready, and reports the ExternalSecret not ready instead.

## Rationale

A broken store would otherwise send every ExternalSecret using it to the provider on every retry.
