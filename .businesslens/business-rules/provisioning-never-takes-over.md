---
appliesTo:
  - { type: capability, id: create-cluster-external-secret }
  - { type: capability, id: refresh-cluster-external-secret }
  - { type: capability, id: create-cluster-push-secret }
  - { type: capability, id: refresh-cluster-push-secret }
  - { type: capability, id: edit-cluster-external-secret }
  - { type: capability, id: edit-cluster-push-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/clusterexternalsecret/clusterexternalsecret_controller.go
  - kind: code
    role: implementation
    target: pkg/controllers/clusterpushsecret/clusterpushsecret_controller.go
---

# Cluster-wide resources never take over namespaced ones they did not create

A ClusterExternalSecret or ClusterPushSecret only creates, updates and deletes the ExternalSecrets and PushSecrets it provisioned; one of the same name created by someone else is left alone and the namespace is reported as failed.
