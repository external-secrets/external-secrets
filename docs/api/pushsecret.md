![PushSecret](../pictures/diagrams-pushsecret-basic.png)

The `PushSecret` is namespaced and it describes what data should be pushed to the secret provider.

* tells the operator what secrets should be pushed by using `spec.selector`.
* you can specify what secret keys should be pushed by using `spec.data`

``` yaml
{% include 'full-pushsecret.yaml' %}
```
