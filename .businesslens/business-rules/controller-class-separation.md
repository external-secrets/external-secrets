---
appliesTo:
  - { type: capability, id: validate-store }
  - { type: capability, id: create-external-secret }
  - { type: capability, id: refresh-external-secret }
  - { type: capability, id: create-push-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/common.go#ShouldProcessStore
  - kind: doc
    role: intent
    target: docs/guides/controller-class.md
---

# A controller processes only stores of its own controller class

A running Product processes stores whose controller class is empty or its own, and ExternalSecrets and PushSecrets only through such stores and generators; everything else is left for the controller of that class.

## Rationale

Several installations of the Product can share one cluster, each serving its own resources.
