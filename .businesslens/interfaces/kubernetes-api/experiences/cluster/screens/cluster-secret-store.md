---
entities:
  - entity: cluster-secret-store
    collects: [Provider, Provider settings, Namespace conditions, Controller class, Refresh interval, Retry settings]
    shows: [Provider, Namespace conditions, Ready, Capabilities]
---

# ClusterSecretStore

One ClusterSecretStore: the shared provider connection, which namespaces may
use it, and whether the Product could validate it.
