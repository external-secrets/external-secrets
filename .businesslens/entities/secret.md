---
domain: core-resources
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.applyOwnership
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_template.go
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_template.go#validateSecretCandidate
  - kind: doc
    role: intent
    target: docs/guides/common-k8s-secret-types.md
---

# Secret

A Kubernetes Secret the Product writes for an ExternalSecret — its target — and
that workloads consume. A PushSecret also reads Secrets as the source of what
it pushes. The Product marks the Secrets it manages so it can tell when one was
changed or deleted behind its back.

## Information kept

- **Data** — the keys and values the workload consumes
- **Type** — the Kubernetes Secret type, such as Opaque, TLS or Docker config
- **Labels and annotations** — metadata rendered from the ExternalSecret's template, plus the Product's own managed marker
- **Owner** — the ExternalSecret that owns it, when its creation policy is Owner
- **Immutable** — whether Kubernetes blocks further changes to its data
- **Service account** — the service account a service-account token Secret is bound to, when its annotations name one
