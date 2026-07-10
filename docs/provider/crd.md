## Kubernetes CRD Provider

External Secrets Operator can read data from arbitrary Kubernetes Custom Resources (CRDs) in the local cluster or in a remote cluster. It uses the same connection model as the [Kubernetes provider](kubernetes.md); remote-cluster access is covered in the [Remote cluster connection](#remote-cluster-connection) section below.

This provider is useful when secrets or secret-like values are already managed inside CRDs (i.e. by an operator) and you want to project them into regular Kubernetes Secrets.

The CRD provider is read-only.

### How it works

1. The provider connects to a Kubernetes API using the same connection model as the Kubernetes provider. To read the local cluster, set `auth.serviceAccount` and omit `server` (the URL defaults to the in-cluster API). To read a remote cluster, set `server` plus `auth` (`serviceAccount`, `token`, or `cert`) or `authRef` (a kubeconfig Secret).
2. It resolves the configured resource (`group`/`version`/`kind`) via Kubernetes discovery.
3. It reads objects using the dynamic Kubernetes client.
4. It applies optional whitelist rules before returning values.

### Remote cluster connection

Set `server` plus `auth` (or `authRef`) to connect to a remote Kubernetes API directly, exactly like the Kubernetes provider. The identity in `auth`/`authRef` is the identity the provider reads CRDs as; there is no separate impersonation step.

Example (`ClusterSecretStore`) connecting to a remote cluster:

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: crd-remote
spec:
  provider:
    crd:
      server:
        url: https://remote-api.example.com
        caProvider:
          type: ConfigMap
          name: remote-cluster-ca
          namespace: external-secrets
          key: ca.crt
      auth:
        serviceAccount:
          name: eso-remote-connector
          namespace: external-secrets
      resource:
        group: example.io
        version: v1alpha1
        kind: Widget
```

Notes:

- `auth` accepts `serviceAccount`, `token`, or `cert`, the same options as the Kubernetes provider.
- `authRef` can be used instead of `auth` when the connection details (a kubeconfig) come from a Secret. In that case `server` is optional because the kubeconfig already carries the API address.

### SecretStore / ClusterSecretStore configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: crd-store
spec:
  provider:
    crd:
      auth:
        serviceAccount:
          name: crd-reader
      resource:
        group: example.io
        version: v1alpha1
        kind: Widget
      # optional. If not empty, at least one rule must match the request
      whitelist:
        rules:
          # Name-only rule: allows objects whose name matches this regexp
          - name: "^app-.*$"

          # Name + properties rule: both must match
          - name: "^team-.*$"
            properties:
              - "^spec\\.password$"
              - "^spec\\.username$"

          # ClusterSecretStore-only rule: allows objects from matching namespaces
          - namespace: "^(prod|staging)$"

          # Properties-only rule: allows a request when the requested property matches
          - properties:
              - "^spec\\.password$"
```

#### Resource fields

- `auth.serviceAccount`: ServiceAccount used for API access to the local cluster (omit `server`).
- `server` + `auth`/`authRef`: Kubernetes API connection and authentication for a remote cluster. Omit `server` to read the local cluster.
- Setting `server.url` requires `auth` or `authRef`; a store that sets `server.url` without credentials is rejected at admission.
- `resource.group`: API group of the resource (empty for core API resources).
- `resource.version`: API version of the resource.
- `resource.kind`: Kind of the resource.

### Whitelist rules

`whitelist.rules` is an array of allow rules.

If `whitelist.rules` is empty or omitted, requests are allowed.
If `whitelist.rules` is not empty, at least one rule must match.

Each rule can define:

- `name` (regexp, optional): matched against requested object name.
- `namespace` (regexp, optional): matched against requested object namespace for `ClusterSecretStore`.
- `properties` (array of regexps, optional): matched against requested property keys.

Rule behavior:

1. `name` only: request is allowed when object name matches.
2. `namespace` only: request is allowed when object namespace matches (`ClusterSecretStore` only).
3. `properties` only: request is allowed when requested property keys match.
4. any combination of `name`, `namespace`, and `properties`: all configured fields in the same rule must match.

A rule with none of `name`, `namespace`, or `properties` is invalid.

#### Notes about `properties` matching

- For `data[].remoteRef.property`, the requested key is that property path (for example `spec.password`).
- For requests without a specific property (for example `GetAllSecrets`), only rules that do not require `properties` can match.
- `namespace` matching is only enforced for `ClusterSecretStore`; `SecretStore` ignores this field.

### Property Expressions (GJSON)

`remoteRef.property` is evaluated using [GJSON path syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md), the same syntax used by the Kubernetes provider and the rest of ESO, so the property dialect is consistent across providers.

Examples:

- `spec.user`
- `spec.password`
- `status.0.key`
- `status.#(key=="rotationDate").val`

If the expression resolves to no value, the provider returns `property not found`.

### Return values

The value returned for a reference depends on how it is requested:

- `remoteRef.property` set: the value at that GJSON path. A string leaf is returned unquoted (`spec.password` -> `hunter2`); objects, arrays, numbers, and booleans are returned as their raw JSON (`spec.replicas` -> `3`).
- `remoteRef.property` omitted: the entire object is returned as JSON.
- Map extraction (`dataFrom`, or a `remoteRef` consumed as a map): the object's top-level keys become secret keys. String values are unwrapped; non-string values keep their raw JSON.
- `dataFrom.find`: each matching object is returned as JSON, keyed by object name (`SecretStore`) or `namespace/objectName` (`ClusterSecretStore`). Use `conversionStrategy` on the `find` block to sanitize keys that are not valid Secret data keys.

A `property` that resolves to no value returns `property not found`; a missing object is reported as "secret not found".

### ExternalSecret examples

#### Fetch a single property

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: crd-store
    kind: SecretStore
  target:
    name: app-credentials
  data:
    - secretKey: password
      remoteRef:
        key: app-backend
        property: spec.password
    - secretKey: username
      remoteRef:
        key: app-backend
        property: spec.username
```

#### ClusterSecretStore with namespace whitelist

Use a namespace-qualified key (`namespace/objectName`) for namespaced resources when referencing a `ClusterSecretStore`.

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: crd-cluster-store
spec:
  provider:
    crd:
      auth:
        serviceAccount:
          name: crd-reader
          namespace: external-secrets
      resource:
        group: example.io
        version: v1alpha1
        kind: Widget
      whitelist:
        rules:
          - namespace: "^(team-a|team-b)$"
            name: "^app-.*$"
            properties:
              - "^spec\\.password$"
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: team-a-widget
  namespace: app
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: crd-cluster-store
    kind: ClusterSecretStore
  target:
    name: team-a-widget
  data:
    - secretKey: password
      remoteRef:
        key: team-a/app-backend
        property: spec.password
```

#### Fetch all matching CRD objects

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-widgets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: crd-store
    kind: SecretStore
  target:
    name: app-widgets
  dataFrom:
    - find:
        name:
          regexp: "^app-.*$"
```

`find.name.regexp` filters the listed objects by name.
Whitelist rules are applied in addition to this filter.

### RBAC

The configured ServiceAccount must be allowed to read the target resource.
At minimum, grant `get` on the selected resource. The `list` verb is only required when an `ExternalSecret` uses `dataFrom.find` (which calls `GetAllSecrets()` internally) — store bootstrap only checks `get`.

The right scope depends on which kind of store you are using:

- **`SecretStore` (namespaced)** — a namespace-scoped `Role` + `RoleBinding` is enough; the controller only reads from the store's own namespace.
- **`ClusterSecretStore`**: the controller may read across namespaces. `dataFrom.find` lists the target resource across all namespaces, and `remoteRef.key` uses the `namespace/objectName` form, so a `ClusterRole` + `ClusterRoleBinding` is required.

#### SecretStore (namespace-scoped) example

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: crd-reader
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: crd-reader
  namespace: default
rules:
  - apiGroups: ["example.io"]
    resources: ["widgets"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: crd-reader
  namespace: default
subjects:
  - kind: ServiceAccount
    name: crd-reader
    namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: crd-reader
```

#### ClusterSecretStore example

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: crd-reader
  namespace: external-secrets
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: crd-reader
rules:
  - apiGroups: ["example.io"]
    resources: ["widgets"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: crd-reader
subjects:
  - kind: ServiceAccount
    name: crd-reader
    namespace: external-secrets
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: crd-reader
```

### ClusterSecretStore note

When using `ClusterSecretStore`:

- for namespaced resources, `remoteRef.key` must be `namespace/objectName`.
- `whitelist.rules[].namespace` can be used to constrain which namespaces are readable.
- with `auth.serviceAccount`, the `namespace` field is optional. When omitted, the ServiceAccount is resolved in the consuming `ExternalSecret`'s own namespace (referent authentication), so a single store can serve many namespaces, each authenticating as its local ServiceAccount. Set `auth.serviceAccount.namespace` to pin one fixed namespace instead.

#### Referent authentication and RBAC

With referent authentication (no `auth.serviceAccount.namespace`), the named ServiceAccount must exist in **every** namespace that consumes the store, and each must be granted `get` (plus `list` for `dataFrom.find`) on the target resource. Grant this with a `ClusterRole` plus a `RoleBinding` per consuming namespace, or a `ClusterRoleBinding` if the same access is acceptable cluster-wide. When you instead pin `auth.serviceAccount.namespace`, only that one ServiceAccount needs the binding.
