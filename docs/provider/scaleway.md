# Scaleway Secret Manager

External Secrets Operator integrates with [Scaleway's Secret Manager](https://developers.scaleway.com/en/products/secret_manager/api/v1alpha1/).

## Creating a SecretStore

You need an api key (access key + secret key) to authenticate with the secret manager.
Both access and secret keys can be specified either directly in the config, or by referencing
a kubernetes secret.

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secret-store
spec:
  provider:
    scaleway:
      region: <REGION>
      projectId: <PROJECT_UUID>
      accessKey:
        value: <ACCESS_KEY>
      secretKey:
        secretRef:
          name: <NAME_OF_KUBE_SECRET>
          key: <KEY_IN_KUBE_SECRET>
```

## Referencing Secrets

Secrets can be referenced by name, id or path, using the prefixes `"name:"`, `"id:"` and `"path:"` respectively.

A PushSecret resource can use name or path references.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
    name: secret
spec:
    refreshInterval: 1h0m0s
    secretStoreRef:
        kind: SecretStore
        name: secret-store
    data:
      - secretKey: <KEY_IN_KUBE_SECRET>
        remoteRef:
          key: id:<SECRET_UUID>
          version: latest_enabled
```

## JSON Secret Values

Scaleway Secret Manager supports storing JSON objects as secrets. You can access values using [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md):

Consider the following JSON object that is stored in a Scaleway secret:

```json
{
  "first": "Tom", 
  "last": "Anderson"
}
```

This is an example on how you would look up keys in the above JSON object:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: extract-data
spec:
  refreshInterval: 1h0m0s
  secretStoreRef:
    kind: SecretStore
    name: secret-store
  target:
    name: secret-data
    creationPolicy: Owner
  data:
  - secretKey: first_name
    remoteRef:
      key: id:<SECRET_UUID>
      property: first # Tom
  - secretKey: last_name
    remoteRef:
      key: id:<SECRET_UUID>
      property: last # Anderson
```

## PushSecret

The provider supports `PushSecret` with name (`name:<NAME>`) and path (`path:/<PATH>/<NAME>`) references.
Secret versions are immutable in Scaleway: every update creates a new version and disables the previous one.

With `property` set, the pushed value is merged into the remote secret's JSON object — only that
property is created or updated, other properties are preserved. Keys containing dots (e.g. `tls.crt`)
are stored as literal top-level keys, unless the remote JSON already contains a matching nested
structure (e.g. `{"tls":{"crt":...}}`), in which case the nested value is updated in place.
If the existing remote value is not a JSON object, the push fails: the provider refuses to
overwrite a value it did not write. Delete or migrate the remote secret first.
With `deletionPolicy: Delete`, deleting a pushed property removes only that key (as a new version);
the remote secret itself is deleted when the last property is removed.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-tls
spec:
  refreshInterval: 1h
  secretStoreRefs:
    - kind: SecretStore
      name: secret-store
  selector:
    secret:
      name: my-tls-secret
  data:
    - match:
        secretKey: tls.crt
        remoteRef:
          remoteKey: "path:/certificates/my-cert"
          property: tls.crt
    - match:
        secretKey: tls.key
        remoteRef:
          remoteKey: "path:/certificates/my-cert"
          property: tls.key
```

Without `secretKey` (or with `spec.dataTo`), the whole Kubernetes Secret is serialized as a JSON
object of its keys and pushed as the remote secret's value. `dataTo` filtering (`match.regexp`) is
applied by the controller before the push.

```yaml
  dataTo:
    - remoteKey: "path:/certificates/my-cert-bundle"
      storeRef:
        kind: SecretStore
        name: secret-store
```

Note: Scaleway limits a secret's payload to 64KiB; the JSON object holding all pushed
properties (or the whole serialized Secret) must stay under that limit. Values pushed with
`property` (or via a whole-secret push) must be valid UTF-8 — binary values cannot be stored
inside a JSON object and are rejected; push them as a single raw secret (a `data` entry
without `property`) instead.

