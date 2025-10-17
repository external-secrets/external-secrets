## Segura® DevOps Secret Manager (DSM)

External Secrets Operator integrates with [Segura®](https://segura.security/) [DevOps Secret Manager (DSM)](https://segura.security/solutions/devops) module to sync application secrets to secrets held on the Kubernetes cluster.

---

## Authentication

Authentication in Segura® uses DevOps Secret Manager (DSM) application authorization schema. Instructions to setup Authorizations and Secrets in Segura® DSM can be found at [Segura docs for DSM](https://docs.senhasegura.io/docs/how-to-manage-authorizations-per-application-in-devops-secret-manager).

You will need to create an Kubernetes Secret with desired auth parameters, for example:


```yaml
{% include 'senhasegura-dsm-secret.yaml' %}
```

---

## Examples

To sync secrets between Segura® DSM and Kubernetes with External Secrets, you need to define a SecretStore or ClusterSecretStore resource with Segura® provider, setting up authentication in the DSM module with the Secret you defined before.

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

In examples below, consider that three secrets (api-settings, db-settings and hsm-settings) are defined in Segura® DSM

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

You can fetch all key/value pairs for a given secret identifier if you leave the remoteRef.property empty. This returns the json-encoded secret value for that path.

If you only need a specific key, you can select it using remoteRef.property as the key name.

In this method, you can overwrites data name in Kubernetes Secret object (e.g API_SETTINGS and API_SETTINGS_TOKEN)

``` yaml
{% include 'senhasegura-dsm-external-secret-single.yaml' %}
```

Kubernetes Secret will be create with follow `.data.X`

```bash
API_SETTINGS='[{"TOKEN":"example-token-value","URL":"https://example.com/api/example"}]'
API_SETTINGS_TOKEN='example-token-value'
```

---

### Sync DSM secrets using Secret Identifiers with automatically name assignments

If your app requires multiples secrets, it is not required to create multiple ExternalSecret resources, as you can aggregate secrets using a single ExternalSecret resource.

In this method, every secret data in Segura® creates a Kubernetes Secret `.data.X` field

``` yaml
{% include 'senhasegura-dsm-external-secret-multiple.yaml' %}
```

Kubernetes Secret will be created with the following `.data.X`

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

Kubernetes Secret will be created with the following `.data.X`

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
