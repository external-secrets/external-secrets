---
domain: generators
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/types_beyondtrustworkloadcredentials.go
  - kind: code
    role: implementation
    target: generators/v1/beyondtrustworkloadcredentials/beyondtrustworkloadcredentials.go
  - kind: doc
    role: intent
    target: docs/api/generator/beyondtrustworkloadcredentials.md
---

# BeyondtrustWorkloadCredentialsDynamicSecret

A generator that requests dynamic credentials from BeyondTrust Workload Credentials.

## Information kept

- **Provider settings** — where BeyondTrust Workload Credentials is and how to authenticate to it
- **Controller class** — which controller instance may use the generator
- **Retry settings** — how provider calls are retried
