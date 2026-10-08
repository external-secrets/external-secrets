---
references:
  - kind: doc
    role: context
    target: hack/api-docs/mkdocs.yml
  - kind: doc
    role: intent
    target: docs/guides/generator.md
---

# Generators

The resources of the `generators.external-secrets.io` API group: the generator
kinds that produce passwords, keys, tokens and short-lived credentials, and the
ClusterGenerators that share one generator with every namespace.

## Boundary

It does not own how generated values reach a Secret or a provider; the
ExternalSecrets and PushSecrets that use a generator own that.
