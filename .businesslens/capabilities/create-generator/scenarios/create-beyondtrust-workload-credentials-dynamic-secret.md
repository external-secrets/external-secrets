---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies a BeyondtrustWorkloadCredentialsDynamicSecret generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: beyondtrust-workload-credentials-dynamic-secret, effect: creates, facts: [Provider settings, Controller class, Retry settings] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create a BeyondtrustWorkloadCredentialsDynamicSecret generator

## Trigger

A namespace needs a generator that requests dynamic credentials from BeyondTrust Workload Credentials.

## Outcome

The BeyondtrustWorkloadCredentialsDynamicSecret generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
