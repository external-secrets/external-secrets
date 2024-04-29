# Bitwarden support using webhook provider

Bitwarden is an integrated open source password management solution for individuals, teams, and business organizations.

## How does it work?

To make external-secrets compatible with Bitwarden, we need:

* External Secrets Operator >= 0.8.0
* Multiple (Cluster)SecretStores using the webhook provider
* BitWarden CLI image running `bw serve`

When you create a new external-secret object, the External Secrets webhook provider will query the Bitwarden CLI pod that is synced with the Bitwarden server.

## Requirements

* Bitwarden account (it also works with Vaultwarden!)
* A Kubernetes secret which contains your Bitwarden credentials
* A Docker image running the Bitwarden CLI. You could use `ghcr.io/charlesthomas/bitwarden-cli:2023.12.1` or build your own.

Here is an example of a Dockerfile used to build the image:
```dockerfile
FROM debian:sid

ENV BW_CLI_VERSION=2023.12.1

RUN apt update && \
    apt install -y wget unzip && \
    wget https://github.com/bitwarden/clients/releases/download/cli-v${BW_CLI_VERSION}/bw-linux-${BW_CLI_VERSION}.zip && \
    unzip bw-linux-${BW_CLI_VERSION}.zip && \
    chmod +x bw && \
    mv bw /usr/local/bin/bw && \
    rm -rfv *.zip

COPY entrypoint.sh /

CMD ["/entrypoint.sh"]
```

And the content of `entrypoint.sh`:
```bash
#!/bin/bash

set -e

bw config server ${BW_HOST}

export BW_SESSION=$(bw login ${BW_USER} --passwordenv BW_PASSWORD --raw)

bw unlock --check

echo 'Running `bw server` on port 8087'
bw serve --hostname 0.0.0.0 #--disable-origin-protection
```

## Deploy Bitwarden credentials

```yaml
{% include 'bitwarden-cli-secrets.yaml' %}
```

## Deploy Bitwarden CLI container

```yaml
{% include 'bitwarden-cli-deployment.yaml' %}
```

> NOTE: Deploying a network policy is recommended since there is no authentication to query the Bitwarden CLI, which means that your secrets are exposed.

> NOTE: In this example the Liveness probe is querying /sync to ensure that the Bitwarden CLI is able to connect to the server and is also synchronised. (The secret sync is only every 2 minutes in this example)

## Deploy (Cluster)SecretStores

There are four possible (Cluster)SecretStores to deploy, each can access different types of fields from an item in the Bitwarden vault. It is not required to deploy them all.

```yaml
{% include 'bitwarden-secret-store.yaml' %}
```

## Usage

(Cluster)SecretStores:

* `bitwarden-login`: Use to get the `username` or `password` fields
* `bitwarden-fields`: Use to get custom fields
* `bitwarden-notes`: Use to get notes
* `bitwarden-attachments`: Use to get attachments

remoteRef:

* `key`: ID of a secret, which can be found in the URL `itemId` parameter:
  `https://myvault.com/#/vault?type=login&itemId=........-....-....-....-............`s

* `property`: Name of the field to access
    * `username` for the username of a secret (`bitwarden-login` SecretStore)
    * `password` for the password of a secret (`bitwarden-login` SecretStore)
    * `name_of_the_custom_field` for any custom field (`bitwarden-fields` SecretStore)
    * `id_or_name_of_the_attachment` for any attachment (`bitwarden-attachment`, SecretStore)

```yaml
{% include 'bitwarden-secret.yaml' %}
```
