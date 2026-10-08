---
id: external-secrets-operator
summary: A Kubernetes operator that keeps Kubernetes Secrets in sync with external secret managers, pushes Secrets back to them and generates credentials on demand.
category: secrets-management
tags: [kubernetes, operator, secrets, gitops]
authors:
  - name: The External Secrets Authors
    url: https://github.com/external-secrets/external-secrets
license: Apache-2.0
languages: [en]
limitations:
  - The Product pulls from secret providers on a schedule or on request; it never watches a provider for changes.
  - Kubernetes authenticates everyone who uses the Product, and Kubernetes RBAC decides who may create, read, change or delete its resources; the Product keeps no accounts of its own and ships view and edit roles.
  - The Product writes target Secrets, provider secrets and generic targets with its own permissions, not with the permissions of the person who declared them.
  - A shared store gives every namespace it admits the same access to the provider; limiting which keys a namespace may read is left to the provider's own access control and to Kubernetes admission control.
  - Whether a provider operation succeeds is decided by the provider and by the credentials a store gives it.
  - Rotating secrets inside a provider is left to the provider; the Product writes to a provider only what a PushSecret pushes.
  - A generator produces new values every time it is used; earlier values are not kept.
  - Credentials issued by a VaultDynamicSecret are never revoked by the Product.
references:
  - kind: doc
    role: intent
    target: docs/introduction/overview.md
    title: API overview, roles and responsibilities
  - kind: doc
    role: context
    target: README.md
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
    title: Lifecycle of synced Secrets
  - kind: doc
    role: context
    target: docs/guides/multi-tenancy.md
  - kind: doc
    role: intent
    target: docs/api/controller-options.md
---

# External Secrets Operator

External Secrets Operator extends Kubernetes with resources that say where
secrets live and how to bring them into a cluster. A SecretStore or
ClusterSecretStore says how to reach and authenticate to an external secret
provider — AWS Secrets Manager, HashiCorp Vault, Google Secret Manager, Azure
Key Vault and dozens more. An ExternalSecret says which values to fetch and the
Product writes them into a Kubernetes Secret, keeping it in sync as the
provider changes. A PushSecret works the other way, writing a Kubernetes
Secret's values into one or more providers. Generators produce values the
Product cannot fetch — passwords, registry tokens, short-lived cloud
credentials — for either direction.

Cluster operators install the Product, choose its controller options, connect
it to providers and publish cluster-wide stores, secrets and generators;
application developers declare the secrets their workloads need inside their
own namespaces. Monitoring systems read the Product's metrics, and the esoctl
tool lets developers try templates before applying them.

## Intent

Let teams keep secrets in the provider their organisation already trusts while
workloads consume ordinary Kubernetes Secrets, with the lifecycle of every
synced Secret — creation, refresh, ownership and deletion — declared on the
resource that asks for it.
