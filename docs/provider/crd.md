## Kubernetes CRD Provider

External Secrets Operator can read data from arbitrary Kubernetes Custom Resources (CRDs) in the same cluster.

This provider is useful when secrets or secret-like values are already managed inside CRDs (i.e. by an operator) and you want to project them into regular Kubernetes Secrets.

The CRD provider is read-only.

### How it works

1. The provider authenticates as the configured `serviceAccountName`.
2. It resolves the configured resource (`group`/`version`/`kind`) via Kubernetes discovery.
3. It reads objects using the dynamic Kubernetes client.
4. It applies optional whitelist rules before returning values.

## SecretStore / ClusterSecretStore configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: crd-store
spec:
  provider:
    crd:
      serviceAccountName: crd-reader
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

          # Properties: allow only these properties to be read
          - properties:
              - "^spec\\.password$"
```

### Resource fields

- `serviceAccountName`: ServiceAccount used for API access.
- `resource.group`: API group of the resource (empty for core API resources).
- `resource.version`: API version of the resource.
- `resource.kind`: Kind of the resource.

## Whitelist rules

`whitelist.rules` is an array of allow rules.

If `whitelist.rules` is empty or omitted, requests are allowed.
If `whitelist.rules` is not empty, at least one rule must match.

Each rule can define:

- `name` (regexp, optional): matched against requested object name.
- `properties` (array of regexps, optional): matched against requested property keys.

Rule behavior:

1. `name` only: request is allowed when object name matches.
2. `properties` only: request is allowed when requested property keys match.
3. both `name` and `properties`: `name` must match and `properties` must match.

A rule with neither `name` nor `properties` is invalid.

### Notes about `properties` matching

- For `data[].remoteRef.property`, the requested key is that property path (for example `spec.password`).
- For requests without a specific property (for example `GetAllSecrets`), only rules that do not require `properties` can match.

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

When using `ClusterSecretStore`, the provider still needs a ServiceAccount namespace context.
If the referencing ExternalSecret namespace is not available, the provider defaults to `default` for ServiceAccount token resolution.
