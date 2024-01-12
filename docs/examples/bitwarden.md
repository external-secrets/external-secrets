# Bitwarden support using webhook provider

Bitwarden is an integrated open source password management solution for individuals, teams, and business organizations.

## How is it working ?

To make external-secret compatible with BitWarden, we need:

* External-Secret >= 0.8.0
* To use the Webhook Provider
* 2 (Cluster)SecretStores
* BitWarden CLI image running `bw serve`

When you create a new external-secret object,
External-Secret Webhook provider will do a query to the Bitwarden CLI pod,
which is synced with the BitWarden server.

## Requirements

* Bitwarden account (it works also with VaultWarden)
* A Kubernetes secret which contains your BitWarden Credentials
* You need a Docker image with BitWarden CLI installed.
  You could use `ghcr.io/charlesthomas/bitwarden-cli:2023.12.1` or build your own.

Here an example of Dockerfile use to build this image:
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

And the content of `entrypoint.sh`
```bash
#!/bin/bash

set -e

bw config server ${BW_HOST}

export BW_SESSION=$(bw login ${BW_USER} --passwordenv BW_PASSWORD --raw)

bw unlock --check

echo 'Running `bw server` on port 8087'
bw serve --hostname 0.0.0.0 #--disable-origin-protection
```


## Deploy Bitwarden Credentials

```yaml
{% include 'bitwarden-cli-secrets.yaml' %}
```

## Deploy Bitwarden CLI container

```yaml
{% include 'bitwarden-cli-deployment.yaml' %}
```

> NOTE: Deploying a network policy is recommended since, there is no authentication to query the BitWarden CLI, which means that your secrets are exposed.

> NOTE: In this example the Liveness probe is quering /sync to ensure that the BitWarden CLI is able to connect to the server and also to sync secrets. (The secret sync is only every 2 minutes in this example)

## Deploy ClusterSecretStore (Or SecretStore)

Here the two ClusterSecretStore to deploy

```yaml
{% include 'bitwarden-secret-store.yaml' %}
```


## How to use it ?

* If you need the `username` or the `password` of a secret, you have to use `bitwarden-login`
* If you need a custom field of a secret, you have to use `bitwarden-fields`
* If you need to use a Bitwarden Note for multiline strings (SSH keys, service account json files), you have to use `bitwarden-notes`
* The `key` is the ID of a secret, which can be find in the URL with the `itemId` value:
  `https://myvault.com/#/vault?itemId=........-....-....-....-............`
* The `property` is the name of the field:
  * `username` for the username of a secret (`bitwarden-login` SecretStore)
  * `password` for the password of a secret (`bitwarden-login` SecretStore)
  * `name_of_the_custom_field` for any custom field (`bitwarden-fields` SecretStore)

```yaml
{% include 'bitwarden-secret.yaml' %}
```
