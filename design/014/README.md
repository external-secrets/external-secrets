```yaml
---
title: Out of Process Providers (Provider v2)
version: v1alpha1
authors: Moritz Johner, Gustavo Carvalho
creation-date: 2025-11-05
last-updated: 2026-11-09
status: draft
---
```

# Out of Process Providers


This design document is split into separate sub-documents. Each sub-document analyzes different approaches and discusses trade-offs.

> This revision supersedes the 2025 draft that was never merged (PR #5544, branch
> `mj-design-oopp`); the directory keeps its original name so that the two can be
> compared. `design/014-feature-flag-consolidation.md` also uses number 014;
> renaming this directory to `017-provider-v2/` at merge time is a one-line
> `git mv` and is left to the maintainers.

## Overview

- [API.md](./API.md) - API Design for Out-of-Process Providers
- [PROTOCOL.md](./PROTOCOL.md) - The ESO ⇄ provider gRPC contract (issue #7215) ([STALE]: to be updated on follow up PRs)
- [VALIDATION.md](./VALIDATION.md) - Who validates what, where, and who writes which object (issue #7214) ([STALE]: to be updated on follow up PRs)
- [mTLS.md](./mTLS.md) - mTLS & Service Discovery for Out-of-Process Providers
- [ADAPTER.md](./ADAPTER.md) - Out-of-Process Provider Adapter Pattern
- [PROVIDER_AUTOGEN.md](./PROVIDER_AUTOGEN.md) - Provider Code Generation
- [TESTING.md](./TESTING.md) - V2 Provider Testing Strategy
- [THREAT_MODEL.md](./THREAT_MODEL.md) - Threat Model for Out-of-Process Providers
- [ROADMAP.md](./ROADMAP.md) - Delivery as PRs of at most 500 hand-written lines ([STALE]: to be updated on follow up PRs)

## Scope

This document covers the technical aspects of out-of-process providers.

Multi-repository structure, user migration tooling beyond `esoctl migrate store`, and provider governance are out of scope. Governance is covered by PR #7124; the provider capability boundary inside the core is covered by PR #6829 and is a prerequisite that this design builds on.

## Problem Statements

For detailed discussion, see: [recording](https://zoom.us/rec/share/fioR9a-blopYRqALdvphBU2rXN-To3BAuMOZK1FdpxUIIe31Qam3oWDxGWFHKTT9.UmkUa-3W5abZww8-), [meeting notes](https://zoom-lfx.platform.linuxfoundation.org/meeting/95703312466-1761822000000/summaries?password=47fac69c-506a-408d-a424-ff3c1c69a0dc), [drawings](https://excalidraw.com/#room=bf94104fddcd05c32c20,BTAv5naTXFc-6m0275D0_g), and [this previous PR](https://github.com/external-secrets/external-secrets/pull/4792). There's also an [old issue](https://github.com/external-secrets/external-secrets/issues/696) with more research and history, although outdated. The umbrella issue is #694; the 2025 initiative issue is #5218.

#### Risk of Supply-Chain Attacks

ESO bundles many dependencies, creating a large attack surface. We need to reduce this surface area to improve security.

#### Users want to maintain their own providers without merging into core ESO

ESO provides limited support for providers built and maintained outside the ESO domain. Enterprises with custom APIs or workflows for managing secrets cannot integrate them effectively. Currently, only the webhook provider supports such integrations, but it lacks features like `GetAllSecrets()` and advanced authentication methods such as OAuth flows.

#### Independence of Provider Maintainers

Provider maintainers need the freedom to change their APIs. This is not possible today because everything is bundled in ESO. The GA SecretStore CRD cannot be changed in breaking ways.

#### The ESO core team does not want to maintain everything

With approximately 40 providers bundled in ESO, maintaining them has become a burden. We want to shift responsibility to the community for provider maintenance. Issue #6651 paused the acceptance of new in-tree providers until this design ships.

## Requirements

#### A Provider must run in a separate Pod

Each provider must run in a separate Pod to enable network isolation. When users run multiple providers, each can have its own network policy. This enforces the principle of least privilege at the network level. Core ESO does not need access to anything outside the cluster. All secrets flow through provider pods.

Core ESO and providers must run with different RBAC permissions. We discourage sidecar deployments. ESO must provide a secure-by-default deployment that allows users to apply NetworkPolicies post-deployment and ensures that ESO Core and providers run with separate RBAC permissions to enforce least privilege.

#### A Provider must be built with a minimum set of dependencies

A provider binary should typically import only the ESO `apis` module, the `proto` module, the provider SDK and the provider-specific SDK. This keeps providers lightweight, easy to audit, and easy to keep up to date.

#### ESO Core must be built without provider dependencies

ESO Core must be built without provider dependencies. This reduces the attack surface by removing unneeded dependencies. This also means the core must not import, register in its scheme, or hold RBAC for any provider-owned CRD; the core only ever handles provider configuration as `apiVersion/kind/name/namespace` strings.

#### Provider code must stay coherent

We must not fork or branch the existing provider source code for the out-of-process provider implementation. Provider code must remain coherent to avoid the maintenance burden of keeping multiple versions in sync. A v2 provider process imports the existing `providers/v1/<name>` module and wraps it with the SDK adapter; it never copies it.

#### Existing manifests keep working, adoption is a store edit

`ExternalSecret` and `PushSecret` do not change. Adopting an out-of-process provider is an edit to the `SecretStore`, and `esoctl migrate store` performs that edit mechanically.

#### No new cluster-level install dependency by default

The default install path must not require cert-manager or any other operator. Certificates are issued by the existing cert-controller deployment; cert-manager remains an option.

## Prior work and how the branches relate

Nine proposals or branches exist. None is merged.

| Ref | Author | Type | Opened | State | Size | One-line summary |
| --- | --- | --- | --- | --- | --- | --- |
| #4792 | gusfcarvalho | design `014-secretstore-generator-v2.md` | 2025-05 | closed 2025-11 | +376 | `secretstore.external-secrets.io/v2alpha1 SecretStore` with `providerConfig{address, auth, providerRef}`; generators too; explicit `authenticationScope`; out-of-tree repos; conformance suite; three-phase migration |
| #5544 | moolen | this directory, 2025 draft | 2025-11 | closed 2026-07 | +1662 | API options (indirection vs direct reference, recommendation left blank), push-model mTLS with label discovery, bidirectional adapter, provider main autogen, testing strategy, threat model T1–T10 |
| #5527 | moolen | code "PoC 1" | 2025-10 | closed 2026-07 in favour of #6245 | +68758/−11286, 475 files | New `Provider`/`ClusterProvider` kind in `external-secrets.io/v1`; `ExternalSecret.secretStoreRef.kind: Provider`; fake, kubernetes, aws, gcp providers; e2e |
| #6245 | moolen → Skarlso | code "PoC 2" | 2026-04 | open draft | +55755/−1359, 498 files | `SecretStore` keeps centre stage: `runtimeRef` → `(Cluster)ProviderClass{address}`, `providerRef` → provider CRD; fake, kubernetes, aws; helm deploys providers; cert-controller mints certs; e2e; ten AI-written spec documents under `docs/superpowers/` |
| #6710 | Skarlso | code "PoC 3" | 2026-07 | open | +21435/−1509, 267 files | Same commits as #6245 minus aws/kubernetes providers, helm provider templates, e2e and the spec documents. Hand-written: +13.5k/−1k (of which +5.8k tests; +3.4k generated Go excluded) |
| #6632 | alekc | design `016-out-of-tree-providers.md` | 2026-07 | closed same day | +435 | Crossplane-style OCI `ProviderPackage`/`ProviderRevision`, dynamic CRD install, gRPC, auth broker; behind an experimental flag |
| #6829 | evrardj-roche | design `015-provider-capabilities-ready-for-v2.md` | 2026-08 | open draft | +289 | Move provider registry and provider-aware validation out of `apis` into `runtime/provider`; transport-neutral capability set; v1 adapter and v2 gRPC adapter implement the same small interfaces |
| #7124 | evrardj-roche | design `016-provider-governance.md` | 2026-10 | open draft | +362 | Provider Representatives, registry, maintained/unmaintained lifecycle; v2 providers external by default |
| #7214 / #7215 | evrardj-roche | issues | 2026-10 | open | – | Validation boundaries and the RPC contract are undocumented |

`#6710` is not an alternative to `#6245`; it is a cut of it. Both contain commit
`90098457b feat: add v2 provider runtime plumbing` (moolen) and the four
follow-up commits; `#6710` adds one commit (`118f84dcd feat: v2 poc take 3`)
that deletes the aws/kubernetes providers, the helm provider templates, the e2e
framework changes and the spec documents. The protobuf contract is byte-for-byte
identical across #5527, #6245 and #6710. `#5527` was closed by its reviewers in
favour of `#6245` because of the user-facing `Provider` kind.


## Assessment of the proof of concept against the goals

The goals, in the order this design weighs them: provider independence,
smaller security footprint of the core, user experience, no new install
requirements.

What #6245/#6710 get right:

- `ExternalSecret` and `PushSecret` are untouched. `SecretStore` stays the
  thing users reference. Migration is an edit to the store.
- Transport (`ProviderClass.spec.endpoint`) is separated from provider
  configuration (the provider's own CRD). This is the Gateway API pattern
  (`GatewayClass` → `Gateway` → `HTTPRoute`) and will be familiar.
- A namespaced `ProviderClass` and a cluster-scoped `ClusterProviderClass`
  give tenants their own runtimes without cluster-admin involvement.
- The provider-owned CRD is a real, schema-validated object. No opaque JSON
  blob that is validated only at runtime.
- The server-side adapter wraps the existing `esv1.Provider` implementations,
  so the forty providers do not fork. That is the single most important
  property for actually finishing the migration.
- The wire contract carries a *reference* to the provider CR, not its content.
  See [THREAT_MODEL.md](./THREAT_MODEL.md) T11 for why this is the safer choice.

What must change before any of it merges:

| Problem in #6710 | Why it matters | Decision |
| --- | --- | --- |
| `cmd/controller/root.go` registers `apis/provider/fake/v2alpha1` in the core scheme; core RBAC grants `get/list/watch` on `provider.external-secrets.io/fakes`; `pkg/register/generators.go` imports `providers/v2/fake/generator` | Core knows about a specific provider. Every new provider would need a core change and a core RBAC change. Goal 1 is defeated | Core never imports, registers or has RBAC on provider CRDs (D3) |
| `runtime/clientmanager` imports `providers/v2/adapter/store` and `providers/v2/common/grpc` | Violates design 015 (`runtime` may import `apis` and an allowlist only). Pulls provider modules into every provider's dependency graph | Proto becomes a root-level module; the client adapter lives in `runtime/provider/grpcclient`; the server SDK lives in `providers/v2/sdk` (D4) |
| One Secret `external-secrets-provider-tls` per namespace holds the CA key, the server cert and the **client** cert; validity 10 years; namespace derived by parsing the address; cert-controller configured with `--provider-service-names` | A provider pod with `secrets get` in its namespace can read ESO's client identity and impersonate the core towards any sibling provider. The rotation story is "never". Discovery by flag list means the chart must know every provider at install time | Per-`ProviderClass` identity, three secrets with distinct readers, 90-day leafs, hot reload, composable transport (`Managed` or `External` mutual TLS, IPC, flagged dev modes), discovery is the `ProviderClass` object itself (D8, D9) |
| `apis/externalsecrets/v1beta1/*` gains the same fields | `v1beta1` is `served: false` in the shipped CRDs. Pure churn plus conversion risk | `v1` only (D6) |
| Helm `v2.enabled: true` by default; `--store-requeue-interval` default changed from 5m to 30s in the same PR; `+kubebuilder:validation:Format=""` added to unrelated aws/github/vault/grafana fields | Behaviour changes hidden in a "plumbing" PR | Everything behind `--experimental-enable-provider-runtime`, default off; no unrelated changes (D11) |
| `providers/v2/fake/store` is an adapted copy of `providers/v1/fake` (340 + 629 test lines, about three quarters identical) | Two implementations to keep in sync, the exact thing the adapter pattern avoids | The v2 fake provider imports the v1 fake module (D5) |
| `providers/v2/fake/main.go` is marked "generated by `providers/v2/hack/generate-provider-main.go`" but the generator is not in the PR | Unbuildable claim | Hand-written `main.go` on top of an `sdk.Run()` helper; a generator only once three or more providers exist ([PROVIDER_AUTOGEN.md](./PROVIDER_AUTOGEN.md)) |
| Server adapter calls `provider.NewClient()` and `Close()` on every RPC | The provider process is a translation shim, not a runtime. #6245's own spec says "avoid creating and destroying a new backend client on every single RPC" | SDK caches clients per (provider CR UID, generation, credential namespace) (D7) |
| 395-line connection pool with health-check goroutines, 575-line client with per-call verbose logging | More code than the problem needs; gRPC `ClientConn` already multiplexes | One `ClientConn` per `ProviderClass` generation, in a small cache (D10) |
| `GeneratorProvider` gRPC service and adapter exist, but no core path ever calls them (`GeneratorRef` has no runtime indirection) | Dead surface in the first milestone | Generators out of scope for milestone 1; separate design (N2) |
| `AuthenticationScope` type added to `apis` and unused | API noise | Dropped; scope rules are expressed by `providerRef.namespace` rules in [API.md](./API.md) |

## Decisions

- **D1. API shape.** `SecretStore`/`ClusterSecretStore` v1 gain
  `spec.runtimeRef` and `spec.providerRef`, mutually exclusive with
  `spec.provider`. `ExternalSecret` and `PushSecret` do not change.
  `ProviderClass` (namespaced) and `ClusterProviderClass` (cluster) in
  `external-secrets.io/v1alpha1` describe a provider runtime. Details in
  [API.md](./API.md).
- **D2. By-reference wire contract.** Requests carry the provider CR
  coordinates and a `RequestContext` (the requesting object, the store it
  resolved through, the caller namespace, labels and annotations), never the CR
  body and never credentials. The provider reads its own CR and the Secrets or
  ServiceAccounts it references. Referent authentication is decided by the core
  and sent as `credential_namespace` and `allow_explicit_credential_namespace`:
  `providerRef.namespace` only says where the one shared CR lives (required on
  `ClusterSecretStore`, forbidden on `SecretStore`), and the references inside
  the CR follow v1's rule, so one CR serves many namespaces. The provider is
  the party that rejects an explicit namespace other than the CR's own through
  a `SecretStore`, since only it sees the CR; a namespace equal to the CR's is
  accepted, as in v1, so inline specs convert unchanged. Rationale in [THREAT_MODEL.md](./THREAT_MODEL.md) T11
  and T16 and in [PROTOCOL.md](./PROTOCOL.md) P1 and P9 ([STALE]: to be updated on follow up PRs).
- **D3. Core is provider-agnostic.** The core binary does not import, register
  or hold RBAC for any provider CRD. Its RBAC gains only `providerclasses` and
  `clusterproviderclasses` (+`/status`). It already holds cluster-wide access to
  Secrets, because it creates and updates target Secrets in every namespace,
  and it reads a class's client TLS Secret with that existing access wherever
  the class places it.
- **D4. Module layout.** `proto/` is a root-level module with no ESO
  dependencies. The gRPC client adapter lives in `runtime/provider/grpcclient`
  and depends on `apis` and `proto`. The provider SDK (server adapter, mTLS
  server, hot reload, metrics, `Run()`) lives in `providers/v2/sdk`. Provider
  modules (`providers/v2/<name>`) depend on the SDK and on their v1
  implementation module. `runtime` never imports `providers/`.
- **D5. No forks.** v2 provider processes wrap the existing v1 `esv1.Provider`
  implementation by importing its module. Provider configuration CRD specs
  reuse the v1 provider spec types where they exist (for example the fake
  provider CR spec is `esv1.FakeProvider`), so `esoctl migrate store` is a
  mechanical transformation.
- **D6. `v1` only.** `v1beta1` is not served and is not touched. A clean
  `v2alpha1 SecretStore` without `spec.provider` is introduced only after the
  `v1` clean subset (`providerRef` + `runtimeRef`) is in use, exactly as the
  #6245 authors concluded. No "compatibility mode" that serialises
  `spec.provider` over the wire.
- **D7. The provider process is a runtime.** The SDK caches backend clients
  per (provider CR UID, generation, credential namespace), checks the
  credential-namespace rule before serving a cache hit, and invalidates on
  change. The caching contract is not written yet; when it is, it must account
  for credential Secrets and ServiceAccount tokens that rotate without changing
  the CR generation (the v1 cache has the same gap, issue #6626): bounded
  client lifetime or invalidation on referenced-object change. Providers whose
  SDK serialises access (GCPSM holds a package-wide mutex from `NewClient` to
  `Close`) keep that serialisation inside the provider process, where it
  belongs; running such a provider out of process lets operators add
  deployments for throughput, which the in-tree model cannot offer. Providers
  that need their own CR status run their own small controller; that is
  optional.
- **D8. Transport is composable per `ProviderClass`.** A class composes an
  `endpoint` (TCP `address` or Unix `socket` for IPC), a `protocol` (`grpc`;
  `http` reserved), `tls` (on or off; the server certificate from the `Managed`
  cert-controller issuer or an `External` one verified through
  `caBundle`/`caRef`) and `auth`, how the core proves itself (`clientCertificate`,
  which is what makes the TLS mutual, issued by the same `Managed` issuer or
  referenced through `secretRef`; `serviceAccountToken` reserved). The default
  for an address is managed mutual TLS; the default for a socket is no TLS and
  no auth. Network compositions without an `auth` mechanism, without TLS or
  without server verification are development-only, require
  `--experimental-allow-insecure-provider-transport` and are reported with an
  `Insecure` condition. With the `Managed` issuer the client credential lives
  only in the ESO namespace and the provider namespace receives the CA and the
  server leaf only; leaf validity 90 days, CA 1 year, rotation with a 35-day
  lookahead, hot reload required in the SDK. Rationale: many clusters cannot
  run an in-cluster PKI, and some bootstrap theirs with secrets ESO delivers.
  Details in [API.md](./API.md) "Transport compositions" and
  [mTLS.md](./mTLS.md).
- **D9. No new install dependency by default.** `Managed` mode reuses the
  existing cert-controller deployment, so the default path adds one
  Deployment, one Service, one ServiceAccount and one Role per provider, plus
  the provider's CRDs and a `ProviderClass`, all shipped by the provider's
  chart. cert-manager is supported through the `External` issuer, not
  required. The IPC composition needs none of this: the provider ships as a
  sidecar of the controller.
- **D10. Small core plumbing.** One `grpc.ClientConn` per `ProviderClass`
  generation held in a cache; per-call deadline from the class; retries only
  for idempotent reads on `UNAVAILABLE`. The client manager stays in
  `pkg/controllers/secretstore`; it gains one dispatch branch.
- **D11. Flag-gated and inert until enabled.** All core behaviour is behind
  `--experimental-enable-provider-runtime` (registered through
  `runtime/feature`, default `false`) and helm `providerRuntime.enabled`
  (default `false`). A PR in the roadmap never changes unrelated defaults.
- **D12. Capabilities are strings on the wire.** `repeated string
  capabilities` named after RPC methods; the core projects them to the
  existing `ReadOnly`/`WriteOnly`/`ReadWrite` status and gates calls on them.
  Unknown strings are ignored. This is the consensus reached in #6620 and
  adopted by PR #6829.
- **D13. Validation boundary.** Content-only rules live in CRD schema and CEL
  (owner: whoever owns the CRD). The ESO webhook never calls provider code or
  RPCs for v2 stores. The ESO controller validates references and policy and is
  the only writer of ESO-owned status. The provider validates its own CR,
  enforces the credential-namespace rule and reports through the RPC response.
  Full matrix in [VALIDATION.md](./VALIDATION.md) ([STALE]: to be updated on follow up PRs).
- **D14. Delivery.** A sequence of PRs of at most 500 hand-written lines each,
  every one leaving `main` releasable, per [ROADMAP.md](./ROADMAP.md) ([STALE]: to be updated on follow up PRs).
  Existing branches are mined, not rebased.
- **D15. Authentication is the transport's, authorization is the provider's.**
  mTLS, a ServiceAccount token (reserved) or the container boundary for IPC
  tells a provider the call comes from the core. Whether a given request may
  touch a given secret is decided by the provider on the `RequestContext` it
  receives, with rules it keeps wherever it likes (its own CRDs, a config file,
  an external policy engine). The core enforces only namespace boundaries, as
  today; a core-side policy engine would make every provider's security depend
  on core policy. [PROTOCOL.md](./PROTOCOL.md) P8 ([STALE]: to be updated on follow up PRs).
## Responsibilities

### Provider (the out-of-process runtime and its chart)

- **Authentication.** Serve TLS with the certificate of its class: the server
  Secret that cert-controller fills (`Managed`) or one the operator issues
  (`External`), hot-reloaded. Verify the core's client certificate against the
  class CA when `auth.clientCertificate` is in use; validate the projected
  ServiceAccount token with `TokenReview` when `auth.serviceAccountToken` is in
  use (reserved). Over IPC, accept the container boundary as the identity.
  Never offer a plaintext network endpoint unless the class says so.
- **Authorization.** Decide whether a given request may read or write a given
  secret, on the `RequestContext` it receives (requesting object, source store,
  caller namespace, labels and annotations). Keep the rules in its own CRDs, a
  configuration file or an external policy engine; refuse with
  `PERMISSION_DENIED`; allow everything from the authenticated core by default.
  [PROTOCOL.md](./PROTOCOL.md) P8 ([STALE]: to be updated on follow up PRs).
- **RBAC on install**, shipped by its chart; the rules below are an example and
  each provider declares the set its backend needs: a `Role` on its own CRD kinds;
  `secrets get` and `serviceaccounts/token create` in the namespaces it serves,
  with the cross-namespace `ClusterRole` as an explicit opt-in for referent
  `ClusterSecretStore`s; the empty server-certificate Secret plus the `Role`
  that lets cert-controller fill it (`Managed`); the `(Cluster)ProviderClass`;
  optionally a `NetworkPolicy`. Never a rule on `external-secrets.io` or
  `generators.external-secrets.io`, never `list`/`watch` on Secrets
  cluster-wide, never access to the issuer or client Secrets.
- **Contract.** Load the configuration CR by the coordinates it receives;
  resolve credential references in `credential_namespace`, honour an explicit
  namespace only when `allow_explicit_credential_namespace` is true or when it
  equals `credential_namespace`, and reject it otherwise; return `NOT_FOUND` for a missing remote secret and for nothing
  else; advertise capabilities honestly and answer `UNIMPLEMENTED` for the
  rest; answer `Validate` with `VALID` or `UNKNOWN`; fill `Resolved` on every
  response; cache backend clients per (CR uid, generation, credential
  namespace); serve `Describe` and gRPC health; own its CRD schema and CEL;
  write status only on its own objects; ship its CRDs, its chart and its class.
  [PROTOCOL.md](./PROTOCOL.md) ([STALE]: to be updated on follow up PRs).

### Core (controller, cert-controller, chart)

- **Authentication.** Dial each class with that class's own client identity:
  the client certificate cert-controller issued (`Managed`) or the referenced
  one (`External`), or a projected ServiceAccount token (reserved); verify the
  server against the class CA; refuse unauthenticated, unencrypted or
  unverified network compositions unless
  `--experimental-allow-insecure-provider-transport` is set, and report them
  with `Insecure=True`. cert-controller issues and rotates one CA, one server
  leaf and one client leaf per `Managed` class; the CA key and the client leaf
  never leave the ESO namespace.
- **Authorization.** Namespace boundaries only, as today:
  `ClusterSecretStore.conditions`; the scope rules of `runtimeRef` and
  `providerRef` (a `SecretStore` reaches a class and a CR in its own namespace
  or a cluster class; a `ClusterSecretStore` reaches a `ClusterProviderClass`
  and a CR in a named namespace); RBAC on the class objects. Compute
  `credential_namespace` and `allow_explicit_credential_namespace` per call and
  send them; verify the provider's `Resolved` self-report and stop dispatch to
  a runtime that violates it. Never evaluate provider-side policy.
- **Formatting the Secret.** Everything between provider bytes and the target
  object stays in the core: `data`/`dataFrom` key mapping, `extract` and
  `find` handling, decoding and conversion strategies, `rewrite`,
  `creationPolicy` and `deletionPolicy`, ownership, labels and annotations,
  generic targets, immutability and hashing.
- **Templating the Secret.** `spec.target.template` rendering (`templateFrom`
  ConfigMaps, Secrets and literals, engine v2, merge policy) runs in the core;
  the wire carries rendered bytes for `PushSecret` and raw bytes for reads. A
  provider never sees a template (#7213 is orthogonal).
- **Lifecycling providers.** Reconcile `(Cluster)ProviderClass`: health,
  `Describe` into `status.serves`, `status.capabilities` and
  `status.protocolVersion`, the `Ready` and `Insecure` conditions,
  `ContractViolation`; keep one connection per class generation; issue
  certificates for `Managed` classes; gate everything behind
  `--experimental-enable-provider-runtime`. The core does not install, upgrade
  or scale provider workloads; their charts do.
- **SecretStore features.** Readiness from the `Validate` RPC into
  `status.conditions`, including `ValidationUnknown` for referent cluster
  stores; capability projection and gating; flood gate; `refreshInterval`,
  `retrySettings` and the `controller` class; PushSecret finalizers;
  `ClusterSecretStore` conditions; metrics and events; every status write on
  ESO objects. `ExternalSecret`, `PushSecret`, `ClusterExternalSecret` and
  generators are unchanged and stay in the core.

## Non-goals

- **N1.** Removing in-tree providers from the default build. That is a separate
  deprecation decision governed by PR #7124 and `DEPRECATING.md`. A
  `core`-flavoured image built with a reduced `PROVIDER=` tag list is a cheap
  early win and is in the roadmap, but it is additive.
- **N2.** Generators over gRPC. `GeneratorRef` has no runtime indirection and
  generators are compiled unconditionally today. The proto module reserves the
  `GeneratorProvider` service so the fake provider can exercise it, but no core
  path dispatches to it in milestone 1. A follow-up design covers it.
- **N3.** OCI packaging and a package manager (#6632). Provider charts are the
  distribution unit.
- **N4.** Providers outside the cluster. `ProviderClass.spec.endpoint.address`
  may point anywhere, and the `External` issuer makes it work, but the `Managed`
  issuer and the default documentation target in-cluster Services only.
- **N5.** *(revised 2026-11)* Sidecar providers over a Unix socket were
  rejected in #4792 and in the 2025 draft for isolation and scaling reasons.
  They are now a supported, opt-in composition (`endpoint.socket`,
  [API.md](./API.md) "IPC deployment model") because clusters without any
  PKI otherwise have no v2 path at all. The trade-offs (shared ServiceAccount,
  no per-provider cloud identity) are documented, and network deployments stay
  the default.
- **N6.** A `v2alpha1 SecretStore` in milestone 1 (see D6).
- **N7.** Changing templating (#7213). Templating stays in the core; the wire
  contract carries rendered bytes for `PushSecret`.

## Open questions

- **Q1.** Default class resolution. Should a store with `providerRef` and no
  `runtimeRef` resolve to the unique `(Cluster)ProviderClass` whose
  `status.serves` contains the referenced kind (like a default
  `StorageClass`)? It removes one field for users at the cost of a lookup in the
  core. Proposed: not in milestone 1; revisit once `status.serves` is populated.
- **Q2.** Cluster-wide credential reads by providers. For `ClusterSecretStore`
  referent authentication the provider must read Secrets in the caller's
  namespace. Proposed: the provider chart ships a namespaced `Role` by default
  and an opt-in `ClusterRole`; document the trade-off.
- **Q3.** Where the SDK and proto eventually live. In-tree modules first (same
  review gates, same release), a separate `external-secrets/provider-sdk`
  repository once the contract reaches `v1`.
- **Q4.** Whether cert-controller should be the issuer at all, or whether
  `Managed` mode should delegate to cert-manager when it is detected. Proposed:
  cert-controller first (no new dependency), detection later.

## Relationship to other proposals

- **Design 015 (module import boundaries)** is a constraint on this design.
  D4 exists to satisfy it, and the `depguard` rules should be extended to
  `runtime/provider` and `proto`.
- **PR #6829 (capability boundary)** is complementary and should merge first
  or in parallel. Its `ClientFactory` / `StoreValidator` /
  `StoreCapabilitiesReader` split is what the gRPC client adapter implements
  on the core side; its "Validation Boundary" section is consistent with
  [VALIDATION.md](./VALIDATION.md) ([STALE]: to be updated on follow up PRs).
- **PR #7124 (governance)** decides *who* maintains a provider; this design
  decides *how* a provider runs. Its "version 2 external by default" policy
  depends on this design shipping.
- **Issue #7213 (templating)** is orthogonal: rendering stays in the core and
  the wire carries rendered bytes.
- **Issue #6651 (pause on new providers)** can be lifted for v2 providers once
  the fake provider and the SDK are released, because new providers no longer
  touch the core.
