# Provider V2 E2E Reuse Refactor Design

**Date:** 2026-04-14

## Goal

Reduce the amount of provider-specific boilerplate required to add new provider v2 e2e coverage by extracting reusable namespaced-provider read cases and by making the shared cluster-provider and push harnesses capability-based instead of implicitly Kubernetes-shaped.

## Scope

In scope:
- Generalizing the current fake-provider read cases so the kubernetes v2 namespaced-provider suite can use the same case builders.
- Centralizing shared namespace lifecycle setup used by provider v2 suites.
- Refactoring shared cluster-provider and push harnesses so providers can opt into only the behaviors they can model truthfully.
- Updating fake v2 and kubernetes v2 suites to use the new shared structure.

Out of scope:
- Reworking provider runtime implementations.
- Changing the controller behavior under test.
- Building a full provider-registration framework for all e2e suites.

## Current Problems

1. The kubernetes v2 namespaced-provider suite reimplements sync, refresh, find, and recovery as bespoke `It(...)` blocks, while fake v2 already uses table-driven case builders.
2. Shared cluster-provider and push cases require callbacks and runtime fields that are only natural for the kubernetes provider, so fake must satisfy them with no-ops or indirect readback helpers.
3. Namespace creation and teardown logic is duplicated between provider suites.

These issues do not currently break coverage, but they make onboarding a new provider to v2 e2e slower and more error-prone than it needs to be.

## Approaches Considered

### Recommended: Incremental shared-case and capability refactor

Extract generic namespaced-provider read cases from the fake-specific ones, introduce explicit capability flags or optional hooks for cluster-provider and push recovery/absence assertions, and move duplicated namespace setup into a common helper.

Why this is the recommendation:
- It removes the main onboarding duplication without changing the underlying test framework contract everywhere.
- It keeps provider semantics honest: unsupported behaviors can be omitted instead of stubbed.
- It is small enough to verify with existing package tests and focused provider slices.

### Alternative: One-shot redesign of all provider e2e runtimes

Replace the existing `framework.TestCase`, cluster-provider, and push harness layers with a new provider capability model in one pass.

Rejected for now because:
- It is broader than the actual pain points.
- It would create unnecessary churn across suites that already work.
- It raises the regression surface without proportionate benefit.

## Design

### 1. Introduce generic namespaced-provider read cases

Create provider-agnostic case builders in `e2e/suites/provider/cases/common` for:
- namespaced sync
- refresh after provider data changes
- `dataFrom.find`

These cases should depend only on the existing `framework.SecretStoreProvider` contract plus test-case fields such as expected secret contents, refresh intervals, and optional auth-break hooks. The existing fake helpers should be renamed or split so they express generic behavior rather than fake-specific naming.

The kubernetes v2 namespaced-provider suite should become a thin `DescribeTable(...)` entrypoint over these common cases, matching the fake v2 suite shape. Recovery coverage can remain a separate case builder if it needs optional auth capabilities.

### 2. Add shared helper(s) for provider-v2 namespace lifecycle

Move the duplicated namespace create-and-cleanup helper into `e2e/suites/provider/cases/common`, with timeouts and polling passed in or set by a narrow common default.

Provider suites should call the shared helper instead of maintaining near-identical copies. This change is mechanical but useful because namespace setup is part of many cluster-provider and push scenarios.

### 3. Make cluster-provider external-secret recovery optional

Change the cluster-provider runtime contract so auth recovery is optional. The common runtime should only run recovery cases when the provider harness advertises working `BreakAuth` and `RepairAuth` hooks.

Concretely:
- keep sync and condition-denial cases generic and always available
- represent recovery support explicitly in the runtime or harness config
- fail fast in common code if a recovery case is accidentally wired without the required hooks

This keeps fake from pretending to support auth failures it cannot model while preserving the richer kubernetes recovery coverage.

### 4. Make push runtime assertions capability-based

Split the current push runtime expectations into provider capabilities rather than a single Kubernetes-oriented shape.

The shared push cases need three separable concepts:
- assert that a pushed value becomes visible at a remote key
- optionally assert that no remote object exists at a location
- optionally create a writable override scope for remote-namespace tests

Providers should implement only the pieces they can support. Cases that require unsupported capabilities should not be registered for that provider. This lets fake continue to cover truthful push behaviors without indirect no-op semantics, while kubernetes can keep richer metadata and absence assertions.

### 5. Keep suite entrypoints thin and aligned

After the refactor:
- fake v2 namespaced-provider and kubernetes v2 namespaced-provider suites should both be tables over the same common read cases
- fake v2 and kubernetes v2 cluster-provider suites should differ primarily in their harness construction, not in the case definitions
- push suites should compose from the same common case inventory, with each provider selecting only supported cases

The target shape is consistent suite wiring, not identical provider behavior.

## Testing Strategy

1. Add or update fast package tests for any new helper logic and for any provider-specific utility behavior changed during extraction.
2. Convert the kubernetes namespaced-provider suite to shared cases and run the provider-case package tests to catch compile and API mismatches.
3. Refactor cluster-provider and push runtime contracts with focused package verification first.
4. Run focused e2e slices for:
   - `fake && v2`
   - `kubernetes && v2`
5. Re-run the fast package suite covering addon/common/fake/kubernetes provider-case packages.

## Risks And Guardrails

- The refactor must not erase meaningful provider differences. If a case depends on Kubernetes `Secret` object semantics, it should stay out of fake.
- Optional capabilities must be explicit; silent no-op fallbacks would hide wiring mistakes.
- The common layer should get smaller and clearer, not more abstract for its own sake.

## Success Criteria

- Kubernetes v2 namespaced-provider coverage is table-driven using shared common cases.
- Fake v2 no longer needs no-op recovery hooks or indirect push assertions where the behavior is unsupported.
- Namespace lifecycle setup is shared.
- Adding a new provider v2 suite mostly means implementing provider setup/harness code and choosing from a reusable case inventory.
