---
appliesTo:
  - { type: entity, id: cluster-external-secret, effect: changes }
permits:
  - { configuredBy: kubernetes-role }
  - { unattended: true, when: [{ entity: controller-options, fact: Cluster external secrets, is: Enabled }] }
references:
  - kind: code
    role: implementation
    target: deploy/charts/external-secrets/templates/rbac.yaml
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: Roles and responsibilities
---

# People a Kubernetes role permits, and the Product while it processes ClusterExternalSecrets, change a ClusterExternalSecret

A ClusterExternalSecret is changed by someone whose Kubernetes roles allow it cluster-wide, and by the Product as it records where it provisioned — only while its controller options have ClusterExternalSecrets processed.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
