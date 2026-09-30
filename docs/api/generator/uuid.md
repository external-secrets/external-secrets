The UUID generator provides random UUIDs that you can feed into your applications. A UUID (Universally Unique Identifier) is a 128-bit label used for information in computer systems. The generator produces version 4 (random) UUIDs as defined in [RFC 9562](https://www.rfc-editor.org/rfc/rfc9562), in the lowercase canonical form.

## Output Keys and Values

| Key  | Description        |
| ---- | ------------------ |
| uuid | the generated UUID |

## Parameters

The UUID generator does not require any additional parameters.

## Example Manifest

```yaml
{% include 'generator-uuid.yaml' %}
```

Example `ExternalSecret` that references the UUID generator:

```yaml
{% include 'generator-uuid-example.yaml' %}
```

Which will generate a `Kind=Secret` with a key called 'uuid' that may look like:

```
21061816-abbd-40ef-8985-6db0fbcbc4c4
```
