# Kubernetes Provider V2 Push Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve Kubernetes `PushSecret` type/metadata semantics and omitted-kind v2 store refs before expanding Kubernetes-provider v2 e2e coverage.

**Architecture:** Keep the existing controller -> clientmanager -> gRPC -> adapter -> v1 Kubernetes provider shape. Fix the reverse path by extending the `PushSecret` transport to carry the secret fields the v1 Kubernetes provider already uses, then normalize resolved v2 store kinds so omitted `PushSecretStoreRef.kind` survives push and delete flows.

**Tech Stack:** Go 1.26.x, protobuf/protoc generation via `make proto`, controller-runtime fake client tests, gRPC client tests, Ginkgo e2e on kind.

---

## File Map

- Modify: `providers/v2/common/proto/provider/secretstore.proto`
  Add the `PushSecret` transport fields needed for Kubernetes writes.
- Modify: `providers/v2/common/proto/provider/secretstore.pb.go`
  Regenerated protobuf bindings.
- Modify: `providers/v2/common/proto/provider/secretstore_grpc.pb.go`
  Regenerated gRPC bindings.
- Modify: `providers/v2/common/types.go`
  Update the internal `v2.Provider` `PushSecret` contract to accept the full source secret shape.
- Modify: `providers/v2/common/grpc/client.go`
  Build the new protobuf request fields from the source secret.
- Modify: `providers/v2/common/grpc/client_test.go`
  Assert the gRPC request contains the added fields.
- Modify: `providers/v2/adapter/store/client.go`
  Forward source secret type, labels, annotations, and data into the v2 provider client.
- Modify: `providers/v2/adapter/store/client_test.go`
  Lock down the reverse-adapter client mapping.
- Modify: `providers/v2/adapter/store/server.go`
  Rebuild the synthetic source secret from the expanded request instead of forcing `Opaque`.
- Modify: `providers/v2/adapter/store/server_test.go`
  Lock down the reverse-adapter server mapping.
- Modify: `pkg/controllers/pushsecret/pushsecret_controller.go`
  Preserve resolved store kind when refs omit `kind`.
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2.go`
  Keep v2 push/delete dispatch aligned with normalized kinds.
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2_test.go`
  Prove omitted-kind refs still resolve to `Provider` and `ClusterProvider`.
- Modify: `e2e/suites/provider/cases/kubernetes/push_v2.go`
  Add focused Kubernetes v2 push specs for metadata/type preservation and omitted-kind refs.

### Task 1: Write Failing Transport Tests For Kubernetes Push Semantics

**Files:**
- Modify: `providers/v2/adapter/store/client_test.go`
- Modify: `providers/v2/adapter/store/server_test.go`
- Modify: `providers/v2/common/grpc/client_test.go`

- [ ] **Step 1: Add a failing reverse-adapter client test for `PushSecret`**

```go
secret := &corev1.Secret{
    Type: corev1.SecretTypeDockerConfigJson,
    ObjectMeta: metav1.ObjectMeta{
        Labels: map[string]string{"team": "platform"},
        Annotations: map[string]string{"owner": "eso"},
    },
    Data: map[string][]byte{".dockerconfigjson": []byte("payload")},
}
```

Assert that the fake v2 provider receives:
- the original secret type
- the original labels
- the original annotations
- the existing `PushSecretData` metadata bytes

- [ ] **Step 2: Add a failing reverse-adapter server test for `PushSecret`**

```go
req := &pb.PushSecretRequest{
    SecretType: "kubernetes.io/dockerconfigjson",
    SecretLabels: map[string]string{"team": "platform"},
    SecretAnnotations: map[string]string{"owner": "eso"},
}
```

Assert that the fake v1 secrets client receives a `*corev1.Secret` with the same type, labels, annotations, and data.

- [ ] **Step 3: Add a failing gRPC client test for the expanded request**

Record the incoming `pb.PushSecretRequest` in the fake gRPC server and assert it includes:
- `secret_data`
- `secret_type`
- `secret_labels`
- `secret_annotations`
- existing `provider_ref`
- existing `source_namespace`

- [ ] **Step 4: Run targeted transport tests and confirm they fail for the expected reason**

Run: `env GOTOOLCHAIN=go1.26.2 go test ./providers/v2/adapter/store ./providers/v2/common/grpc -count=1`

Expected: FAIL because the current `PushSecret` contract only transports `secret_data`.

- [ ] **Step 5: Commit the failing test slice**

```bash
git add providers/v2/adapter/store/client_test.go \
  providers/v2/adapter/store/server_test.go \
  providers/v2/common/grpc/client_test.go
git commit -m "test: lock down kubernetes v2 push transport semantics"
```

### Task 2: Implement The Expanded `PushSecret` Transport

**Files:**
- Modify: `providers/v2/common/proto/provider/secretstore.proto`
- Modify: `providers/v2/common/proto/provider/secretstore.pb.go`
- Modify: `providers/v2/common/proto/provider/secretstore_grpc.pb.go`
- Modify: `providers/v2/common/types.go`
- Modify: `providers/v2/common/grpc/client.go`
- Modify: `providers/v2/adapter/store/client.go`
- Modify: `providers/v2/adapter/store/server.go`
- Modify: `providers/v2/adapter/store/client_test.go`
- Modify: `providers/v2/adapter/store/server_test.go`
- Modify: `providers/v2/common/grpc/client_test.go`

- [ ] **Step 1: Extend the protobuf contract**

Add these fields to `PushSecretRequest`:

```proto
string secret_type = 5;
map<string, string> secret_labels = 6;
map<string, string> secret_annotations = 7;
```

- [ ] **Step 2: Regenerate protobuf bindings**

Run: `make proto`

Expected: updated `secretstore.pb.go` and `secretstore_grpc.pb.go`

- [ ] **Step 3: Update the internal provider contract**

Change `providers/v2/common/types.go` so `PushSecret` accepts the full source secret object instead of only `secretData`.

```go
PushSecret(ctx context.Context, secret *corev1.Secret, pushSecretData *pb.PushSecretData, providerRef *pb.ProviderReference, sourceNamespace string) error
```

- [ ] **Step 4: Update the reverse-adapter client**

Forward the original `*corev1.Secret` from `providers/v2/adapter/store/client.go` to the v2 provider client without stripping type/metadata first.

- [ ] **Step 5: Update the gRPC client**

Build `pb.PushSecretRequest` from the source secret:
- `SecretData: secret.Data`
- `SecretType: string(secret.Type)`
- `SecretLabels: secret.Labels`
- `SecretAnnotations: secret.Annotations`

- [ ] **Step 6: Update the reverse-adapter server**

Reconstruct the synthetic source secret using the transported fields:

```go
secret := &corev1.Secret{
    Data: req.SecretData,
    Type: corev1.SecretType(req.SecretType),
    ObjectMeta: metav1.ObjectMeta{
        Labels: req.SecretLabels,
        Annotations: req.SecretAnnotations,
    },
}
```

- [ ] **Step 7: Re-run the targeted transport tests**

Run: `env GOTOOLCHAIN=go1.26.2 go test ./providers/v2/adapter/store ./providers/v2/common/grpc -count=1`

Expected: PASS

- [ ] **Step 8: Commit the transport implementation**

```bash
git add providers/v2/common/proto/provider/secretstore.proto \
  providers/v2/common/proto/provider/secretstore.pb.go \
  providers/v2/common/proto/provider/secretstore_grpc.pb.go \
  providers/v2/common/types.go \
  providers/v2/common/grpc/client.go \
  providers/v2/common/grpc/client_test.go \
  providers/v2/adapter/store/client.go \
  providers/v2/adapter/store/client_test.go \
  providers/v2/adapter/store/server.go \
  providers/v2/adapter/store/server_test.go
git commit -m "fix: preserve kubernetes v2 push secret metadata"
```

### Task 3: Write Failing Omitted-Kind Controller Tests And Fix Kind Normalization

**Files:**
- Modify: `pkg/controllers/pushsecret/pushsecret_controller.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2.go`
- Modify: `pkg/controllers/pushsecret/pushsecret_controller_v2_test.go`

- [ ] **Step 1: Add a failing `resolvedStoreInfo` test for omitted-kind v2 refs**

Use `esv1.Provider` and `esv1.ClusterProvider` objects with:

```go
esapi.PushSecretStoreRef{
    Name: "provider",
    Kind: "",
}
```

Assert the resolved kind becomes `Provider` or `ClusterProvider` based on the concrete object type, not `SecretStore`.

- [ ] **Step 2: Add failing push/delete path tests for omitted-kind refs**

Exercise:
- `PushSecretToProvidersV2()`
- `DeleteSecretFromProvidersV2()`

Assert the manager lookup and synced map keys use:
- `Provider/<name>` for namespaced providers
- `ClusterProvider/<name>` for cluster providers

- [ ] **Step 3: Run the controller tests and confirm they fail**

Run: `env GOTOOLCHAIN=go1.26.2 go test ./pkg/controllers/pushsecret -count=1`

Expected: FAIL because omitted-kind refs still normalize to `SecretStore`.

- [ ] **Step 4: Implement kind normalization**

Introduce the smallest helper needed so `resolvedStoreInfo()` preserves the resolved kind when:
- `store` is `*esv1.Provider`
- `store` is `*esv1.ClusterProvider`
- existing v1 store types keep their current defaults

Keep the logic centralized so `validateDataToMatchesResolvedStores()`, `PushSecretToProvidersV2()`, and `DeleteSecretFromProvidersV2()` all consume the same normalized kind.

- [ ] **Step 5: Re-run the controller tests**

Run: `env GOTOOLCHAIN=go1.26.2 go test ./pkg/controllers/pushsecret -count=1`

Expected: PASS

- [ ] **Step 6: Commit the controller fix**

```bash
git add pkg/controllers/pushsecret/pushsecret_controller.go \
  pkg/controllers/pushsecret/pushsecret_controller_v2.go \
  pkg/controllers/pushsecret/pushsecret_controller_v2_test.go
git commit -m "fix: preserve v2 push store kinds when kind is omitted"
```

### Task 4: Add Focused Kubernetes V2 Push E2E Coverage

**Files:**
- Modify: `e2e/suites/provider/cases/kubernetes/push_v2.go`

- [ ] **Step 1: Add a namespaced-provider spec for secret type and metadata preservation**

Create a source secret with:
- non-default type such as `kubernetes.io/dockerconfigjson`
- labels
- annotations

Push it through the v2 provider path and assert the remote Kubernetes secret preserves the same type and metadata alongside the pushed data.

- [ ] **Step 2: Add an omitted-kind namespaced-provider spec**

Create a `PushSecret` whose `secretStoreRefs[0]` keeps `Name` but leaves:
- `Kind: ""`
- `APIVersion: ""`

Assert the push still succeeds through the v2 namespaced provider flow.

- [ ] **Step 3: Add an omitted-kind cluster-provider spec if the existing fixture shape stays simple**

Reuse the existing Kubernetes cluster-provider helpers. If adding the cluster-provider variant would require substantial fixture duplication, stop after the namespaced-provider e2e and rely on the controller unit tests for the cluster-provider branch in this PR.

- [ ] **Step 4: Run the focused Kubernetes provider package tests**

Run: `cd e2e && env GOTOOLCHAIN=go1.26.2 go test ./suites/provider/cases/kubernetes -count=1`

Expected: PASS

- [ ] **Step 5: Run the focused v2 e2e make target**

Run: `make -C e2e test.v2 V2_GINKGO_LABELS='v2 && kubernetes && push-secret'`

Expected: PASS for the Kubernetes v2 push-secret slice on kind.

- [ ] **Step 6: Commit the e2e coverage**

```bash
git add e2e/suites/provider/cases/kubernetes/push_v2.go
git commit -m "test: add kubernetes v2 push regression coverage"
```

## Final Verification

- [ ] Run: `env GOTOOLCHAIN=go1.26.2 go test ./providers/v2/adapter/store ./providers/v2/common/grpc ./pkg/controllers/pushsecret -count=1`
- [ ] Run: `cd e2e && env GOTOOLCHAIN=go1.26.2 go test ./suites/provider/cases/kubernetes -count=1`
- [ ] Run: `make -C e2e test.v2 V2_GINKGO_LABELS='v2 && kubernetes && push-secret'`
- [ ] Run: `git status --short`

## Exit Criteria

- Kubernetes v2 `PushSecret` preserves source secret type, labels, and annotations across the reverse adapter.
- Omitted-kind refs resolve to `Provider` / `ClusterProvider` consistently on push and delete flows.
- Transport and controller fixes are covered by targeted unit tests.
- Kubernetes-only v2 push e2e proves the new semantics on kind before broader provider rollout.
