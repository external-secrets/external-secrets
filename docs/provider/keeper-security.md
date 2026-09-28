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
    * `remoteRef.key` is equated to a Record's ID
    * `remoteRef.property` is equated to one of the following options:
        * Fields: Record's field's Label (if present), otherwise [Record's field's Type](https://docs.keeper.io/secrets-manager/secrets-manager/about/field-record-types)
        * CustomFields: Record's field's Label
        * Files: Record's file's Name
        * If empty, defaults to the complete Record in JSON format
    * `remoteRef.version` is currently not supported.
* `dataFrom`:
    * `find.path` is currently not supported.
    * `find.name.regexp` is equated to one of the following options:
        * Fields: Record's field's Label (if present), otherwise Record's field's Type
        * CustomFields: Record's field's Label
        * Files: Record's file's Name
    * `find.tags` are not supported at this time.

**NOTE:** For complex [types](https://docs.keeper.io/secrets-manager/secrets-manager/about/field-record-types), like name, phone, bankAccount, which does not match with a single string value, external secrets will return the complete json string. Use the json template functions to decode.

**NOTE:** A record that cannot be found is reported as a missing secret, which is what `deletionPolicy: Delete` and `deletionPolicy: Merge` act on. Keeper Secrets Manager returns an empty record set both for a record that was deleted and for one that is simply no longer shared with the KSM application, and the two are indistinguishable to the provider. So with `deletionPolicy: Delete`, revoking the application's access to a record removes the key from the target Secret exactly as if the record had been deleted. Use the default `deletionPolicy: Retain` if that is not what you want.

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
* Using path `find.path` is not supported at the moment

## Push Secrets

PushSecret creates and updates only Keeper records of the custom `externalSecrets` type. The configured `folderID` must grant write access to create a record.

### Behavior
* `selector`:
  * `secret.name`: name of the kubernetes secret to be pushed
* `data.match`:
    * Whole-record target: omit both `secretKey` and `remoteRef.property`. ESO pushes every key in the selected Secret to the Keeper record named by `remoteRef.remoteKey`. With `updatePolicy: Replace`, keys removed from the source Secret are removed from the Keeper record.
    * Property target: set `remoteRef.remoteKey` to the Keeper record name and `remoteRef.property` to the target field name. Set `secretKey` to push one source key, or omit it to store the selected Secret as JSON in that property. Multiple entries can target different properties of the same Keeper record.
    * Legacy target: `remoteRef.remoteKey: record-name/key` with `secretKey` remains supported. Standard legacy keys (`login`, `username`, `password`, `url`, and `baseurl`) use Keeper's default standard fields.

Keeper standard keys are stored as standard fields: `login` and `username` use `login`, `password` uses `password`, and `url` and `baseurl` use `url`. Other keys are stored as custom `secret` fields.

With `deletionPolicy: Delete`, deleting a whole-record target deletes the Keeper record. Deleting a property target removes only that property; if it was the record's final field, ESO deletes the empty Keeper record.

### Creating push secret
To create a Keeper Security record from kubernetes a `Kind=PushSecret` is needed.

```yaml
{% include 'keepersecurity-push-secret.yaml' %}
```
