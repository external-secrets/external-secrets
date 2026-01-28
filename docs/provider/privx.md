# PrivX

External Secrets Operator integrates with SSH PrivX for secret management.
See [PrivX](https://www.ssh.com/products/privileged-access-management-privx)

This provider uses the PrivX Vault API to read and write secrets. See [PrivX API](https://privx.docs.ssh.com/v42/api)

Secrets are stored as objects within PrivX Vault.

## Usage example

First, create the connection credentials (this example uses OAuth)

```bash
kubectl create secret generic privx-secret \
--from-literal=privx_api_oauth_client_id='<SECRET-VALUE>' \
--from-literal=privx_api_oauth_client_secret='<SECRET-VALUE>' \
--from-literal=privx_api_client_id='<SECRET-VALUE>' \
--from-literal=privx_api_client_secret='<SECRET-VALUE>'
```

Now, create a SecretStore with a PrivX backend. Enter following with `kubectl apply -f`

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secretstore-privx
spec:
  provider:
    privx:
      host: <privx host url>
      auth:
        oauth:
          clientIDRef:
            name: privx-secret
            key: privx_api_oauth_client_id
          clientSecretRef:
            name: privx-secret
            key: privx_api_oauth_client_secret
          apiClientIDRef:
            name: privx-secret
            key: privx_api_client_id
          apiClientSecretRef:
            name: privx-secret
            key: privx_api_client_secret
```

And finally, create the external secret definition

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: privx-test
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: secretstore-privx
    kind: SecretStore
  target:
    name: privx-test-secret
    creationPolicy: Owner
  data:
    - secretKey: test_value
      remoteRef:
        key: <name of secret in PrivX>
```

The secret from PrivX is now available in Kubernetes secret `privx-test-secret`, with key `test_value`.
Note that the *OAuth user* must have a *role* in PrivX that is listed in the *readers of the secret*.

### Fetching Multiple Secrets

The PrivX provider supports dataFrom.find.

Find by Name (RegExp)
dataFrom:
- find:
    name:
      regexp: "app-.*"

Returns all secrets whose name matches the regular expression.


# Authentication

## OAuth Authentication



# PushSecret

PrivX supports PushSecret to write Kubernetes Secret values into PrivX.

apiVersion: external-secrets.io/v1
kind: PushSecret
metadata:
  name: push-to-privx
spec:
  secretStoreRefs:
  - name: privx-backend
  selector:
    secret:
      name: source-secret
  data:
  - match:
      secretKey: password
      remoteRef:
        remoteKey: my-app-secret
        property: password

## Requirements

