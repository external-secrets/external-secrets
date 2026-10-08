---
kind: primary
routes:
  metrics: Metrics
steps:
  - text: The Monitoring System requests the metrics endpoint
    kind: actor
    actor: monitoring-system
    entities: []
    contexts: { metrics: { place: metrics } }
  - text: The Product returns the status condition of every ExternalSecret, PushSecret and store it processes, with its sync counts and durations
    kind: product
    actor: monitoring-system
    entities:
      - { entity: external-secret, effect: reads, facts: [Ready] }
      - { entity: push-secret, effect: reads, facts: [Ready] }
      - { entity: cluster-external-secret, effect: reads, facts: [Ready] }
      - { entity: secret-store, effect: reads, facts: [Ready] }
      - { entity: cluster-secret-store, effect: reads, facts: [Ready] }
    contexts: { metrics: { place: metrics } }
---

# Scrape the metrics

## Trigger

A monitoring system polls the Product to alert on failing syncs.

## Outcome

The monitoring system holds current counts, durations and conditions for every
resource the Product processes.

## Edge cases

- With metrics authentication on, a request without Kubernetes credentials allowed to read the endpoint is refused.
- With extended metric labels on, the resource's recommended Kubernetes labels are added to its metrics.
