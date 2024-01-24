## Chef

`Chef External Secrets provider` will enable users to seamlessly integrate their Chef-based secret management with Kubernetes through the existing External Secrets framework.

In many enterprises, legacy applications and infrastructure are still tightly integrated with Chef/Chef Infra Server/Chef Server Cluster for configuration and secrets management. Teams often rely on [Chef data bags](https://docs.chef.io/data_bags/) to securely store sensitive information such as application secrets and infrastructure configurations. These data bags serve as a centralized repository for managing and distributing sensitive data across the Chef ecosystem.

**NOTE:** `Chef External Secrets provider` is designed only to fetch data form the Chef data bags into Kubernetes secrets, it won't update/delete any item in the data bags. 

### Authentication

Every request made to the Chef Infra server needs to be authenticated. [Authentication](https://docs.chef.io/server/auth/) is done using the Private keys of the Chef Users.  The User needs to have appropriate [Permissions](https://docs.chef.io/server/server_orgs/#permissions) to the data bags containing the data that they want to fetch using the External Secrets Operator.

The following command can be used to create Chef Users:
```sh
chef-server-ctl user-create USER_NAME FIRST_NAME [MIDDLE_NAME] LAST_NAME EMAIL 'PASSWORD' (options)
```

More details on the above command are available here https://docs.chef.io/server/server_users/#user-create. The above command will return the default private key (PRIVATE_KEY_VALUE), which we will use for authentication. Additionally, a Chef User with access to specific data bags, private key pair with expiration date can be created with the help of [knife user key](https://docs.chef.io/server/auth/#knife-user-key) command.

### Create a secret containing your private key

We need to store above User's apikey into a secret resource.
Example:
```sh
kubectl create secret generic chef-user-secret -n vivid --from-literal=user-private-key='PRIVATE_KEY_VALUE'
```

### Createing ClusterSecretStore

The Chef `ClusterSecretStore` is a cluster scoped SecretStore that can be referenced by all Chef `ExternalSecrets` from all namespaces. You can follow below example to create `ClusterSecretStore` resource.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: vivid-clustersecretstore # name of ClusterSecretStore
spec:
  provider:
    chef:
      username: user # Chef User name
      serverurl: https://manage.chef.io/organizations/testuser/ # Chef server URL
      auth:
        secretRef:
          privateKeySecretRef:
            key: user-private-key # name of the key inside Secret resource
            name: chef-user-secret # name of Kubernetes Secret resource containing the Chef User's private key
            namespace: vivid # the namespace in which above Secret resource resides
```


### Createing ExternalSecret

The Chef `ExternalSecret` describes what data should be fetched from Chef Data bags, how the data should be transformed and saved as a Kind=Secret.

You can follow below example to create `ExternalSecret` resource.
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
  data:
  - secretKey: USERNAME
    remoteRef:
      key: vivid_prod/global_user # databagName/dataItemName
      property: username # a json key in dataItem
  - secretKey: PASSWORD
    remoteRef:
      key: vivid_prod/global_user
      property: password
  - secretKey: APIKEY
    remoteRef:
      key: vivid_global/apikey
      property: api_key
- secretKey: APP_PROPERTIES
    remoteRef:
      key: vivid_global/app_properties # databagName/dataItemName , it will fetch all key-vlaues present in the dataItem
  target:
    name: vivid-credentials # name of kubernetes Secret resource that will be created and will contain the obtained secrets
    creationPolicy: Owner
    template:
      mergePolicy: Replace    
      engineVersion: v2
      data:
        secrets.json: |
          {
            "username": "{{.USERNAME}}",
            "password": "{{.PASSWORD}}",
            "app_apikey": "{{.APIKEY}}"
          }

```

When above `ClusterSecretStore` and `ExternalSecret` resources are created, the `ExternalSecret` will connect to the Chef Server using the private key and will fetch the data bags contains into `vivid-credentials` secret resource.


follow : apis/externalsecrets/v1beta1/secretstore_chef_types.go for more info