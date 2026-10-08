---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_mfa.go
  - kind: code
    role: implementation
    target: generators/v1/mfa/mfa.go
  - kind: doc
    role: intent
    target: docs/api/generator/mfa.md
---

# MFA

A generator that produces a time-based one-time password from a stored seed.

## Information kept

- **Seed** — the Secret key holding the TOTP seed
- **Length** — the number of digits
- **Time period** — how long a code is valid
- **Algorithm** — the hash algorithm
- **When** — a fixed time to generate the code for
