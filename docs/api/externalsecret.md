The `ExternalSecret` describes what data should be fetched, how the data should
be transformed and saved as a `Kind=Secret`:

* tells the operator what secrets should be synced by using `spec.data` to
  explicitly sync individual keys or use `spec.dataFrom` to get **all values**
  from the external API.
* you can specify how the secret should look like by specifying a
  `spec.target.template`

## Template

When the controller reconciles the `ExternalSecret` it will use the `spec.template` as a blueprint to construct a new `Kind=Secret`. You can use golang templates to define the blueprint and use template functions to transform secret values. You can also pull in `ConfigMaps` that contain golang-template data using `templateFrom`. See [advanced templating](../guides/templating.md) for details.

## Update behavior with 3 different refresh policies

You can control how and when the `ExternalSecret` is refreshed by setting the `spec.refreshPolicy` field. If not specified, the default behavior is `Periodic`.

### CreatedOnce

With `refreshPolicy: CreatedOnce`, the controller will:
- Create the `Kind=Secret` only if it does not exist yet
- Never update the `Kind=Secret` afterwards if the source data changes
- Update/ Recreate the `Kind=Secret` if it gets changed/Deleted
- Useful for immutable credentials or when you want to manage updates manually

Example:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshPolicy: CreatedOnce
  # other fields...
```

### Periodic

With `refreshPolicy: Periodic` (the default behavior), the controller will:
- Create the `Kind=Secret` if it doesn't exist
- Update the `Kind=Secret` regularly based on the `spec.refreshInterval` duration
- When `spec.refreshInterval` is set to zero, it will only create the secret once and not update it afterward
- When `spec.refreshInterval` is set to a value greater than zero, the controller will update the `Kind=Secret` at the specified interval or when the `ExternalSecret` specification changes

Example:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshPolicy: Periodic
  refreshInterval: 1h  # Update every hour
  # other fields...
```

### OnChange

With `refreshPolicy: OnChange`, the controller will:
- Create the `Kind=Secret` if it doesn't exist
- Update the `Kind=Secret` only when the `ExternalSecret`'s metadata or specification changes
- This policy is independent of the `refreshInterval` value
- Useful when you want to manually control when the secret is updated, by modifying the `ExternalSecret` resource

Example:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshPolicy: OnChange
  # other fields...
```

## Manual Refresh

If supported by the configured `refreshPolicy`, you can manually trigger a refresh of the `Kind=Secret` by updating the annotations of the `ExternalSecret`:

```
kubectl annotate es my-es force-sync=$(date +%s) --overwrite
```

## Features

Individual features are described in the [Guides section](../guides/introduction.md):

* [Find many secrets / Extract from structured data](../guides/getallsecrets.md)
* [Templating](../guides/templating.md)
* [Using Generators](../guides/generator.md)
* [Secret Ownership and Deletion](../guides/ownership-deletion-policy.md)
* [Key Rewriting](../guides/datafrom-rewrite.md)
* [Decoding Strategy](../guides/decoding-strategy.md)

## Example

Take a look at an annotated example to understand the design behind the
`ExternalSecret`.

``` yaml
{% include 'full-external-secret.yaml' %}
```
