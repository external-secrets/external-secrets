---
availability:
  - { place: metrics }
references:
  - kind: doc
    role: intent
    target: docs/api/metrics.md
  - kind: code
    role: implementation
    target: cmd/controller/root.go
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/esmetrics/esmetrics.go
  - kind: code
    role: context
    target: pkg/controllers/metrics/labels.go
---

# Read metrics

Scrape the controller's metrics: sync counts and errors, reconcile durations,
the status condition of each ExternalSecret, PushSecret, ClusterExternalSecret,
SecretStore and ClusterSecretStore, and the calls made to each provider. The
metrics describe what happened, never the values synced.
