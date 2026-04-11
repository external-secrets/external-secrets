![PushSecret](../pictures/diagrams-pushsecret-basic.png)

The `PushSecret` is namespaced and it describes what data should be pushed to the secret provider.

* tells the operator what secrets should be pushed by using `spec.selector`.
* you can specify what secret keys should be pushed by using `spec.data`.
* you can bulk-push secrets using pattern matching with `spec.dataTo`.
* you can also template the resulting property values using [templating](#templating).

## Example

Below is an example of the `PushSecret` in use.

``` yaml
{% include 'full-pushsecret.yaml' %}
```

The result of the created Secret object will look like:

```yaml
# The destination secret that will be templated and pushed by PushSecret.
apiVersion: v1
kind: Secret
metadata:
  name: destination-secret
stringData:
  best-pokemon-dst: "PIKACHU is the really best!"
```

## DataTo

The `spec.dataTo` field enables bulk pushing of secrets without explicit per-key configuration. This is useful when you need to push multiple related secrets and want to avoid verbose YAML.

### Basic Example

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-db-secrets
spec:
  secretStoreRefs:
    - name: aws-secret-store
  selector:
    secret:
      name: app-secrets
  dataTo:
    - storeRef:
        name: aws-secret-store
      match:
        regexp: "^db-.*"  # Push all keys starting with "db-"
      rewrite:
        - regexp:
            source: "^db-"
            target: "myapp/database/"  # db-host -> myapp/database/host
```

### Fields

#### `storeRef` (required)

Specifies which SecretStore to push to. Each `dataTo` entry must include a `storeRef` to target a specific store.

- **`name`** (string, optional): Name of the SecretStore to target.
- **`labelSelector`** (object, optional): Select stores by label. Use either `name` or `labelSelector`, not both.
- **`kind`** (string, optional): `SecretStore` or `ClusterSecretStore`. Defaults to `SecretStore`.

```yaml
dataTo:
  # Target a specific store by name
  - storeRef:
      name: aws-secret-store

  # Target stores by label
  - storeRef:
      labelSelector:
        matchLabels:
          env: production
```

!!! note "storeRef vs spec.data"
    Unlike `spec.data` entries which can omit store targeting, every `dataTo` entry requires a `storeRef`.
    This prevents accidental "push to all stores" behavior. The `storeRef` must reference a store
    listed in `spec.secretStoreRefs`.

#### `match` (optional)

Defines which keys to select from the source Secret.

- **`regexp`** (string, optional): Regular expression pattern to match keys. If omitted, all keys are matched.

**Examples:**
```yaml
# Match all keys
dataTo:
  - storeRef:
      name: my-store

# Match keys starting with "db-"
dataTo:
  - storeRef:
      name: my-store
    match:
      regexp: "^db-.*"

# Match keys ending with "-key"
dataTo:
  - storeRef:
      name: my-store
    match:
      regexp: ".*-key$"
```

#### `rewrite` (array, optional)

Array of rewrite operations to transform key names. Operations are applied sequentially.

Each rewrite can be either:

**Regexp Rewrite:**
```yaml
rewrite:
  - regexp:
      source: "^db-"      # Regex pattern to match
      target: "app/db/"   # Replacement string (supports capture groups like $1, $2)
```

**Transform Rewrite (Go Template):**
{% raw %}
```yaml
rewrite:
  - transform:
      template: "secrets/{{ .value | upper }}"  # .value contains the key name
```
{% endraw %}

**Chained Rewrites:**
```yaml
rewrite:
  - regexp: {source: "^db-", target: ""}     # Remove "db-" prefix
  - regexp: {source: "^", target: "prod/"}   # Add "prod/" prefix
```

#### `metadata` (object, optional)

Provider-specific metadata to attach to all pushed secrets. Structure depends on the provider.

```yaml
dataTo:
  - storeRef:
      name: my-store
    match:
      regexp: "^db-.*"
    metadata:
      labels:
        app: myapp
        env: production
```

#### `conversionStrategy` (string, optional)

Strategy for converting secret key names before matching and rewriting. The conversion is applied to keys (not values), and `match` patterns and `rewrite` operations operate on the converted key names. Default: `"None"`

- `"None"`: No conversion
- `"ReverseUnicode"`: Reverse Unicode escape sequences in key names (useful when paired with ExternalSecret's `Unicode` strategy)

```yaml
dataTo:
  - storeRef:
      name: my-store
    conversionStrategy: ReverseUnicode
```

### Combining dataTo with data

You can use both `dataTo` and `data` fields. Explicit `data` entries override `dataTo` for the same source key:

```yaml
spec:
  secretStoreRefs:
    - name: my-store
  dataTo:
    - storeRef:
        name: my-store  # Push all keys with original names
  data:
    - match:
        secretKey: db-host
        remoteRef:
          remoteKey: custom-db-host  # Override for db-host only
```

In this example, all keys are pushed via `dataTo`, but `db-host` uses the custom remote key from `data` instead.

### Multiple dataTo Entries

You can specify multiple `dataTo` entries with different patterns:

```yaml
spec:
  secretStoreRefs:
    - name: my-store
  dataTo:
    # Push db-* keys with database/ prefix
    - storeRef:
        name: my-store
      match:
        regexp: "^db-.*"
      rewrite:
        - regexp: {source: "^db-", target: "database/"}
    # Push api-* keys with api/ prefix
    - storeRef:
        name: my-store
      match:
        regexp: "^api-.*"
      rewrite:
        - regexp: {source: "^api-", target: "api/"}
```

### Error Handling

- **Invalid regular expression**: PushSecret enters error state with details in status
- **Duplicate remote keys**: Operation fails if rewrites produce duplicate keys
- **No matching keys**: Warning logged, PushSecret remains Ready

See the [PushSecret dataTo guide](../guides/pushsecret-datato.md) for more examples and use cases.

## Template

When the controller reconciles the `PushSecret` it will use the `spec.template` as a blueprint to construct a new property.
You can use golang templates to define the blueprint and use template functions to transform the defined properties.
You can also pull in `ConfigMaps` that contain golang-template data using `templateFrom`.
See [advanced templating](../guides/templating.md) for details.
