---
singleton: true
references:
  - kind: code
    role: implementation
    target: cmd/controller/root.go
  - kind: doc
    role: intent
    target: docs/api/controller-options.md
  - kind: doc
    role: intent
    target: docs/guides/disable-cluster-features.md
  - kind: code
    role: context
    target: deploy/charts/external-secrets/values.yaml
  - kind: code
    role: context
    target: runtime/statemanager/statemanager.go
---

# Controller options

How a cluster operator configured one running Product, chosen at installation
through the Helm chart's values and the controller's command-line options.
Each installation of the Product has its own.

## Information kept

- **Controller class** — the class this installation processes; stores naming another class are left to another installation
- **Watched namespace** — the one namespace this installation watches, or every namespace when empty
- **Secret stores** — whether SecretStores are processed: Enabled or Disabled
- **Cluster secret stores** — whether ClusterSecretStores are processed, and ExternalSecrets naming one synced: Enabled or Disabled
- **Cluster external secrets** — whether ClusterExternalSecrets are processed: Enabled or Disabled
- **Push secrets** — whether PushSecrets are processed: Enabled or Disabled
- **Cluster push secrets** — whether ClusterPushSecrets are processed: Enabled or Disabled
- **Store requeue interval** — how often stores without a refresh interval of their own are validated again; five minutes by default
- **Flood gate** — whether ExternalSecrets wait for their store to be ready before syncing: On, the default, or Off
- **Generic targets** — whether ExternalSecrets may write ConfigMaps and custom resources: Allowed or Not allowed, the default
- **Generator state** — whether the Product records what stateful generators issued for ExternalSecrets, so it can revoke it later: On, the default, or Off
- **Metrics authentication** — whether the metrics endpoint requires Kubernetes authentication and authorization: On or Off, the default
