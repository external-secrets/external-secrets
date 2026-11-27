# Provider Capability Boundary Design

## Context

ESO v1 compiles provider implementations into the controller. Provider dispatch,
maintenance metadata, and coarse capabilities currently involve API-level
interfaces and registries. JP's experimental `feature-flags` branch
attempted to consolidate these into `runtime/provider`,
but preserved API lookup functions by
installing runtime callbacks from package `init()` functions.

That callback approach avoids a Go import cycle but leaves the `apis` module
behaviorally dependent on `runtime`. It also does not fit ESO v2, where providers
run out of tree and expose operations over gRPC.

The Kubernetes v1 API remains stable. Exported Go interfaces and package layout do
not need source compatibility.

## Goals

- Keep `apis` independent from runtime implementation and initialization.
- Preserve `SecretStore.status.capabilities` with its current `ReadOnly`,
  `WriteOnly`, and `ReadWrite` values.
- Derive that status from provider-declared features, not runtime health or
  effective credentials.
- Optionally expose all those provider-declared features (not their runtime status) into
  a new field.
- Keep v1 in-process dispatch as a compatibility adapter.
- Give v1 and v2 a common controller-facing capability contract.
- Update our existing provider catalog/metadata based on declared capabilities
- Generate {code-version,capabilities} metadata separately from our code base.
- Generate our provider catalog documentation based
  on {code-version,capabilities} metadata.

## Non-Goals

- Do not infer capabilities from IAM permissions, credentials, reachability, or
  backend health and report it back to API.
- Do not use the v1 registry as v2 provider discovery.
- Do not route v1 provider calls through loopback gRPC.
- Do not preserve exported Go registry or provider interfaces in `apis`.
- Do not define v2 provider governance or catalog inclusion policy here.
- Fix bloated interfaces. (This will come with a follow up design document).

## Module Boundary

The dependency direction is:

```text
apis
  ^
runtime/provider
  ^
providers and controllers
```

The `apis` module owns:

- Kubernetes CRD and status types.
- Serialization and defaulting.
- Deterministic structural validation.
- The existing coarse `SecretStoreCapabilities` API values.

The `apis` module does not own:

- Provider implementations or clients.
- Provider registries.
- Runtime lookup callbacks.
- gRPC discovery or connection state.
- Provider-aware runtime validation.

Controllers import both API and runtime packages. They obtain runtime facts, map
them to API values, and write the Kubernetes status subresource. Runtime data does
not need to flow through executable code in the API module.

## Runtime Contracts

Provider responsibilities are split into small interfaces under
`runtime/provider`:

```go
type ClientFactory interface {
	NewClient(context.Context, esv1.GenericStore, string) (SecretsClient, error)
}

type StoreValidator interface {
	ValidateStore(context.Context, esv1.GenericStore) (admission.Warnings, error)
}

type StoreCapabilitiesReader interface {
	Capabilities(context.Context, esv1.GenericStore) (CapabilitySet, error)
}

type StoreBackend interface {
	ClientFactory
	StoreValidator
	StoreCapabilitiesReader
}
```

The implementation object owns dependencies such as the Kubernetes client,
connection pool, and v1 registry. The source namespace passed to `NewClient` is
the namespace of the resource using the store. The semantic contract is:

- `Capabilities` returns features declared by the provider implementation.
- It does not probe effective permissions or report runtime health.
- `CapabilitySet` is detailed and transport-neutral (local/grpc).
- Current kubernetes status can use the `CapabilitySet` to declare
  `ReadOnly`, `WriteOnly`, `ReadWrite`.
- A future detailed Kubernetes status can reuse the same set.

V1 provider implementations do not need a repository-wide method rename. A v1
adapter reads capabilities from provider metadata. A v2 adapter reads them from
the gRPC capability RPC. These are the direct implementations of
`StoreCapabilitiesReader`.

## V1 Compatibility Registry

The v1 registry is private runtime infrastructure:

```go
type RegistryEntry struct {
	ClientFactory
	StoreValidator
	Metadata
}
```

Registration has no separate dispatch-name argument:

```go
Register(NewProvider(), ProviderSpec(), Metadata())
```

The dispatch key is derived from the single discriminator in `ProviderSpec`.
Display names and documentation slugs belong in metadata and do not select an
implementation.

The registry is used only for:

- Selecting an in-process v1 provider implementation.
- Reading v1 provider metadata at runtime.
- Supporting selective provider builds.

It is not the v2 discovery mechanism and is not exposed through `apis`.

## V1 And V2 Dispatch

The existing provider client manager is the dispatch boundary. A separate generic
`Resolver` interface is not introduced.

```text
SecretStore
  -> provider manager
     -> inline v1 provider -> v1 registry -> in-process adapter
     -> v2 runtimeRef      -> gRPC adapter
```

Controllers depend on manager behavior rather than registry maps. V2 core still
owns reference resolution, TLS, gRPC clients, connection pooling, retries,
protocol compatibility, lifecycle, and metrics. It does not register remote Go
implementations.

Connection pools and observation caches are runtime state, not provider
registries.

## Capability Status Flow

V1 capability flow:

```text
provider Metadata().Capabilities
  -> v1 StoreCapabilitiesReader
  -> CapabilitySet
```

V2 capability flow:

```text
out-of-tree provider capability RPC
  -> gRPC StoreCapabilitiesReader
  -> CapabilitySet
```

Controller projection:

```text
CapabilitySet
  -> derive ReadOnly / WriteOnly / ReadWrite
  -> SecretStore.status.capabilities
```

The controller writes capability status only after successful discovery. A failed
gRPC query must not become an implicit `ReadOnly` result and must not overwrite a
previously known value. Reconciliation and readiness error handling remain
separate from capability semantics.

## Validation Boundary

The API module retains validation that depends only on object content, including:

- Exactly one inline v1 provider.
- Required references.
- Enum and field-shape checks.
- Namespace and referent rules expressible without provider execution.

Runtime-aware validation moves out of `apis`:

- V1 provider-specific `ValidateStore` runs from a runtime webhook handler.
- V2 provider-owned CRDs validate their own configuration.
- ESO validates v2 references structurally and performs remote validation during
  reconciliation.

This removes `SetRegistryHooks`, side-effect imports, and API tests that import
runtime.

## Maintenance Metadata

V1 maintenance and deprecation information belongs on our provider catalog
documentation. This needs to be generated from provider implementation.

For v2, public maintenance history still belongs to the provider catalog unless a future
Kubernetes API requirement explicitly needs it. It is not part of capability or
runtime-health reporting. We use the same process for v1 and v2, but the
_generation point_ is different: In v1, we generate from our code. In v2,
we request the provider to push metadata for compliance reason on a regular
basis (to be defined in a governance call).

## Documentation Metadata

Documentation uses two stages with separate ownership.

### Code-Version Extraction

- The v1 extractor imports all built-in provider packages using a generated
  package list and emits a versioned metadata artifact.
- The v2 extractor runs in each out-of-tree provider repository and emits the same
  artifact schema.
- The artifact describes one code version.
- V2 support declarations must be backed by conformance tests. Generated gRPC
  interfaces alone do not prove support because unimplemented server methods can
  still compile.

The extractor does not require the production runtime registry of the controller.

### Public Catalog Rendering

- Provider maintainers submit catalog metadata by pull request.
- Entries state when a capability became supported, such as `PushSecret` since
  version `2.3` (conformance check happen post release).
- Validation checks the metadata schema and version ranges.
- A renderer generates the public Markdown capability matrix.

The extraction artifact is version-specific evidence. Catalog metadata is the
maintained public statement. Core documentation does not import out-of-tree
provider implementations.

## Failure Rules

- Invalid or duplicate v1 registration fails during process startup.
- Missing v1 capability metadata is a registration error, not `ReadOnly`.
- Unknown capability values fail metadata extraction or gRPC decoding.
- Failed v2 capability discovery preserves the previous status value.
- The standalone `apis` module remains fully functional without runtime imports.

## Verification

- Run standalone `apis` module tests to prove runtime independence.
- Test invalid, duplicate, selective-build, and successful v1 registration.
- Test coarse derivation for read-only, write-only, read-write, empty, and invalid
  capability sets.
- Test that v1 metadata and a fake gRPC response produce equivalent SecretStore
  status.
- Test that capability discovery errors do not overwrite known status.
- Validate extraction artifacts against a versioned schema.
- Keep provider behavior and conformance tests aligned with declared detailed
  capabilities.

## Implementation Order

1. Add runtime metadata and transport-neutral capability types.
2. Add the split runtime interfaces and v1 adapters.
3. Introduce the private v1 compatibility registry.
4. Move provider lookup, provider-aware validation, and maintenance lookup out of
   `apis`.
5. Remove API registry functions and callback hooks.
6. Project capabilities through the SecretStore controller.
7. Add the independent v1 metadata extractor.
8. Implement the same runtime contracts in the v2 gRPC adapter.
