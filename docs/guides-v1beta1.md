# Upgrading CRD versions

From version v0.5.0, `v1alpha1` version is deprecated, and `v1beta1` is in place. This guide will cover the main differences between the two versions, and a procedure on how to safely upgrade it.

## Differences between versions
Versions v1alpha1 and v1beta1 are fully-compatible for SecretStores and ClusterSecretStores. For ExternalSecrets, there is a difference on the `dataFrom` method.

While in v1alpha1, we could define a `dataFrom` with the following format:

```
spec:
  dataFrom:
    - key: my-key
    - key: my-other-key
```

In v1beta1 is possible to use two methods. One of them is `Extract` and has the exact same behavior as `dataFrom` in v1alpha1. The other is `Find`, which allows finding multiple external secrets and map them into a single Kubernetes secret. Here is an example of `Find`:

```
spec:
  dataFrom:
    - find:
        name:  #matches any secret name ending in foo-bar
          regexp: .*foo-bar$
    - find:
        tags: #matches any secrets with the following metadata.
            env: dev  
            app: web
```

## Upgrading

If you already have an installation of ESO using `v1alpha1`, we recommend you to upgrade to `v1beta1`. If you do not use `dataFrom` in your ExternalSecrets, or if you deploy the CRDs using the official Helm charts, the upgrade can be done with no risk of losing data. 

If you are installing CRDs manually, you will need to deploy the bundle CRD file available at `deploys/crds/bundle.yaml`. This bundle file contains `v1beta1` definition and a conversion webhook configuration. This configuration will ensure that new requests to handle any CRD object will only be valid after the upgrade is successfully complete - so there are no risks of losing data due to an incomplete upgrade. Once the new CRDs are applied, you can proceed to upgrade the controller version.

Once the upgrade is finished, at each reconcile, any `ExternalSecret`, `SecretStore`,  and `ClusterSecretStore` stored in `v1alpha1` will be automatically converted to `v1beta1`. 