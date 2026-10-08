---
appliesTo:
  - { type: entity, id: external-secret, effect: reads, contexts: [{ place: metrics }] }
  - { type: entity, id: push-secret, effect: reads, contexts: [{ place: metrics }] }
  - { type: entity, id: cluster-external-secret, effect: reads, contexts: [{ place: metrics }] }
  - { type: entity, id: secret-store, effect: reads, contexts: [{ place: metrics }] }
  - { type: entity, id: cluster-secret-store, effect: reads, contexts: [{ place: metrics }] }
permits:
  - { actors: [monitoring-system], when: [{ entity: controller-options, fact: Metrics authentication, is: Off }] }
  - { configuredBy: kubernetes-role, when: [{ entity: controller-options, fact: Metrics authentication, is: On }] }
references:
  - kind: code
    role: implementation
    target: cmd/controller/root.go
  - kind: doc
    role: intent
    target: docs/api/metrics.md
---

# Anyone who reaches the metrics endpoint reads it, unless metrics authentication is on

While metrics authentication is off, the default, the metrics endpoint answers
every request. While it is on, the endpoint is served over HTTPS only and
answers only callers whose Kubernetes credentials a role allows to read it.

## Rationale

Metrics name every resource the Product processes and whether it is failing,
which some clusters treat as sensitive.
