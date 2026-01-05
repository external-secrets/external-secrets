```yaml
---
title: Centralized Metrics Registry
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-01-05
status: draft
---
```

# Centralized Metrics Registry

## Table of Contents

<!-- toc -->
<!-- /toc -->

## Summary

Consolidate the distributed controller metrics definitions across 8 packages into a centralized metrics registry with type-safe accessors. Provider metrics remain decoupled, allowing providers to be compiled independently and removed without impacting ESO core.

## Motivation

The External Secrets Operator defines controller metrics in 8 separate packages with significant duplication:

| Package         | Location                                            | Metrics Defined               |
|-----------------|-----------------------------------------------------|-------------------------------|
| esmetrics       | `pkg/controllers/externalsecret/esmetrics/`         | 4 metrics                     |
| psmetrics       | `pkg/controllers/pushsecret/psmetrics/`             | 2 metrics                     |
| cesmetrics      | `pkg/controllers/clusterexternalsecret/cesmetrics/` | 2 metrics                     |
| cpsmetrics      | `pkg/controllers/clusterpushsecret/cpsmetrics/`     | 2 metrics                     |
| ssmetrics       | `pkg/controllers/secretstore/ssmetrics/`            | 2 metrics                     |
| cssmetrics      | `pkg/controllers/secretstore/cssmetrics/`           | 2 metrics                     |
| commonmetrics   | `pkg/controllers/secretstore/metrics/`              | 1 helper                      |
| runtime/metrics | `runtime/metrics/`                                  | 1 metric (provider API calls) |

**Key problems:**

1. **Code duplication**: 7 nearly identical `status_condition` gauge definitions
2. **Code duplication**: 6 nearly identical `reconcile_duration` gauge definitions
3. **Code duplication**: 4 identical `RemoveMetrics()` function implementations
4. **No type safety**: Metrics accessed by string keys, risking typos at runtime
5. **No central catalog**: Users cannot easily discover all available metrics

### Goals

1. Create a single metrics registry for **controller metrics only**
2. Provide type-safe accessor functions (no string key lookups)
3. Eliminate duplicated metric definitions and helper functions
4. Keep provider metrics decoupled from ESO core
5. Maintain 100% backward compatibility for metric names and labels

### Non-Goals

- Changing metric names (would break dashboards)
- Changing label names or cardinality
- Centralizing provider-specific metrics (providers own their metrics)
- Adding new metrics (separate effort)

## Proposal

### Architecture: Controller vs Provider Metrics

**Clear separation of concerns:**

```
┌─────────────────────────────────────────────────────────────────┐
│                        ESO Core                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Controller Metrics Registry                │    │
│  │  • externalsecret_sync_calls_total                      │    │
│  │  • externalsecret_status_condition                      │    │
│  │  • pushsecret_reconcile_duration                        │    │
│  │  • secretstore_status_condition                         │    │
│  │  • ... (14 controller metrics total)                    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │           Generic Provider API Metric                   │    │
│  │  • externalsecret_provider_api_calls_count              │    │
│  │    Labels: provider, call, status                       │    │
│  │    (Provider name is a label, not hardcoded)            │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────┐
│                    Providers (Separate Modules)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │    Vault     │  │     AWS      │  │   Custom     │             │
│  │  (optional)  │  │  (optional)  │  │  (user-built)│             │
│  │              │  │              │  │              │             │
│  │ Can register │  │ Can register │  │ Can register │             │
│  │ own metrics  │  │ own metrics  │  │ own metrics  │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
│                                                                   │
│  Providers call: metrics.ObserveAPICall("vault", "GetSecret", err)│
│  ESO core doesn't know about specific providers                   │
└───────────────────────────────────────────────────────────────────┘
```

ESO core metrics registry will not contain knowledge of specific providers. Providers can:
- Be compiled in or out via build tags
- Be removed without changing ESO core
- Register their own custom metrics independently
- Be built by users as external modules

### Phase 1: Create Controller Metrics Registry

Create a centralized package for controller metrics only:

**Location:** `pkg/metrics/registry.go`

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Subsystems for controller metrics
const (
    SubsystemExternalSecret        = "externalsecret"
    SubsystemPushSecret            = "pushsecret"
    SubsystemClusterExternalSecret = "clusterexternalsecret"
    SubsystemClusterPushSecret     = "clusterpushsecret"
    SubsystemSecretStore           = "secretstore"
    SubsystemClusterSecretStore    = "clustersecretstore"
)

// Metric names
const (
    MetricSyncCallsTotal    = "sync_calls_total"
    MetricSyncCallsError    = "sync_calls_error"
    MetricStatusCondition   = "status_condition"
    MetricReconcileDuration = "reconcile_duration"
)

// ControllerRegistry holds all ESO controller metrics
// NOTE: This registry intentionally excludes provider-specific metrics.
// Providers register their own metrics independently.
type ControllerRegistry struct {
    // ExternalSecret metrics
    ESSyncCallsTotal    *prometheus.CounterVec
    ESSyncCallsError    *prometheus.CounterVec
    ESStatusCondition   *prometheus.GaugeVec
    ESReconcileDuration *prometheus.GaugeVec

    // PushSecret metrics
    PSStatusCondition   *prometheus.GaugeVec
    PSReconcileDuration *prometheus.GaugeVec

    // ClusterExternalSecret metrics
    CESStatusCondition   *prometheus.GaugeVec
    CESReconcileDuration *prometheus.GaugeVec

    // ClusterPushSecret metrics
    CPSStatusCondition   *prometheus.GaugeVec
    CPSReconcileDuration *prometheus.GaugeVec

    // SecretStore metrics
    SSStatusCondition   *prometheus.GaugeVec
    SSReconcileDuration *prometheus.GaugeVec

    // ClusterSecretStore metrics
    CSSStatusCondition   *prometheus.GaugeVec
    CSSReconcileDuration *prometheus.GaugeVec
}

var globalRegistry *ControllerRegistry

// Initialize creates and registers all controller metrics.
// Call once at startup from cmd/controller/root.go.
func Initialize(extendedLabels bool) *ControllerRegistry {
    if globalRegistry != nil {
        return globalRegistry
    }

    labels := buildLabelNames(extendedLabels)

    globalRegistry = &ControllerRegistry{
        // ExternalSecret - the only controller with sync counters
        ESSyncCallsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Subsystem: SubsystemExternalSecret,
                Name:      MetricSyncCallsTotal,
                Help:      "Total number of External Secret sync calls",
            },
            labels.NonCondition,
        ),
        ESSyncCallsError: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Subsystem: SubsystemExternalSecret,
                Name:      MetricSyncCallsError,
                Help:      "Total number of External Secret sync errors",
            },
            labels.NonCondition,
        ),
        ESStatusCondition:   newStatusConditionGauge(SubsystemExternalSecret, labels.Condition),
        ESReconcileDuration: newReconcileDurationGauge(SubsystemExternalSecret, labels.NonCondition),

        // PushSecret
        PSStatusCondition:   newStatusConditionGauge(SubsystemPushSecret, labels.Condition),
        PSReconcileDuration: newReconcileDurationGauge(SubsystemPushSecret, labels.NonCondition),

        // ClusterExternalSecret
        CESStatusCondition:   newStatusConditionGauge(SubsystemClusterExternalSecret, labels.Condition),
        CESReconcileDuration: newReconcileDurationGauge(SubsystemClusterExternalSecret, labels.NonCondition),

        // ClusterPushSecret
        CPSStatusCondition:   newStatusConditionGauge(SubsystemClusterPushSecret, labels.Condition),
        CPSReconcileDuration: newReconcileDurationGauge(SubsystemClusterPushSecret, labels.NonCondition),

        // SecretStore
        SSStatusCondition:   newStatusConditionGauge(SubsystemSecretStore, labels.Condition),
        SSReconcileDuration: newReconcileDurationGauge(SubsystemSecretStore, labels.NonCondition),

        // ClusterSecretStore
        CSSStatusCondition:   newStatusConditionGauge(SubsystemClusterSecretStore, labels.Condition),
        CSSReconcileDuration: newReconcileDurationGauge(SubsystemClusterSecretStore, labels.NonCondition),
    }

    // Register all controller metrics
    ctrlmetrics.Registry.MustRegister(
        globalRegistry.ESSyncCallsTotal,
        globalRegistry.ESSyncCallsError,
        globalRegistry.ESStatusCondition,
        globalRegistry.ESReconcileDuration,
        globalRegistry.PSStatusCondition,
        globalRegistry.PSReconcileDuration,
        globalRegistry.CESStatusCondition,
        globalRegistry.CESReconcileDuration,
        globalRegistry.CPSStatusCondition,
        globalRegistry.CPSReconcileDuration,
        globalRegistry.SSStatusCondition,
        globalRegistry.SSReconcileDuration,
        globalRegistry.CSSStatusCondition,
        globalRegistry.CSSReconcileDuration,
    )

    return globalRegistry
}

// Get returns the global controller registry.
// Panics if Initialize() was not called.
func Get() *ControllerRegistry {
    if globalRegistry == nil {
        panic("metrics.Initialize() must be called before metrics.Get()")
    }
    return globalRegistry
}

// Helper functions eliminate duplication
func newStatusConditionGauge(subsystem string, labels []string) *prometheus.GaugeVec {
    return prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Subsystem: subsystem,
            Name:      MetricStatusCondition,
            Help:      "The status condition of the resource",
        },
        labels,
    )
}

func newReconcileDurationGauge(subsystem string, labels []string) *prometheus.GaugeVec {
    return prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Subsystem: subsystem,
            Name:      MetricReconcileDuration,
            Help:      "The duration time to reconcile the resource",
        },
        labels,
    )
}
```

### Phase 2: Provider API Metric (Unchanged)

The generic provider API metric stays in `runtime/metrics/` and remains provider-agnostic:

```go
// runtime/metrics/metrics.go - NO CHANGES NEEDED

var providerAPICalls = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Subsystem: "externalsecret",
        Name:      "provider_api_calls_count",
        Help:      "Number of API calls towards the secret provider",
    },
    []string{"provider", "call", "status"},  // Provider is a LABEL
)

func init() {
    metrics.Registry.MustRegister(providerAPICalls)
}

// ObserveAPICall records a provider API call.
// Providers call this with their name - ESO core doesn't know provider details.
func ObserveAPICall(provider, call string, err error) {
    status := "success"
    if err != nil {
        status = "error"
    }
    providerAPICalls.WithLabelValues(provider, call, status).Inc()
}
```

- When Vault provider is compiled in, it calls `ObserveAPICall("vault", "ReadSecret", err)`
- When AWS provider is compiled in, it calls `ObserveAPICall("aws", "GetSecretValue", err)`
- If a provider is removed, its label values simply stop appearing
- Custom providers can use any name they want

### Phase 3: Type-Safe Accessor Methods

Add helper methods for common operations:

```go
// pkg/metrics/helpers.go

// Labels provides pre-built label maps for metrics
type Labels map[string]string

// ForExternalSecret creates labels for ExternalSecret metrics
func ForExternalSecret(namespace, name string) Labels {
    return Labels{
        "namespace": namespace,
        "name":      name,
    }
}

// ForSecretStore creates labels for SecretStore metrics
func ForSecretStore(namespace, name string) Labels {
    return Labels{
        "namespace": namespace,
        "name":      name,
    }
}

// RecordESSyncCall records a sync call for ExternalSecret
func (r *ControllerRegistry) RecordESSyncCall(labels Labels) {
    r.ESSyncCallsTotal.With(prometheus.Labels(labels)).Inc()
}

// RecordESSyncError records a sync error for ExternalSecret
func (r *ControllerRegistry) RecordESSyncError(labels Labels) {
    r.ESSyncCallsError.With(prometheus.Labels(labels)).Inc()
}

// RecordESReconcileDuration records reconciliation duration
func (r *ControllerRegistry) RecordESReconcileDuration(labels Labels, seconds float64) {
    r.ESReconcileDuration.With(prometheus.Labels(labels)).Set(seconds)
}

// SetESCondition sets the status condition metric for ExternalSecret
func (r *ControllerRegistry) SetESCondition(labels Labels, condition, status string, value float64) {
    l := prometheus.Labels(labels)
    l["condition"] = condition
    l["status"] = status
    r.ESStatusCondition.With(l).Set(value)
}

// RemoveESMetrics removes all metrics for a deleted ExternalSecret
func (r *ControllerRegistry) RemoveESMetrics(labels Labels) {
    pl := prometheus.Labels(labels)
    r.ESSyncCallsTotal.Delete(pl)
    r.ESSyncCallsError.Delete(pl)
    r.ESReconcileDuration.Delete(pl)
    // Note: status_condition requires iterating conditions
}

// Similar methods for PS, CES, CPS, SS, CSS...
```

### Phase 4: Migrate Controllers

Update controllers to use the central registry:

**Before (distributed, string-based):**
```go
// pkg/controllers/externalsecret/externalsecret_controller.go
import "github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // String-based lookup - typos cause runtime errors
    gauge := esmetrics.GetGaugeVec(esmetrics.ExternalSecretReconcileDurationKey)
    gauge.With(labels).Set(duration)
}
```

**After (centralized, type-safe):**
```go
// pkg/controllers/externalsecret/externalsecret_controller.go
import "github.com/external-secrets/external-secrets/pkg/metrics"

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    labels := metrics.ForExternalSecret(req.Namespace, req.Name)

    // Type-safe - compiler catches errors
    metrics.Get().RecordESReconcileDuration(labels, duration)
}
```

### Phase 5: Provider Custom Metrics (Optional Pattern)

If providers want their own metrics beyond `provider_api_calls_count`, they register independently:

```go
// providers/v1/vault/metrics.go

var (
    vaultTokenCacheHits = prometheus.NewCounter(
        prometheus.CounterOpts{
            Subsystem: "vault",
            Name:      "token_cache_hits_total",
            Help:      "Number of Vault token cache hits",
        },
    )
    vaultTokenCacheMisses = prometheus.NewCounter(
        prometheus.CounterOpts{
            Subsystem: "vault",
            Name:      "token_cache_misses_total",
            Help:      "Number of Vault token cache misses",
        },
    )
)

func init() {
    // Provider registers its own metrics
    // ESO core has no knowledge of these
    ctrlmetrics.Registry.MustRegister(
        vaultTokenCacheHits,
        vaultTokenCacheMisses,
    )
}
```

This has multiple advantages:

- Provider removal removes its metrics automatically
- Users building custom providers can add metrics
- No ESO core changes needed for provider metric changes

At the same time, it risks us not having an oversight of the metrics in each provider.
We might want to clarify that to our users.

### Resulting Package Structure

```
pkg/metrics/                              # NEW: Controller metrics registry
├── registry.go                           # Central metric definitions
├── helpers.go                            # Type-safe accessor methods
├── labels.go                             # Label management (moved from pkg/controllers/metrics/)
└── doc.go                                # Package documentation

runtime/metrics/                          # UNCHANGED: Generic provider API metric
└── metrics.go                            # ObserveAPICall() - provider-agnostic

providers/v1/vault/                       # OPTIONAL: Provider-specific metrics
└── metrics.go                            # Vault-specific metrics (if needed)

pkg/controllers/externalsecret/esmetrics/ # DEPRECATED: Old package
└── esmetrics.go                          # Forwards to pkg/metrics, removed in v1.x or v2
```

### User Stories

**As an operator**, I want to find all controller metrics in one place, so I can build comprehensive dashboards.

**As a provider maintainer**, I want to add provider-specific metrics without changing ESO core, so I can iterate independently.

**As a user building a custom provider**, I want my provider to work without forking ESO, so I can maintain it separately.

**As a maintainer**, I want to remove an unmaintained provider without breaking metrics, so cleanup is safe.

### API

No API changes.

### Behavior

No metric name or label changes. Existing dashboards and alerts continue to work.

### Drawbacks

1. **Migration effort**: All controllers need updates
   - **Mitigation**: Incremental migration with deprecated shims
2. **Two metric locations**: Controller metrics in `pkg/metrics/`, provider API metric in `runtime/metrics/`
   - **Mitigation**: Clear documentation; separation is intentional for decoupling

### Acceptance Criteria

**Phase 1 (Controller Registry)**:
- [ ] All 14 controller metrics defined in `pkg/metrics/registry.go`
- [ ] Single `Initialize()` call replaces 6 `SetUpMetrics()` calls
- [ ] Metrics output identical to current implementation

**Phase 2 (Provider API Metric)**:
- [ ] `runtime/metrics/metrics.go` unchanged
- [ ] Documentation clarifies the separation

**Phase 3 (Type-Safe Accessors)**:
- [ ] Helper methods for all metric operations
- [ ] No string-based metric key lookups in controller code

**Phase 4 (Migration)**:
- [ ] All controllers migrated to central registry
- [ ] Old packages marked deprecated
- [ ] Test coverage maintained

**Phase 5 (Provider Pattern)**:
- [ ] Documentation for provider-specific metrics
- [ ] Example in one provider (e.g., Vault cache metrics)

**Rollout:**
- Phase 1-3: Internal refactoring, no user impact
- Phase 4: Incremental controller migration
- Phase 5: Documentation and examples

At this point, we might even consider to generate the docs around our metrics.

**Monitoring:**
- Verify metric output unchanged via e2e tests
- Compare Prometheus scrape output before/after

## Alternatives

### Alternative 1: Centralize All Metrics Including Providers

Put provider-specific metrics in the central registry.

**Pros**: Single location for all metrics
**Cons**: Couples ESO core to providers, breaks provider independence, can't remove providers cleanly

**Decision**: Rejected - violates provider decoupling goal

### Alternative 2: Keep Current Structure

Accept the distributed controller metrics.

**Pros**: No migration effort
**Cons**: Continued duplication, maintenance burden, string-based typos

**Decision**: Rejected - duplication is already causing inconsistencies

### Alternative 3: Generic Metrics Interface

Define a `MetricsRecorder` interface each controller implements.

**Pros**: Flexibility
**Cons**: Still duplicates implementations across controllers

**Decision**: Rejected - doesn't address root cause of duplication

## Compatibility with Other Designs

- **design/007-provider-versioning-strategy**: When providers become separate CRDs, the provider API metric (`provider_api_calls_count`) continues to work unchanged. Providers register additional metrics in their own CRD controllers if needed.

- **design/013-controller-responsibility-decomposition**: The ConditionReporter component will use `metrics.Get().SetESCondition()` for status condition metrics, providing a clean integration point.
