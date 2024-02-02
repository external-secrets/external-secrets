## Chef

`Chef External Secrets provider` will enable users to seamlessly integrate their Chef-based secret management with Kubernetes through the existing External Secrets framework.

In many enterprises, legacy applications and infrastructure are still tightly integrated with the Chef/Chef Infra Server/Chef Server Cluster for configuration and secrets management. Teams often rely on [Chef data bags](https://docs.chef.io/data_bags/) to securely store sensitive information such as application secrets and infrastructure configurations. These data bags serve as a centralized repository for managing and distributing sensitive data across the Chef ecosystem.

**NOTE:** `Chef External Secrets provider` is designed only to fetch data from the Chef data bags into Kubernetes secrets, it won't update/delete any item in the data bags. 

### Authentication

Every request made to the Chef Infra server needs to be authenticated. [Authentication](https://docs.chef.io/server/auth/) is done using the Private keys of the Chef Users.  The User needs to have appropriate [Permissions](https://docs.chef.io/server/server_orgs/#permissions) to the data bags containing the data that they want to fetch using the External Secrets Operator.

The following command can be used to create Chef Users:
```sh
chef-server-ctl user-create USER_NAME FIRST_NAME [MIDDLE_NAME] LAST_NAME EMAIL 'PASSWORD' (options)
```

More details on the above command are available here [Chef User Create Option](https://docs.chef.io/server/server_users/#user-create). The above command will return the default private key (PRIVATE_KEY_VALUE), which we will use for authentication. Additionally, a Chef User with access to specific data bags, a private key pair with an expiration date can be created with the help of the  [knife user key](https://docs.chef.io/server/auth/#knife-user-key) command.

### Create a secret containing your private key

We need to store the above User's API key into a secret resource.
Example:
```sh
kubectl create secret generic chef-user-secret -n vivid --from-literal=user-private-key='PRIVATE_KEY_VALUE'
```

### Creating ClusterSecretStore

The Chef `ClusterSecretStore` is a cluster-scoped SecretStore that can be referenced by all Chef `ExternalSecrets` from all namespaces. You can follow the below example to create a `ClusterSecretStore` resource.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: vivid-clustersecretstore # name of ClusterSecretStore
spec:
  provider:
    chef:
      username: user # Chef User name
      serverUrl: https://manage.chef.io/organizations/testuser/ # Chef server URL
      auth:
        secretRef:
          privateKeySecretRef:
            key: user-private-key # name of the key inside Secret resource
            name: chef-user-secret # name of Kubernetes Secret resource containing the Chef User's private key
            namespace: vivid # the namespace in which the above Secret resource resides
```

### Creating SecretStore

Chef `SecretStores` are bound to a namespace and can not reference resources across namespaces. For cross-namespace SecretStores, you must use Chef `ClusterSecretStores`.

You can follow the below example to create a `SecretStore` resource.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vivid-secretstore # name of SecretStore
  namespace: vivid # must be required for kind: SecretStore
spec:
  provider:
    chef:
      username: user # Chef User name
      serverUrl: https://manage.chef.io/organizations/testuser/ # Chef server URL
      auth:
        secretRef:
          privateKeySecretRef:
            name: chef-user-secret # name of Kubernetes Secret resource containing the Chef User's private key
            key: user-private-key # name of the key inside Secret resource
            namespace: vivid # the ns where the k8s secret resource containing Chef User's private key resides

```

### Creating ExternalSecret

The Chef `ExternalSecret` describes what data should be fetched from Chef Data bags, and how the data should be transformed and saved as a Kind=Secret.

You can follow the below example to create an `ExternalSecret` resource.
```yaml
{% include 'chef-external-secret.yaml' %}
```

When the above `ClusterSecretStore` and `ExternalSecret` resources are created, the `ExternalSecret` will connect to the Chef Server using the private key and will fetch the data bags contained in the `vivid-credentials` secret resource.

To get all data items inside the data bag, you can use the `dataFrom` directive:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: vivid-external-secrets # name of ExternalSecret
  namespace: vivid # namespace inside which the ExternalSecret will be created
  annotations:
    company/contacts: user.a@company.com, user.b@company.com
    company/team: vivid-dev
  labels:
    app.kubernetes.io/name: external-secrets
spec:
  refreshInterval: 15m
  secretStoreRef:
    name: vivid-clustersecretstore # name of ClusterSecretStore
    kind: ClusterSecretStore
  dataFrom:
  - extract:
      key: vivid_global # only data bag name
  target:
    name: vivid_global_all_cred # name of Kubernetes Secret resource that will be created and will contain the obtained secrets
    creationPolicy: Owner

```

follow : [this file](https://github.com/external-secrets/external-secrets/blob/main/apis/externalsecrets/v1beta1/secretstore_chef_types.go) for more info
