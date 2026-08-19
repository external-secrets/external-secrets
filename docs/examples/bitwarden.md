# Bitwarden support using webhook provider

Bitwarden is an integrated open source password management solution for individuals, teams, and business organizations.

!!! note

    This documentation is for Bitwarden **Password Manager**.
    It is different from [Bitwarden Secrets Manager](https://bitwarden.com/products/secrets-manager/), which enables developers, DevOps, and cybersecurity teams to centrally store, manage, and deploy secrets at scale.
    To integrate with Bitwarden **Secrets Manager**, reference the [provider documentation](../provider/bitwarden-secrets-manager.md).

## How does it work?

To make external-secrets compatible with Bitwarden, we need:

* External Secrets Operator >= 0.8.0
* Multiple (Cluster)SecretStores using the webhook provider
* Bitwarden CLI image running `bw serve`

When you create a new external-secret object, the External Secrets webhook provider will query the Bitwarden CLI pod that is synced with the Bitwarden server.

## Requirements

* Bitwarden account (it also works with Vaultwarden!)
* A Kubernetes secret which contains your Bitwarden credentials
* A Docker image running the Bitwarden CLI. You could use `ghcr.io/charlesthomas/bitwarden-cli:2026.3.0` or build your own.

Here is an example of a Dockerfile used to build the image:
```dockerfile
FROM debian:sid

ARG BW_CLI_VERSION=2025.12.1
ARG TARGETARCH

RUN apt update && \
    apt install -y wget unzip && \
    if [ "$TARGETARCH" = "arm64" ]; then \
      BW_ARCH="-arm64"; \
    else \
      BW_ARCH=""; \
    fi && \
    wget https://github.com/bitwarden/clients/releases/download/cli-v${BW_CLI_VERSION}/bw-oss-linux${BW_ARCH}-${BW_CLI_VERSION}.zip && \
    unzip bw-oss-linux${BW_ARCH}-${BW_CLI_VERSION}.zip && \
    chmod +x bw && \
    mv bw /usr/local/bin/bw && \
    rm -rfv *.zip

COPY entrypoint.sh /

CMD ["/entrypoint.sh"]
```

And the content of `entrypoint.sh`:
```bash
#!/usr/bin/env bash

set -e

bw config server ${BW_HOST}

if [ -n "$BW_CLIENTID" ] && [ -n "$BW_CLIENTSECRET" ]; then
    echo "Using apikey to log in"
    bw login --apikey --raw
    export BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw)
else
    echo "Using username and password to log in"
    export BW_SESSION=$(bw login ${BW_USER} --passwordenv BW_PASSWORD --raw)
fi

bw unlock --check

echo 'Running `bw server` on port 8087'
bw serve --hostname all
```

!!! warning "Bitwarden CLI 2026.6.0 and later: use `--hostname all`"

    `bw serve` 2026.6.0 added a Host header allowlist on top of its existing
    Origin header check. With `--hostname 0.0.0.0` the allowlist is built from
    the bound hostname, so it contains only `localhost:8087`, `127.0.0.1:8087`,
    `[::1]:8087` and `0.0.0.0:8087`.

    The webhook provider sends whatever authority the SecretStore `url` carries
    as the `Host` header. For a store pointing at
    `http://bitwarden-cli.bitwarden.svc:8087` that authority is not on the
    allowlist, so `bw serve` answers `403` and logs:

    ```
    Blocking request with disallowed Host "bitwarden-cli.bitwarden.svc:8087"
    ```

    Every ExternalSecret backed by these stores fails.

    `--hostname all` binds every interface and skips the Host allowlist, which
    is why the example above uses it. It is also accepted by older releases, so
    the same entrypoint works either way.

    Prefer this over `--disable-origin-protection`. That flag turns off the
    Origin header check as well, whereas `--hostname all` leaves it in place.
    The webhook provider does not send an `Origin` header, so it is unaffected.

    Pinning the Host from the store does not work as a substitute. Entries in
    the store's `headers` are applied with `Header.Add`, and Go takes the
    request Host from the URL rather than from `Header["Host"]`, so a
    `Host: localhost:8087` header is silently ignored.

    Neither option authenticates callers. The NetworkPolicy below is what
    restricts access to `bw serve`, so deploy it.

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

There are five possible (Cluster)SecretStores to deploy, each can access different types of fields from an item in the Bitwarden vault. It is not required to deploy them all.

```yaml
{% include 'bitwarden-secret-store.yaml' %}
```

## Usage

(Cluster)SecretStores:

* `bitwarden-login`: Use to get the `username` or `password` fields
* `bitwarden-fields`: Use to get custom fields
* `bitwarden-notes`: Use to get notes
* `bitwarden-attachments`: Use to get attachments
* `bitwarden-ssh`: Use to get ssh key stored in `privateKey` (other possible fields are `publicKey` and `keyFingerprint`)

remoteRef:

* `key`: ID of a secret, which can be found in the URL `itemId` parameter:
  `https://myvault.com/#/vault?type=login&itemId=........-....-....-....-............`s

* `property`: Name of the field to access
    * `username` for the username of a secret (`bitwarden-login` SecretStore)
    * `password` for the password of a secret (`bitwarden-login` SecretStore)
    * `name_of_the_custom_field` for any custom field (`bitwarden-fields` SecretStore)
    * `id_or_name_of_the_attachment` for any attachment (`bitwarden-attachment`, SecretStore)
    * `name_of_the_ssh_field` for any ssh field (`bitwarden-ssh` SecretStore) possible fields are `publicKey`, `privateKey` and `keyFingerprint`

```yaml
{% include 'bitwarden-secret.yaml' %}
```
