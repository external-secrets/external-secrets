---
appliesTo:
  - { type: entity, id: secret-store, facts: [Provider settings] }
  - { type: capability, id: create-secret-store }
  - { type: capability, id: create-external-secret }
references:
  - kind: code
    role: implementation
    target: runtime/esutils/resolvers/secret_ref.go#SecretKeyRef
---

# A SecretStore reads credentials only from the namespace it is used in

Credentials a SecretStore references are always read from the namespace of the resource using it; only a ClusterSecretStore may name a credential Secret in another namespace. Credentials a generator references are always read from the namespace of the resource using it.

## Rationale

Otherwise anyone able to create a SecretStore could borrow credentials from any namespace through the Product's own privileges.
