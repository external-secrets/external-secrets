---
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/provider.go
  - kind: doc
    role: context
    target: docs/introduction/stability-support.md
---

# Provider secret

A secret kept in an external secret provider — a Secrets Manager secret, a
Vault path, a Key Vault secret — addressed by its key. ExternalSecrets read
provider secrets; PushSecrets create, update and delete them.

## Information kept

- **Key** — the provider's name or path for the secret
- **Value** — the secret's content, either one value or structured properties
- **Version** — the provider's version of the value, where the provider keeps versions
- **Metadata** — provider-side tags or labels, where the provider supports them
