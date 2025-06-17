## senhasegura DevOps Secrets Management (DSM)

External Secrets Operator integrates with [senhasegura](https://senhasegura.com/) [DevOps Secrets Management (DSM)](https://senhasegura.com/devops) module to sync application secrets to secrets held on the Kubernetes cluster.

---

## Authentication

Authentication in senhasegura uses DevOps Secrets Management (DSM) application authorization schema

You need to create an Kubernetes Secret with desired auth parameters, for example:

Instructions to setup authorizations and secrets in senhasegura DSM can be found at [senhasegura docs for DSM](https://helpcenter.senhasegura.io/docs/3.22/dsm) and [senhasegura YouTube channel](https://www.youtube.com/channel/UCpDms35l3tcrfb8kZSpeNYw/search?query=DSM%2C%20en-US)

```yaml
{% include 'senhasegura-dsm-secret.yaml' %}
```

---

## Examples

To sync secrets between senhasegura and Kubernetes with External Secrets, we need to define an SecretStore or ClusterSecretStore resource with senhasegura provider, setting authentication in DSM module with Secret defined before

### SecretStore

``` yaml
{% include 'senhasegura-dsm-secretstore.yaml' %}
```

### ClusterSecretStore

``` yaml
{% include 'senhasegura-dsm-clustersecretstore.yaml' %}
```

---

## Syncing secrets

In examples below, consider that three secrets (api-settings, db-settings and hsm-settings) are defined in senhasegura DSM

---

**Secret Identifier: ** api-settings

**Secret data:** 

```bash
URL=https://example.com/api/example
TOKEN=example-token-value
```

---

**Secret Identifier: ** db-settings

**Secret data:** 

```bash
DB_HOST='db.example'
DB_PORT='5432'
DB_USERNAME='example'
DB_PASSWORD='example'
```

---

**Secret Identifier: ** hsm-settings

**Secret data:** 

```bash
HSM_ADDRESS='hsm.example'
HSM_PORT='9223'
```


---

### Sync DSM secrets using Secret Identifiers

You can fetch all key/value pairs for a given secret identifier If you leave the remoteRef.property empty. This returns the json-encoded secret value for that path.

If you only need a specific key, you can select it using remoteRef.property as the key name.

In this method, you can overwrites data name in Kubernetes Secret object (e.g API_SETTINGS and API_SETTINGS_TOKEN)

``` yaml
{% include 'senhasegura-dsm-external-secret-single.yaml' %}
```

Kubernetes Secret will be create with follow `.data.X`

```bash
API_SETTINGS='[{"TOKEN":"example-token-value","URL":"https://example.com/api/example"}]'
API_SETTINGS_TOKEN='example-token-value'
API_SETTINGS_TOKEN_ENCRYPT='hAZJktRFdzSkGxxiiSE46T271veCgwvC0GrY+AwDYA/KeuFZFdPgZsJ74awu1WR6x4BrbMLTXNpQw4UqChdbaM7VoKUCkPTcCU1jsveqYNisM2MNF98QjNjvp+9jXHfAsClLA5AvJxe3GjfWIi18E4PieFpATn/BTrmoklx4rSkWmfifZol7Wcny0D2fhrj/JOdxEIqowUB/tNwYzNd+lXgm55wea+G3YnD3Fr4ARaCCaQMUcdW9Kgx7mmZGZE3xDAhs8WMfpe9xVZ17Ca7Sw2r1JKS0o0fYiZNHUmCXVsP9O+//+0sfEtETiVUF0jItrwlK4GL8+bVcXQ9N2TW7+g=='
API_SETTINGS_TOKEN_DECRYPT='a1b2c3d4'
```

---

### Sync DSM secrets using Secret Identifiers with automatically name assignments

If your app requires multiples secrets, it is not required to create multiple ExternalSecret resources, you can aggregate secrets using a single ExternalSecret resource

In this method, every secret data in senhasegura creates an Kubernetes Secret `.data.X` field

``` yaml
{% include 'senhasegura-dsm-external-secret-multiple.yaml' %}
```

Kubernetes Secret will be create with follow `.data.X`

```bash
URL='https://example.com/api/example'
TOKEN='example-token-value'
DB_HOST='db.example'
DB_PORT='5432'
DB_USERNAME='example'
DB_PASSWORD='example'
```

<!-- https://github.com/external-secrets/external-secrets/pull/830#discussion_r858657107 -->

<!-- ### Sync all secrets from DSM authorization

You can sync all secrets that your authorization in DSM has using find, in a future release you will be able to filter secrets by name, path or tags

``` yaml
{% include 'senhasegura-dsm-external-secret-all.yaml' %}
```

Kubernetes Secret will be create with follow `.data.X`

```bash
URL='https://example.com/api/example'
TOKEN='example-token-value'
DB_HOST='db.example'
DB_PORT='5432'
DB_USERNAME='example'
DB_PASSWORD='example'
HSM_ADDRESS='hsm.example'
HSM_PORT='9223'
``` -->
