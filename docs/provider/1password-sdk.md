## 1Password Secrets with SDK

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

Caching applies to read operations (`GetSecret`, `GetSecretMap`). Write operations (`PushSecret`, `DeleteSecret`) automatically invalidate relevant cache entries.

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
  source-key: "my-secret"
```

Looks like this:

```yaml
{% include '1passwordsdk-push-secret.yaml' %}
```

Once all fields of a secret are deleted, the entire secret is deleted if the PushSecret object is removed and
policy is set to `delete`.

### Supported Functionality

Please check the documentation on 1password for [Supported Functionality](https://developer.1password.com/docs/sdks/functionality).
