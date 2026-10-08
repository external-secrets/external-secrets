---
domain: core-resources
availability:
  - { place: kubernetes-api::namespace }
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/secretstore_validator.go
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/common.go
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/secretstore_controller.go
  - kind: doc
    role: intent
    target: docs/api/secretstore.md
---

# Create a SecretStore

Declare a connection to one secret provider inside a namespace. The Product
checks the declaration as it is submitted, then connects to the provider and
reports whether the store is ready to use and whether it can read, write or
both.
