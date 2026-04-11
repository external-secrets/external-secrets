# V2 provider runtime plumbing with Kubernetes provider as the first hardened path

## Summary

This PR introduces the shared v2 provider runtime for out-of-process providers and wires the Kubernetes provider through the full controller -> clientmanager -> gRPC -> adapter path.

The Kubernetes provider is the first fully exercised provider on this path. Other providers can build on the same runtime pieces later, but the implementation and verification in this PR are intentionally centered on Kubernetes.

## What is in scope

- add the shared v2 API surface:
  - `Provider` and `ClusterProvider`
  - provider config CRDs
  - gRPC protobufs and transport helpers
- add controller support for `Provider` and `ClusterProvider`
- teach `runtime/clientmanager` how to resolve, cache, and invalidate v2 provider clients
- add the forward adapter:
  - gRPC -> synthetic v1 store -> existing provider interface
- add the reverse adapter:
  - v2 gRPC client -> `esv1.SecretsClient`
- add the first concrete v2 provider implementation for Kubernetes
- extend the e2e harness so Kubernetes v2 can run through the normal provider suite

## Kubernetes path polish in this branch

The WIP follow-up work in this branch tightened the Kubernetes v2 path in a few places that were previously incomplete or ambiguous:

- preserve `storeRef.kind` from the controllers into the gRPC `ProviderReference`
- preserve cluster provider auth scope through the reverse adapter path
- stop defaulting PushSecret store kinds in places where omission must remain distinguishable
- preserve PushSecret metadata payloads through the transport
- make Kubernetes config mapping prefer `providerRef.namespace` and only fall back to `sourceNamespace`
- add coverage for provider-namespace vs manifest-namespace auth scope
- add recovery coverage around provider and cluster-provider v2 PushSecret flows
- harden the focused v2 e2e loop and trim Docker build context noise for faster reruns

## Suggested review path

If you want to review this in the runtime order secrets flow through, this is the shortest path:

1. Controller entrypoints
   - `pkg/controllers/provider/controller.go`
   - `pkg/controllers/clusterprovider/controller.go`
   - `pkg/controllers/pushsecret/pushsecret_controller_v2.go`
2. Client resolution and caching
   - `runtime/clientmanager/manager.go`
3. gRPC transport
   - `providers/v2/common/proto/provider/secretstore.proto`
   - `providers/v2/common/grpc/client.go`
   - `providers/v2/common/grpc/pool.go`
   - `providers/v2/common/grpc/tls.go`
4. Adapter boundary
   - `providers/v2/adapter/store/server.go`
   - `providers/v2/adapter/store/client.go`
   - `providers/v2/adapter/store/synthetic_store.go`
5. Kubernetes provider
   - `providers/v2/kubernetes/main.go`
   - `providers/v2/kubernetes/config.go`

## Behavior that is now locked down

- namespaced `Provider` auth always resolves against the manifest namespace
- `ClusterProvider` namespace conditions are enforced before client creation
- `ClusterProvider.authenticationScope=ProviderNamespace` requires `spec.config.providerRef.namespace`
- TLS secret namespace resolution follows the effective auth namespace
- v2 client caching is generation-aware and namespace-sensitive
- pooled gRPC connections are released when the manager closes
- every RPC carries both `ProviderReference` and `SourceNamespace`
- the reverse adapter preserves remote refs, store kinds, and PushSecret metadata
- Kubernetes provider config lookup honors explicit provider-ref namespace before manifest fallback

## Verification

Focused Kubernetes v2 e2e coverage passed:

- `push-secret`
- `cluster-provider`
- `namespaced-provider`
- `capabilities`
- `metrics`

Fresh package verification passed:

- `runtime/clientmanager`
- `providers/v2/common/grpc`
- `providers/v2/adapter`
- `providers/v2/kubernetes`
- `cmd/controller`
- `pkg/controllers/provider`
- `pkg/controllers/clusterprovider`
- targeted v2 `pkg/controllers/pushsecret` tests

## Notes for reviewers

- this PR contains shared runtime pieces that future providers will reuse, but Kubernetes is the only provider path intentionally polished and covered end-to-end here
- the adapter `Capabilities()` comment is still an architectural TODO, but it is not a correctness blocker for the current Kubernetes implementation
- the OVH constructor fix is unrelated plumbing repair needed to keep the wider tree building during the rebase

## Follow-up

- extend Kubernetes v2 e2e coverage from this baseline
- bring additional providers onto the same runtime path one at a time
- revisit the longer-term provider/generator capability advertisement model once more than one v2 provider is live
