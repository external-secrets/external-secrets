
![aws sm](./pictures/eso-az-kv-azure-kv.png)

## Azure Key vault

External Secrets Operator integrates with [Azure Key vault](https://azure.microsoft.com/en-us/services/key-vault/) for secrets, certificates and Keys management.

### Authentication

At the moment, we only support [service principals](https://docs.microsoft.com/en-us/azure/key-vault/general/authentication) authentication.

#### Service Principal key authentication

A service Principal client and Secret is created and the JSON keyfile is stored in a `Kind=Secret`. The `ClientID` and `ClientSecret` should be configured for the secret. This service principal should have proper access rights to the keyvault to be managed by the operator

```yaml
{% include 'azkv-credentials-secret.yaml' %}
```

### Update secret store
Be sure the `azkv` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'azkv-secret-store.yaml' %}
```

### Object Types

Azure KeyVault manages different [object types](https://docs.microsoft.com/en-us/azure/key-vault/general/about-keys-secrets-certificates#object-types), we support `keys`, `secrets` and `certificates`. Simply prefix the key with `key`, `secret` or `cert` to retrieve the desired type (defaults to secret).

| Object Type   | Return Value                                                                                                                                                                                                                      |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `secret`      | the raw secret value.                                                                                                                                                                                                             |
| `key`         | A JWK which contains the public key. Azure KeyVault does **not** export the private key. You may want to use [template functions](guides-templating.md) to transform this JWK into PEM encoded PKIX ASN.1 DER format. |
| `certificate` | The raw CER contents of the x509 certificate. You may want to use [template functions](guides-templating.md) to transform this into your desired encoding                                                             |


### Creating external secret

To create a kubernetes secret from the Azure Key vault secret a `Kind=ExternalSecret` is needed.

You can manage keys/secrets/certificates saved inside the keyvault , by setting a "/" prefixed type in the secret name , the default type is a `secret`. other supported values are `cert` and `key`

to select all secrets inside the key vault , you can use the `dataFrom` directive

```yaml
{% include 'azkv-external-secret.yaml' %}
```

The operator will fetch the Azure Key vault secret and inject it as a `Kind=Secret`
```
kubectl get secret secret-to-be-created -n <namespace> | -o jsonpath='{.data.dev-secret-test}' | base64 -d
```
