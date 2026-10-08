---
kind: person
acts: external
references:
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: Roles and responsibilities
  - kind: doc
    role: context
    target: docs/guides/multi-tenancy.md
---

# Cluster Operator

The person responsible for running the Product in a cluster: installing it,
choosing its controller options, managing access policies to secret providers
and publishing the cluster-wide stores, secrets and generators namespaces
share. What they may do is decided by the Kubernetes roles bound to them; in a
small cluster one person can be both Cluster Operator and Application
Developer.
