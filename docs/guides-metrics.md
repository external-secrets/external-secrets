# Metrics

The External Secrets Operator exposes its Prometheus metrics in the `/metrics` path. To enable it, set the `serviceMonitor.enabled` Helm flag to `true`. In addition you can also set `webhook.serviceMonitor.enabled=true` and `certController.serviceMonitor.enabled=true` to create `ServiceMonitor` resources for the other components.

The Operator has the metrics inherited from Kubebuilder plus some custom metrics with the `externalsecret` prefix.

## External Secret Metrics

| Name                            | Type    | Description                                        |
| ------------------------------- | ------- | -------------------------------------------------- |
| externalsecret_sync_calls_total | Counter | Total number of the External Secret sync calls     |
| externalsecret_sync_calls_error | Counter | Total number of the External Secret sync errors    |
| externalsecret_status_condition | Gauge   | The status condition of a specific External Secret |
