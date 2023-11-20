
Contrary to what `ExternalSecret` does by pulling secrets from secret providers and creating `kind=Secret` in your cluster, `PushSecret` reads a local `kind=Secret` and pushes its content to a secret provider.

If there's already a secret in the secrets provided with the intended name of the secret to be created by the `PushSecret` you'll see the `PushSecret` in Error state, and when described you'll see a message saying `secret not managed by external-secrets`.

By default, the secret created in the secret provided will not be deleted even after deleting the `PushSecret`, unless you set `spec.deletionPolicy` to Delete. 

``` yaml
{% include 'full-pushsecret.yaml' %}
```

## Backup use case

An interesting use case for `kind=PushSecret` is backing up your current secret from one provider to another one.

Imagine you have your secrets in GCP and you want to back them up in Azure Key Vault. You would then create a `SecretStore` for each provider, and an `ExternalSecret` to pull the secrets from GCP. This will generetae `kind=Secret` in your cluster that you can use as the source of a `PushSecret` configured with the Azure `SecretStore`. 

![PushSecretBackup](../pictures/diagrams-pushsecret-backup.png)