## 1Password Secrets with SDK

1Password released [developer SDKs](https://developer.1password.com/docs/sdks/) to ease the usage of the secret provider
without the need for any external devices. This provides a much better user experience for automated processes without
the need of the connect server.

### Valid Requests

Valid secret references should use the following key format: `op://<vault>/<item>/[section/]<field>`.

For a one-time password use the following key format: `op://<vault>/<item>/[section/]one-time password?attribute=otp`.

### Fetching all secrets

### Supported Functionality

Please check the documentation on 1password for [Supported Functionality](https://developer.1password.com/docs/sdks/functionality).
