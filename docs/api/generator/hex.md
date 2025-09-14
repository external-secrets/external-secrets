The Hex generator provides random hexadecimal strings that you can feed into your applications. It uses cryptographically secure random number generation to produce hex strings of configurable length and format.

## Output Keys and Values

| Key | Description                |
| --- | -------------------------- |
| hex | the generated hex string   |

## Parameters

You can influence the behavior of the generator by providing the following args

| Key       | Default | Description                                                                 |
| --------- | ------- | --------------------------------------------------------------------------- |
| length    | 16      | Length of the hex string to be generated (number of hex characters).       |
| uppercase | false   | Use uppercase letters (A-F) instead of lowercase (a-f).                    |
| prefix    | ""      | Optional prefix to add to the hex string (e.g., "0x").                     |

## Example Manifest

```yaml
{% include 'generator-hex.yaml' %}
```

Example `ExternalSecret` that references the Hex generator:
```yaml
{% include 'generator-hex-example.yaml' %}
```

Which will generate a `Kind=Secret` with a key called 'hex' that may look like:

```
0xABCD1234567890EF1234567890ABCDEF
```

With default values (length: 16, lowercase, no prefix) you would get something like:

```
a1b2c3d4e5f67890
```

With different configurations:

```
# length: 8, uppercase: true, prefix: "0x"
0xDEADBEEF

# length: 32, uppercase: false, prefix: ""  
a1b2c3d4e5f678901234567890abcdef1234
```
