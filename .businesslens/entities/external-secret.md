---
domain: core-resources
relations:
  - entity: secret
    verb: owns
    cardinality: one-to-one
  - entity: generic-target
    verb: writes
    cardinality: one-to-one
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/externalsecret_types.go
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.Reconcile
  - kind: doc
    role: intent
    target: docs/api/externalsecret.md
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
---

# ExternalSecret

A namespaced declaration of which values to fetch from secret providers and how
to write them into a Kubernetes Secret — or, where allowed, into another kind of
resource. The Product keeps the target in sync with it for as long as it
exists.

## Information kept

- **Secret store** — the SecretStore or ClusterSecretStore values come from unless an entry names its own
- **Data** — single values to fetch: each target key with its provider key and optional property, version, decoding and conversion strategy, and optionally its own store or generator
- **Data from** — sources that bring many keys at once: extracting a structured provider secret, finding provider secrets by name, path or tags, or a generator, each with optional key rewriting
- **Target name** — the name of the Secret to write; the ExternalSecret's own name when empty
- **Creation policy** — how the Product treats the target: Owner (the default), Orphan, Merge, CreateOrMerge or None
- **Deletion policy** — what happens to the target when the provider returns no data: Retain (the default), Delete or Merge
- **Template** — how the target's type, labels, annotations and data are rendered from the fetched values, from inline templates or templates kept in ConfigMaps and Secrets
- **Immutable** — whether the target Secret is marked immutable so its data is never rewritten once it exists
- **Target kind** — the kind of resource written when it is not a Secret: a ConfigMap or a custom resource
- **Refresh policy** — Periodic (the default), OnChange or CreatedOnce
- **Refresh interval** — how long between periodic refreshes, one hour by default; zero syncs only once
- **Force sync** — the force-sync annotation; giving it a new value asks for a refresh straight away
- **Sync windows** — scheduled windows that allow, or deny, periodic refreshes
- **Ready** — whether the last sync succeeded, with its reason: SecretSynced, SecretSyncedError, SecretDeleted, SecretMissing, SecretOwnedByOther or SecretImmutable, or for a generic target ResourceSynced, ResourceSyncedError, ResourceDeleted or ResourceMissing
- **Refresh time** — when the Product last synced the target
