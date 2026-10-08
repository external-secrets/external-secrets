---
appliesTo:
  - { type: entity, id: secret-store, effect: changes }
permits:
  - { configuredBy: kubernetes-role }
  - { unattended: true, when: [{ entity: controller-options, fact: Secret stores, is: Enabled }] }
references:
  - kind: code
    role: implementation
    target: deploy/charts/external-secrets/templates/rbac.yaml
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: Roles and responsibilities
---

# People a Kubernetes role permits, and the Product while it processes SecretStores, change a SecretStore

A SecretStore is changed by someone whose Kubernetes roles allow it, and by the Product as it records the store's readiness — only while its controller options have SecretStores processed.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
