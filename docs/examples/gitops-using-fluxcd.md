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

## The solution

Let's see the conditions to start working on a solution:

* The External Secrets operator is deployed with Helm, and admits disabling the CRDs deployment
* The race condition only affects the deployment of `CustomResourceDefinition` and the CRs needed later
* CRDs can be deployed directly from the Git repository of the project using a Flux `Kustomization`
* Required CRs can be deployed using a Flux `Kustomization` too, allowing dependency between CRDs and CRs
* All previous manifests can be applied with a Kubernetes `kustomization`

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

## Deploy the operator

The operator is deployed using a HelmRelease manifest to deploy the Helm package, but due to the special race condition,
the deployment must be disabled in the `values` of the manifest called `deployment.yaml`, as follows:

```yaml
{% include 'gitops/deployment.yaml' %}
```

## Deploy the CRs

Now, be ready for the arcane magic. Create a Kustomization manifest called `deployment-crs.yaml` with the following content:

```yaml
{% include 'gitops/deployment-crs.yaml' %}
```

There are several interesting details to see here, that finally solves the race condition:

1. First one is the field `dependsOn`, which points to a previous Kustomization called `external-secrets-crds`. This
   dependency forces this deployment to wait for the other to be ready, before start being deployed.
2. The reference to the place where to find the CRs
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

At the end, the required files tree is shown in the following picture:

![FluxCD files tree](../pictures/screenshot_gitops_final_directory_tree.png)
