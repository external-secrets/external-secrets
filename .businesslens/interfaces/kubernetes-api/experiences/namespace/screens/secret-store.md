---
entities:
  - entity: secret-store
    collects: [Provider, Provider settings, Controller class, Refresh interval, Retry settings]
    shows: [Provider, Ready, Capabilities]
---

# SecretStore

One SecretStore in a namespace: the provider connection a developer declares
and whether the Product could validate it.
