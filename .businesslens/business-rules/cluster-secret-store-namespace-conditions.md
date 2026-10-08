---
appliesTo:
  - { type: entity, id: cluster-secret-store, facts: [Namespace conditions] }
  - { type: capability, id: create-external-secret }
  - { type: capability, id: create-push-secret }
  - { type: capability, id: refresh-external-secret }
  - { type: capability, id: refresh-push-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/client_manager.go#Manager.shouldProcessSecret
  - kind: doc
    role: intent
    target: docs/api/clustersecretstore.md
---

# A ClusterSecretStore serves only the namespaces its conditions admit

An ExternalSecret or PushSecret may use a ClusterSecretStore only from a namespace matching one of its conditions by name, label selector or name pattern; a store without conditions serves every namespace.
