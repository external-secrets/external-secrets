# Metrics

The External Secrets Operator exposes its Prometheus metrics in the `/metrics` path. To enable it, set the `prometheus.enabled` Helm flag to `true`.

The Operator has the metrics inherited from Kubebuilder plus some custom metrics with the `externalsecret` prefix.
