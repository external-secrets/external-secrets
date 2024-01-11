## Chef

`Chef External Secrets provider` will enable users to seamlessly integrate their Chef-based secret management with Kubernetes through the existing External Secrets framework.

In many enterprises, legacy applications and infrastructure are still tightly integrated with Chef/Chef Infra Server/Chef Server Cluster for configuration and secrets management. Teams often rely on [Chef data bags](https://docs.chef.io/data_bags/) to securely store sensitive information such as application secrets and infrastructure configurations. These data bags serve as a centralized repository for managing and distributing sensitive data across the Chef ecosystem.

### Authentication

Every request made to the Chef Infra server needs to be authenticated. [Authentication](https://docs.chef.io/server/auth/) is done using the Private keys of the Chef Users.  The User needs to have appropriate [Permissions](https://docs.chef.io/server/server_orgs/#permissions) to the data bags containing the data that they want to fetch using the External Secrets Operator.

The following command can be used to create Chef Users:
```sh
chef-server-ctl user-create USER_NAME FIRST_NAME [MIDDLE_NAME] LAST_NAME EMAIL 'PASSWORD' (options)
```

More details on the above command are available here https://docs.chef.io/server/server_users/#user-create. The above command will return the default private key, which we will use for authentication. Additionally, a Chef User can create more private keys with the help of [knife user key](https://docs.chef.io/server/auth/#knife-user-key) command.

#### Create a secret containing your private key

```sh
kubectl create secret generic my-chef-infra-secret --from-literal=my-private-key='PRIVATE_KEY_VALUE'
```
