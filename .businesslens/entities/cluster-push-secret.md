---
domain: core-resources
relations:
  - entity: push-secret
    verb: provisions
    cardinality: one-to-many
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1alpha1/pushsecret_types.go
  - kind: code
    role: implementation
    target: pkg/controllers/clusterpushsecret/clusterpushsecret_controller.go
  - kind: doc
    role: intent
    target: docs/api/clusterpushsecret.md
---

# ClusterPushSecret

A cluster-wide declaration that provisions the same PushSecret into every
namespace it selects, and keeps that set current as namespaces change.

## Information kept

- **Push secret spec** — the PushSecret to provision in each selected namespace
- **Push secret name** — the name of the provisioned PushSecrets; the ClusterPushSecret's own name when empty
- **Push secret metadata** — labels and annotations given to the provisioned PushSecrets
- **Namespace selection** — the namespaces chosen by label selectors
- **Refresh interval** — how often the Product checks again which namespaces match
- **Provisioned namespaces** — the namespaces where its PushSecret exists
- **Failed namespaces** — the namespaces where provisioning failed, each with its reason
- **Ready** — whether every selected namespace was provisioned
