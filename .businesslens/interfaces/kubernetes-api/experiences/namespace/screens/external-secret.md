---
entities:
  - entity: external-secret
    collects: [Secret store, Data, Data from, Target name, Creation policy, Deletion policy, Template, Immutable, Target kind, Refresh policy, Refresh interval, Sync windows, Force sync]
    shows: [Secret store, Target name, Refresh interval, Ready, Refresh time]
---

# ExternalSecret

One ExternalSecret in a namespace: what it fetches, where it writes it and the
lifecycle of its target, with the outcome of its last sync.
