---
type: api
actors: [monitoring-system]
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
---

# Metrics

The controller's Prometheus endpoint at `/metrics`: how often and how long the
Product reconciles each kind of resource, the status condition of every
ExternalSecret, PushSecret, ClusterExternalSecret, SecretStore and
ClusterSecretStore, and how many calls it made to each secret provider. It is
served over HTTP by default, over HTTPS when the cluster operator turns that
on, and behind Kubernetes authentication and authorization when the cluster
operator asks for it.
