# AWS Parameter Store V2 Parity Design

## Goal

Bring AWS Parameter Store v2 to parity with the existing AWS Secrets Manager v2 implementation and e2e structure.

Parity here means matching the current Secrets Manager v2 shape for provider auth support and e2e coverage, not inventing a larger test matrix.

## Current State

Parameter Store v2 already has:
- provider implementation in `providers/v2/aws/store/parameterstore`
- namespaced-provider v2 e2e coverage for static auth
- cluster-provider v2 e2e coverage
- push-secret v2 e2e coverage

Secrets Manager v2 additionally has:
- namespaced-provider auth coverage for `static`, `external-id`, and `session-tags`
- managed v2 IRSA coverage for:
  - referenced service account
  - mounted IRSA
- reusable auth/profile helpers for assume-role probing and provider config generation

## Scope

In scope:
- add missing Parameter Store v2 auth support in e2e helper/config generation for:
  - static
  - external-id
  - session-tags
  - referenced-irsa
  - mounted-irsa
- add missing Parameter Store v2 e2e coverage so it matches Secrets Manager v2:
  - namespaced provider: static, external-id, session-tags
  - managed v2 IRSA: referenced service account, mounted IRSA
- keep existing Parameter Store v2 cluster-provider and push-secret coverage
- add or update focused helper tests for the new auth/profile behavior

Out of scope:
- expanding cluster-provider or push-secret auth matrices beyond current Secrets Manager v2 parity
- changing CI wiring
- broader AWS v2 helper deduplication beyond what is directly needed for this slice

## Design

### 1. Parameter Store v2 auth profiles

Extend `e2e/suites/provider/cases/aws/parameterstore/provider_support_v2.go` to mirror the Secrets Manager v2 auth-profile model.

This includes:
- auth profile enum/constants for `static`, `external-id`, `session-tags`, `referenced-irsa`, `mounted-irsa`
- auth-aware config generation for Parameter Store provider configs
- assume-role probe reuse for `external-id` and `session-tags`
- provider preparation helpers for namespaced-provider and managed-provider cases

The Parameter Store v2 helper should stay Parameter Store specific, but its behavior should align with Secrets Manager v2 so the two suites remain structurally parallel.

### 2. Namespaced-provider parity

Update `e2e/suites/provider/cases/aws/parameterstore/provider_v2.go` to match the Secrets Manager v2 auth-case matrix where applicable.

Resulting namespaced-provider coverage:
- static auth: existing sync, refresh, find, versioned, status-not-updated cases remain
- external-id auth: add the same simple sync coverage style used by Secrets Manager v2
- session-tags auth: add the same simple sync coverage style used by Secrets Manager v2

This keeps the richer Parameter Store-specific cases on static auth only, while matching Secrets Manager v2’s extra auth validation path.

### 3. Managed v2 IRSA parity

Add a new Parameter Store v2 managed test file mirroring `secretsmanager_v2_managed.go`.

Coverage should match Secrets Manager v2 exactly:
- referenced IRSA with a cluster provider reference
- mounted IRSA with a namespaced provider address in the service account namespace
- case set limited to:
  - `common.SimpleDataSync`
  - provider-specific `FindByName`

This preserves the existing AWS managed test strategy and avoids adding a larger managed matrix than Secrets Manager v2 already has.

### 4. Tests

Add or update focused helper tests to verify:
- Parameter Store v2 config generation picks the correct kind and auth fields for each profile
- referenced IRSA store refs use cluster-provider kind where required
- any new profile-specific config fields are wired as expected

Verification commands for implementation should include:
- focused `go test` for Parameter Store v2 helper packages
- focused Parameter Store v2 v2 e2e runs using the existing `make` targets

## Risks

- assume-role coverage depends on the current static credentials being allowed to assume the expected roles; tests should skip cleanly when denied, matching Secrets Manager v2 behavior
- mounted IRSA v2 coverage is sensitive to provider pod namespace/address wiring; reuse of the Secrets Manager v2 pattern should minimize divergence
- Parameter Store remote key naming rules differ from Secrets Manager; existing Parameter Store path conventions should remain unchanged

## Success Criteria

This work is complete when:
- Parameter Store v2 has the same auth profile coverage shape as Secrets Manager v2
- Parameter Store v2 managed IRSA coverage exists and mirrors Secrets Manager v2
- focused helper/unit tests pass
- focused Parameter Store v2 e2e runs pass for the added coverage
