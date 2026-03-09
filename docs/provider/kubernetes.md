External Secrets Operator allows to retrieve secrets from a Kubernetes Cluster - this can be either a remote cluster or the local one where the operator runs in.

A `SecretStore` points to a **specific namespace** in the target Kubernetes Cluster. You are able to retrieve all secrets from that particular namespace given you have the correct set of RBAC permissions.

The `SecretStore` reconciler checks if you have read access for secrets in that namespace using `SelfSubjectRulesReview` and will fallback to `SelfSubjectAccessReview` when that fails. See below on how to set that up properly.

### External Secret Spec

This provider supports the use of the `Property` field. With it you point to the key of the remote secret. If you leave it empty it will json encode all key/value pairs.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: database-credentials
spec:
  refreshInterval: 1h0m0s
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

  # metadataPolicy to fetch all the labels and annotations in JSON format
  - secretKey: tags
    remoteRef:
      metadataPolicy: Fetch
      key: database-credentials

  # metadataPolicy to fetch all the labels in JSON format
  - secretKey: labels
    remoteRef:
      metadataPolicy: Fetch
      key: database-credentials
	  property: labels

  # metadataPolicy to fetch a specific label (dev) from the source secret
  - secretKey: developer
    remoteRef:
      metadataPolicy: Fetch
      key: database-credentials
	  property: labels.dev

```

#### find by tag & name

You can fetch secrets based on labels or names matching a regexp:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: fetch-tls-and-nginx
spec:
  refreshInterval: 1h0m0s
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

The servers `url` can be omitted and defaults to `kubernetes.default`. If no `caBundle` or `caProvider` is specified, the operator uses the system certificate roots from the container image. Both the default (`distroless/static`) and UBI images include standard CA certificates, so connections to servers using well-known CAs (e.g., Let's Encrypt) work without explicit CA configuration.
For your convenience, each namespace has a ConfigMap `kube-root-ca.crt` that contains the CA certificate of the internal API Server (see `RootCAConfigMap` [feature gate](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/)).
Use that if you want to connect to the same API server.
If you want to connect to a remote API Server you need to fetch it and store it inside the cluster as ConfigMap or Secret.
You may also define it inline as base64 encoded value using the `caBundle` property.

```yaml
apiVersion: external-secrets.io/v1
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

!!! note
    System CA roots only cover certificates signed by well-known CAs. Internal Kubernetes API servers typically use self-signed or cluster-internal CAs — you still need to provide explicit `caBundle` or `caProvider` for those.

If the remote server uses a certificate from a well-known CA, you can omit CA configuration entirely:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: k8s-store-system-ca
spec:
  provider:
    kubernetes:
      remoteNamespace: default
      server:
        url: "https://my-proxy.example.com"
        # No caBundle or caProvider — uses system CA roots
      auth:
        token:
          bearerToken:
            name: my-token
            key: token
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
apiVersion: external-secrets.io/v1
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
apiVersion: external-secrets.io/v1
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
apiVersion: external-secrets.io/v1
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


### Access from different namespace in same cluster

If you don't have cluster wide access to create a `ClusterExternalSecret`, you can still access a secret from a dedicated namespace via a bearer token to a service connection within that namespace:

```YAML
# shared-secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: user-credentials
  namespace: shared-secrets
type: Opaque
stringData:
  username: peter
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: eso-store-role
  namespace: shared-secrets
rules:
  - apiGroups: [""]
    resources:
      - secrets
    verbs:
      - get
      - list
      - watch
  # This will allow the role `eso-store-role` to perform **permission reviews** for itself within the defined namespace:
  - apiGroups:
      - authorization.k8s.io
    resources:
      - selfsubjectrulesreviews # used to review or fetch the list of permissions a user or service account currently has.
    verbs:
      - create # `create` allows creating a `selfsubjectrulesreviews` request.
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: eso-service-account
  namespace: shared-secrets
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: bind-eso-store-role-to-eso-service-account
  namespace: shared-secrets
subjects:
  - kind: ServiceAccount
    name: eso-service-account
    namespace: shared-secrets
roleRef:
  kind: Role
  name: eso-store-role
  apiGroup: rbac.authorization.k8s.io
```

After `kubectl apply -f shared-secrets.yaml`, create a bearer token for the service account with `kubectl create token eso-service-account`, then use that bearer token to access the `remoteNamespace` via secret in the target namespace:

```YAML
apiVersion: v1
kind: Secret
metadata:
  name: eso-token
  namespace: target-namespace
stringData:
  token: "<paste-bearer-token-here>"
---
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: kubernetes-secret-store
  namespace: target-namespace
spec:
  provider:
    kubernetes:
      remoteNamespace: shared-secrets
      server:
        # Skip url cause we are in the same cluster
        caProvider:
          type: ConfigMap
          name: kube-root-ca.crt
          key: ca.crt
      auth:
        token:
          bearerToken:
            name: eso-token
            key: token
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: eso-kubernetes-secret
  namespace: target-namespace
spec:
  secretStoreRef:
    kind: SecretStore
    name: kubernetes-secret-store
  target:
    name: eso-kubernetes-secret
  data:
    - secretKey: username
      remoteRef:
        key: user-credentials
        property: username
```

### PushSecret

The PushSecret functionality facilitates the replication of a Kubernetes Secret from one namespace or cluster to another. This feature proves useful in scenarios where you need to share sensitive information, such as credentials or configuration data, across different parts of your infrastructure.

To configure the PushSecret resource, you need to specify the following parameters:

* **Selector**: Specify the selector that identifies the source Secret to be replicated. This selector allows you to target the specific Secret you want to share.

* **SecretKey**: Set the SecretKey parameter to indicate the key within the source Secret that you want to replicate. This ensures that only the relevant information is shared.

* **RemoteRef.Property**: In addition to the above parameters, the Kubernetes provider requires you to set the `remoteRef.property` field. This field specifies the key of the remote Secret resource where the replicated value should be stored.


Here's an example:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: example
spec:
  refreshInterval: 1h0m0s
  secretStoreRefs:
    - name: k8s-store-remote-ns
      kind: SecretStore
  selector:
    secret:
      name: pokedex-credentials
  data:
    - match:
        secretKey: best-pokemon
        remoteRef:
          remoteKey: remote-best-pokemon
          property: best-pokemon
```

To use the PushSecret feature effectively, the referenced `SecretStore` requires specific permissions on the target cluster. In particular, it requires `create`, `read`, `update` and `delete` permissions on the Secret resource:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: remote
  name: eso-store-push-role
rules:
- apiGroups: [""]
  resources:
  - secrets
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
- apiGroups:
  - authorization.k8s.io
  resources:
  - selfsubjectrulesreviews
  verbs:
  - create
```

It is possible to override the target secret type with the `.template.type` property. By default the secret type is copied from the source secret. If none is specified, the default type `Opaque` will be used. The type can be set to any valid Kubernetes secret type, such as `kubernetes.io/dockerconfigjson`, `kubernetes.io/tls`, etc.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: example
spec:
  refreshInterval: 1h0m0s
  secretStoreRefs:
    - name: k8s-store-remote-ns
      kind: SecretStore
  selector:
    secret:
      name: pokedex-credentials
  template:
    type: kubernetes.io/dockerconfigjson
  data:
    - match:
        secretKey: dockerconfigjson
        remoteRef:
          remoteKey: remote-dockerconfigjson
          property: ".dockerconfigjson"
```

#### PushSecret Metadata

The Kubernetes provider is able to manage both `metadata.labels` and `metadata.annotations` of the secret on the target cluster.

Users have different preferences on what metadata should be pushed. ESO, by default, pushes both labels and annotations to the target secret and merges them with the existing metadata.

You can specify the metadata in the `spec.template.metadata` section if you want to decouple it from the existing secret.

```yaml
{% raw %}
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: example
spec:
  # ...
  template:
    metadata:
      labels:
        app.kubernetes.io/part-of: argocd
    data:
      mysql_connection_string: "mysql://{{ .hostname }}:3306/{{ .database }}"
  data:
  - match:
      secretKey: mysql_connection_string
      remoteRef:
        remoteKey: backend_secrets
        property: mysql_connection_string
{% endraw %}
```

Further, you can leverage the `.data[].metadata` section to fine-tine the behavior of the metadata merge strategy. The metadata section is a versioned custom-resource _similar_ structure, the behavior is detailed below.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: example
spec:
  # ...
  data:
  - match:
      secretKey: example-1
      remoteRef:
        remoteKey: example-remote-secret
        property: url

    metadata:
      apiVersion: kubernetes.external-secrets.io/v1alpha1
      kind: PushSecretMetadata
      spec:
        sourceMergePolicy: Merge # or Replace
        targetMergePolicy: Merge # or Replace / Ignore
        labels:
          color: red
        annotations:
          yes: please

```

| Field             | Type                                 | Description                                                                                                                                                                                                                                                                                                                                       |
|-------------------|--------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| sourceMergePolicy | string: `Merge`, `Replace`           | The sourceMergePolicy defines how the metadata of the source secret is merged. `Merge` will merge the metadata of the source secret with the  metadata defined in `.data[].metadata`. With `Replace`, the metadata in `.data[].metadata` replaces the source metadata.                                                                            |
| targetMergePolicy | string: `Merge`, `Replace`, `Ignore` | The targetMergePolicy defines how ESO merges the metadata produced by the sourceMergePolicy with the target secret. With `Merge`, the source metadata is merged with the existing metadata from the target secret. `Replace` will replace the target metadata with the metadata defined in the source. `Ignore` leaves the target metadata as is. |
| labels            | `map[string]string`                  | The labels.                                                                                                                                                                                                                                                                                                                                       |
| annotations       | `map[string]string`                  | The annotations.                                                                                                                                                                                                                                                                                                                                  |
| remoteNamespace   | string                               | The Namespace in which the remote Secret will created in if defined.                                                                                                                                                                                                                                                                              |

#### Implementation Considerations

When using the PushSecret feature and configuring the permissions for the SecretStore, consider the following:

* **RBAC Configuration**: Ensure that the Role-Based Access Control (RBAC) configuration for the SecretStore grants the appropriate permissions for creating, reading, and updating resources in the target cluster.

* **Least Privilege Principle**: Adhere to the principle of least privilege when assigning permissions to the SecretStore. Only provide the minimum required permissions to accomplish the desired synchronization between Secrets.

* **Namespace or Cluster Scope**: Depending on your specific requirements, configure the SecretStore to operate at the desired scope, whether it is limited to a specific namespace or encompasses the entire cluster. Consider the security and access control implications of your chosen scope.
