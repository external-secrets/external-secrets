---
references:
  - kind: doc
    role: context
    target: hack/api-docs/mkdocs.yml
---

# Core Resources

The resources of the `external-secrets.io` API group: the stores that connect
the Product to secret providers, the ExternalSecrets and ClusterExternalSecrets
that bring provider secrets into the cluster, the PushSecrets and
ClusterPushSecrets that send Secrets out, and the Secrets and other targets the
Product writes.

## Boundary

It does not own generators or the values they produce; those belong to
Generators. It does not own the provider secrets themselves, which live in the
providers.
