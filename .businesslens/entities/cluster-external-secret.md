---
domain: core-resources
relations:
  - entity: external-secret
    verb: provisions
    cardinality: one-to-many
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/clusterexternalsecret_types.go
  - kind: code
    role: implementation
    target: pkg/controllers/clusterexternalsecret/clusterexternalsecret_controller.go
  - kind: doc
    role: intent
    target: docs/api/clusterexternalsecret.md
---

# ClusterExternalSecret

A cluster-wide declaration that provisions the same ExternalSecret into every
namespace it selects, and keeps that set current as namespaces come, go or
change their labels.

## Information kept

- **External secret spec** — the ExternalSecret to provision in each selected namespace
- **External secret name** — the name of the provisioned ExternalSecrets; the ClusterExternalSecret's own name when empty
- **External secret metadata** — labels and annotations given to the provisioned ExternalSecrets
- **Namespace selection** — the namespaces chosen by label selectors or, in older declarations, by name; any match selects a namespace
- **Refresh interval** — how often the Product checks again which namespaces match
- **Force sync** — the force-sync annotation, copied to every provisioned ExternalSecret
- **Provisioned namespaces** — the namespaces where its ExternalSecret exists
- **Failed namespaces** — the namespaces where provisioning failed, each with its reason
- **Ready** — whether every selected namespace was provisioned
