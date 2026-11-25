# 1Password Secrets with SDK

1Password released [developer SDKs](https://developer.1password.com/docs/sdks/) to ease the usage of the secret provider
without the need for any external devices. This provides a much better user experience for automated processes without
the need of the connect server.

_Note_: In order to use ESO with 1Password SDK, documents must have unique label names. Meaning, if there is a label
that has the same title as another label we won't know which one to update and an error is thrown:
`found multiple labels with the same key`.

### Store Configuration

A store is per vault. This is to prevent a single ExternalSecret potentially accessing ALL vaults.

A sample store configuration looks like this:

```yaml
{% include '1passwordsdk-secret-store.yaml' %}
```

### Client-Side Caching

Optional client-side caching reduces 1Password API calls. Configure TTL and cache size in the store:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: 1password-cached
spec:
  provider:
    onepasswordSDK:
      vault: production
      auth:
        serviceAccountSecretRef:
          name: op-token
          key: token
      cache:
        ttl: 5m      # Optional, default: 5m
        maxSize: 100 # Optional, default: 100
```

Caching applies to read operations (`GetSecret`, `GetSecretMap`, `GetAllSecrets`). Write operations (`PushSecret`, `DeleteSecret`) automatically invalidate relevant cache entries.

!!! warning "Experimental"
    This is an experimental feature and if too long of a TTL is set, secret information might be out of date.

### GetSecret

Valid secret references should use the following key format: `<item>/[section/]<field>`.

This is described here: [Secret Reference Syntax](https://developer.1password.com/docs/cli/secret-reference-syntax/).

For a one-time password use the following key format: `<item>/[section/]one-time password?attribute=otp`.

```yaml
{% include '1passwordsdk-external-secret.yaml' %}
```

### PushSecret

Pushing a secret is also supported. For example a push operation with the following secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: source-secret
stringData:
  api-key: "my-api-key"
  api-url: "https://example.com/api"
```

Looks like this:

```yaml
{% include '1passwordsdk-push-secret.yaml' %}
```

Once all fields of a secret are deleted, the entire secret is deleted if the PushSecret object is removed and
policy is set to `delete`.

To sync the entire secret into a single 1Password item, the following configuration can be used:

```yaml
{% include '1passwordsdk-push-secret-all-keys.yaml' %}
```

#### Item category and websites

New items are created with the `Server` category by default. Set `category` in the `PushSecretMetadata` to create
another kind of item, for example a `Login` that 1Password can autofill. The category is only applied when the item is
created: 1Password does not allow changing the category of an existing item. `Document` is not supported.

For `Login` items, push to the `username` and `password` properties to fill the built-in fields, and set `websites`
so 1Password suggests the item on these websites. Like `tags`, `websites` are set on the whole item and the last write
wins if entries targeting the same item differ.

```yaml
  data:
    - match:
        secretKey: username
        remoteRef:
          remoteKey: my-app
          property: username
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          category: Login
          fieldType: text
          websites: ["https://my-app.example.com"]
    - match:
        secretKey: password
        remoteRef:
          remoteKey: my-app
          property: password
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          category: Login
          websites: ["https://my-app.example.com"]
```

### Environments

1Password has added [Environments](https://developer.1password.com/docs/environments) functionality as a BETA feature.
This is only supported by 1Password SDK and not the connect server.

Environments are an alternative to Vaults. To use the environment define the `environment` id in your Store configuration
instead of the `vault` value.

The rest of the settings should remain the same.

The SDK, as of this writing, does not support filtering client side, which means that each call always returns everything.
To tackle this problem, the cache will cache the individual values so if ever the same object is requested again within the
TTL of the cache it will only fetch that single value.

It also caches ALL the values with a special key, so if repeated All calls are made, that shouldn't be a problem either.

This is a BETA feature. Please use with caution.

### Supported Functionality

Please check the documentation on 1password for [Supported Functionality](https://developer.1password.com/docs/sdks/functionality).
