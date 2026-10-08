---
entities:
  - entity: cluster-push-secret
    collects: [Push secret spec, Push secret name, Push secret metadata, Namespace selection, Refresh interval]
    shows: [Namespace selection, Provisioned namespaces, Failed namespaces, Ready]
---

# ClusterPushSecret

One ClusterPushSecret: the PushSecret it provisions, the namespaces it selects,
and where provisioning succeeded or failed.
