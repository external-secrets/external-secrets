# Getting started

External-secrets runs within your Kubernetes cluster as a deployment resource.
It utilizes CustomResourceDefinitions to configure access to secret providers through SecretStore resources
and manages Kubernetes secret resources with ExternalSecret resources.

> Note: The minimum supported version of Kubernetes is `1.16.0`. Users still running Kubernetes v1.15 or below should upgrade
> to a supported version before installing external-secrets.

## Installing with Helm

To automatically install and manage the CRDs as part of your Helm release, you must add the --set installCRDs=true flag to your Helm installation command.

Uncomment the relevant line in the next steps to enable this.

### Option 1: Install from chart repository

``` bash
helm repo add external-secrets https://charts.external-secrets.io

helm install external-secrets \
   external-secrets/external-secrets \
    -n external-secrets \
    --create-namespace \
  # --set installCRDs=true
```

### Option 2: Install chart from local build

Build and install the Helm chart locally after cloning the repository.

``` bash
make helm.build

helm install external-secrets \
    ./bin/chart/external-secrets.tgz \
    -n external-secrets \
    --create-namespace \
  # --set installCRDs=true
```


### Create your first SecretStore

``` yaml
{% include 'basic-secret-store.yaml' %}
```

### Create your first ExternalSecret

``` yaml
{% include 'basic-external-secret.yaml' %}
```

``` bash
kubectl describe externalsecret example
# [...]
Name:  example
Status:
  Conditions:
    Last Transition Time:  2021-02-24T16:45:23Z
    Message:               Secret was synced
    Reason:                SecretSynced
    Status:                True
    Type:                  Ready
  Refresh Time:            2021-02-24T16:45:24Z
Events:                    <none>
```

For more advanced examples, please read the other
[guides](guides-introduction.md).

## Uninstalling

Before continuing, ensure that all external-secret resources that have been created by users have been deleted.
You can check for any existing resources with the following command:

```bash
kubectl get SecretStores,ClusterSecretStores,ExternalSecrets --all-namespaces
```

Once all these resources have been deleted you are ready to uninstall external-secrets.

### Uninstalling with Helm

Uninstall the helm release using the delete command.

```bash
helm delete external-secrets --namespace external-secrets
```
