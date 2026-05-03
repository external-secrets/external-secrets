The `VaultDynamicSecret` Generator provides an interface to HashiCorp Vault's
[Secrets engines](https://developer.hashicorp.com/vault/docs/secrets). Specifically,
it enables obtaining dynamic secrets not covered by the
[HashiCorp Vault provider](../../provider/hashicorp-vault.md).

Any Vault authentication method supported by the provider can be used here
(`provider` block of the spec).

All secrets engines should be supported by providing matching `path`, `method`
and `parameters` values to the Generator spec (see example below).

Exact output keys and values depend on the Vault secret engine used; nested values
are stored into the resulting Secret in JSON format. The generator exposes `data`
section of the response from Vault API by default. To adjust the behaviour, use
`resultType` key.

### Passing parameters

- `parameters` is a JSON body sent on write methods (POST, PUT, etc.) and
  supports arbitrary nested JSON. It is **ignored** on `GET` and `LIST`.
- `getParameters` is a `map[string][]string` sent as the query string on `GET`
  calls. Each key may map to multiple values, matching HTTP query-string
  semantics. It is ignored for non-GET methods.

## Example manifest

Write method (POST) with a JSON body:

```yaml
{% include 'generator-vault.yaml' %}
```

GET method with query-string parameters:

```yaml
{% include 'generator-vault-get.yaml' %}
```

Example `ExternalSecret` that references the Vault generator:
```yaml
{% include 'generator-vault-example.yaml' %}
```
