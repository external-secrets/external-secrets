# CRD provider – test fixtures

This directory contains both **static YAML manifests** for manual cluster
inspection and an **automated E2E test suite** (`e2e_test.go`) that exercises
the CRD provider against a real Kubernetes cluster.

---

## Static YAML manifests

| File | Purpose |
|---|---|
| `crd.yaml` | `DBSpec` (namespaced) and `ClusterDBSpec` (cluster-scoped) CRD definitions |
| `sa.yaml` | `ServiceAccount`, `Role`/`RoleBinding`, `ClusterRole`/`ClusterRoleBinding` |
| `test-crd.yaml` | `SecretStore` + `ExternalSecret` for the namespaced `DBSpec` kind |
| `test-crd-cluster.yaml` | `ClusterSecretStore` + `ExternalSecret` for the `ClusterDBSpec` kind |
| `test-crd-namespace.yaml` | `ClusterSecretStore` targeting `DBSpec` objects across all namespaces |
| `test-crd-provider.yaml` | `SecretStore` reading from a core `ConfigMap` |

Apply them in order:
```bash
kubectl apply -f crd.yaml
kubectl apply -f sa.yaml
kubectl apply -f test-crd.yaml        # SecretStore example
kubectl apply -f test-crd-cluster.yaml # ClusterSecretStore (cluster-scoped CRD)
kubectl apply -f test-crd-namespace.yaml # ClusterSecretStore (namespaced CRD, cross-NS)
```

---

## Automated E2E suite

`e2e_test.go` is compiled only when the `e2e` build tag is present.
No ESO controller installation is required — only a reachable cluster.

### Prerequisites

* A running Kubernetes cluster (e.g. `kind create cluster`)
* `KUBECONFIG` pointing at it (or the default `~/.kube/config`)
* Cluster-admin permissions (the test creates CRDs, ClusterRoles, etc.)

The suite creates and cleans up all required fixtures automatically:

| Fixture | Details |
|---|---|
| Namespaces | `crd-e2e-test`, `crd-e2e-test-2` |
| CRDs | `dbspecs.test.external-secrets.io` (namespaced), `clusterdbspecs.test.external-secrets.io` (cluster-scoped) |
| ServiceAccount | `crd-e2e-reader` in `crd-e2e-test` |
| ClusterRole/Binding | `crd-e2e-dbspec-reader` – get/list on both CRD kinds |
| Test objects | `DBSpec` in each namespace, `ClusterDBSpec` cluster-wide |

### Running

`providers/v1/crd` has its own `go.mod`. The repository root also has a
`go.work` that is always active while you are anywhere inside the workspace
tree. Because Go resolves relative `./...` patterns against the **workspace's
main module** (the root), not the local `go.mod`, you must disable the
workspace with `GOWORK=off` so the local module is used directly.

```bash
# From the repository root:
cd providers/v1/crd
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v

# Target specific scenarios:
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v -run TestSecretStore
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v -run TestClusterSecretStore
```

> **Why `GOWORK=off`?** The `go.work` at the repository root is active for
> every directory inside the workspace tree. Without `GOWORK=off`, `go test
> ./test/...` resolves the `./` pattern against the root module
> (`github.com/external-secrets/external-secrets`), which does not own this
> package. `GOWORK=off` forces Go to use the local `go.mod` instead.

### Test coverage

| Test | What it verifies |
|---|---|
| `TestSecretStore_GetSecret_ByProperty` | Fetch a single field from a namespaced CR |
| `TestSecretStore_GetSecret_WholeObject` | Omitting Property returns the full serialised object |
| `TestSecretStore_GetSecret_SlashInKeyRejected` | `'/'` in the key is rejected for SecretStore |
| `TestSecretStore_GetSecret_MissingObjectReturnsNoSecretError` | Missing CR maps to `NoSecretError` |
| `TestSecretStore_GetSecretMap` | Sub-object returned as a flat `map[string][]byte` |
| `TestSecretStore_GetAllSecrets` | Lists objects in the store namespace only |
| `TestSecretStore_GetAllSecrets_RegexpFilter` | Regex filter applied during listing |
| `TestClusterSecretStore_NamespacedKey_Success` | `namespace/name` key fetches from the correct namespace |
| `TestClusterSecretStore_NamespacedKey_SecondNamespace` | Key namespace segment correctly targets a second namespace |
| `TestClusterSecretStore_BareNameRejected` | Bare name rejected for namespaced kind with ClusterSecretStore |
| `TestClusterSecretStore_GetAllSecrets_AcrossNamespaces` | Lists all namespaces; keys are `namespace/name` |
| `TestClusterSecretStore_ClusterScopedKind_GetSecret` | Cluster-scoped CR fetched with bare name |
| `TestClusterSecretStore_ClusterScopedKind_SlashInKeyRejected` | `'/'` rejected for cluster-scoped kind |
| `TestClusterSecretStore_ClusterScopedKind_GetAllSecrets` | Listing cluster-scoped kind returns bare object names |

