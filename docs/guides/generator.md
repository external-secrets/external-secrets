Generators allow you to generate values. They are used through a ExternalSecret `spec.DataFrom`. They are referenced from a custom resource using `sourceRef.generatorRef`.

If the External Secret should be refreshed via `spec.refreshInterval` the generator produces a map of values with the `generator.spec` as input. The generator does not keep track of the produced values. Every invocation produces a new set of values.

These values can be used with the other features like `rewrite` or `template`. I.e. you can modify, encode, decode, pack the values as needed.

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

## Cluster Generate Resource

It's possible to use a `Cluster` scoped generator. At the moment of this writing, this Generator
will only help in locating the Generator cluster-wide. It doesn't mean that the generator can create resources in all
namespaces. It will still only create a resource in the given namespace where the referencing `ExternalSecret` lives.

To define a `ClusterGenerator` use the following config:

```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: ClusterGenerator
metadata:
  name: my-generator
spec:
  kind: Password
  generator:
    passwordSpec:
      length: 42
      digits: 5
      symbols: 5
      symbolCharacters: "-_$@"
      noUpper: false
      allowRepeat: true
```

All the generators are available as a ClusterGenerator spec. The `kind` field MUST match the kind of the Generator
exactly. The following Spec fields are available:

```go
type GeneratorSpec struct {
	ACRAccessTokenSpec        *ACRAccessTokenSpec        `json:"acrAccessTokenSpec,omitempty"`
	ECRAuthorizationTokenSpec *ECRAuthorizationTokenSpec `json:"ecrRAuthorizationTokenSpec,omitempty"`
	FakeSpec                  *FakeSpec                  `json:"fakeSpec,omitempty"`
	GCRAccessTokenSpec        *GCRAccessTokenSpec        `json:"gcrAccessTokenSpec,omitempty"`
	GithubAccessTokenSpec     *GithubAccessTokenSpec     `json:"githubAccessTokenSpec,omitempty"`
	PasswordSpec              *PasswordSpec              `json:"passwordSpec,omitempty"`
	STSSessionTokenSpec       *STSSessionTokenSpec       `json:"stsSessionTokenSpec,omitempty"`
	UUIDSpec                  *UUIDSpec                  `json:"uuidSpec,omitempty"`
	VaultDynamicSecretSpec    *VaultDynamicSecretSpec    `json:"vaultDynamicSecretSpec,omitempty"`
	WebhookSpec               *WebhookSpec               `json:"webhookSpec,omitempty"`
}
```
