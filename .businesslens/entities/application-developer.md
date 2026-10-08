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

# Application Developer

The person who defines the secrets an application needs inside the namespaces
they work in: the stores those secrets come from, the ExternalSecrets that bring
them into the cluster, the PushSecrets that send values out, and the
generators that produce values. What they may do in each namespace is decided by
the Kubernetes roles bound to them.
