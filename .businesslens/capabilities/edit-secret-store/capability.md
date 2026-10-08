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
---

# Edit a SecretStore

Change how a SecretStore reaches or authenticates to its provider. The Product
checks the change as it is submitted and validates the store again straight
away. ExternalSecrets and PushSecrets using the store pick up the change at
their next refresh.
