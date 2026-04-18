# Provider API Removal And V2 E2E Migration Design

**Date:** 2026-04-18

## Goal

Remove the transitional `Provider` and `ClusterProvider` APIs from core, make `ProviderStore` and `ClusterProviderStore` the only out-of-process store surface, and migrate the existing v2 e2e coverage to the clean store API.

## Scope

In scope:
- removing the `Provider` and `ClusterProvider` API types, CRDs, controllers, and runtime lookup paths
- updating `ExternalSecret` and `PushSecret` runtime-backed store handling to only use `ProviderStore` and `ClusterProviderStore`
- removing the obsolete `--enable-v2-providers` gate from the clean store path
- migrating the current v2 e2e helper layer and suites from `Provider` / `ClusterProvider` to `ProviderStore` / `ClusterProviderStore`

Out of scope:
- removing `v1` `SecretStore` / `ClusterSecretStore`
- changing the compatibility `runtimeRef` flow on `v1` stores
- redesigning provider-owned backend CRDs
- broad public docs refresh beyond what must change to keep tests/builds correct

## Problem

The branch already has the long-term clean store model:

- `ProviderStore`
- `ClusterProviderStore`
- `runtimeRef`
- `backendRef`

But it still carries the older `Provider` and `ClusterProvider` runtime path. That creates three problems:

1. two different out-of-process store APIs exist in core
2. `runtime/clientmanager` and controllers still maintain duplicate routing logic
3. the v2 e2e suite validates the transitional API rather than the approved long-term API

If we keep both models, the code and UX stay muddled. The clean path should become the only out-of-process path.

## Decision

Remove `Provider` and `ClusterProvider` entirely and migrate all current v2-path tests to `ProviderStore` and `ClusterProviderStore`.

This includes:

- deleting the old API types and CRDs
- deleting the dedicated `provider` and `clusterprovider` controllers
- removing `runtime/clientmanager` support for `Provider` and `ClusterProvider`
- removing consumer-kind handling for those old kinds
- switching the e2e framework helpers from provider-connection objects to store objects
- making `ProviderStore` / `ClusterProviderStore` always available instead of hiding them behind `--enable-v2-providers`

## Approaches Considered

### Recommended: hard cut to ProviderStore

Remove the old runtime-backed CRDs and move the tests in the same change.

Why this is recommended:
- matches the approved architecture
- removes duplicate controller and clientmanager logic
- makes e2e coverage validate the API we actually want to ship
- simplifies user guidance to one out-of-process model

### Alternative: keep Provider as compatibility alias

Rejected because:
- it preserves duplicate semantics and code paths
- it keeps the wrong API alive in e2e and examples
- it delays cleanup without solving migration inside the repo

### Alternative: migrate tests first and remove code later

Rejected because:
- it prolongs the overlap
- it still forces reviewers to reason about both models
- the codebase already has the replacement API

## API And Controller Impact

### 1. Remove the old runtime-backed CRDs

Delete:

- `Provider`
- `ClusterProvider`

Delete their registration and generated CRD output from:

- Go API registration
- `config/crds`
- `deploy/crds/bundle.yaml`
- Helm CRD templates and snapshots

### 2. Keep only ClusterProviderClass plus clean stores

The out-of-process control-plane objects become:

- `ClusterProviderClass`
- `ProviderStore`
- `ClusterProviderStore`

Semantics stay:

- `ClusterProviderClass` is transport-only
- `ProviderStore` carries `runtimeRef` and `backendRef`
- `ClusterProviderStore` carries `runtimeRef`, `backendRef`, and namespace `conditions`

### 3. Consumer references

`ExternalSecret` and `PushSecret` no longer need to recognize:

- `Provider`
- `ClusterProvider`

The clean runtime-backed kinds become only:

- `ProviderStore`
- `ClusterProviderStore`

Compatibility remains through `SecretStore` / `ClusterSecretStore` with `runtimeRef`.

### 4. Runtime client manager

Delete:

- namespaced `Provider` fetch path
- cluster `ClusterProvider` fetch path
- cache keys, metrics labels, errors, and tests specific to those resources

Keep:

- compatibility `runtimeRef` path for `SecretStore` / `ClusterSecretStore`
- clean-store path for `ProviderStore` / `ClusterProviderStore`

### 5. Feature gate

Remove `--enable-v2-providers`.

Reason:
- the flag name is tied to the deleted API
- the clean store API should not remain behind a stale gate once it is the only out-of-process path

Result:
- `ProviderStore` and `ClusterProviderStore` are always enabled
- providerstore metrics and gRPC metrics register unconditionally when the controller starts

## E2E Migration Design

### 1. Replace helper vocabulary

In `e2e/framework/v2/helpers.go`, replace provider-connection helpers with store helpers:

- `CreateProviderConnection` -> create `ProviderStore`
- `CreateClusterProviderConnection` -> create `ClusterProviderStore`
- `WaitForProviderConnectionReady` -> wait for `ProviderStore`
- `WaitForClusterProviderReady` -> wait for `ClusterProviderStore`

The helper input model remains the same:

- runtime address
- backend CR `apiVersion`
- backend CR `kind`
- backend CR `name`
- backend CR `namespace`

The helper implementation changes to:

- resolve/create `ClusterProviderClass`
- create `ProviderStore` / `ClusterProviderStore`
- populate `runtimeRef` and `backendRef`

### 2. Migrate common harnesses first

The common harness layer under `e2e/suites/provider/cases/common` currently hard-codes:

- `SecretStoreRef.Kind = Provider`
- `SecretStoreRef.Kind = ClusterProvider`

That layer should switch to:

- `ProviderStore`
- `ClusterProviderStore`

Once the common harness is updated, most provider-specific v2 suites only need helper renames and expected-message updates.

### 3. Keep the provider-owned backend CRs

The backend CRs used by e2e stay provider-owned:

- fake backend CR
- kubernetes backend CR
- aws backend CRs

The migration only changes the ESO-facing runtime/store object, not the provider backend object model.

## Testing

Required verification:

- API registration tests for removal of `Provider` / `ClusterProvider`
- controller tests for `cmd/controller/root.go` and reconciler wiring
- `runtime/clientmanager` tests proving only `ProviderStore` / `ClusterProviderStore` remain on the clean path
- pushsecret / externalsecret tests updated to remove old kind assumptions
- e2e framework helper tests updated to the new store objects
- targeted v2 e2e suites for fake, kubernetes, and aws still passing against the store-based API
- final repo verification with `make test`

## Risks And Mitigations

Risk:
- some tests still implicitly depend on `ProviderKindStr` defaults or helper constants

Mitigation:
- remove old constants from consumer logic, then let the compiler and focused tests surface every remaining reference

Risk:
- e2e helper migration touches many files

Mitigation:
- change the shared helper and harness layer first, then fix provider-specific suites in focused batches

Risk:
- removing the gate might accidentally change controller startup behavior

Mitigation:
- verify metrics registration and controller startup tests in `cmd/controller`

## Success Criteria

The work is complete when:

- no `Provider` or `ClusterProvider` API types remain in core
- no core controller reconciles those resources
- `ExternalSecret` and `PushSecret` clean-path logic only uses `ProviderStore` / `ClusterProviderStore`
- the v2 e2e suite provisions and asserts only against `ProviderStore` / `ClusterProviderStore`
- `make test` passes on the branch
