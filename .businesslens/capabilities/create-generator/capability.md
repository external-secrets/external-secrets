---
domain: generators
availability:
  - { place: kubernetes-api::namespace }
references:
  - kind: code
    role: implementation
    target: apis/generators/v1alpha1/generator_types.go
  - kind: code
    role: implementation
    target: pkg/register/generators.go
  - kind: doc
    role: intent
    target: docs/guides/generator.md
  - kind: doc
    role: intent
    target: docs/api/generator/index.md
---

# Create a generator

Declare, inside a namespace, how new values are produced: random passwords,
UUIDs, SSH keys and one-time codes; registry tokens for Azure, AWS, Google and
Quay; short-lived AWS, GitHub, GitLab, Cloudsmith, Grafana, Vault and
BeyondTrust credentials; values returned by an HTTP endpoint; or fixed values
for testing. A generator produces nothing on its own: ExternalSecrets and
PushSecrets in the namespace name it, and it produces fresh values each time
they sync.
