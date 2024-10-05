```yaml
---
title: Generator State
version: v1alpha1
authors: Moritz Johner
creation-date: 2024-10-05
status: draft
---
```

# Generator State

## Problem Description

Generators always have been stateless to avoid complexity. It has brought us a lot of limitations, like lack of support for generating more complex secrets for e.g. GCP Service Accounts, AWS IAM users, Grafana Cloud Service Accounts or Azure Service Principals. 
Having the ability to store state created by a generator within ESO/Kubernetes would allow us to manage user or system accounts for databases systems, message brokers or managed service providers.

This will not only help us to clean up secrets previously created by a generator, 
it will also significantly help with the use-case of rotating secrets.


## Proposed Solution

Let's assume we want to implement a generator for Grafana Service Accounts. The workflow is as follows ([see docs](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#create-service-account)):

1. Create a Service Account with a name and role.
```
POST /api/serviceaccounts
{
  "name": "test",
  "role": "Editor",
}

--- response
{
	"id": 42,
	"name": "test",
	"login": "sa-test",
	"role": "Editor",
	// .. omitted for previty
}
```

2. Create Token with name
```
POST /api/serviceaccounts/42/tokens
{
	"name": "eso-gen" # token name
}
--- response
{
	"id": 7,
	"name": "eso-gen",
	"key": "eyJrIjoXXXXXXXXX=="
}

```

We should not create hundreds of thousands of tokens every time we want to rotate a secret. 
Instead, we want to be a good citizen and delete old service account tokens after a reasonable amount of time.

For the sake of this example we have to store the following things:

1. Service Account ID (`42` from above)
2. Service Account Token ID (`7` from above)
3. everything from the generator spec (grafana URL, organization ID, service account name + role)

This state is stored on the custom resource status field.
Depending on the custom resource type `Kind=ExternalSecret` or `Kind=PushSecret` we have different schemas, because generators can be referenced only once (`Kind=PushSecret`) or multiple times (`Kind=ExternalSecret`).

I propose to simply let the `Generate()` function to return the state and let the CR controller decide on how to store and handle that.

In addition, we need a `Cleanup()` function which simply takes the state and everything else that is needed to cleanup the generated secret.

```go
type GeneratorState *apiextensions.JSON

type Generator interface {
	// Generate creates a new secret or set of secrets.
	// The returned map is a mapping of secret names to their respective values.
	// The status is an optional field that can be used to store any generator-specific
	// state which can be used during the Cleanup phase.
	Generate(
		ctx context.Context,
		generatorResource *apiextensions.JSON,
		k8sClient client.Client,
		namespace string,
	) (map[string][]byte, GeneratorState, error)

	// Cleanup deletes any resources created during the Generate phase.
	// Cleanup is idempotent and should not return an error if the resources
	// have already been deleted.
	Cleanup(
		ctx context.Context,
		generatorResource *apiextensions.JSON,
		status GeneratorState,
		k8sClient client.Client,
		namespace string,
	) error
}
```

As for the `Cleanup()` we need to take a close look at the `Kind=PushSecret` implementation. 
We can notice that we need to deal with the following cases to properly clean up the generated secret:

###### 1. The `Kind=PushSecret` resource re-generates a secret due to `spec.refreshInterval` or manual reconciliation

When the secret is being rotated, then we should not immediately call `Cleanup()`, because it will take some time until the new secret is available to the consumer. If we, e.g. use the `kubernetes` provider to push a secret into a different namespace or cluster, then [it will take 60-90 seconds until it is propagated](https://ahmet.im/blog/kubernetes-secret-volumes-delay/) when consumed as a volumeMount. When used as a environment variable, then it even needs a pod restart (e.g. using [stakater/Reloader](https://github.com/stakater/Reloader)).

With that being said, we should not immediately cleanup the secret, but instead flag it as *to-be-deleted* and wait a `grace period` before we finally delete it.


###### 2. A user deletes the `Kind=PushSecret` resource

In contrast to 👆, we should follow `PushSecret.spec.deletionPolicy` to either orphan the generated secret (`deletionPolicy=None`) or immediately delete it when using `deletionPolicy=Delete`. A finalizer blocks the deletion until the secret has been removed from the target store **and** the generated secret (if any) has been cleaned up. Since we delete it in the target store we should be fine to immediately delete it.


###### 3. A user changes `spec.selector.generatorRef` to `spec.selector.secret` or vice versa

In this case we follow the same procedure as described in `1.`: wait for grace period and then delete the secret. 

When jumping from `selector.generatorRef` to `selector.secret` back and forth multiple times, **the generator implementation must ensure that the generated state is unique for every invocation**, otherwise we may run into a race condition and accidentally delete a newly generated secret if the timings align.
In our grafana example above, the Service Account Token ID (`7` from above) is monotonically incrementing with every invocation, hence this is not a problem here.
Alternatively, we can consider to embed a `UUID` for every `.Generate()` invocation.

### Storing Generator State

We'll store the state in a dedicated custom resource `GeneratorState`.
We need to store the full generator CR, because it contains the all the configuration
needed to create/cleanup the secret. Because the generator spec can change at any time, 
we can not rely on it to be available later on in the process.

If someone decides to, e.g. `kubectl delete -f ./all-the-things.yaml`, then this will likely cause orphaned data.
This will result in support issues and maintainer fatigue.

Overview over the state management process:

- when a secret is **generated initially**, the generator resource and the returned state will be stored in a `GeneratorState` resource. We identify the "latest" state by it's creationTimestamp. We can  map the GeneratorState to the resource it's originating from by using a label on the GeneratorState which stores a key which points to it.
- when a secret is **rotated**, a controller sets the `GeneratorState.spec.garbageCollectionDeadline`. That will eventually trigger the deletion of the generator state after a configurable grace period. 
- A separate GeneratorState controller will ensure that the resource gets eventually deleted after the gc deadline has passed.

In the first iteration we can go with a global grace period, which can be configured by the user with a flag `--generator-gc-grace-period`. In the future we can consider adding it to the generator spec, or embedding it in the state returned by the generator.

```yaml
Kind: GeneratorState
metadata:
  labels:
    generators.external-secrets.io/owner-key: "externalsecret-foobar-0"
  finalizers:
    - generatorstate.externalsecrets.io/finalizer
spec: 
  garbageCollectionDeadline: "2024-12-10T22:30:05Z"
  resource: { }
  state: { }
```
