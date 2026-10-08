---
entities:
  - entity: cluster-external-secret
    collects: [External secret spec, External secret name, External secret metadata, Namespace selection, Refresh interval, Force sync]
    shows: [Namespace selection, Provisioned namespaces, Failed namespaces, Ready]
---

# ClusterExternalSecret

One ClusterExternalSecret: the ExternalSecret it provisions, the namespaces it
selects, and where provisioning succeeded or failed.
