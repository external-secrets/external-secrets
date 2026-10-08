---
appliesTo:
  - { type: entity, id: secret-store, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: cluster-secret-store, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: external-secret, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: cluster-external-secret, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: push-secret, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: cluster-push-secret, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: secret, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: generic-target, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: acr-access-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: beyondtrust-workload-credentials-dynamic-secret, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: cloudsmith-access-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: ecr-authorization-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: fake, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: gcr-access-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: github-access-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: gitlab-deploy-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: grafana, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: mfa, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: password, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: quay-access-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: ssh-key, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: sts-session-token, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: uuid, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: vault-dynamic-secret, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: webhook, effect: reads, contexts: [{ place: kubernetes-api }] }
  - { type: entity, id: cluster-generator, effect: reads, contexts: [{ place: kubernetes-api }] }
permits:
  - { configuredBy: kubernetes-role }
  - { unattended: true }
references:
  - kind: code
    role: implementation
    target: deploy/charts/external-secrets/templates/rbac.yaml
  - kind: doc
    role: context
    target: docs/guides/multi-tenancy.md
---

# Only people a Kubernetes role permits read the Product's resources and target Secrets

Through the Kubernetes API, a store, ExternalSecret, PushSecret, cluster-wide
resource, generator, target Secret or generic target is read only by someone
whose Kubernetes roles allow reading it. The Product itself reads them with its
own permissions as it reconciles.

## Rationale

Reading a resource's status, or the Secret it writes, is governed by the
cluster's roles like every other Kubernetes read; the view role the Product
ships covers its own resources but not Secrets.
