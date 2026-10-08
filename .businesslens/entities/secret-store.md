---
domain: core-resources
relations:
  - entity: external-secret
    verb: serves
    cardinality: one-to-many
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/secretstore_types.go
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/common.go
  - kind: doc
    role: intent
    target: docs/api/secretstore.md
---

# SecretStore

A namespaced connection to one external secret provider: where the provider is
and how to authenticate to it. ExternalSecrets and PushSecrets in the same
namespace name it to say where their values come from or go to. Credentials it
references are read from its own namespace.

## Information kept

- **Provider** — the one secret provider the store connects to, from the Product's supported list
- **Provider settings** — where the provider is and how to authenticate to it, including references to credential Secrets
- **Controller class** — which controller instance processes the store; empty means every instance
- **Refresh interval** — how often the Product validates the store again
- **Retry settings** — how many times, and how long apart, provider calls are retried
- **Ready** — whether the last validation succeeded, with its reason: Valid, InvalidProviderConfig, ProviderNotFound or ValidationUnknown
- **Capabilities** — whether the provider can be read, written or both: ReadOnly, WriteOnly or ReadWrite
