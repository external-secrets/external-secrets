---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_vault.go
  - kind: code
    role: implementation
    target: generators/v1/vault/vault.go
  - kind: doc
    role: intent
    target: docs/api/generator/vault.md
---

# VaultDynamicSecret

A generator that reads dynamic credentials from a HashiCorp Vault secrets engine.

## Information kept

- **Provider settings** — where Vault is and how to authenticate to it
- **Path** — the Vault path to call
- **Method** — the HTTP method of the call
- **Parameters** — the request parameters
- **Result type** — which part of Vault's response is returned
- **Allow empty response** — whether an empty response is accepted
- **Retry settings** — how Vault calls are retried
- **Controller class** — which controller instance may use the generator
