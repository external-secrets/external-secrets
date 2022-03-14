External Secrets Operator allows to retrieve in-cluster secrets or from a remote Kubernetes Cluster.

### Authentication

It's possible to authenticate against the Kubernetes API using client certificates, a bearer token or a service account (not implemented yet). The operator enforces that exactly one authentication method is used.

## Example

### K8s Cluster Secret


```
apiVersion: v1
kind: Secret
metadata:
  name: cluster-secrets
data:
  # Fill with your encoded base64 CA
  ca: Cg==
  # Fill with your encoded base64 Certificate
  certificate: Cg==
  # Fill with your encoded base64 Key
  key: Cg==
stringData:
  # Fill with your a string Token
  bearerToken: "my-token"
```

## SecretStore

The `Server` section specifies the url of the Kubernetes API and the location to fetch the CA. The `auth` section indicates the type of authentication to use, `cert`, `token` or `serviceAccount` and includes the path to fetch the certificates or the token.

```
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: example
spec:
  provider:
      kubernetes:
        # If not remoteNamesapce is provided, default namespace is used
        remoteNamespace: default  
        server: 
          url:  https://127.0.0.1:36473
          # Add your encoded base64 to caBundle or a referenced caProvider
          # if both are provided caProvider will be ignored
          caBundle: Cg==
          caProvider: 
            type: Secret
            name : cluster-secrets
            key: ca
        auth:
          # Add a referenced bearerToken or client certificates, 
          # if both are provided client certificates will be ignored
          token:
            bearerToken:
              name: cluster-secrets
              key: bearerToken
          cert:
            clientCert: 
                name: cluster-secrets
                key: certificate
            clientKey: 
                name: cluster-secrets
                key: key
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-example
data:
  extra: YmFyCg==
```
        
### ExternalSecret

```
apiVersion: external-secrets.io/v1beta1
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
  - secretKey: extra
    remoteRef:
      key: secret-example
      property: extra
```