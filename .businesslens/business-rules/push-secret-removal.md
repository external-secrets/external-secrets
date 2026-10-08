---
appliesTo:
  - { type: entity, id: push-secret, effect: removes }
permits:
  - { configuredBy: kubernetes-role }
  - { unattended: true, when: [{ entity: controller-options, fact: Cluster push secrets, is: Enabled }] }
references:
  - kind: code
    role: implementation
    target: deploy/charts/external-secrets/templates/rbac.yaml
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: Roles and responsibilities
---

# People a Kubernetes role permits, and the Product for a ClusterPushSecret, delete a PushSecret

A PushSecret is deleted by someone whose Kubernetes roles allow it in that namespace, and by the Product on its own schedule when its namespace leaves the selection of the ClusterPushSecret that provisioned it — only while its controller options have ClusterPushSecrets processed.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
