The UUID generator provides random UUIDs that you can feed into your applications. A UUID (Universally Unique Identifier) is a 128-bit label used for information in computer systems. Please see below for the format in use.

## Output Keys and Values

| Key  | Description        |
| ---- | ------------------ |
| uuid | the generated UUID |

## Parameters

The UUID generator does not require any additional parameters.

## Example Manifest

```yaml
{ % include 'generator-uuid.yaml' % }
```

Example `ExternalSecret` that references the UUID generator:

```yaml
{ % include 'generator-uuid-example.yaml' % }
```

Which will generate a `Kind=Secret` with a key called 'uuid' that may look like:

```
EA111697-E7D0-452C-A24C-8E396947E865
```

With default values you would get something like:

```
4BEE258F-64C9-4755-92DC-AFF76451471B
```
