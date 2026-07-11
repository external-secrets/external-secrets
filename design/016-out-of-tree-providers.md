```yaml
---
title: Out-of-Tree Providers (Provider Packages)
version: v1alpha1
authors: Alexander Chernov
creation-date: 2026-07-11
status: draft
---
```

# Out-of-Tree Providers (Provider Packages)

> **Experimental feature, off by default.** Everything described here is gated behind an
> experimental feature flag, `--experimental-enable-provider-packages` (default `false`), following
> the `experimental-*` convention from [014-feature-flag-consolidation](014-feature-flag-consolidation.md).
> While the flag is disabled, none of this machinery is registered or reconciled: the packaging
> controllers do not run, the `pkg.external-secrets.io` CRDs are inert, and ESO behaves exactly as
> it does today (in-tree providers only). The flag is both the opt-in and the kill switch for the
> entire alpha lifetime, and the packaging CRDs ship as `v1alpha1` and may change without notice.
> It graduates to a stable `--enable-*` flag only once the design is proven, per the
> [Acceptance Criteria](#acceptance-criteria).

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->

## Summary

Today every provider is compiled into the `external-secrets` core binary. Adding, updating, or
removing a provider requires a core release, and the binary statically links the SDK of every
supported backend (AWS, GCP, Azure, Vault, and roughly forty others), which drives image size,
the dependency graph, and the CVE surface.

This proposal introduces **out-of-tree providers** distributed as OCI packages, modelled on the
Crossplane provider package mechanism. A cluster admin declares a provider (by OCI image
reference) or installs it through Helm. ESO pulls the image, installs the provider's own
configuration CRD(s), and runs the provider as an independently-managed workload. The core talks
to the provider over a stable gRPC contract. Each provider ships, versions, and is garbage
collected independently of the core and of every other provider.

The change is **additive and opt-in**. In-tree providers keep working unchanged. The only
breaking step (removing providers from the core binary) is deferred to a later major release,
gated behind a deprecation window, and is explicitly a non-goal here.

This design composes with, and depends on, [007-provider-versioning-strategy](007-provider-versioning-strategy.md):
007 defines *provider configuration as its own CRD* plus `SecretStore.spec.providerRef`; this
document defines *how that CRD and its controller are packaged, delivered, run, and retired*
out-of-tree.

## Motivation

WRT: https://github.com/external-secrets/external-secrets/issues/694 (provider separation) and
the broader "v2 provider plugin" discussion.

Concrete pain points with the current all-in-one binary:

* **Release coupling.** A fix or a new field in one provider needs a full ESO release. Community
  or vendor-maintained providers move at core's cadence, not their own.
* **Footprint and supply chain.** The core links every backend SDK. Users who run one provider
  still carry the transitive dependencies and CVEs of all of them. The multi-module `go.work`
  layout already splits providers at build time, but the shipped artifact is still monolithic.
* **Third-party providers.** There is no supported path for an organisation to ship a private
  provider without forking core.
* **Independent maturity.** A `v1alpha1` community provider and a `v1` mature provider live in
  the same binary and the same API surface today.

### Goals

* Deliver providers as self-contained OCI packages that ESO can install, upgrade, and remove at
  runtime without a core release.
* Let each provider own and ship its own configuration CRD(s), versioned on its own schedule
  (the 007 config-CRD model, delivered dynamically).
* Run each provider as an independently-managed workload with its own lifecycle, its own
  ServiceAccount and RBAC, and therefore its own cloud identity.
* Keep a stable, versioned runtime contract between core and providers so the two evolve
  independently.
* Preserve full backward compatibility: existing `SecretStore.spec.provider.<name>` manifests and
  in-tree providers continue to function unchanged.

### Non-Goals

* **Removing any in-tree provider from the core binary.** That is a separate, later,
  major-version decision with its own deprecation plan.
* Changing the `ExternalSecret`, `PushSecret`, or generator user-facing APIs.
* Defining a marketplace, a package registry service, or a curation process. This design assumes
  standard OCI registries.
* Cross-cluster or hosted (control-plane-in-another-cluster) provider execution.
* Replacing the existing `webhook` provider (it remains a lightweight in-tree option).

## Proposal

The design has three planes. Two are new here; the third is 007.

1. **Packaging and lifecycle plane** (new): OCI package format, the `ProviderPackage` and
   `ProviderRevision` CRDs, and the package-manager controller that pulls, verifies, installs, and
   runs a provider.
2. **Runtime plane** (new): a gRPC service contract that mirrors the existing `Provider` and
   `SecretsClient` Go interfaces, plus a version/capability handshake and an auth-resolution
   broker.
3. **Configuration plane** (007): per-provider configuration CRDs and `SecretStore.spec.providerRef`.
   The provider package *is* what delivers these CRDs and their reconciler out-of-tree.

### User Stories

* *As a cluster admin*, I add `helm install eso-provider-vault external-secrets/provider-vault`
  (or apply a `ProviderPackage` referencing `ghcr.io/external-secrets/provider-vault:v1.2.0`), and
  the Vault config CRD plus its controller appear in my cluster. I never touched the core release.
* *As a cluster admin*, I upgrade a provider by bumping the image tag on the `ProviderPackage`. A
  new `ProviderRevision` is created and activated; if it is unhealthy I roll back to the previous
  revision.
* *As a platform team*, I ship a private provider for our internal secret store as an OCI image in
  our own registry and install it exactly like a public one.
* *As a security engineer*, I require provider images to be cosign-signed and pulled by digest,
  and I confine provider workloads to a dedicated namespace with a scoped ServiceAccount per
  provider.
* *As an existing user*, I upgrade ESO and every one of my current `SecretStore` manifests keeps
  working with zero changes, because the providers I use are still in-tree.

### API

Two new CRD groups are introduced for packaging, kept distinct from the 007 configuration groups
to avoid ambiguity:

* Packaging/lifecycle: `pkg.external-secrets.io/v1alpha1`, kinds `ProviderPackage` (cluster
  scoped) and `ProviderRevision` (cluster scoped). Analogous to Crossplane's `Provider` and
  `ProviderRevision`.
* Configuration (from 007): `providers.external-secrets.io` and `cluster.providers.external-secrets.io`,
  one kind per provider (for example `Vault`, `AWS`), *installed dynamically by the package*.

#### ProviderPackage

```yaml
apiVersion: pkg.external-secrets.io/v1alpha1
kind: ProviderPackage
metadata:
  name: vault
spec:
  # OCI image containing the package. Digest pinning is recommended and can be enforced by policy.
  package: ghcr.io/external-secrets/provider-vault:v1.2.0
  packagePullPolicy: IfNotPresent
  packagePullSecrets:
    - name: registry-creds
  # Automatic (default): a new, healthy revision is activated automatically.
  # Manual: a human activates the new revision.
  revisionActivationPolicy: Automatic
  revisionHistoryLimit: 3
  # Signature / provenance verification policy for this package.
  verification:
    cosign:
      enabled: true
      # keyless (Fulcio/Rekor) or a referenced public key
  # Runtime knobs for the provider workload the manager creates.
  runtime:
    replicas: 1
    resources: {}
    serviceAccountName: ""      # empty => manager creates a scoped SA
    nodeSelector: {}
    tolerations: []
status:
  currentRevision: vault-8f3a1c2
  conditions:
    - type: Installed
      status: "True"
    - type: Healthy
      status: "True"
```

#### ProviderRevision

One `ProviderRevision` exists per unpacked image version. It is the owner of everything the
package installs (the config CRDs, the Deployment, the RBAC), which is what makes uninstall and
rollback clean.

```yaml
apiVersion: pkg.external-secrets.io/v1alpha1
kind: ProviderRevision
metadata:
  name: vault-8f3a1c2
  ownerReferences:
    - apiVersion: pkg.external-secrets.io/v1alpha1
      kind: ProviderPackage
      name: vault
spec:
  image: ghcr.io/external-secrets/provider-vault@sha256:...
  revision: 7
  # Active: serving traffic and owning its CRDs. Inactive: unpacked but dormant (kept for rollback).
  desiredState: Active
status:
  foundVersion: ghcr.io/external-secrets/provider-vault:v1.2.0
  # Runtime contract version negotiated with core.
  runtimeAPIVersion: v1alpha1
  providedKinds:
    - group: providers.external-secrets.io
      kind: Vault
    - group: cluster.providers.external-secrets.io
      kind: ClusterVault
  conditions:
    - type: Healthy
      status: "True"
    - type: RuntimeReachable
      status: "True"
```

#### Package format (OCI)

The package is an OCI image with a well-known layout, mirroring Crossplane's `package.yaml`
convention:

* `package.yaml` (or an OCI annotation pointer to it): package metadata, containing the provider
  name, the runtime API version(s) it implements, the list of CRDs it provides, the capabilities
  it supports (`ReadWrite` / `ReadOnly`), and the RBAC it requires to run.
* The CRD manifests for the provider's configuration kinds.
* The provider binary as the image entrypoint. The image is simultaneously the *metadata carrier*
  and the *runnable provider*: when the manager runs it as a Deployment, the entrypoint serves the
  gRPC contract.

#### SecretStore reference (007)

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: team-a
spec:
  providerRef:                      # 007: reference a provider config object
    group: providers.external-secrets.io
    kind: Vault
    name: team-a-vault
---
apiVersion: providers.external-secrets.io/v1
kind: Vault                         # CRD installed by the vault ProviderPackage
metadata:
  name: team-a-vault
spec:
  server: https://vault.example.com
  path: secret
  auth:
    kubernetes:
      role: eso
      serviceAccountRef:
        name: eso-vault
```

The legacy inline form (`spec.provider.vault: {...}`) remains valid for in-tree providers and is
untouched.

#### gRPC runtime contract

A single service mirrors the existing `esv1.Provider` and `esv1.SecretsClient` interfaces
one-to-one, so an in-tree provider can be wrapped as a server with a thin adapter:

```proto
service SecretsProvider {
  // Provider interface
  rpc Capabilities(CapabilitiesRequest) returns (CapabilitiesResponse);
  rpc ValidateStore(ValidateStoreRequest) returns (ValidateStoreResponse);

  // SecretsClient interface (client identity established via NewClient semantics in the request)
  rpc GetSecret(GetSecretRequest) returns (GetSecretResponse);
  rpc GetSecretMap(GetSecretMapRequest) returns (GetSecretMapResponse);
  rpc GetAllSecrets(GetAllSecretsRequest) returns (GetAllSecretsResponse);
  rpc PushSecret(PushSecretRequest) returns (PushSecretResponse);
  rpc DeleteSecret(DeleteSecretRequest) returns (DeleteSecretResponse);
  rpc SecretExists(SecretExistsRequest) returns (SecretExistsResponse);

  // Lifecycle / negotiation
  rpc Handshake(HandshakeRequest) returns (HandshakeResponse); // runtime API version + capabilities
}
```

The provider's config object is passed as its serialized CRD spec (the API server has already
validated it against the installed schema), so there is no opaque, unvalidated JSON blob on this
path.

### Behavior

#### Install and run (package-manager controller)

On a `ProviderPackage` create/update, the manager:

1. Resolves and pulls the OCI image (honouring `packagePullSecrets` and `packagePullPolicy`).
2. Verifies provenance per `spec.verification` (cosign signature, digest). On failure the package
   is marked `Installed=False` with reason and no workload is created.
3. Reads `package.yaml`, and creates or updates a `ProviderRevision` for this image digest.
4. Installs the provider's config CRDs, labelled and owner-referenced by the `ProviderRevision`.
   A CRD already owned by a *different* provider is a conflict and blocks activation.
5. Creates the provider workload: a Deployment (serving gRPC), a Service, and a scoped
   ServiceAccount + Role/RoleBinding derived from the RBAC declared in `package.yaml`.
6. Performs the gRPC `Handshake`. If the runtime API version is outside the range core supports,
   the revision is `Healthy=False` and is not activated.
7. On success under `revisionActivationPolicy: Automatic`, marks the revision `Active`, registers
   its provided GVKs into the dynamic provider registry (the runtime successor to 007's
   `RefRegister`), and retires older revisions beyond `revisionHistoryLimit` (config CRDs of a
   retired revision are kept only while it stays `Inactive` for rollback).

#### Secret resolution (steady state)

For a `SecretStore` whose `providerRef` points at a config kind served by an installed package,
the ClientManager (per 007) resolves the config object, then dials the provider's Service and
calls the gRPC methods instead of an in-process client. From the `ExternalSecret` /`PushSecret`
reconciler's point of view nothing changes: it still calls `GetProvider(store).NewClient(...)`
and then `GetSecret(...)`; the returned `SecretsClient` is a gRPC-backed implementation.

If a `providerRef` targets a kind that is not installed, the `SecretStore` gets a clear
`Ready=False` condition (`ProviderPackageNotInstalled`) rather than a generic failure.

#### Authentication across the process boundary

This is the hardest part and the main behavioural change. Today a provider receives a live
controller-runtime `client.Client` and reads referenced Secrets / ServiceAccount tokens itself.
That cannot cross a process boundary. Two supported models, default first:

* **Broker (recommended default).** The provider runs with its own ServiceAccount and, for
  referenced Kubernetes material, calls back to a core-hosted *auth broker* RPC to resolve a typed
  reference (a `SecretKeySelector`, or a projected ServiceAccount token via TokenRequest) within
  the correct (possibly referent) namespace. Referent-auth namespace logic already lives in the
  ClientManager per 007, so it stays in core; the provider never needs blanket Secret read access,
  and raw kubeconfig is never shipped. This uses bidirectional gRPC (the HashiCorp go-plugin
  broker pattern).
* **Direct RBAC.** The provider reads referenced material itself using its own ServiceAccount RBAC
  (CSI-like). Simpler, but grants the provider broader standing access and pushes referent-auth
  handling into every provider. Offered for providers that genuinely need arbitrary cluster reads.

**Ambient cloud identity** (IRSA, GKE Workload Identity, Azure Managed Identity) attaches to the
provider pod's ServiceAccount, so identity becomes per-provider and least-privilege by default.
This is desirable for new out-of-tree providers, and it is precisely why moving an *existing*
in-tree provider out-of-tree is a behavioural change (its effective identity moves from the core
pod to the provider pod). That migration is out of scope here.

#### Upgrade and rollback

Bumping `spec.package` creates a new `ProviderRevision`. Under `Automatic` activation the new
revision must pass verification, CRD install (including CRD schema conversion if the config CRD
version changed), and handshake before it takes over the GVK registrations. The previous revision
is kept `Inactive`. Rollback is setting the package back (or activating the prior revision under
`Manual`).

#### Uninstall and garbage collection

Deleting a `ProviderPackage` cascades to its `ProviderRevision`s. A revision uses a finalizer:
if any config CRs (for example `Vault` objects) still exist for a CRD it owns, deletion is blocked
and surfaced as a condition, so provider config CRDs are never yanked out from under live
`SecretStore`s. A `deletionPolicy` (`Orphan` vs `Delete`) governs whether the installed CRDs are
removed with the package.

#### Versioning

* The **runtime contract** carries its own semantic version negotiated at `Handshake`. Core
  supports a documented range; a provider outside it is refused activation rather than run
  half-compatibly.
* The **config CRD** versions independently per 007, so a community provider can stay `v1alpha1`
  while a mature one is `v1`, each shipped in its own package.

### Drawbacks

* **Complexity.** Three planes, a package-manager controller, dynamic CRD management, and a gRPC
  transport add significant surface compared to the in-tree model.
* **Supply chain.** In-tree providers are vetted by ESO maintainers and released as one signed
  artifact. Out-of-tree packages shift trust to the image source. Signature verification, digest
  pinning, and an allowed-registry policy mitigate but do not remove this. This is the single
  biggest new risk and must be documented prominently.
* **Operational overhead.** Each provider becomes at least one more Deployment, Service, SA, and
  RBAC binding to run and observe.
* **New failure modes.** Image pull failure, signature failure, CRD conflict, provider
  crash-loop, gRPC unreachable, and runtime-version mismatch are all new states the operator must
  understand.
* **Privileged manager.** Installing CRDs and creating RBAC/Deployments requires a powerful
  package-manager ServiceAccount; a compromised package source is high impact.
* **Auth model migration.** The broker path is new machinery, and the identity shift for any
  future in-tree-to-out-of-tree move is a real compatibility nuance.

Note one thing this model does *not* lose: because the provider installs a real config CRD,
provider configuration is fully schema-validated by the API server. A pure gRPC-with-opaque-JSON
plugin design would have sacrificed that; this design keeps it.

### Acceptance Criteria

* **Rollout / rollback.** Ship behind the experimental feature flag
  `--experimental-enable-provider-packages` (default `false`) at `alpha`, per the notice at the top
  of this document and the `experimental-*` convention in 014. The whole mechanism is opt-in and
  off by default. `ProviderRevision` history provides in-mechanism rollback; disabling the flag
  leaves in-tree providers fully functional. The flag graduates to a stable `--enable-*` form only
  when the feature leaves alpha.
* **Test roadmap.**
  * Unit tests for the package-manager reconcile logic (pull, verify, revision lifecycle, GC).
  * Envtest for CRD install/conflict/removal and ownerReference/finalizer behaviour.
  * A reference provider package built from the existing `fake` provider via the gRPC adapter,
    exercised end-to-end (install package, create `SecretStore.providerRef`, resolve an
    `ExternalSecret`, upgrade, roll back, uninstall).
  * Contract tests that run the same provider both in-process and over gRPC and assert identical
    behaviour.
* **Observability.** Conditions on `ProviderPackage` and `ProviderRevision`
  (`Installed`, `Healthy`, `RuntimeReachable`) plus Kubernetes events. gRPC client metrics per
  provider (request rate, error rate, latency) and provider Deployment health.
* **Monitoring.** Example alerts: revision not Healthy, provider unreachable, package signature
  verification failing, provider error rate above threshold. A dashboard per provider.
* **Troubleshooting.** A documented decision tree keyed on the condition reasons above, covering
  each failure mode in Drawbacks (image pull, signature, CRD conflict, crash-loop, unreachable,
  version mismatch).

## Alternatives

* **Status quo plus 007 only (in-tree, config-CRD split).** Simplest, and it delivers per-provider
  config CRDs and opt-out at build time. It does not deliver runtime install/upgrade, independent
  lifecycle, third-party/private providers without a fork, or a slimmer core artifact. This design
  is a strict superset that reuses 007 as its configuration plane.
* **Pure gRPC plugin with opaque config (no dynamic CRDs).** A single `plugin` provider type on the
  existing oneOf whose config is opaque JSON validated only by a `ValidateStore` RPC. Much simpler
  packaging, but it loses server-side schema validation and per-provider API versioning, and it
  gives a worse config UX. Rejected in favour of shipping real CRDs.
* **Webhook-only providers.** ESO already has a `webhook` provider that fetches over HTTP. It is
  useful and stays, but it is read-oriented, has no rich typed configuration, and is not a general
  replacement for native providers.
* **HashiCorp go-plugin (subprocess, not pods).** Run providers as subprocesses of the core pod
  over the go-plugin gRPC broker. Lighter operationally (no extra Deployments) and it solves the
  auth broker elegantly, but it couples provider lifecycle and identity to the core pod, which
  directly contradicts the independent-lifecycle and per-provider-identity goals. The broker
  *pattern* is still borrowed for auth resolution; the *process model* (separate workloads) is
  kept.
* **Crossplane-style OCI packages plus dynamic CRDs plus gRPC runtime (this proposal).** Highest
  complexity, but the only option that meets all stated goals: runtime lifecycle, independent
  versioning, third-party distribution, per-provider identity, and preserved schema validation,
  while staying fully backward compatible with in-tree providers.

## Open Questions

* Reuse Crossplane's `crossplane-runtime` package machinery / `xpkg` format directly, or define a
  minimal ESO-specific package format? Reuse buys maturity at the cost of a heavy dependency.
* Broker vs direct-RBAC as the *only* supported auth model, or both? Both increases surface.
* Does the package manager live in the core controller, or as a separate optional controller
  (so clusters that never use packages do not run it at all)?
* Migration path and tooling for republishing selected in-tree providers as packages, and the
  eventual (major-version) deprecation schedule for removing them from core.
