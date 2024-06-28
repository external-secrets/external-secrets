## Pulumi ESC

Sync environments, configs and secrets from [Pulumi ESC](https://www.pulumi.com/product/esc/) to Kubernetes using the External Secrets Operator.

### Authentication

Pulumi [Access Tokens](https://www.pulumi.com/docs/pulumi-cloud/access-management/access-tokens/) are recommended to access Pulumi ESC.

### Creating a SecretStore

A Pulumi SecretStore can be created by specifying the `organization` and `environment` and referencing a Kubernetes secret containing the `accessToken`.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secret-store
spec:
  provider:
    pulumi:
      organization: <NAME_OF_THE_ORGANIZATION>
      environment: <NAME_OF_THE_ENVIRONMENT>
      accessToken:
        secretRef:
          name: <NAME_OF_KUBE_SECRET>
          key: <KEY_IN_KUBE_SECRET>
```

If required, the API URL (`apiUrl`) can be customized as well. If not specified, the default value is `https://api.pulumi.com`.

### Referencing Secrets

Secrets can be referenced by defining the `key` containing the JSON path to the secret. Pulumi ESC secrets are internally organized as a JSON object.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: secret
spec:
  refreshInterval: 5m
  secretStoreRef:
    kind: SecretStore
    name: secret-store
  data:
  - secretKey: <KEY_IN_KUBE_SECRET>
    remoteRef:
      key: <PULUMI_PATH_SYNTAX>
```

**Note:** `key` is not following the JSON Path syntax, but rather the Pulumi path syntax.

#### Examples

* root
* root.nested
* root["nested"]
* root.double.nest
* root["double"].nest
* root["double"]["nest"]
* root.array[0]
* root.array[100]
* root.array[0].nested
* root.array[0][1].nested
* root.nested.array[0].double[1]
* root["key with \"escaped\" quotes"]
* root["key with a ."]
* ["root key with \"escaped\" quotes"].nested
* ["root key with a ."][100]
* root.array[*].field
* root.array["*"].field

See [Pulumi's documentation](https://www.pulumi.com/docs/concepts/options/ignorechanges/) for more information.
