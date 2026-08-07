# GitOps using FluxCD (v2)

FluxCD is a GitOps operator for Kubernetes. It synchronizes the status of the cluster from manifests allocated in
different repositories (Git or Helm). This approach fits perfectly with External Secrets on clusters which are dynamically
created, to get credentials with no manual intervention from the beginning.

## Advantages

This approach has several advantages as follows:

* **Homogenize environments** allowing developers to use the same toolset in Kind in the same way they do in the cloud
  provider distributions such as EKS or GKE. This accelerates the development
* **Reduce security risks**, because credentials can be easily obtained, so temptation to store them locally is reduced.
* **Application compatibility increase**: Applications are deployed in different ways, and sometimes they need to share
  credentials. This can be done using External Secrets as a wire for them at real time.
* **Automation by default** oh, come on!

## The approach

FluxCD is composed by several controllers dedicated to manage different custom resources. The most important
ones are **Kustomization** (to clarify, Flux one, not Kubernetes' one) and **HelmRelease** to deploy using the approaches
of the same names.

External Secrets can be deployed using Helm [as explained here](../introduction/getting-started.md). The deployment includes the
CRDs if enabled on the `values.yaml`, but after this, you need to deploy some `SecretStore` to start
getting credentials from your secrets manager with External Secrets.

> The idea of this guide is to deploy the whole stack, using flux, needed by developers not to worry about the credentials,
> but only about the application and its code.

## The problem

This can sound easy, but External Secrets is deployed using Helm, which is managed by the HelmController,
and your custom resources, for example a `ClusterSecretStore` and the related `Secret`, are often deployed using a
`kustomization.yaml`, which is deployed by the KustomizeController.

Both controllers manage the resources independently, at different moments, with no possibility to wait each other.
This means that we have a wonderful race condition where sometimes the CRs (`SecretStore`,`ClusterSecretStore`...) tries
to be deployed before than the CRDs needed to recognize them.

A second, subtler race exists around the **admission webhook**. External Secrets ships a `ValidatingWebhookConfiguration`
that is registered in the API server as soon as the HelmRelease is applied, before the webhook pod has had time to start
serving. If a Kustomization tries to apply an `ExternalSecret` or `ClusterSecretStore` in that brief window, the API
server performs a dry-run validation, the webhook endpoint returns `connection refused`, and the reconciliation fails with:

```
Internal error occurred: failed calling webhook "validate.externalsecret.external-secrets.io": ...
dial tcp <ip>:443: connect: connection refused
```

Flux retries after `interval` (typically 10 minutes), so everything works on the second attempt, but the initial
deployment always fails. The fix is a three-level dependency chain that ensures the webhook pod is healthy before
any CR is applied.

## The solution

Let's see the conditions to start working on a solution:

* The External Secrets operator is deployed with Helm, and admits disabling the CRDs deployment
* The race condition only affects the deployment of `CustomResourceDefinition` and the CRs needed later
* CRDs can be deployed directly from the Git repository of the project using a Flux `Kustomization`
* The operator HelmRelease is wrapped in its own Flux `Kustomization` with `wait: true` so it only
  reports Ready after all deployed resources (including the webhook pod) are healthy
* Required CRs can be deployed using a Flux `Kustomization` that depends on the operator Kustomization,
  not just the CRDs, guaranteeing the webhook is serving before any CR dry-run is attempted
* All previous manifests can be applied with a Kubernetes `kustomization`

The dependency chain is:

```
external-secrets-crds --> external-secrets-operator (wait: true) --> external-secrets-crs
```

## Create the main kustomization

To have a better view of things needed later, the first manifest to be created is the `kustomization.yaml`

```yaml
{% include 'gitops/kustomization.yaml' %}
```

## Create the secret

To access your secret manager, External Secrets needs some credentials. They are stored inside a Secret, which is intended
to be deployed by automation as a good practise. This time, a placeholder called `secret-token.yaml` is show as an example:

```yaml
# The namespace.yaml first
{% include 'gitops/namespace.yaml' %}
```

```yaml
{% include 'gitops/secret-token.yaml' %}
```

## Creating the references to repositories

Create a manifest called `repositories.yaml` to store the references to external repositories for Flux

```yaml
{% include 'gitops/repositories.yaml' %}
```

## Deploy the CRDs

As mentioned, CRDs can be deployed using the official Helm package, but to solve the race condition, they will be deployed
from our git repository using a Kustomization manifest called `deployment-crds.yaml` as follows:

```yaml
{% include 'gitops/deployment-crds.yaml' %}
```

Note the `wait: true` field. This ensures the Kustomization only reports Ready after all CRDs have been fully
established in the API server, so the operator can register its validation webhooks and controllers cleanly.

## Deploy the operator

The operator HelmRelease is placed inside an `operator/` subdirectory and wrapped with its own Flux Kustomization.
Create a manifest called `deployment-operator.yaml`:

```yaml
{% include 'gitops/deployment-operator.yaml' %}
```

The `wait: true` here is the key to solving the webhook race condition: Flux will not mark
`external-secrets-operator` as Ready until every resource the HelmRelease created -- including the webhook
Deployment -- has reached a healthy state. Only then will the CRs Kustomization be allowed to proceed.

Inside the `operator/` subdirectory, place the HelmRelease manifest (`operator/deployment.yaml`):

```yaml
{% include 'gitops/operator/deployment.yaml' %}
```

## Deploy the CRs

Now, be ready for the arcane magic. Create a Kustomization manifest called `deployment-crs.yaml` with the following content:

```yaml
{% include 'gitops/deployment-crs.yaml' %}
```

There are several interesting details to see here, that finally solves the race condition:

1. The `dependsOn` field now points to `external-secrets-operator` rather than `external-secrets-crds`. This
   dependency forces this deployment to wait for the operator (including its webhook) to be fully ready before
   any CR is applied, eliminating the webhook race condition.
2. `retryInterval: 1m` makes Flux retry quickly if the very first reconcile still catches a brief startup window.
3. The reference to the place where to find the CRs
   ```yaml
   path: ./infrastructure/external-secrets/crs
   sourceRef:
    kind: GitRepository
    name: flux-system
   ```
   Custom Resources will be searched in the relative path `./infrastructure/external-secrets/crs` of the GitRepository
   called `flux-system`, which is a reference to the same repository that FluxCD watches to synchronize the cluster.
   With fewer words, a reference to itself, but going to another directory called `crs`

Of course, allocate inside the mentioned path `./infrastructure/external-secrets/crs`, all the desired CRs to be deployed,
for example, a manifest `clusterSecretStore.yaml` to reach your Hashicorp Vault as follows:

```yaml
{% include 'gitops/crs/clusterSecretStore.yaml' %}
```

## Results

At the end, the required files tree is:

```
./infrastructure/external-secrets/
  kustomization.yaml
  namespace.yaml
  secret-token.yaml
  repositories.yaml
  deployment-crds.yaml
  deployment-operator.yaml
  operator/
    kustomization.yaml
    deployment.yaml
  deployment-crs.yaml
  crs/
    kustomization.yaml
    clusterSecretStore.yaml
```
