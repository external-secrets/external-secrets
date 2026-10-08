---
domain: core-resources
relations:
  - entity: external-secret
    verb: serves
    cardinality: one-to-many
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/secretstore_types.go
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/clustersecretstore_controller.go
  - kind: code
    role: implementation
    target: pkg/controllers/secretstore/client_manager.go#Manager.shouldProcessSecret
  - kind: doc
    role: intent
    target: docs/api/clustersecretstore.md
---

# ClusterSecretStore

A cluster-wide connection to one external secret provider that ExternalSecrets
and PushSecrets in any admitted namespace can name — the central gateway a
cluster operator publishes so namespaces need no provider credentials of their
own. Credential references may name a namespace explicitly; without one they
are read from the namespace of the resource using the store.

## Information kept

- **Provider** — the one secret provider the store connects to, from the Product's supported list
- **Provider settings** — where the provider is and how to authenticate to it, including references to credential Secrets
- **Namespace conditions** — the namespaces allowed to use the store, by name, label selector or name pattern; none means every namespace
- **Controller class** — which controller instance processes the store; empty means every instance
- **Refresh interval** — how often the Product validates the store again
- **Retry settings** — how many times, and how long apart, provider calls are retried
- **Ready** — whether the last validation succeeded, with its reason: Valid, InvalidProviderConfig, ProviderNotFound or ValidationUnknown
- **Capabilities** — whether the provider can be read, written or both: ReadOnly, WriteOnly or ReadWrite
