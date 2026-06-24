# AWS V2 SecretsManager Design

**Date:** 2026-04-14

## Goal

Add complete provider-v2 coverage for the AWS `SecretsManager` provider path by wiring the existing runtime into the v2 e2e flow and by reusing the common provider-v2 e2e cases wherever they already express the right behavior.

## Scope

In scope:
- AWS provider-v2 plumbing for local and CI e2e runs.
- Provider-v2 e2e coverage for AWS `SecretsManager`.
- Static credential coverage, including session-token passthrough.
- AWS-specific auth coverage for assume-role, external ID, and session tags.
- Reuse of the common provider-v2 namespaced-provider, cluster-provider, and push-secret case builders where AWS semantics match them.
- Managed-only IRSA coverage for the v2 `SecretsManager` path.

Out of scope:
- AWS `ParameterStore` provider-v2 support. This is the next pass.
- Adding a new AWS provider-v2 API kind beyond `SecretsManager`.
- Reworking the underlying AWS store implementation unless e2e work exposes a real defect.
- Changing managed test infrastructure beyond what is needed to point it at the v2 path.

## Current Gaps

1. The AWS provider-v2 runtime already exists, but the v2 e2e bootstrap only installs fake and kubernetes providers.
2. The `test.v2` e2e flow does not build or load `provider-aws`.
3. There is no AWS provider-v2 e2e harness for namespaced `Provider`, `ClusterProvider`, or `PushSecret`.
4. Existing AWS e2e coverage is still centered on legacy `SecretStore` and `ClusterSecretStore` usage.
5. Local and CI credential handling already passes `AWS_SESSION_TOKEN` through `run.sh`, but the v2 test coverage does not currently exercise that path.

## Approaches Considered

### Recommended: Add AWS to the existing provider-v2 framework and reuse common cases

Build an AWS-specific harness around the existing `SecretsManager` v2 resource, then compose it with the shared namespaced-provider, cluster-provider, and push-secret cases plus AWS-specific auth cases.

Why this is the recommendation:
- It matches the current provider-v2 architecture instead of introducing a second testing style.
- It immediately benefits from the recent common-case refactors for fake and kubernetes.
- It gives strong coverage for the real supported API surface without inventing abstractions first.

### Alternative: Implement a bespoke AWS v2 suite first and refactor later

Write AWS-only provider-v2 tests without trying to fit the common case builders, then generalize afterward if patterns emerge.

Rejected for now because:
- It would duplicate behavior that the shared provider-v2 cases already cover.
- It would make AWS onboarding the exception instead of validating the common framework.
- It would defer the real goal, which is reusable provider-v2 onboarding.

### Alternative: Extract a generic cloud-provider harness before adding AWS

Refactor fake, kubernetes, and AWS into a new higher-level provider-v2 harness layer first, then add AWS on top.

Rejected for now because:
- The right common shape is easier to judge after AWS is implemented once.
- It raises abstraction cost before proving what AWS actually needs.
- The current framework is already close enough to support AWS with modest provider-specific glue.

## Design

### 1. Add AWS provider-v2 install and image plumbing

Add a new addon mutator alongside `WithV2KubernetesProvider()` and `WithV2FakeProvider()` that enables `provider-aws` in the v2 ESO install. Extend the v2 e2e make target so it builds and kind-loads the AWS provider image in the same way it already handles fake and kubernetes.

The result should be that `E2E_PROVIDER_MODE=v2` installs:
- the provider-v2 CRDs and controller
- `provider-kubernetes`
- `provider-fake`
- `provider-aws`

No custom bootstrap path should be required for AWS.

### 2. Keep credential flow on standard AWS environment variables

The e2e runner already passes `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_SA_NAME`, and `AWS_SA_NAMESPACE` into the test pod. The AWS v2 suite should continue to read those standard variables.

Credential expectations:
- local kind runs may omit `AWS_SESSION_TOKEN` if the active credentials do not use one
- CI runs must continue to work with session credentials, so the auth secret wiring must preserve the session token field
- AWS-targeted suites should skip cleanly when required AWS credentials are absent instead of failing unrelated provider-v2 runs

This keeps local `~/.aws/credentials` export flows and CI environment injection aligned.

### 3. Add an AWS v2 namespaced-provider harness

Create an AWS provider-v2 test harness that:
- provisions a namespaced `provider.external-secrets.io/v2alpha1` `SecretsManager` resource
- provisions the matching namespaced `external-secrets.io/v1` `Provider`
- waits for the `Provider` to become ready
- uses the existing AWS SDK client helper pattern to create and delete remote secrets in real AWS Secrets Manager

The namespaced provider suite should reuse the common cases for:
- sync through a namespaced `Provider`
- refresh after remote secret updates
- `dataFrom.find`

AWS-specific auth scenarios should then cover:
- static credentials
- assume-role with external ID
- assume-role with session tags

These auth scenarios should stay thin: they should vary provider config and assertions, not reimplement generic sync behavior.

### 4. Add an AWS v2 ClusterProvider harness

Create a cluster-provider harness that reuses namespaced `SecretsManager` config objects behind `ClusterProvider` wrappers. It should support both `AuthenticationScopeManifestNamespace` and `AuthenticationScopeProviderNamespace` so the shared cluster-provider cases can be reused directly.

The harness should support:
- cluster-provider sync from both auth scopes
- denial by `spec.conditions`

Recovery-after-auth-repair should only be added if the AWS harness can model a truthful auth break and repair cycle without introducing flaky, AWS-specific timing behavior. If it cannot, the harness should opt out explicitly rather than pretending to support it.

### 5. Exercise provider-v2 push coverage when AWS semantics match

The AWS v2 provider reports read/write capabilities, so the initial implementation should attempt to cover `PushSecret` through the shared push-case inventory rather than declaring push out of scope.

The AWS push harness should:
- verify pushed values against real AWS Secrets Manager data
- support cluster-provider push cases where the runtime behavior maps cleanly
- only opt into optional common push capabilities that AWS can assert truthfully

If a specific shared push case depends on Kubernetes-only semantics, the AWS suite should omit that case instead of distorting the harness.

### 6. Keep IRSA managed-only and point it at the v2 path

IRSA coverage stays in the managed AWS suite. The work here is not to make kind emulate IRSA; it is to ensure the managed AWS path validates provider-v2 `SecretsManager` configuration instead of only the legacy store path.

That means:
- keep IRSA labels and managed test assumptions intact
- create provider-v2 `SecretsManager` resources in managed tests
- use namespaced `Provider` or `ClusterProvider` connections as appropriate for the existing managed scenarios

### 7. Review reuse opportunities after AWS lands

After the AWS harness exists, compare it with fake and kubernetes to identify any repeated provider-v2 harness code worth extracting. This should be a targeted cleanup only if the duplication is concrete and improves onboarding for the next provider.

The intent is not a framework rewrite. The intent is to confirm whether a small helper extraction becomes justified once AWS is implemented.

## Testing Strategy

1. Add or update focused tests for any touched addon or makefile plumbing.
2. Run targeted Go tests for touched AWS provider-v2 and e2e helper packages.
3. Run focused provider-v2 e2e coverage for AWS `SecretsManager` with local AWS credentials.
4. Run managed AWS v2 coverage for IRSA-oriented scenarios in the appropriate environment.
5. Re-run the affected provider-v2 e2e slices to confirm fake and kubernetes coverage still works after any shared-harness changes.

## Risks And Guardrails

- Do not broaden scope into `ParameterStore` during this pass.
- Do not add AWS-specific duplicated cases when a shared provider-v2 case already covers the same behavior.
- Do not claim auth-lifecycle recovery support unless the AWS harness can break and repair auth deterministically.
- Keep session-token handling explicit in auth secret setup so CI credentials continue to work.
- Skip AWS suites cleanly when credentials are unavailable instead of making provider-v2 mode globally brittle.

## Success Criteria

- `provider-aws` is installed and loaded automatically in v2 e2e mode.
- AWS `SecretsManager` provider-v2 coverage exists for namespaced `Provider`, `ClusterProvider`, and push flows where supported.
- Shared provider-v2 common cases are reused instead of rebuilding AWS-only equivalents.
- AWS-specific auth coverage exists for static credentials, external ID, and session tags.
- Managed IRSA coverage exercises the provider-v2 `SecretsManager` path.
- `ParameterStore` remains untouched for this pass and is the next scoped follow-up.
