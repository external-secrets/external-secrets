---
hide:
  - toc
---

# Metrics

The External Secrets Operator exposes its Prometheus metrics in the `/metrics` path. To enable it, set the `serviceMonitor.enabled` Helm flag to `true`. In addition you can also set `webhook.serviceMonitor.enabled=true` and `certController.serviceMonitor.enabled=true` to create `ServiceMonitor` resources for the other components.

If you are using a different monitoring tool that also needs a `/metrics` endpoint, you can set the `metrics.service.enabled` Helm flag to `true`. In addition you can also set `webhook.metrics.service.enabled` and `certController.metrics.service.enabled` to scrape the other components.

The Operator has the metrics inherited from Kubebuilder plus some custom metrics with the `externalsecret` prefix.

## External Secret Metrics

| Name                                           | Type      | Description                                                                                                                                                                                                            |
| ---------------------------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `externalsecret_sync_calls_total`              | Counter   | Total number of the External Secret sync calls                                                                                                                                                                         |
| `externalsecret_sync_calls_error`              | Counter   | Total number of the External Secret sync errors                                                                                                                                                                        |
| `externalsecret_status_condition`              | Gauge     | The status condition of a specific External Secret                                                                                                                                                                     |
| `externalsecret_reconcile_duration`            | Gauge     | The duration time to reconcile the External Secret                                                                                                                                                                     |
| `controller_runtime_reconcile_total`           | Counter   | Holds the totalnumber of reconciliations per controller. It has two labels. controller label refers to the controller name and result label refers to the reconcile result i.e success, error, requeue, requeue_after. |
| `controller_runtime_reconcile_errors_total`    | Counter   | Total number of reconcile errors per controller                                                                                                                                                                        |
| `controller_runtime_reconcile_time_seconds`    | Histogram | Length of time per reconcile per controller                                                                                                                                                                            |
| `controller_runtime_reconcile_queue_length`    | Gauge     | Length of reconcile queue per controller                                                                                                                                                                               |
| `controller_runtime_max_concurrent_reconciles` | Gauge     | Maximum number of concurrent reconciles per controller                                                                                                                                                                 |
| `controller_runtime_active_workers`            | Gauge     | Number of currently used workers per controller                                                                                                                                                                        |
