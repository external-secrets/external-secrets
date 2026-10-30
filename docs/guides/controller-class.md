# Controller Classes

> NOTE: this feature is experimental and not highly tested

Controller classes are a property set during the deployment that allows multiple controllers to work in a group of workload. It works by separating which secretStores are going to be attributed to which controller. For the behavior of a single controller, no extra configuration is needed.

## Setting up Controller Class

In order to deploy the controller with a specific class, install the helm charts specifying the controller class, and create a `SecretStore` with the appropriate `spec.controller` values:
```
helm install custom-external-secrets external-secrets/external-secrets --set controllerClass=custom
```
``` yaml
{% include 'controller-class-store.yaml' %}
```

Now, any `ExternalSecret` bound to this secret store will be evaluated by the operator with the controllerClass custom. If an `ExternalSecret` references multiple stores (including overrides under `spec.data[].sourceRef` or `spec.dataFrom[].sourceRef`), a controller skips it when any referenced store belongs to a different controller class.

> Note: Any SecretStore without `spec.controller` set will be considered as valid by any operator, regardless of their respective controllerClasses. The same applies to ClusterSecretStores.

Skipped `ExternalSecrets` (not belonging to this controllerClass) are retried after the controller's requeue "fallbackUnmanagedRetryInterval" interval (one hour), even if their ExternalSecret refresh interval is zero.
This is done to make it possible to pick up changes from a referenced store or generator's class.
