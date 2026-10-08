---
appliesTo:
  - { type: entity, id: external-secret, effect: changes }
permits:
  - { configuredBy: kubernetes-role }
  - { unattended: true }
references:
  - kind: code
    role: implementation
    target: deploy/charts/external-secrets/templates/rbac.yaml
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: Roles and responsibilities
---

# People a Kubernetes role permits, and the Product, change an ExternalSecret

An ExternalSecret is changed by someone whose Kubernetes roles allow it in that namespace, and by the Product as it records each sync.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
