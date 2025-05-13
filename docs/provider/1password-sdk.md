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
