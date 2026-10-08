---
appliesTo:
  - { type: entity, id: external-secret, effect: creates }
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

# People a Kubernetes role permits, and the Product for a ClusterExternalSecret, create an ExternalSecret

An ExternalSecret is created by someone whose Kubernetes roles allow it in that namespace, and by the Product on its own schedule as it provisions a ClusterExternalSecret into a newly selected namespace — only while its controller options have ClusterExternalSecrets processed.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
