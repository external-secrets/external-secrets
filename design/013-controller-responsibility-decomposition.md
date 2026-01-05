```yaml
---
title: Controller Responsibility Decomposition
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-01-05
status: draft
---
```

# Controller Responsibility Decomposition

## Table of Contents

<!-- toc -->
<!-- /toc -->

## Summary

Incrementally decompose the ExternalSecret controller from a monolithic 2,500+ line implementation into focused, single-responsibility components. This proposal maintains full backward compatibility while improving testability, reducing cognitive load, and enabling safer changes.

## Motivation

The ExternalSecret controller has grown organically to handle multiple concerns within a single reconciliation loop. The main `Reconcile()` function spans 450 lines with 12+ distinct responsibilities, nested closures, and duplicated logic between the Secret and Generic Target paths.

**Current state:**
- `externalsecret_controller.go`: 1,250 lines
- `externalsecret_controller_secret.go`: 292 lines
- `externalsecret_controller_manifest.go`: 295 lines
- `externalsecret_controller_template.go`: 159 lines

**Key problems:**
- Hard to maintain: Changes to one concern (e.g., refresh policy) can inadvertently break another (e.g., secret mutation)
- Code duplication not explained: The Secret path and Generic Target path duplicate ~100 lines of policy evaluation logic - Why did we do it? Is that the best solution?
- Difficult to Test as we must test a full reconciliation setup even for isolated logic
- I like spaghetti but at some point I am afraid the meal won't look good.

### Goals

1. Extract clearly-defined responsibilities into five components with explicit interfaces
2. Unify Secret and Generic Target paths behind a common `TargetWriter` interface
3. Reduce the main `Reconcile()` function to orchestration-only logic (~100-150 lines)
4. Enable independent testing of each component
5. Maintain 100% backward compatibility - no API changes, no behavior changes

### Non-Goals

- Changing the ExternalSecret API or behavior
- Splitting into multiple controllers or CRDs
- Addressing test file organization (separate effort)
- Performance optimization (orthogonal concern)

## Proposal

### Target Architecture

The controller will be decomposed into five components:

```
┌─────────────────────────────────────────────────────────────────┐
│                    ExternalSecret Reconciler                    │
│                      (Orchestration Only)                       │
└─────────────────────────────────────────────────────────────────┘
        │           │              │            │           │
        ▼           ▼              ▼            ▼           ▼
┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
│   Sync    │ │  Target   │ │  Secret   │ │  Target   │ │ Condition │
│ Scheduler │ │  Policy   │ │   Data    │ │  Writer   │ │ Reporter  │
│           │ │ Evaluator │ │  Fetcher  │ │ Interface │ │           │
└───────────┘ └───────────┘ └───────────┘ └───────────┘ └───────────┘
                                               │
                                    ┌──────────┴──────────┐
                                    ▼                     ▼
                             ┌────────────┐        ┌────────────┐
                             │   Secret   │        │  Manifest  │
                             │   Writer   │        │   Writer   │
                             └────────────┘        └────────────┘
```

### Component 1: SyncScheduler

**Purpose:** Decides *when* to reconcile based on refresh intervals and secret state.

**Location:** `pkg/controllers/externalsecret/sync/scheduler.go`

**Interface:**
```go
type SyncScheduler interface {
    // ShouldSync determines if the ExternalSecret needs synchronization
    ShouldSync(es *esv1.ExternalSecret, target client.Object) bool

    // GetRequeueAfter returns the duration until next reconciliation
    GetRequeueAfter(es *esv1.ExternalSecret, syncErr error) time.Duration
}
```

**Extracted functions:**
- `shouldRefresh()` (externalsecret_controller.go:1087-1108)
- `shouldRefreshPeriodic()` (externalsecret_controller.go:1110-1133)
- `isSecretValid()` (externalsecret_controller.go:1136-1158)
- `getRequeueResult()` (externalsecret_controller.go:723-759)

**Benefits:**
- Refresh logic testable without Kubernetes client
- Clear boundary for refresh policy changes
- Reusable across future controllers (e.g., PushSecret)

### Component 2: TargetPolicyEvaluator

**Purpose:** Decides *how* to handle the target based on creation and deletion policies.

**Location:** `pkg/controllers/externalsecret/policy/evaluator.go`

**Interface:**
```go
type TargetAction int

const (
    ActionCreate TargetAction = iota
    ActionUpdate
    ActionMerge
    ActionDelete
    ActionOrphan
    ActionSkip
)

type TargetPolicyEvaluator interface {
    // EvaluateCreation determines what action to take for target creation/update
    EvaluateCreation(es *esv1.ExternalSecret, existing client.Object, hasData bool) TargetAction

    // EvaluateDeletion determines what action to take when data is empty
    EvaluateDeletion(es *esv1.ExternalSecret, existing client.Object) TargetAction
}
```

**Extracted logic:**
- Creation policy switch (externalsecret_controller.go:534-572 and 656-694)
- Deletion policy switch (externalsecret_controller.go:406-438 and 618-647)

**Benefits:**
- Eliminates ~100 lines of duplicated switch statements
- Policy logic testable with simple unit tests
- Single place to modify policy behavior

### Component 3: SecretDataFetcher

**Purpose:** Gets data *from* external sources (providers, generators, extract/find).

**Location:** `pkg/controllers/externalsecret/data/fetcher.go`

**Interface:**
```go
type SecretDataFetcher interface {
    // Fetch retrieves secret data from all configured sources
    Fetch(ctx context.Context, es *esv1.ExternalSecret) (map[string][]byte, error)
}
```

**Extracted functions:**
- `GetProviderSecretData()` (externalsecret_controller_secret.go:42-121)
- `handleSecretData()` (externalsecret_controller_secret.go:123-145)
- `handleGenerateSecrets()` (externalsecret_controller_secret.go:156-193)
- `handleExtractSecrets()` (externalsecret_controller_secret.go:202-241)
- `handleFindAllSecrets()` (externalsecret_controller_secret.go:243-282)

**Dependencies:**
- `secretstore.Manager` for provider client creation
- `statemanager.Manager` for generator state (when enabled)

**Benefits:**
- Data fetching testable with mock providers
- Clear integration point for design/007-provider-versioning-strategy
- Encapsulates GeneratorState complexity (design/011-generator-state)

### Component 4: TargetWriter (Interface + Implementations)

**Purpose:** Manages the *output* Kubernetes resource with a unified interface.

**Location:** `pkg/controllers/externalsecret/target/writer.go`

**Interface:**
```go
type TargetWriter interface {
    // Get retrieves the current target resource, or nil if not found
    Get(ctx context.Context, es *esv1.ExternalSecret) (client.Object, error)

    // Create creates a new target resource with the provided data
    Create(ctx context.Context, es *esv1.ExternalSecret, data map[string][]byte) (client.Object, error)

    // Update updates an existing target resource with new data
    Update(ctx context.Context, es *esv1.ExternalSecret, existing client.Object, data map[string][]byte) error

    // Delete removes the target resource
    Delete(ctx context.Context, es *esv1.ExternalSecret, existing client.Object) error

    // DeleteOrphaned removes targets that are no longer referenced
    DeleteOrphaned(ctx context.Context, es *esv1.ExternalSecret) error
}
```

**Implementation 1: SecretWriter**

**Location:** `pkg/controllers/externalsecret/target/secret_writer.go`

Handles standard `corev1.Secret` targets. Extracted functions:
- `createSecret()` (externalsecret_controller.go:858-886)
- `updateSecret()` (externalsecret_controller.go:888-962)
- `deleteOrphanedSecrets()` (externalsecret_controller.go:826-855)
- `ApplyTemplate()` (externalsecret_controller_template.go:40-108)
- `setMetadata()` (externalsecret_controller_template.go:111-159)

Handles:
- Owner reference management
- Immutability checking
- Template application
- Field manager tracking

**Implementation 2: ManifestWriter**

**Location:** `pkg/controllers/externalsecret/target/manifest_writer.go`

Handles generic `unstructured.Unstructured` targets. Extracted functions:
- `createGenericResource()` (externalsecret_controller_manifest.go:105-132)
- `updateGenericResource()` (externalsecret_controller_manifest.go:134-145)
- `deleteGenericResource()` (externalsecret_controller_manifest.go:148-168)
- `applyTemplateToManifest()` (externalsecret_controller_manifest.go:171-201)
- `renderTemplatedManifest()` (externalsecret_controller_manifest.go:231-295)

**Factory:**
```go
func NewTargetWriter(client client.Client, es *esv1.ExternalSecret) TargetWriter {
    if es.Spec.Target.Template != nil && es.Spec.Target.Template.Metadata.Kind != "" {
        return NewManifestWriter(client)
    }
    return NewSecretWriter(client)
}
```

**Benefits:**
- Unified target handling eliminates duplicated reconciliation paths
- Easy to add new target types in the future
- Each implementation testable independently

### Component 5: ConditionReporter

**Purpose:** Reports *status* back to users via conditions and events.

**Location:** `pkg/controllers/externalsecret/status/reporter.go`

**Interface:**
```go
type ConditionReporter interface {
    // ReportSynced marks the ExternalSecret as successfully synced
    ReportSynced(ctx context.Context, es *esv1.ExternalSecret, target client.Object) error

    // ReportFailed marks the ExternalSecret as failed with error details
    ReportFailed(ctx context.Context, es *esv1.ExternalSecret, err error) error

    // ReportDeleting marks the ExternalSecret as being deleted
    ReportDeleting(ctx context.Context, es *esv1.ExternalSecret) error
}
```

**Extracted functions:**
- `markAsDone()` (externalsecret_controller.go:761-786)
- `markAsFailed()` (externalsecret_controller.go:788-793)
- `NewExternalSecretCondition()` (util.go:31-39)
- `SetExternalSecretCondition()` (util.go:53-74)
- `GetExternalSecretCondition()` (util.go:42-49)

**Benefits:**
- Consistent condition handling
- Event recording centralized
- Status update retry logic in one place

### Resulting Reconcile Function

After decomposition, the main `Reconcile()` becomes pure orchestration.

I asked claude to write me a pure orchestration Reconciliation loop, here's what it gave me:

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch ExternalSecret
    es := &esv1.ExternalSecret{}
    if err := r.Get(ctx, req.NamespacedName, es); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Handle deletion
    if !es.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, es)
    }

    // 3. Select target writer (Secret or Manifest)
    writer := target.NewTargetWriter(r.Client, es)

    // 4. Get current target state
    existing, err := writer.Get(ctx, es)
    if err != nil {
        return ctrl.Result{}, err
    }

    // 5. Check if sync needed
    if !r.syncScheduler.ShouldSync(es, existing) {
        return ctrl.Result{RequeueAfter: r.syncScheduler.GetRequeueAfter(es, nil)}, nil
    }

    // 6. Fetch secret data from providers
    data, fetchErr := r.dataFetcher.Fetch(ctx, es)

    // 7. Handle empty data case
    if len(data) == 0 && fetchErr == nil {
        action := r.policyEvaluator.EvaluateDeletion(es, existing)
        return r.executeAction(ctx, es, writer, existing, nil, action)
    }

    // 8. Evaluate creation/update policy
    action := r.policyEvaluator.EvaluateCreation(es, existing, len(data) > 0)

    // 9. Execute the action
    result, err := r.executeAction(ctx, es, writer, existing, data, action)

    // 10. Report status
    if err != nil {
        r.conditionReporter.ReportFailed(ctx, es, err)
    } else {
        r.conditionReporter.ReportSynced(ctx, es, existing)
    }

    return result, err
}
```

### User Stories

**As a contributor**, I want to fix a bug in the refresh policy without understanding secret mutation logic impact, so I can make targeted changes confidently.

**As a maintainer**, I want to review PRs that touch only one responsibility, so I can assess impact accurately.

**As an operator**, I want the controller to be more stable, so my secret synchronization is reliable.

### API

No API changes. This is purely internal refactoring.

### Behavior

No behavior changes. Each extraction must pass the existing test suite without modification (beyond import path changes).

### Drawbacks

1. **More packages to navigate**: Developers must understand the component structure
2. **Interface overhead**: Small performance cost from interface dispatch (negligible for controller reconciliation)
3. **Migration period**: During incremental extraction, some code paths may temporarily coexist

### Acceptance Criteria

Thanks to claude for generating the following list:

**Phase 1: SyncScheduler + TargetPolicyEvaluator** (Low Risk)
- [ ] SyncScheduler extracted with dedicated unit tests
- [ ] TargetPolicyEvaluator extracted, duplicated switch statements removed
- [ ] Main Reconcile() reduced by ~150 lines
- [ ] All existing tests pass

**Phase 2: SecretDataFetcher** (Medium Risk)
- [ ] SecretDataFetcher extracted with mock provider tests
- [ ] GeneratorState integration preserved
- [ ] No changes to external behavior

**Phase 3: TargetWriter Interface** (Medium Risk)
- [ ] SecretWriter implementation extracted
- [ ] ManifestWriter implementation extracted
- [ ] Factory selects correct writer based on ExternalSecret spec
- [ ] Generic Target reconciliation path removed from main Reconcile()

**Phase 4: ConditionReporter** (Low Risk)
- [ ] ConditionReporter consolidated from util.go and inline code
- [ ] Consistent condition handling across all paths
- [ ] Event recording centralized

**Rollout:**
- Each phase is an independent PR that can be merged and released separately
- Rollback is simply reverting the PR - no data migration needed
- Feature flags are not needed since behavior is unchanged

**Testing:**
- Each component has dedicated unit tests with mocked dependencies
- Integration tests verify components work together
- Existing e2e tests validate no behavior change

**Monitoring:**
- Existing `externalsecret_reconcile_duration` metric continues to work
- Future: per-component timing metrics can be added

## Alternatives

### Alternative 1: Keep Current Structure

Accept the current monolithic design.

This is Increasing maintenance burden, contributor friction, stability risk compounds over time.

**Decision**: Rejected - tech debt is already impacting reliability and velocity

### Alternative 2: Full Rewrite

Rewrite the controller from scratch with new architecture.

Clean slate design based on what we learned.

The downside is the high risk, long timeline, likely regressions and we will loose our battle-tested edge case handling

**Decision**: Rejected - incremental extraction preserves existing behavior

## Compatibility with Other Designs

- **design/007-provider-versioning-strategy**: The SecretDataFetcher component provides a clean integration point for the new provider CRDs. When providers become separate CRDs, only SecretDataFetcher needs to change.

- **design/011-generator-state**: GeneratorState management is encapsulated within SecretDataFetcher, making future generator enhancements (like the Cleanup() function) easier to implement without touching orchestration logic.
