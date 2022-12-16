External Secrets Operator allows to retrieve secrets from a Kubernetes Cluster - this can be either a remote cluster or the local one where the operator runs in.

A `SecretStore` points to a **specific namespace** in the target Kubernetes Cluster. You are able to retrieve all secrets from that particular namespace given you have the correct set of RBAC permissions.

The `SecretStore` reconciler checks if you have read access for secrets in that namespace using `SelfSubjectRulesReview`. See below on how to set that up properly.

### External Secret Spec

This provider supports the use of the `Property` field. With it you point to the key of the remote secret. If you leave it empty it will json encode all key/value pairs.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: SecretStore
    name: k8s-store             # name of the SecretStore (or kind specified)
  target:
    name: database-credentials  # name of the k8s Secret to be created
  data:
  - secretKey: username
    remoteRef:
      key: database-credentials
      property: username

  - secretKey: password
    remoteRef:
      key: database-credentials
      property: password
```

#### find by tag & name

You can fetch secrets based on labels or names matching a regexp:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: fetch-tls-and-nginx
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: SecretStore
    name: k8s-store
  target:
    name: fetch-tls-and-nginx
  dataFrom:
  - find:
      name:
        # match secret name with regexp
        regexp: "tls-.*"
  - find:
      tags:
        # fetch secrets based on label combination
        app: "nginx"
```

### Target API-Server Configuration

The servers `url` can be omitted and defaults to `kubernetes.default`. You **have to** provide a CA certificate in order to connect to the API Server securely.
For your convenience, each namespace has a ConfigMap `kube-root-ca.crt` that contains the CA certificate of the internal API Server (see `RootCAConfigMap` [feature gate](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/)).
Use that if you want to connect to the same API server.
If you want to connect to a remote API Server you need to fetch it and store it inside the cluster as ConfigMap or Secret.
You may also define it inline as base64 encoded value using the `caBundle` property.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: k8s-store-default-ns
spec:
  provider:
    kubernetes:
      # with this, the store is able to pull only from `default` namespace
      remoteNamespace: default
      server:
        url: "https://myapiserver.tld"
        caProvider:
          type: ConfigMap
          name: kube-root-ca.crt
          key: ca.crt
```

### Authentication

It's possible to authenticate against the Kubernetes API using client certificates, a bearer token or service account. The operator enforces that exactly one authentication method is used. You can not use the service account that is mounted inside the operator, this is by design to avoid reading secrets across namespaces.

**NOTE:** `SelfSubjectRulesReview` permission is required in order to validation work properly. Please use the following role as reference:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default
  name: eso-store-role
rules:
- apiGroups: [""]
  resources:
  - secrets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - authorization.k8s.io
  resources:
  - selfsubjectrulesreviews
  verbs:
  - create
```

#### Authenticating with BearerToken

Create a Kubernetes secret with a client token. There are many ways to acquire such a token, please refer to the [Kubernetes Authentication docs](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#authentication-strategies).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-token
data:
  token: "...."
```

Create a SecretStore: The `auth` section indicates that the type `token` will be used for authentication, it includes the path to fetch the token. Set `remoteNamespace` to the name of the namespace where your target secrets reside.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: k8s-store-token-auth
spec:
  provider:
    kubernetes:
      # with this, the store is able to pull only from `default` namespace
      remoteNamespace: default
      server:
        # ...
      auth:
        token:
          bearerToken:
            name: my-token
            key: token
```

#### Authenticating with ServiceAccount

Create a Kubernetes Service Account, please refer to the [Service Account Tokens Documentation](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#service-account-tokens) on how they work and how to create them.

```
$ kubectl create serviceaccount my-store
```

This Service Account needs permissions to read `Secret` and create `SelfSubjectRulesReview` resources. Please see the above role.

```
$ kubectl create rolebinding my-store --role=eso-store-role --serviceaccount=default:my-store
```

Create a SecretStore: the `auth` section indicates that the type `serviceAccount` will be used for authentication.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: k8s-store-sa-auth
spec:
  provider:
    kubernetes:
      # with this, the store is able to pull only from `default` namespace
      remoteNamespace: default
      server:
        # ...
      auth:
        serviceAccount:
          name: "my-store"
```

#### Authenticating with Client Certificates

Create a Kubernetes secret which contains the client key and certificate. See [Generate Certificates Documentations](https://kubernetes.io/docs/tasks/administer-cluster/certificates/) on how to create them.

```
$ kubectl create secret tls tls-secret --cert=path/to/tls.cert --key=path/to/tls.key
```

Reference the `tls-secret` in the SecretStore

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: k8s-store-cert-auth
spec:
  provider:
    kubernetes:
      # with this, the store is able to pull only from `default` namespace
      remoteNamespace: default
      server:
        # ...
      auth:
        cert:
          clientCert:
            name: "tls-secret"
            key: "tls.crt"
          clientKey:
            name: "tls-secret"
            key: "tls.key"
```
