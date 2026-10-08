---
appliesTo:
  - { type: entity, id: secret-store, effect: creates }
  - { type: entity, id: cluster-secret-store, effect: creates }
  - { type: entity, id: cluster-external-secret, effect: creates }
  - { type: entity, id: cluster-push-secret, effect: creates }
  - { type: entity, id: acr-access-token, effect: creates }
  - { type: entity, id: beyondtrust-workload-credentials-dynamic-secret, effect: creates }
  - { type: entity, id: cloudsmith-access-token, effect: creates }
  - { type: entity, id: ecr-authorization-token, effect: creates }
  - { type: entity, id: fake, effect: creates }
  - { type: entity, id: gcr-access-token, effect: creates }
  - { type: entity, id: github-access-token, effect: creates }
  - { type: entity, id: gitlab-deploy-token, effect: creates }
  - { type: entity, id: grafana, effect: creates }
  - { type: entity, id: mfa, effect: creates }
  - { type: entity, id: password, effect: creates }
  - { type: entity, id: quay-access-token, effect: creates }
  - { type: entity, id: ssh-key, effect: creates }
  - { type: entity, id: sts-session-token, effect: creates }
  - { type: entity, id: uuid, effect: creates }
  - { type: entity, id: vault-dynamic-secret, effect: creates }
  - { type: entity, id: webhook, effect: creates }
  - { type: entity, id: cluster-generator, effect: creates }
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

# Only people a Kubernetes role permits create stores, cluster-wide resources and generators

A SecretStore, ClusterSecretStore, ClusterExternalSecret, ClusterPushSecret, generator or ClusterGenerator is created only by someone whose Kubernetes roles allow creating that kind of resource in that namespace, or cluster-wide for cluster-wide kinds.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
