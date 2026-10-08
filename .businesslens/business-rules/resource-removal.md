---
appliesTo:
  - { type: entity, id: secret-store, effect: removes }
  - { type: entity, id: cluster-secret-store, effect: removes }
  - { type: entity, id: cluster-external-secret, effect: removes }
  - { type: entity, id: cluster-push-secret, effect: removes }
  - { type: entity, id: acr-access-token, effect: removes }
  - { type: entity, id: beyondtrust-workload-credentials-dynamic-secret, effect: removes }
  - { type: entity, id: cloudsmith-access-token, effect: removes }
  - { type: entity, id: ecr-authorization-token, effect: removes }
  - { type: entity, id: fake, effect: removes }
  - { type: entity, id: gcr-access-token, effect: removes }
  - { type: entity, id: github-access-token, effect: removes }
  - { type: entity, id: gitlab-deploy-token, effect: removes }
  - { type: entity, id: grafana, effect: removes }
  - { type: entity, id: mfa, effect: removes }
  - { type: entity, id: password, effect: removes }
  - { type: entity, id: quay-access-token, effect: removes }
  - { type: entity, id: ssh-key, effect: removes }
  - { type: entity, id: sts-session-token, effect: removes }
  - { type: entity, id: uuid, effect: removes }
  - { type: entity, id: vault-dynamic-secret, effect: removes }
  - { type: entity, id: webhook, effect: removes }
  - { type: entity, id: cluster-generator, effect: removes }
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

# Only people a Kubernetes role permits delete stores, cluster-wide resources and generators

A SecretStore, ClusterSecretStore, ClusterExternalSecret, ClusterPushSecret, generator or ClusterGenerator is deleted only by someone whose Kubernetes roles allow deleting it.

## Rationale

The Product authorizes nobody itself: Kubernetes checks every request against the roles bound to the person making it, including the view and edit roles the Product ships.
