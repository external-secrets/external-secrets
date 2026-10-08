---
type: api
actors: [application-developer, cluster-operator]
references:
  - kind: doc
    role: intent
    target: docs/api/components.md
  - kind: doc
    role: intent
    target: docs/api/spec.md
  - kind: code
    role: implementation
    target: cmd/controller/webhook.go
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/externalsecret_validator.go
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/secretstore_validator.go
---

# Kubernetes API

The Product's custom resources in the `external-secrets.io` and
`generators.external-secrets.io` API groups, served by the cluster's Kubernetes
API. People create, change and delete the resources with any Kubernetes client
and read the outcome from each resource's status and events. Where the
Product's admission webhook is installed, as it is by default, ExternalSecrets,
SecretStores and ClusterSecretStores are checked as they are created or
changed, and a refused request is not stored.

## Intent

Make secret synchronization declarative: everything a person asks of the
Product is a resource they can keep in Git and apply like the rest of their
cluster configuration.
