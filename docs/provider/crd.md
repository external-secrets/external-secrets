## Kubernetes CRD Provider

External Secrets Operator can read data from arbitrary Kubernetes Custom Resources (CRDs) within the cluster or from remote clusters when configured. Remote-cluster access is covered in the [Remote cluster connection](#remote-cluster-connection-and-service-account-impersonation) section below.

This provider is useful when secrets or secret-like values are already managed inside CRDs (i.e. by an operator) and you want to project them into regular Kubernetes Secrets.

The CRD provider is read-only.

### How it works

1. The provider authenticates as the configured `serviceAccountRef` (simple mode), or with `server` + `auth`/`authRef` (explicit mode).
2. It resolves the configured resource (`group`/`version`/`kind`) via Kubernetes discovery.
3. It reads objects using the dynamic Kubernetes client.
4. It applies optional whitelist rules before returning values.

### Remote cluster connection and service account impersonation

When `server` + `auth` (or `authRef`) is configured, the CRD provider uses **explicit mode** and connects to the target Kubernetes API directly.

In explicit mode there are two identities:

1. **Connection identity** (`auth`/`authRef`): used to establish the API connection.
2. **Optional impersonated identity** (`serviceAccountRef`): if set, the provider sends `Impersonate-User: system:serviceaccount:<namespace>:<name>` on requests.

This lets you connect with one identity and read CRDs as a specific service account role in the remote cluster.

Example (`ClusterSecretStore`) connecting to a remote cluster and impersonating a remote service account:

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
      # optional impersonation target on the remote cluster
      serviceAccountRef:
        name: tenant-reader
        namespace: tenant-a
      resource:
        group: example.io
        version: v1alpha1
        kind: Widget
```

Notes:

- `serviceAccountRef` is optional in explicit mode; set it only when you want impersonation.
- For `ClusterSecretStore`, `serviceAccountRef.namespace` is required when impersonation is enabled.
- For `SecretStore`, the impersonation namespace is the ExternalSecret/store namespace.
- `authRef` can be used instead of `auth` when your connection details come from a Secret.

## SecretStore / ClusterSecretStore configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: crd-store
spec:
  provider:
    crd:
      serviceAccountRef:
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

          # Properties: allow only these properties to be read
          - properties:
              - "^spec\\.password$"
```

### Resource fields

- `serviceAccountRef`: ServiceAccount used for API access.
- `server` + `auth`/`authRef` (optional): explicit Kubernetes API connection/authentication.
- `remoteNamespace` (optional): for `SecretStore`, overrides the store namespace as the default namespace for all namespaced Get and List operations. For `ClusterSecretStore`, limits `dataFrom` Find (list) to that namespace only (keys become bare object names instead of `namespace/objectName`); it does **not** replace the required `namespace/objectName` key format when calling `GetSecret` on a namespaced resource.
- `resource.group`: API group of the resource (empty for core API resources).
- `resource.version`: API version of the resource.
- `resource.kind`: Kind of the resource.

## Whitelist rules

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

### Notes about `properties` matching

- For `data[].remoteRef.property`, the requested key is that property path (for example `spec.password`).
- For requests without a specific property (for example `GetAllSecrets`), only rules that do not require `properties` can match.
- `namespace` matching is only enforced for `ClusterSecretStore`; `SecretStore` ignores this field.

## Property Expressions (JMESPath)

`remoteRef.property` is evaluated using [JMESPath](https://jmespath.org/).

Examples:

- `spec.user`
- `spec.password`
- `status[0].key`
- `status[?key=='rotationDate'].val | [0]`

If an expression is invalid, the provider returns an error.
If an expression is valid but resolves to no value, the provider returns `property not found`.

## ExternalSecret examples

### Fetch a single property

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

### ClusterSecretStore with namespace whitelist

Use a namespace-qualified key (`namespace/objectName`) for namespaced resources when referencing a `ClusterSecretStore`.

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: crd-cluster-store
spec:
  provider:
    crd:
      serviceAccountRef:
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

### Fetch all matching CRD objects

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

## RBAC

The configured ServiceAccount must be allowed to read the target resource.
At minimum, grant `get` and `list` on the selected resource (and namespace-scoped permissions where applicable).

Example:

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

## ClusterSecretStore note

When using `ClusterSecretStore`:

- for namespaced resources, `remoteRef.key` should be `namespace/objectName`.
- `whitelist.rules[].namespace` can be used to constrain which namespaces are readable.
- for service account token resolution, if no namespace can be derived the provider defaults to `default`.
