External Secrets Operator allows to retrieve in-cluster secrets or from a remote Kubernetes Cluster.

### Authentication

It's possible to authenticate against the Kubernetes API using client certificates or a bearer token. Authentication using a service account has not yet been implemented. The operator enforces that exactly one authentication method is used.

**NOTE:** `SelfSubjectAccessReview` permission is required for the service account in order to validation work properly.

## Example

### In-cluster secrets using a Token

1. Create a K8s Secret with a client token for the default service account

```
apiVersion: v1
kind: Secret
metadata:
  name: mydefaulttoken
  annotations:
    kubernetes.io/service-account.name: default
type: kubernetes.io/service-account-token
```
2. Create a SecretStore

The Servers `url` won't be present as it will default to `kubernetes.default`, add a proper value if needed. In this example the Certificate Authority is fetched using the referenced `caProvider`.

The `auth` section indicates that the type `token` will be used for authentication, it includes the path to fetch the token. Set `remoteNamespace` to the name of the namespace where your target secrets reside.

```
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: example
spec:
  provider:
    kubernetes:
      server: 
        caProvider: 
          type: Secret
          name: mydefaulttoken
          key: ca.crt
        auth:
          token:
            bearerToken: 
              name: mydefaulttoken
              key: token
        remoteNamespace: default
```
3. Create the local secret that will be synced 
              
```
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-example
data:
  extra: YmFyCg==
```     
4. Finally create the ExternalSecret resource

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

### Remote Secret using a Token

1. Create a K8s Secret with the encoded base64 ca and client token.
   
```
apiVersion: v1
kind: Secret
metadata:
  name: cluster-secrets
data:
  # Fill with your encoded base64 CA
  certificate-authority-data: Cg==
stringData:
  # Fill with your string Token
  bearerToken: "my-token"
```
2. Create a SecretStore

The Server section specifies the `url` of the remote Kubernetes API. In this example the Certificate Authority is fetch using the encoded base64 `caBundle`. 

The `auth` section indicates that the  `token` type will be used for authentication, it includes the path to fetch the token.

```
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: example
spec:
  provider:
      kubernetes: 
        # If not remoteNamesapce is provided, default     namespace is used
        remoteNamespace: remote-namespace
        server: 
          url: https://remote.kubernetes.api-server.address
          # Add your encoded base64 to caBundle
          caBundle: Cg==
        auth:
          # Adds referenced bearerToken
          token:
            bearerToken:
              name: cluster-secrets
              key: bearerToken
```     
4. Finally create the ExternalSecret resource

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
      key: secret-remote-example
      property: extra
```
