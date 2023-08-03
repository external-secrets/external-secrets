The `VaultDynamicSecret` Generator provides an interface to HashiCorp Vault's
[Secrets engines](https://developer.hashicorp.com/vault/docs/secrets). Specifically,
it enables obtaining dynamic secrets not covered by the
[HashiCorp Vault provider](../../provider/hashicorp-vault.md).

Any Vault authentication method supported by the provider can be used here
(`provider` block of the spec).

All secrets engines should be supported by providing matching `path`, `method`
and `parameters` values to the Generator spec (see example below).

Exact output keys and values depend on the Vault secret engine used; nested values
are stored into the resulting Secret in JSON format.

## Example manifest

```yaml
{% include 'generator-vault.yaml' %}
```

Example `ExternalSecret` that references the Vault generator:
```yaml
{% include 'generator-vault-example.yaml' %}
```
