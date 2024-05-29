# Getting started

External-secrets runs within your Kubernetes cluster as a deployment resource.
It utilizes CustomResourceDefinitions to configure access to secret providers through SecretStore resources
and manages Kubernetes secret resources with ExternalSecret resources.

> Note: The minimum supported version of Kubernetes is `1.16.0`. Users still running Kubernetes v1.15 or below should upgrade
> to a supported version before installing external-secrets.

## Installing with Helm

The default install options will automatically install and manage the CRDs as part of your helm release. If you do not want the CRDs to be automatically upgraded and managed, you must set the `installCRDs` option to `false`. (e.g. `--set installCRDs=false`)

You can install those CRDs outside of `helm` using:
```bash
kubectl apply -k "https://raw.githubusercontent.com/external-secrets/external-secrets/<replace_with_your_version>/deploy/crds/bundle.yaml"
```

Uncomment the relevant line in the next steps to disable the automatic install of CRDs.

### Option 1: Install from chart repository

```bash
helm repo add external-secrets https://charts.external-secrets.io

helm install external-secrets \
   external-secrets/external-secrets \
    -n external-secrets \
    --create-namespace \
  # --set installCRDs=false
```

### Option 2: Install chart from local build

Build and install the Helm chart locally after cloning the repository.

```bash
make helm.build

helm install external-secrets \
    ./bin/chart/external-secrets.tgz \
    -n external-secrets \
    --create-namespace \
  # --set installCRDs=false
```

### Create a secret containing your AWS credentials

```shell
echo -n 'KEYID' > ./access-key
echo -n 'SECRETKEY' > ./secret-access-key
kubectl create secret generic awssm-secret --from-file=./access-key --from-file=./secret-access-key
```

### Create your first SecretStore

Create a file 'basic-secret-store.yaml' with the following content.

```yaml
{% include 'basic-secret-store.yaml' %}
```

Apply it to create a SecretStore resource.

```
kubectl apply -f "basic-secret-store.yaml"
```

### Create your first ExternalSecret

Create a file 'basic-external-secret.yaml' with the following content.

```yaml
{% include 'basic-external-secret.yaml' %}
```

Apply it to create an External Secret resource.

```
kubectl apply -f "basic-external-secret.yaml"
```

```bash
kubectl describe externalsecret example
# [...]
Name:  example
Status:
  Binding:
    Name:                  secret-to-be-created
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
[guides](../guides/introduction.md).

## Installing with OLM

External-secrets can be managed by [Operator Lifecycle Manager](https://olm.operatorframework.io/) (OLM) via an installer operator. It is made available through [OperatorHub.io](https://operatorhub.io/), this installation method is suited best for OpenShift. See installation instructions on the [external-secrets-operator](https://operatorhub.io/operator/external-secrets-operator) package.

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
