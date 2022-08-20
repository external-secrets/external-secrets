
Generators allow you to generate values. They are used through a ExternalSecret `spec.DataFrom`. They can be defined inline using a `sourceRef.generator` or referenced from a custom resource using `sourceRef.generatorRef`.

If the External Secret should be refreshed via `spec.refreshInterval` the generator produces a map of values with the `generator.spec` as input. The generator does not keep track of the produced values. Every invocation produces a new set of values.

These values can be used with the other features like `rewrite` or `template`. I.e. you can modify, encode, decode, pack the values as needed.



## Inline Definition

Generators can be defined inline in a ExternalSecret. **Every invocation creates a new set of values**. I.e. you can not share the same value produced by a generator across different `spec.dataFrom[]` entries.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: "ecr-token"
spec:
  refreshInterval: "30m"
  target:
    name: ecr-token
  dataFrom:
  - sourceRef:
      generator:
        apiVersion: generators.external-secrets.io/v1alpha1
        kind: ECRAuthorizationToken
        spec:
          region: eu-west-1
          auth:
            jwt:
              serviceAccountRef:
                name: "oci-token-sync"
```


## Reference Custom Resource

Generators can be defined as a custom resource and re-used across different ExternalSecrets. **Every invocation creates a new set of values**. I.e. you can not share the same value produced by a generator across different `ExternalSecrets` or `spec.dataFrom[]` entries.

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: "ecr-token"
spec:
  refreshInterval: "30m"
  target:
    name: ecr-token
  dataFrom:
  - sourceRef:
      generatorRef:
        apiVersion: generators.external-secrets.io/v1alpha1
        kind: ECRAuthorizationToken
        name: "my-ecr"
```