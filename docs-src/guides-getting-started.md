# Getting started with Services APIs

## Installing CRDs

This project provides a collection of Custom Resource Definitions (CRDs) that can
be installed into any Kubernetes (>= 1.16) cluster.

To install the CRDs, please execute:

``` console
kubectl kustomize "github.com/external-secrets/external-secrets/config/crd" \
| kubectl apply -f -
```

## Install the controller

### Create your first SecretStore

``` yaml
{% include 'basic-secret-store.yaml' %}
```

### Create your first ExternalSecret

``` yaml
{% include 'basic-external-secret.yaml' %}
```

``` console
kubectl describe externalsecret example

# TODO
```

For more advanced examples, please read the other [guides](guides-getting-started.md).

## Uninstalling the CRDs

To uninstall the CRDs and all resources created with them, run the following
command. Note that this will remove all ExternalSecrets and SecretStore resources in
your cluster. If you have been using these resources for any other purpose do
not uninstall these CRDs.

```
kubectl kustomize "github.com/external-secrets/external-secrets/config/crd" \
| kubectl delete -f -
```
