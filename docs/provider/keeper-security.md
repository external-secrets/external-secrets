## Keeper Security

External Secrets Operator integrates with [Keeper Security](https://www.keepersecurity.com/) for secret management by using [Keeper Secrets Manager](https://docs.keeper.io/secrets-manager/secrets-manager/about).


## Authentication

### Secrets Manager Configuration (SMC)

KSM can authenticate using *One Time Access Token* or *Secret Manager Configuration*. In order to work with External Secret Operator we need to configure a Secret Manager Configuration.

#### Creating Secrets Manager Configuration

You can find the documentation for the Secret Manager Configuration creation [here](https://docs.keeper.io/secrets-manager/secrets-manager/about/secrets-manager-configuration). Make sure you add the proper permissions to your device in order to be able to read and write secrets

Once you have created your SMC, you will get a config.json file containing the following keys:
- `hostname`
- `clientId`
- `privateKey`
- `serverPublicKeyId`
- `appKey`
- `appOwnerPublicKey`

This config will be required to create your secretStores

## Important note about this documentation
_**The KepeerSecurity calls the entries in vaults 'Records'. These docs use the same term.**_

### Update secret store
Be sure the `keepersecurity` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'keepersecurity-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` for `SecretAccessKeyRef` with the namespace of the secret that we just created.

## External Secrets
### Behavior
* How a Record is equated to an ExternalSecret:
    * `remoteRef.key` is equated to a Record's Title
    * `remoteRef.property` is equated to one of the following options:
        * Fields: Record's field's Type
        * CustomFields: Record's field's Label
        * Files: Record's file's Name
        * If empty, defaults to the complete Record in JSON format
    * `remoteRef.version` is currently not supported.
* `dataFrom`:
    * `find.path` is currently not supported.
    * `find.name.regexp` is equated to one of the following options:
        * Fields: Record's field's Type
        * CustomFields: Record's field's Label
        * Files: Record's file's Name
    * `find.tags` are not supported at this time.

### Creating external secret
To create a kubernetes secret from the GCP Secret Manager secret a `Kind=ExternalSecret` is needed.

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

Push Secret will only work with a custom KeeperSecurity Record type `ExternalSecret`

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
* If the record with the selected name exists but the key does not exists the record can not be updated. See [Ability to add custom fields to existing secret #17](https://github.com/Keeper-Security/secrets-manager-go/issues/17)
