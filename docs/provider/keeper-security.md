## Keeper Security

External Secrets Operator integrates with [Keeper Security](https://www.keepersecurity.com/) for secret management by using [Keeper Secrets Manager](https://docs.keeper.io/secrets-manager/secrets-manager/about).


## Authentication

### Secrets Manager Configuration (SMC)

KSM can authenticate using *One Time Access Token* or *Secret Manager Configuration*. In order to work with External Secret Operator we need to configure a Secret Manager Configuration.

#### Creating Secrets Manager Configuration

You can find the documentation for the Secret Manager Configuration creation [here](https://docs.keeper.io/secrets-manager/secrets-manager/about/secrets-manager-configuration). Make sure you add the proper permissions to your device in order to be able to read and write secrets

Once you have created your SMC, you will get a config.json file or a base64 json encoded string containing the following keys:

- `hostname`
- `clientId`
- `privateKey`
- `serverPublicKeyId`
- `appKey`
- `appOwnerPublicKey`

This base64 encoded jsong string will be required to create your secretStores

## Important note about this documentation
_**The KeeperSecurity calls the entries in vaults 'Records'. These docs use the same term.**_

### Update secret store
Be sure the `keepersecurity` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'keepersecurity-secret-store.yaml' %}
```

**NOTE 1:** `folderID` target the folder ID where the secrets should be pushed to. It requires write permissions within the folder

**NOTE 2:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `SecretAccessKeyRef` with the namespace of the secret that we just created.

## External Secrets
### Behavior
* How a Record is equated to an ExternalSecret:
    * `remoteRef.key` is equated to a Record's ID, **or** a [Keeper Notation](https://docs.keeper.io/en/secrets-manager/secrets-manager/about/keeper-notation) expression when prefixed with `keeper://` (see [Keeper Notation](#keeper-notation) below). ID lookup is the default; record **title** lookup only happens when `getByTitleFallback` is enabled on the SecretStore.
    * `remoteRef.property` is equated to one of the following options:
        * Fields: Record's field's Label (if present), otherwise [Record's field's Type](https://docs.keeper.io/secrets-manager/secrets-manager/about/field-record-types)
        * CustomFields: Record's field's Label
        * Files: Record's file's Name
        * If empty, defaults to the complete Record in JSON format
    * `remoteRef.version` is currently not supported.
* `dataFrom`:
    * `find.path` filters records by their **folder path** (e.g. `Production/Databases`). Matching is by prefix, so the path and everything beneath it is returned.
    * `find.name.regexp` matches the record's **title** (regular expression).
    * `find.tags` are not supported at this time.

### Keeper Notation

`remoteRef.key` accepts a [Keeper Notation](https://docs.keeper.io/en/secrets-manager/secrets-manager/about/keeper-notation) expression to pull a specific field, custom field, or file out of a record. It must be prefixed with `keeper://`:

```yaml
  data:
    - secretKey: username
      remoteRef:
        key: keeper://<recordUID|title>/field/login
    - secretKey: password
      remoteRef:
        key: keeper://<recordUID|title>/field/password
    - secretKey: token
      remoteRef:
        key: keeper://<recordUID|title>/custom_field/API Token
```

Array indexes and predicates are supported (e.g. `keeper://<uid>/field/phone[0]`, `keeper://<uid>/field/name[first]`). A key without the `keeper://` prefix is treated as a plain record key — a UID by default, or a title only when `getByTitleFallback` is enabled on the SecretStore.

### Finding records by folder path

`dataFrom.find.path` returns every record at (or beneath) a folder path:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: by-folder
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: SecretStore
    name: keeper
  target:
    name: by-folder
  dataFrom:
    - find:
        path: "Production/Databases"   # all records in this folder and its subfolders
```

**NOTE:** For complex [types](https://docs.keeper.io/secrets-manager/secrets-manager/about/field-record-types), like name, phone, bankAccount, which does not match with a single string value, external secrets will return the complete json string. Use the json template functions to decode.

### Creating external secret
To create a kubernetes secret from Keeper Secret Manager secret a `Kind=ExternalSecret` is needed.

```yaml
{% include 'keepersecurity-external-secret.yaml' %}
```

The operator will fetch the Keeper Secret Manager secret and inject it as a `Kind=Secret`
```
kubectl get secret secret-to-be-created -n <namespace> | -o jsonpath='{.data.dev-secret-test}' | base64 -d
```

## Limitations

There are some limitations using this provider.

* Keeper Secret Manager does not work with `General` Records types nor legacy non-typed records
* Using tags `find.tags` is not supported by KSM

## Performance & rate-limit tuning

Keeper Secrets Manager throttles per device (a `403 {"error":"throttled"}` at the application layer, and a `429 Too Many Requests` at the edge). Two built-in behaviors keep the provider within those limits at scale:

* **Throttle/429 retry (on by default).** Read and write calls that hit a throttle (`403`) or `429` are retried with exponential backoff + jitter, instead of surfacing as an `ExternalSecret` sync error. Non-rate-limit errors are returned immediately.
* **Shared record cache (opt-in).** A single `get_secret` call returns every record the application can read. Enabling the cache lets one fetch serve a whole reconcile wave of `ExternalSecrets` (and `Validate`) instead of one backend call per `ExternalSecret`, which is the main lever for large fleets.

Configure via environment variables on the `external-secrets` controller:

| Variable | Default | Description |
|---|---|---|
| `KEEPER_RECORD_CACHE_TTL_MS` | `0` (disabled) | TTL in milliseconds for the shared record/folder cache. Set e.g. `30000` to dedupe bursts; writes invalidate the cache immediately. |
| `KEEPER_THROTTLE_RETRY_ATTEMPTS` | `4` | Max attempts (including the first) for a throttled call. |
| `KEEPER_THROTTLE_RETRY_BASE_MS` | `500` | Base backoff in milliseconds (doubles each attempt). |
| `KEEPER_THROTTLE_RETRY_MAX_MS` | `10000` | Backoff ceiling in milliseconds. |

For very large deployments, also stagger `refreshInterval` across `ExternalSecrets` and raise the controller's `--concurrent` flag.

## Push Secrets

Push Secret will only work with a custom KeeperSecurity Record type `externalSecrets`

### Behavior
* `selector`:
  * `secret.name`: name of the kubernetes secret to be pushed
* `data.match`:
  * `secretKey`: key on the selected secret to be pushed
  * `remoteRef.remoteKey`: Secret and key to be created on the remote provider
    * Format: SecretName/SecretKey

### Creating push secret
To create a Keeper Security record from kubernetes a `Kind=PushSecret` is needed.

```yaml
{% include 'keepersecurity-push-secret.yaml' %}
```

### Limitations
* Only possible to push one key per secret at the moment
* If the record with the selected name exists but the key does not exist, the record cannot be updated. See [Ability to add custom fields to existing secret #17](https://github.com/Keeper-Security/secrets-manager-go/issues/17)
