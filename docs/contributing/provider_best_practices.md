# Provider Best Practices

This document outlines best practices code-wise when implementing a new Provider.

The following coding guidelines should be covered:
- where to put new code
- how to look for code that already exists
- standards new code should follow
- what to use and when to use it
- specifics for certain providers

## Utils

### PushSecrets Metadata handling

![PushSecret Metadata](../../design/010-pushsecret-metadata.md) introduced the way we handle metadata with PushSecret.
Metadata provides configuration options for the provider. I.e. further fine-tuning how the data is pushed. For example,
secret types for AWS, region for GCP or tags and vaults for 1Password.

In code, there is a utils function under utils/metadata called `ParseMetadataParameters`. This is a generic function
that adds the following required struct values to the metadata:

```go
type PushSecretMetadata[T any] struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Spec       T      `json:"spec,omitempty"`
}
```

From here, further fine-tuning is done in code, by defining the `Spec` like this:

```go
type PushSecretMetadataSpec struct {
	ExpirationDate string `json:"expirationDate,omitempty"`
}
```

Then, you can use the `ParseMetadataParameters` function to parse that data and use it in code like this:

```go
metadata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data.GetMetadata())
if err != nil {
    return fmt.Errorf("failed to parse push secret metadata: %w", err)
}

if metadata != nil && metadata.Spec.ExpirationDate != "" {
    t, err := time.Parse(time.RFC3339, metadata.Spec.ExpirationDate)
    if err != nil {
        return fmt.Errorf("error parsing expiration date in metadata: %w. Expected format: YYYY-MM-DDTHH:MM:SSZ (RFC3339). Example: 2024-12-31T20:00:00Z", err)
    }
    unixTime := date.UnixTime(t)
    expires = &unixTime
}
```

This allows a uniform way to handle Metadata and versioning it. The `ParseMetadataParameters` function handles the version
part and the provider specific part is handled in the provider code.
