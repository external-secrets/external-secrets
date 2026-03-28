# CRD provider – documentation & test fixtures

This directory contains **static YAML manifests** for manual cluster inspection
and an **automated E2E test suite** (`e2e_test.go`) that exercises the CRD
provider against a real Kubernetes cluster.

---

## Overview

The CRD provider reads data from any Kubernetes custom resource (or even core
resources such as `ConfigMap`) and exposes field values as External Secret
entries. It is configured via the `spec.provider.crd` block of a `SecretStore`
or `ClusterSecretStore`.

```yaml
spec:
  provider:
    crd:
      serviceAccountRef:      # identity used to read CRDs
        name: crd-reader
        namespace: default    # required for ClusterSecretStore; ignored for SecretStore
      resource:
        group: example.io
        version: v1alpha1
        kind: MyResource
      remoteNamespace: default  # optional default namespace for namespaced Get/List
      whitelist:                # optional access control rules
        rules:
          - name: "^allowed-.*$"
      # --- explicit / remote-cluster mode ---
      server:
        url: https://remote.k8s.example
        caBundle: LS0tLS...
      auth:
        serviceAccount:         # SA token used to authenticate to the remote cluster
          name: crd-bridge
          namespace: default
```

---

## Authentication

### Legacy mode (in-cluster, local API server)

Set `serviceAccountRef` only (no `server`/`auth`/`authRef`). The controller
mints a short-lived token for the referenced ServiceAccount and uses it against
the local Kubernetes API server.

| Store kind | `serviceAccountRef.namespace` |
|---|---|
| `SecretStore` | Ignored — the SA must live in the store's own namespace. |
| `ClusterSecretStore` | Required — tells the controller where the SA lives. Defaults to `"default"` when omitted. |

```yaml
# SecretStore — namespace field is optional (always uses store namespace)
serviceAccountRef:
  name: crd-reader

# ClusterSecretStore — namespace is required
serviceAccountRef:
  name: crd-reader
  namespace: ops
```

### Explicit mode (remote cluster)

Set `server` together with `auth` (or `authRef`). The same `server`/`auth`
fields as the Kubernetes provider are supported.

`serviceAccountRef` is **optional** in this mode. When present the controller
connects to the remote cluster using the credentials from `auth`, then sends a
`Impersonate-User: system:serviceaccount/<ns>/<name>` header for every API
request — effectively assuming the identity of the referenced ServiceAccount on
the remote cluster. This lets a single privileged bridge SA authenticate while
scoped reader SAs enforce the actual RBAC.

| Store kind | Impersonation namespace |
|---|---|
| `SecretStore` | Always the store's own namespace (the `namespace` field on `serviceAccountRef` is ignored). |
| `ClusterSecretStore` | Must be set explicitly in `serviceAccountRef.namespace`. |

```yaml
crd:
  server:
    url: https://remote.k8s.example
    caBundle: LS0tLS...
  auth:
    serviceAccount:          # authenticates to the remote cluster
      name: crd-bridge
      namespace: default
  serviceAccountRef:         # impersonated on the remote cluster (optional)
    name: crd-scoped-reader
    namespace: team-a
  resource:
    group: example.io
    version: v1alpha1
    kind: Widget
```

---

## Remote reference key rules

The meaning of `remoteRef.key` differs by store kind and resource scope:

| Store kind | Resource scope | `remoteRef.key` format | Notes |
|---|---|---|---|
| `SecretStore` | namespaced | `objectName` | `/` is rejected. Namespace = store namespace (or `remoteNamespace` when set). |
| `SecretStore` | cluster-scoped | `objectName` | `/` is rejected. |
| `ClusterSecretStore` | namespaced | `namespace/objectName` | Namespace segment required; bare names are rejected. |
| `ClusterSecretStore` | cluster-scoped | `objectName` | `/` is rejected (no namespace concept). |

For `dataFrom` Find (listing), `ClusterSecretStore` with a namespaced kind
lists all namespaces unless `remoteNamespace` is set, and result map keys are
`namespace/objectName`. When `remoteNamespace` is set, the listing is limited
to that namespace and keys are bare object names.

---

## Whitelist

The optional `whitelist` block restricts which objects and properties the
provider is allowed to read. If any rules are configured, a request must match
at least one rule; requests that match no rule are denied.

Each rule consists of three independent filters — a request is allowed when
**all non-empty filters in that rule** match:

| Field | Applies to | Description |
|---|---|---|
| `name` | object name (bare, no namespace prefix) | Regular expression matched against the bare object name. |
| `namespace` | object namespace | Regular expression matched against the object namespace. **Only applied for `ClusterSecretStore`; ignored for `SecretStore`.** |
| `properties` | requested property paths | List of regular expressions; every requested property must match at least one pattern. |

A rule must define at least one of `name`, `namespace`, or `properties`.

### Examples

```yaml
whitelist:
  rules:
    # Allow only objects whose name starts with "prod-".
    - name: "^prod-.*$"

    # Allow objects in the "prod" or "staging" namespaces (ClusterSecretStore only).
    - namespace: "^(prod|staging)$"

    # Combine: objects named "db-*" in any "app-*" namespace.
    - name: "^db-.*$"
      namespace: "^app-.*$"

    # Allow any object, but only the spec.password property.
    - properties:
        - "^spec\\.password$"

    # Allow "db-*" objects, but only spec.user and spec.password.
    - name: "^db-.*$"
      properties:
        - "^spec\\.user$"
        - "^spec\\.password$"
```

All `name`, `namespace`, and `properties` values are compiled as Go regular
expressions. Invalid patterns are rejected at `ValidateStore` time.

---

## Static YAML manifests

Located in `static-manifests/`. Apply them to a cluster for manual inspection:

| File | Purpose |
|---|---|
| `crd.yaml` | `DBSpec` (namespaced) and `ClusterDBSpec` (cluster-scoped) CRD definitions |
| `sa.yaml` | `ServiceAccount`, `Role`/`RoleBinding`, `ClusterRole`/`ClusterRoleBinding` for `crd-reader` |
| `test-crd.yaml` | `SecretStore` + `ExternalSecret` for the namespaced `DBSpec` kind |
| `test-crd-cluster.yaml` | `ClusterSecretStore` + `ExternalSecret` for the cluster-scoped `ClusterDBSpec` kind |
| `test-crd-namespace.yaml` | `ClusterSecretStore` targeting `DBSpec` objects across all namespaces (cross-NS example) |
| `test-crd-provider.yaml` | `SecretStore` reading from a core `ConfigMap`; also shows the explicit/impersonation mode |

Apply in order:

```bash
kubectl apply -f static-manifests/crd.yaml
kubectl apply -f static-manifests/sa.yaml
kubectl apply -f static-manifests/test-crd.yaml
kubectl apply -f static-manifests/test-crd-cluster.yaml
kubectl apply -f static-manifests/test-crd-namespace.yaml
```

---

## Automated E2E suite

`e2e_test.go` is compiled only when the `e2e` build tag is present.
No ESO controller installation is required — only a reachable cluster.

### Prerequisites

* A running Kubernetes cluster (e.g. `kind create cluster`)
* `KUBECONFIG` pointing at it (or the default `~/.kube/config`)
* Cluster-admin permissions (the suite creates CRDs, ClusterRoles, etc.)

The suite creates and cleans up all required fixtures automatically:

| Fixture | Details |
|---|---|
| Namespaces | `crd-e2e-test`, `crd-e2e-test-2` |
| CRDs | `dbspecs.test.external-secrets.io` (namespaced), `clusterdbspecs.test.external-secrets.io` (cluster-scoped) |
| ServiceAccounts | `crd-e2e-reader` (has CRD access), `crd-e2e-noaccess` (no CRD access) |
| ClusterRole/Binding | `crd-e2e-dbspec-reader` — get/list on both CRD kinds |
| ClusterRole/Binding | `crd-e2e-impersonator` — allows `crd-e2e-reader` to impersonate other SAs |
| Test objects | `DBSpec` in each namespace, `ClusterDBSpec` cluster-wide |

### Running

`providers/v1/crd` has its own `go.mod`. The repository root also has a
`go.work` that is always active while you are anywhere inside the workspace
tree. Because Go resolves relative `./...` patterns against the **workspace's
main module** (the root), not the local `go.mod`, you must disable the
workspace with `GOWORK=off` so the local module is used directly.

```bash
# From inside providers/v1/crd:
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v

# Target specific scenarios:
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v -run TestSecretStore
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v -run TestClusterSecretStore
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v -run TestLegacyMode
GOWORK=off KUBECONFIG=~/.kube/config go test -tags e2e ./test/... -v -run TestImpersonation
```

> **Why `GOWORK=off`?** Without it `go test ./test/...` resolves `./` against
> the root module (`github.com/external-secrets/external-secrets`), which does
> not own this package. `GOWORK=off` forces Go to use the local `go.mod`.

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
| `TestLegacyMode_NoAccessSA_NewClientFails` | `NewClient` fails when the SA has no CRD access (legacy mode) |
| `TestImpersonation_NoAccessSA_NewClientFails` | `NewClient` fails when impersonating an SA without CRD access (explicit mode) |
