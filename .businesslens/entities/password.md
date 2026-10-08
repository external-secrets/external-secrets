---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_password.go
  - kind: code
    role: implementation
    target: generators/v1/password/password.go
  - kind: doc
    role: intent
    target: docs/api/generator/password.md
---

# Password

A generator that produces random passwords.

## Information kept

- **Length** — the password length
- **Digits** — how many digits it contains
- **Symbols** — how many symbols it contains
- **Symbol characters** — the symbols to choose from
- **No uppercase** — whether uppercase letters are left out
- **Allow repeat** — whether characters may repeat
- **Secret keys** — the keys to generate, each with its own password
- **Encoding** — how the value is encoded
- **Prefix** — text placed before the password
- **Suffix** — text placed after the password
