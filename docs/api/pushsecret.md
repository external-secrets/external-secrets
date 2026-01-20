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
    - match:
        regexp: "^db-.*"  # Push all keys starting with "db-"
      rewrite:
        - regexp:
            source: "^db-"
            target: "myapp/database/"  # db-host -> myapp/database/host
```

### Fields

#### `match` (optional)

Defines which keys to select from the source Secret.

- **`regexp`** (string, optional): Regular expression pattern to match keys. If omitted, all keys are matched.

**Examples:**
```yaml
# Match all keys
dataTo:
  - {}

# Match keys starting with "db-"
dataTo:
  - match:
      regexp: "^db-.*"

# Match keys ending with "-key"
dataTo:
  - match:
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
```yaml
rewrite:
  - transform:
      template: "secrets/{{ .value | upper }}"  # .value contains the key name
```

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
  - match:
      regexp: "^db-.*"
    metadata:
      labels:
        app: myapp
        env: production
```

#### `conversionStrategy` (string, optional)

Strategy for converting secret values. Default: `"None"`

- `"None"`: No conversion
- `"ReverseUnicode"`: Reverse Unicode escape sequences (useful when paired with ExternalSecret's `Unicode` strategy)

```yaml
dataTo:
  - conversionStrategy: ReverseUnicode
```

### Combining dataTo with data

You can use both `dataTo` and `data` fields. Explicit `data` entries override `dataTo` for the same source key:

```yaml
spec:
  dataTo:
    - {}  # Push all keys with original names
  data:
    - match:
        secretKey: db-host
        remoteRef:
          remoteKey: custom-db-host  # Override for db-host only
```

### Multiple dataTo Entries

You can specify multiple `dataTo` entries with different patterns:

```yaml
spec:
  dataTo:
    # Push db-* keys with database/ prefix
    - match:
        regexp: "^db-.*"
      rewrite:
        - regexp: {source: "^db-", target: "database/"}
    # Push api-* keys with api/ prefix
    - match:
        regexp: "^api-.*"
      rewrite:
        - regexp: {source: "^api-", target: "api/"}
```

### Error Handling

- **Invalid regexp**: PushSecret enters error state with details in status
- **Duplicate remote keys**: Operation fails if rewrites produce duplicate keys
- **No matching keys**: Warning logged, PushSecret remains Ready

See the [PushSecret dataTo guide](../guides/pushsecret-datato.md) for more examples and use cases.

## Template

When the controller reconciles the `PushSecret` it will use the `spec.template` as a blueprint to construct a new property.
You can use golang templates to define the blueprint and use template functions to transform the defined properties.
You can also pull in `ConfigMaps` that contain golang-template data using `templateFrom`.
See [advanced templating](../guides/templating.md) for details.
