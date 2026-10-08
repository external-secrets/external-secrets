---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_sshkey.go
  - kind: code
    role: implementation
    target: generators/v1/sshkey/sshkey.go
  - kind: doc
    role: intent
    target: docs/api/generator/sshkey.md
---

# SSHKey

A generator that produces an SSH key pair.

## Information kept

- **Key type** — RSA, ECDSA or Ed25519; RSA by default
- **Key size** — the key size
- **Comment** — the comment added to the public key
