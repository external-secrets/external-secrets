---
hide:
  - toc
---

# Components

## Overview

Exernal Secrets comes with three components: `Core Controller`, `Webhook` and `Cert Controller`.

This is due to the need to implement conversion webhooks in order to convert custom resources between api versions and
to provide a ValidatingWebhook for the `ExternalSecret` and `SecretStore` resources.

These features are optional but highly recommended. You can disable them with hem chart values `certController.create=false` and `webhook.create=false`.

<br/>
![Component Overview](../pictures/diagrams-component-overview.png)