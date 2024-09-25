
![Previder Secret Vault](../pictures/previder-provider.png)

## Previder Secret Vault Manager

External Secrets Operator integrates with [Previder Secrets Vault](https://vault.previder.io) for secure secret management.

### Authentication

We support Access Token authentication using a Secrets Vault ReadWrite or ReadOnly token.

This token can be created with the [vault-cli](https://github.com/previder/vault-cli) using an Environment token which can be acquired via the [Previder Portal](https://portal.previder.nl).

#### Access Token authentication

To use the access token, first create it as a regular Kubernetes Secret and then associate it with the Previder Secret Store.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: previder-vault-sample-secret
data:
  previder-vault-token: cHJldmlkZXIgdmF1bHQgZXhhbXBsZQ==
```

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: previder-secretstore-sample
spec:
  provider:
    previder:
      auth:
        secretRef:
          accessToken:
            name: previder-vault-sample-secret
            key: previder-vault-token
```


### Creating external secret

To create a kubernetes secret from the Previder Secret Vault, create an ExternalSecret with a reference to a Vault secret.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: previder-secretstore-sample
    kind: SecretStore
  target:
    name: example-secret
    creationPolicy: Owner
  data:
    - secretKey: local-secret-key
      remoteRef:
        key: token-name-or-id
```
