---
appliesTo:
  - { type: entity, id: acr-access-token, effect: changes }
  - { type: entity, id: beyondtrust-workload-credentials-dynamic-secret, effect: changes }
  - { type: entity, id: cloudsmith-access-token, effect: changes }
  - { type: entity, id: ecr-authorization-token, effect: changes }
  - { type: entity, id: fake, effect: changes }
  - { type: entity, id: gcr-access-token, effect: changes }
  - { type: entity, id: github-access-token, effect: changes }
  - { type: entity, id: gitlab-deploy-token, effect: changes }
  - { type: entity, id: grafana, effect: changes }
  - { type: entity, id: mfa, effect: changes }
  - { type: entity, id: password, effect: changes }
  - { type: entity, id: quay-access-token, effect: changes }
  - { type: entity, id: ssh-key, effect: changes }
  - { type: entity, id: sts-session-token, effect: changes }
  - { type: entity, id: uuid, effect: changes }
  - { type: entity, id: vault-dynamic-secret, effect: changes }
  - { type: entity, id: webhook, effect: changes }
  - { type: entity, id: cluster-generator, effect: changes }
permits:
  - { configuredBy: kubernetes-role }
references:
  - kind: code
    role: implementation
    target: deploy/charts/external-secrets/templates/rbac.yaml
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: Roles and responsibilities
---

# Only people a Kubernetes role permits change a generator

A generator or ClusterGenerator is changed only by someone whose Kubernetes roles allow it; the Product itself never changes one.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
