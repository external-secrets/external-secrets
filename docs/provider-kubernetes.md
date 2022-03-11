External Secrets Operator allows to retrieve in-cluster secrets or from a remote Kubernetes Cluster.

### Authentication

It's possible to authenticate against the Kubernetes API using client certificates, a bearer token or a service account (not implemented yet). The operator enforces that exactly one authentication method is used.

## Example

### SecretStore

The `Server` section specifies the url of the Kubernetes API and the location to fetch the CA. The `auth` section indicates the type of authentication to use, `cert`, `token` or `serviceAccount` and includes the path to fetch the certificates or the token.


```
apiVersion: external-secrets.io/v1alpha1
kind: SecretStore
metadata:
  name: example
spec:
  provider:
      kubernetes:  
        server: 
          url:  https://127.0.0.1:36473
          caProvider: 
            type: Secret
            name : kind-cluster-secrets
            key: ca
        auth:
          cert:
            clientCert: 
                name: kind-cluster-secrets
                key: certificate
            clientKey: 
                name: kind-cluster-secrets
                key: key
```
        
### ExternalSecret

```
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshInterval: 1h           
  secretStoreRef:
    kind: SecretStore
    name: example               # name of the SecretStore (or kind specified)
  target:
    name: secret-to-be-created  # name of the k8s Secret to be created
    creationPolicy: Owner
  data:
  - secretKey: extra #
    remoteRef:
      key: secret-example
      property: extra
```