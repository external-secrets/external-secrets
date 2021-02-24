# Getting started with Services APIs

## Installing CRDs

This project provides a collection of Custom Resource Definitions (CRDs) that can
be installed into any Kubernetes (>= 1.16) cluster.

To install the CRDs, please execute:

``` bash
kubectl kustomize "github.com/external-secrets/external-secrets/config/crd" \
| kubectl apply -f -
```

## Install the controller

``` bash
kubectl kustomize "github.com/external-secrets/external-secrets/config/default" \
| kubectl apply -f -
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

For more advanced examples, please read the other [guides](guides-introduction.md).

## Uninstalling the CRDs

To uninstall the CRDs and all resources created with them, run the following
command. Note that this will remove all ExternalSecrets and SecretStore resources in
your cluster. If you have been using these resources for any other purpose do
not uninstall these CRDs.

```
kubectl kustomize "github.com/external-secrets/external-secrets/config/crd" \
| kubectl delete -f -
```
