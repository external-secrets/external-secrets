## 1Password Secrets with SDK

1Password released [developer SDKs](https://developer.1password.com/docs/sdks/) to ease the usage of the secret provider
without the need for any external devices. This provides a much better user experience for automated processes without
the need of the connect server.

### Store Configuration

A store is per vault. This is to prevent a single ExternalSecret potentially accessing ALL vaults.

A sample store configuration looks like this:

```yaml
{% include '1passwordsdk-secret-store.yaml' %}
```

### Valid Requests

Valid secret references should use the following key format: `<item>/[section/]<field>`.

For a one-time password use the following key format: `<item>/[section/]one-time password?attribute=otp`.

### Supported Functionality

Please check the documentation on 1password for [Supported Functionality](https://developer.1password.com/docs/sdks/functionality).

### Sample External Secret

```yaml
{% include '1passwordsdk-external-secret.yaml' %}
```
