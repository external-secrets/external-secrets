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

Now, any `ExternalSecret` bound to this secret store will be evaluated by the operator with the controllerClass custom.

> Note: Any SecretStore without `spec.controller` set will be considered as valid by any operator, regardless of their respective controllerClasses.
