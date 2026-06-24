# PR Walkthrough

This PR is large, but the core change is narrow: runtime selection moved from the old v2 `ProviderStore` model to `ProviderClass`, with a new namespaced `ProviderClass` as the default.

If you want a guided tour, review it in this order. The goal is to confirm the semantic contract first, then check the controller/runtime enforcement, then validate that the e2es actually prove the contract.

## 20-Minute Review Plan

### 1. API contract and invariants (5 min)

Start here:

- `apis/externalsecrets/v1/secretstore_types.go`
- `apis/externalsecrets/v1/externalsecret_types.go`
- `apis/externalsecrets/v1beta1/externalsecret_types.go`
- `apis/externalsecrets/v1alpha1/pushsecret_types.go`

Focus on these questions:

- Is the default runtime class kind now clearly namespaced `ProviderClass`?
- Is it explicit that namespaced `ProviderClass` resolution uses the `SecretStore` namespace?
- Is it explicit that `ClusterSecretStore` may only target `ClusterProviderClass`?
- Are kind defaults and validation rules obvious from the types, not just from controller behavior?

What matters:

- If this layer is ambiguous, the rest of the PR is hard to reason about.
- Most regression risk comes from mismatched assumptions between API shape and controller resolution.

### 2. Runtime resolution and controller wiring (6 min)

Read next:

- `pkg/controllers/externalsecret/externalsecret_controller.go`
- `pkg/controllers/pushsecret/pushsecret_controller.go`
- `pkg/controllers/providerclass/controller.go`
- `cmd/controller/root.go`

Focus on these questions:

- Where is the default class kind applied when the kind is omitted?
- Does a namespaced `ProviderClass` always resolve in the `SecretStore` namespace, not the workload namespace?
- Is the `ClusterSecretStore -> ProviderClass` path rejected early and clearly?
- Did controller setup and watches change in a way that could miss reconciles or break ownership assumptions?

What matters:

- This is the highest-value review area.
- The main failure mode is “valid YAML, wrong runtime object selected”.

### 3. Migration away from ProviderStore (3 min)

Skim the removals as one story:

- deleted `apis/externalsecrets/v2alpha1/*`
- deleted `pkg/controllers/providerstore/*`
- deleted `runtime/clientmanager/providerstore.go`

Focus on these questions:

- Is anything still conceptually depending on the old v2 providerstore layer?
- Were there any compatibility shims that silently carried behavior and are now gone?
- Does the new `ProviderClass` path fully replace the removed stack, or is there a hidden gap?

What matters:

- The diff is deletion-heavy, so this is less about line review and more about architecture sanity.

### 4. PushSecret edge cases (3 min)

Give this a separate pass:

- `pkg/controllers/pushsecret/pushsecret_controller.go`
- `pkg/controllers/pushsecret/pushsecret_controller_test.go`

Focus on these questions:

- Does omitted kind now normalize consistently to `SecretStore`?
- Does PushSecret resolve runtime refs the same way ExternalSecret does?
- Are tests covering the defaulting path, not only explicit kind declarations?

What matters:

- There was a follow-up fix here, so this is a known sharp edge.

### 5. E2E proof, not just migration churn (3 min)

Finish in the e2es:

- `e2e/framework/v2/helpers.go`
- `e2e/suites/provider/cases/fake/runtime_ref_v2.go`
- `e2e/suites/provider/cases/fake/operational_v2.go`
- `e2e/suites/provider/cases/common/*`

Focus on these questions:

- Do the tests cover both namespaced `ProviderClass` and `ClusterProviderClass` resolution?
- Is there a negative case for `ClusterSecretStore -> ProviderClass`?
- Do the fake-provider tests exercise the real runtime-ref path, or do helpers hide too much?
- Did the migration keep the tests behavior-focused, or did it drift into fixture churn?

What matters:

- This tells you whether the new contract is enforced end-to-end, not just unit-tested.

## What To Skim Last

Only after the semantic review above:

- `config/crds/*`
- `deploy/crds/bundle.yaml`
- `docs/api/spec.md`
- `tests/__snapshot__/*`

Use these as confirmation that the public surface matches the implementation. Do not start here.

## Things Worth Calling Out In The Tour

If you are presenting this PR live, I would center the walkthrough on four claims:

1. The default is now namespaced `ProviderClass`.
2. Namespaced `ProviderClass` resolution is anchored to the `SecretStore` namespace.
3. `ClusterSecretStore` is intentionally restricted to `ClusterProviderClass`.
4. The old v2 `ProviderStore` stack was intentionally removed rather than kept as a compatibility layer.

## Suggested Tour Script

If you want a compact live narrative, use this:

1. Start at the API types and state the new invariants.
2. Jump to controller resolution and show where those invariants are enforced.
3. Call out the ProviderStore removal as deliberate simplification, not incidental cleanup.
4. Show the PushSecret defaulting fix as a concrete edge case.
5. End with the e2e runtime-ref coverage to show the new model is exercised end-to-end.
