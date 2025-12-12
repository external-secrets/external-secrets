# esoctl

This tool contains external-secrets-operator related activities and helpers.

## Templates

`cmd/esoctl` -> `esoctl template`

The purpose is to give users the ability to rapidly test and iterate on templates in a PushSecret/ExternalSecret.

For a more in-dept description read [Using esoctl Tool](../../docs/guides/using-esoctl-tool.md).

This project doesn't have its own go mod files to allow it to grow together with ESO instead of waiting for new ESO
releases to import it.

## Fetch

Fetch secrets directly from a provider without needing an ExternalSecret resource. Useful for debugging store configurations or quick secret retrieval.

### Basic Usage

```bash
# fetch a secret (requires kubeconfig for auth credentials)
esoctl fetch --store vault-store.yaml --key secret/data/myapp/password

# extract a specific property
esoctl fetch --store vault-store.yaml --key secret/data/db --property host

# output formatting
esoctl fetch --store vault-store.yaml --key secret/data/config --format json
```

### Standalone Mode ( no Kubernetes cluster required )

Run without a Kubernetes cluster using `--standalone`. Auth credentials come from environment variables or a yaml file:

```bash
# using environment variables
export VAULT_TOKEN=hvs.xyz
esoctl fetch --store vault-store.yaml --key secret/data/myapp --standalone

# using custom auth file
esoctl fetch --store vault-store.yaml --key secret/data/myapp --standalone --auth-file ./secrets.yaml
```

**Supported environment variables:**

- `VAULT_TOKEN`
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
- `GCP_SERVICE_ACCOUNT_KEY`
- `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`
- `ESO_SECRET_<NAME>_<KEY>` pattern for custom secrets

**NOTE:** Standalone mode doesn't support pod identity, IRSA, workload identity, service account token exchange, or other
cluster-based auth mechanisms. Use static credentials only.

### Output Formats

- `text`: plain text (default for single values)
- `json`: json output
- `env`: environment format (`KEY=value`)
- `dotenv`: export format (`export KEY="value"`)

## AWS Example

Store:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: aws-secretsmanager
spec:
  provider:
    aws:
      service: SecretsManager
      role: arn:aws:iam::<account>:role/external-secrets-role
      region: eu-central-1
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: awssm-secret
            key: access-key
          secretAccessKeySecretRef:
            name: awssm-secret
            key: secret-access-key
```

Secret:

```yaml
apiVersion: v1
data:
  access-key: <base64>
  secret-access-key: <base64>
kind: Secret
metadata:
  name: awssm-secret
```

Secret in AWS:
```json
{
  "name": {"first": "Tom", "last": "Anderson"},
  "friends": [
    {"first": "Dale", "last": "Murphy"},
    {"first": "Roger", "last": "Craig"},
    {"first": "Jane", "last": "Murphy"}
  ]
}
```

The command to fetch a single value:
```
esoctl fetch --auth-file secret.yaml --store secret-store.yaml --key friendlist --property name.first --standalone
INFO: Loading store from: secret-store.yaml
INFO: Store loaded: /aws-secretsmanager (kind: SecretStore)
INFO: Fetching secret with key: friendlist
Tom
```

The command to fetch a map:
```
esoctl fetch-map --auth-file secret.yaml --store secret-store.yaml --key friendlist --standalone --format json
INFO: Loading store from: secret-store.yaml
INFO: Store loaded: /aws-secretsmanager (kind: SecretStore)
INFO: Fetching secret map with key: friendlist
INFO: Retrieved 2 secret(s)
{
  "friends": "[\n    {\"first\": \"Dale\", \"last\": \"Murphy\"},\n    {\"first\": \"Roger\", \"last\": \"Craig\"},\n    {\"first\": \"Jane\", \"last\": \"Murphy\"}\n  ]",
  "name": "{\"first\": \"Tom\", \"last\": \"Anderson\"}"
}
```

Note that all `INFO` entries are on _stderr_ so they don't interfere with the actual secret value output.
