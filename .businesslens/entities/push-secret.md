---
domain: core-resources
relations:
  - entity: provider-secret
    verb: pushes
    cardinality: one-to-many
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1alpha1/pushsecret_types.go
  - kind: code
    role: implementation
    target: pkg/controllers/pushsecret/pushsecret_controller.go#Reconciler.Reconcile
  - kind: doc
    role: intent
    target: docs/api/pushsecret.md
  - kind: doc
    role: intent
    target: docs/guides/pushsecrets.md
---

# PushSecret

A namespaced declaration that writes values from Kubernetes Secrets, or from a
generator, into one or more secret providers and keeps them there.

## Information kept

- **Secret stores** — the stores to push into, by name or by label selector
- **Source** — what is pushed: a Secret by name, the Secrets matching a label selector, or a generator
- **Data** — single entries: a source key and the provider key, and optional property and metadata, it is written to
- **Data to** — bulk entries that push many source keys to selected stores at once, with key matching and rewriting
- **Template** — how pushed values are rendered from the source
- **Update policy** — Replace (the default), or IfNotExists, which leaves a provider secret that already exists unchanged
- **Deletion policy** — None (the default), or Delete, which removes pushed provider secrets once they are no longer pushed
- **Refresh interval** — how long between pushes, one hour by default; zero pushes again only when the PushSecret changes
- **Synced push secrets** — which provider keys were pushed to which stores
- **Ready** — whether the last push succeeded, with its reason: Synced, Errored or SourceDeleted
- **Refresh time** — when the Product last pushed
