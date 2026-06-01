# External Secrets Operator

Kubernetes operator that synchronizes secrets from external providers (AWS Secrets Manager, Vault, GCP Secret Manager, Azure Key Vault, etc.) into Kubernetes Secrets.

## Build and Test

Use `make` targets — refer to the Makefile for available commands. Do not run `go test`, `golangci-lint`, or `helm` directly.

## Project Layout

Single binary built from `main.go`. The **controller** reconciles ExternalSecrets into K8s Secrets. The **webhook** (validates and defaults CRDs) and **certcontroller** (manages webhook TLS) are subcommands registered via `rootCmd.AddCommand()`.

Multi-module repo: `apis/`, `runtime/`, `e2e/`, and each `providers/v1/*/` have their own `go.mod`.

## Non-Obvious Patterns

- `make reviewable` is the gate for PRs. Run it, not individual checks.
- Helm chart is the source of truth for deploy manifests. `make manifests` generates static YAML from it.
- Provider docs use MkDocs snippet transclusion (`--8<--`). Auth docs are shared across providers via `docs/snippets/`.
- CRD tests use snapshot testing. Run `make test.crds.update` to update snapshots after CRD changes.
- `make update-deps` updates dependencies across all modules at once.
- Add a `git notes add HEAD` entry on every non-trivial commit. Record key design decisions, trade-offs, and gotchas. Queryable via `git notes show <sha>`.
- If you discover a non-obvious pattern while implementing, add it here before the PR is merged. Keep entries general — applicable across the codebase, not specific to one provider or feature.
- Never edit `zz_generated.*` files by hand. They are owned by controller-gen. Modify the source types and run `make generate` (included in `make reviewable`).
- After everything is committed - **ALWAYS RUN `make check-diff`** - this is the first step where PRs fall apart that LLMs forget - there are a lot of generated code outside of the main `make reviewable` spec like helm chart tests, docs, etc.

## Adding a Provider

A provider is its own Go module under `providers/v1/<name>/` with no build tags on the package itself.
Build tags live in `pkg/register/<name>.go`.

### API types

- New spec goes in `apis/externalsecrets/v1/secretstore_<name>_types.go`.
- Add a one-line slot to the discriminator union in `apis/externalsecrets/v1/secretstore_types.go`
  (the `SecretStoreProvider` struct). The JSON tag is the provider name; `apis/externalsecrets/v1/provider_schema.go`
  resolves it from the first JSON key of the marshaled union.
- Auth: nested `*<Name>Auth` struct. Multi-method auth uses `+kubebuilder:validation:MaxProperties=1`. Selector types
  are `esmeta.SecretKeySelector` and `esmeta.ServiceAccountSelector`.
- CA: include `CABundle []byte` and `CAProvider *CAProvider` if the backend speaks TLS.
- v1 API is frozen by default. Net-new provider slots are fine

### Runtime helpers (use these, do not roll your own)

- `runtime/esutils/resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, ref)` for credential resolution. It enforces
  `ClusterSecretStore` vs `SecretStore` namespace scoping. Pass `store.GetKind()` and the ES namespace.
- `runtime/esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{...})` for CA bundles.
- `runtime/esutils.ValidateSecretSelector` / `ValidateReferentSecretSelector` / `ValidateServiceAccountSelector` for spec validation.
- `runtime/esutils/metadata` for parsing `PushSecretMetadata` into a typed spec.
- `runtime/constants` for metric label values.

### `SecretsClient` contract

Defined at `apis/externalsecrets/v1/provider.go`. All eight methods are mandatory; `Close` may be a no-op.

- Return `esv1.NoSecretErr` from `GetSecret` when the secret is missing. The reconciler depends on this for `deletionPolicy`.
- Set `Capabilities()` honestly: `SecretStoreReadOnly`, `SecretStoreWriteOnly`, or `SecretStoreReadWrite`. Read-only
  providers still implement Push/Delete but return a sentinel error! Do _NOT_ return `nil`!
- `gjson` is the conventional path extractor for `ref.Property` on JSON payloads.

### Caching (skip unless construction is expensive)

- Per-Provider client cache: `runtime/cache.Must[T](size, cleanup)`. Keyed by `cache.Key{Name, Namespace, Kind}`,
  versioned by `store.GetObjectMeta().ResourceVersion`. Use this for OIDC, vault leases, token exchange, etc. Default to no cache.
- Per-secret cache (in the SecretsClient): `expirable.LRU[string, []byte]` with a user-facing `CacheConfig{TTL, MaxSize}`
  field on the spec.

### Feature flags

Pipeline: helm value to deployment `extraArgs` to cmd flag to `feature.Register` to `Initialize()`.

- Register flags from the provider's `init()` using `runtime/feature.Feature{Flags, Initialize}`.
- `cmd/controller/root.go` collects them and runs `Initialize` after manager startup.
- Helm wiring is `extraArgs` in `deploy/charts/external-secrets/values.yaml`, rendered by `templates/deployment.yaml`.
  Out-of-process SDKs (e.g. bitwarden) ship as a sidecar subchart.

### Registration

Provider package exports three symbols: `NewProvider() esv1.Provider`, `ProviderSpec() *esv1.SecretStoreProvider`,
`MaintenanceStatus() esv1.MaintenanceStatus`. `ProviderSpec()` must set exactly one field on the union.

Registration lives in `pkg/register/<name>.go`:

```go
//go:build <name> || all_providers
package register

import (
    esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
    foo "github.com/external-secrets/external-secrets/providers/v1/foo"
)

func init() {
    esv1.Register(foo.NewProvider(), foo.ProviderSpec(), foo.MaintenanceStatus())
}
```

Maintenance values: `MaintenanceStatusMaintained`, `NotMaintained`, `Deprecated` (`apis/externalsecrets/v1/provider_schema_maintenance.go`).

### Wiring

- Add `providers/v1/<name> => ./providers/v1/<name>` to root `go.mod` (alphabetized).
- `Makefile` honors `PROVIDER ?= all_providers` and passes it as `go build -tags`.

### Documentation

- Write `docs/provider/<slug>.md`. Conventional sections: intro, Authentication or Store Configuration, External Secret Spec / GetSecret, optional PushSecret.
- YAML examples live in `docs/snippets/<name>-secret-store.yaml`, `<name>-external-secret.yaml`, `<name>-push-secret.yaml`.
  Pull them in via `{% include '<name>-secret-store.yaml' %}`.
- Add nav entry to the `Provider:` block in `hack/api-docs/mkdocs.yml`. Order is historical; append at the bottom.

## Adding a Generator

A generator is its own Go module under `generators/v1/<name>/`. Generators are **v1alpha1 only** and are
**unconditionally compiled** into the binary (no build tags, unlike providers).

The repo ships a scaffold: `esoctl bootstrap generator --name <Name>` (`cmd/esoctl/generator/bootstrap.go`).
Run it first; the manual steps below are the audit checklist for what it produced and what it skipped.

### What `esoctl bootstrap generator` wires for you

- Creates `apis/generators/v1alpha1/types_<pkg>.go` (CRD types).
- Creates `generators/v1/<pkg>/{<pkg>.go,<pkg>_test.go,go.mod,go.sum}` from templates in `cmd/esoctl/generator/templates/`.
- Patches `pkg/register/generators.go` with the import and `genv1alpha1.Register(<pkg>.Kind(), <pkg>.NewGenerator())`.
- Patches `apis/generators/v1alpha1/types_cluster.go`: enum value, `GeneratorKind<Name>` const, and a field on
  `GeneratorSpec` (the discriminator union).
- Adds the `replace` directive to root `go.mod`.
- Patches `runtime/esutils/resolvers/generator.go` `clusterGeneratorToVirtual` switch.
- Patches `apis/generators/v1alpha1/register.go` (`<Name>Kind` var + `SchemeBuilder.Register`).
- Patches `apis/externalsecrets/v1/externalsecret_types.go` `GeneratorRef.Kind` enum. This is the one v1 enum write the
  bootstrap performs; it is documentation-class, not behavioral.

What it does **NOT** do: ClusterRole RBAC, mkdocs nav, docs, snippets, helm.

### API types

- All generators live in `apis/generators/v1alpha1/`. No v1beta1, no v1.
- Per-generator file is `types_<name>.go`. Standard shape: `<Name>Spec`, `<Name>` (TypeMeta + ObjectMeta + Spec), `<Name>List`.
  Most generators have no Status field.
- Standard markers: `+kubebuilder:object:root=true`, `+kubebuilder:storageversion`, `+kubebuilder:subresource:status`,
  `+kubebuilder:metadata:labels="external-secrets.io/component=controller"`,
  `+kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}`.
- All concrete generators are `scope=Namespaced`. Cluster-scoped use is delivered by the single `ClusterGenerator`
  umbrella type (`apis/generators/v1alpha1/types_cluster.go`) which embeds a `GeneratorSpec` discriminator union with
  `MaxProperties=1` / `MinProperties=1`. Do **NOT** write a `Cluster<Name>` type. Add one field to that union and one
  `GeneratorKind` enum value.

### `Generator` interface

Defined at `apis/generators/v1alpha1/generator_interfaces.go`. Two methods:

```go
Generate(ctx, obj *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, GeneratorProviderState, error)
Cleanup(ctx, obj *apiextensions.JSON, status GeneratorProviderState, kube client.Client, namespace string) error
```

- Spec arrives as raw `apiextensions.JSON`. YAML-unmarshal it inside `Generate`.
- Returns the full `map[string][]byte` of generated keys at once. There is no per-key `GetSecret`.
- `GeneratorProviderState` is `*apiextensions.JSON`, an opaque blob persisted between `Generate` and `Cleanup`.
- `Cleanup` MUST be idempotent.

### Runtime helpers

- `runtime/esutils/resolvers.SecretKeyRef(ctx, kube, resolvers.EmptyStoreKind, ns, ref)` for credential refs. Generators
  pass `EmptyStoreKind` because they have no SecretStore; namespace scoping does not apply.
- `runtime/esutils.FetchServiceAccountToken` for SA-token auth, `esutils.ExtractJWTExpiration` for JWT parsing.
- AWS-family generators reuse the provider's auth path:
  `awsauth "github.com/external-secrets/external-secrets/providers/v1/aws/auth"` then `awsauth.NewGeneratorSession(...)`.
  Vault generator imports `providers/v1/vault` and calls `provider.NewGeneratorClient`. Cross-module imports of providers
  are normal; wire via `replace` in the generator's own `go.mod`.

### State and lifecycle

Stateless **by default**. Return `nil` for `GeneratorProviderState` from `Generate` and a no-op `Cleanup` (uuid, password,
ecr, sts all do this).

Stateful generators return a non-nil state. `runtime/statemanager` persists it to a `GeneratorState` CR
(`apis/generators/v1alpha1/generator_state_types.go`). The `generatorstate` controller runs a finalizer that calls
`Cleanup` on deletion. If state persistence fails post-Generate, statemanager invokes Cleanup as rollback; if Cleanup
itself errors, it creates a `GeneratorState` with an immediate `GarbageCollectionDeadline`.

No generator currently uses `runtime/cache.Must` style client caching.

### Registration

Generator package exports two symbols: `NewGenerator() genv1alpha1.Generator` and `Kind() string`. Registration lives in
`pkg/register/generators.go`:

```go
import (
    genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
    foo "github.com/external-secrets/external-secrets/generators/v1/foo"
)

func init() {
    genv1alpha1.Register(foo.Kind(), foo.NewGenerator())
}
```

`Register` panics on duplicate kinds. Scheme registration is separate, in `apis/generators/v1alpha1/register.go`:
`<Name>Kind = reflect.TypeFor[<Name>]().Name()` and `SchemeBuilder.Register(&<Name>{}, &<Name>List{})`.

The runtime resolver (`runtime/esutils/resolvers/generator.go`) loads the typed object via the scheme then dispatches to
the registered `Generator` by kind. ClusterGenerator goes through `clusterGeneratorToVirtual` which materializes a
synthetic namespaced object from the union spec; every generator must have a case there.

### Feature flags

No precedent. None of the existing generators register `runtime/feature` flags. If you need one, follow the provider
pattern, but expect to be the first.

### Documentation

- Generator docs live at `docs/api/generator/<name>.md`.
- YAML snippets in `docs/snippets/<name>-...yaml`, transcluded via `--8<--`.
- Nav entry goes under `Reference: -> API: -> Generators:` in `hack/api-docs/mkdocs.yml`. Append at the bottom.

### Manual checklist after bootstrap

- ClusterRole rules in `deploy/charts/external-secrets/templates/rbac.yaml` for any new resources the generator reads.
- Docs page + snippets.
- mkdocs nav entry.
- After adding the module to `go.work`, run `go work use` to reconcile the `go` directive version.
