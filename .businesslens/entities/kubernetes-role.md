---
references:
  - kind: code
    role: implementation
    target: deploy/charts/external-secrets/templates/rbac.yaml
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: Roles and responsibilities
  - kind: doc
    role: context
    target: docs/guides/multi-tenancy.md
---

# Kubernetes role

A Kubernetes Role or ClusterRole and its bindings, defined in the cluster's
configuration outside the Product, that decides which people may create, read,
change and delete the Product's resources. The Product ships two such roles: a
view role and an edit role, added to the cluster's view, edit and admin roles
unless the cluster operator turns that off. The edit role covers ClusterSecretStores,
ClusterPushSecrets and ClusterGenerators while the Product processes them, but
not ClusterExternalSecrets; a role bound inside one namespace never reaches
cluster-wide resources.

## Information kept

- **Name** — the role's name
- **Permissions** — which of the Product's resources it allows, and which operations on them
- **Members** — the users, groups and service accounts bound to it
- **Scope** — the namespace it is bound in, or the whole cluster
