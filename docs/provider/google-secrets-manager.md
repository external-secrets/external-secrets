External Secrets Operator integrates with the [Google Cloud Secret Manager](https://cloud.google.com/secret-manager).

## Authentication

### Workload Identity Federation

Through [Workload Identity Federation](https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity) (WIF), [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine) (GKE) workloads can authenticate with Google Cloud Platform (GCP) services like Secret Manager without using static, long-lived credentials.

Authenticating through WIF is the recommended approach when using the External Secrets Operator (ESO) on GKE clusters. ESO supports three options:

- **Using a Kubernetes service account as a GCP IAM principal**: The `SecretStore` (or `ClusterSecretStore`) references a [Kubernetes service account](https://kubernetes.io/docs/concepts/security/service-accounts) that is authorized to access Secret Manager secrets.
- **Linking a Kubernetes service account to a GCP service account:** The `SecretStore` (or `ClusterSecretStore`) references a Kubernetes service account, which is linked to a [GCP service account](https://cloud.google.com/iam/docs/service-accounts) that is authorized to access Secret Manager secrets. This requires that the Kubernetes service account is annotated correctly and granted the `iam.workloadIdentityUser` role on the GCP service account.
- **Authorizing the Core Controller Pod:** The ESO Core Controller Pod's service account is authorized to access Secret Manager secrets. No authentication is required for `SecretStore` and `ClusterSecretStore` instances.

In the following, we will describe each of these options in detail.

#### Prerequisites

* Ensure that [Workload Identity Federation is enabled](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) for the GKE cluster.

_Note that while Google Cloud WIF [is available for AKS, EKS, and self-hosted Kubernetes clusters](https://cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes), ESO currently supports WIF authentication only for GKE ([Issue #1038](https://github.com/external-secrets/external-secrets/issues/1038))._

#### Using a Kubernetes service account as a GCP IAM principal

The `SecretStore` (or `ClusterSecretStore`) references a Kubernetes service account that is authorized to access Secret Manager secrets.

To demonstrate this approach, we'll create a `SecretStore` in the `demo` namespace.

First, create a Kubernetes service account in the `demo` namespace:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: demo-secrets-sa
  namespace: demo
```

To grant a Kubernetes service account access to Secret Manager secret(s), you need to know four values:

* `PROJECT_ID`: Your GCP project ID, which you can find under "Project Info" on your console dashboard. Note that this might be different from your project's _name_.
* `PROJECT_NUMBER`: Your GCP project number, which you can find under "Project Info" on your console dashboard or through `gcloud projects describe $PROJECT_ID --format="value(projectNumber)"`.
* `K8S_SA`: The name of the Kubernetes service account you created. (In our example, `demo-secrets-sa`.)
* `K8S_NAMESPACE`: The namespace where you created the Kubernetes service account (In our example, `demo`.)

For example, the following CLI call grants the Kubernetes service account access to a secret `demo-secret`:

```shell
gcloud secrets add-iam-policy-binding demo-secret \
  --project=$PROJECT_ID \
  --role="roles/secretmanager.secretAccessor" \
  --member="principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/${K8S_NAMESPACE}/sa/${K8S_SA}"
```

You can also grant the Kubernetes service account access to _all_ secrets in a GCP project:

```shell
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --role="roles/secretmanager.secretAccessor" \
  --member="principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/${K8S_NAMESPACE}/sa/${K8S_SA}"
```

Note that this allows anyone who can create `ExternalSecret` resources referencing a `SecretStore` instance using this service account access to all secrets in the project.

_For more information about WIF and Secret Manager permissions, refer to:_

* _[Authenticate to Google Cloud APIs from GKE workloads](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) in the GKE documentation._
* _[Access control with IAM](https://cloud.google.com/secret-manager/docs/access-control) in the Secret Manager documentation._

Next, create a `SecretStore` that references the `demo-secrets-sa` Kubernetes service account:

```yaml
{% include 'gcpsm-wif-iam-secret-store.yaml' %}
```

In the case of a `ClusterSecretStore`, you additionally have to define the service account's `namespace` under `auth.workloadIdentity.serviceAccountRef`.

Finally, you can create an `ExternalSecret` for the `demo-secret` that references this `SecretStore`:

```yaml
{% include 'gcpsm-wif-externalsecret.yaml' %}
```

#### Linking a Kubernetes service account to a GCP service account

The `SecretStore` (or `ClusterSecretStore`) references a Kubernetes service account, which is linked to a GCP service account that is authorized to access Secret Manager secrets.

To demonstrate this approach, we'll create a `SecretStore` in the `demo` namespace.

To set up the Kubernetes service account, you need to know or choose the following values:

* `PROJECT_ID`: Your GCP project ID, which you can find under "Project Info" on your console dashboard. Note that this might be different from your project's _name_.
* `GCP_SA`: The name of the GCP service account you are going to create and use (e.g., `external-secrets`).

First, create the Kubernetes service account with an annotation that references the GCP service account:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: demo-secrets-sa
  namespace: demo
  annotations:
    iam.gke.io/gcp-service-account: [GCP_SA]@[PROJECT_ID].iam.gserviceaccount.com
```

Next, create the GCP service account:

```shell
gcloud iam service-accounts create $GCP_SA \
  --project=$PROJECT_ID
```

To finalize the link between the GCP service account and the Kubernetes service account, you need two additional values:

* `K8S_SA`: The name of the Kubernetes service account you created. (In our example, `demo-secrets-sa`.)
* `K8S_NAMESPACE`: The namespace where you created the Kubernetes service account (In our example, `demo`.)

Grant the Kubernetes service account the `iam.workloadIdentityUser` role on the GCP service account:

```shell
gcloud iam service-accounts add-iam-policy-binding \
  ${GCP_SA}@${PROJECT_ID}.iam.gserviceaccount.com \
  --role="roles/iam.workloadIdentityUser" \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[${K8S_NAMESPACE}/${K8S_SA}]"
```

Next, grant the GCP service account access to a secret in the Secret Manager.
For example, the following CLI call grants it access to a secret `demo-secret`:

```shell
gcloud secrets add-iam-policy-binding demo-secret \
  --project=$PROJECT_ID \
  --role="roles/secretmanager.secretAccessor"
  --member "serviceAccount:${GCP_SA}@${PROJECT_ID}.iam.gserviceaccount.com"
```

You can also grant the GCP service account access to _all_ secrets in a GCP project:

```shell
gcloud project add-iam-policy-binding $PROJECT_ID \
  --role="roles/secretmanager.secretAccessor"
  --member "serviceAccount:${GCP_SA}@${PROJECT_ID}.iam.gserviceaccount.com"
```

Note that this allows anyone who can create `ExternalSecret` resources referencing a `SecretStore` instance using this service account access to all secrets in the project.

_For more information about WIF and Secret Manager permissions, refer to:_

* _[Authenticate to Google Cloud APIs from GKE workloads](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) in the GKE documentation._
* _[Access control with IAM](https://cloud.google.com/secret-manager/docs/access-control) in the Secret Manager documentation._

Next, create a `SecretStore` that references the `demo-secrets-sa` Kubernetes service account:

```yaml
{% include 'gcpsm-wif-sa-secret-store.yaml' %}
```

In the case of a `ClusterSecretStore`, you additionally have to define the service account's `namespace` under `auth.workloadIdentity.serviceAccountRef`.

Finally, you can create an `ExternalSecret` for the `demo-secret` that references this `SecretStore`:

```yaml
{% include 'gcpsm-wif-externalsecret.yaml' %}
```

#### Authorizing the Core Controller Pod

Instead of managing authentication at the `SecretStore` and `ClusterSecretStore` level, you can give the [Core Controller](../api/components.md) Pod's service account access to Secret Manager secrets using one of the two WIF approaches described in the previous sections.

To demonstrate this approach, we'll assume you installed ESO using Helm into the `external-secrets` namespace, with `external-secrets` as the release name:

```shell
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets \
  --namespace external-secrets --create-namespace
```

This creates a Kubernetes service account `external-secrets` in the `external-secrets` namespace, which is used by the Core Controller Pod.

To verify this (or to determine the service account's name in a different setup), you can run:

```shell
kubectl get pods --namespace external-secrets \
  --selector app.kubernetes.io/name=external-secrets \
  --output jsonpath='{.items[0].spec.serviceAccountName}'
```

Use WIF to grant this Kubernetes service account access to the Secret Manager secrets.
You can use either of the approaches described in the previous two sections.

_For details and further information on WIF and Secret Manager permissions, refer to:_

* _[Authenticate to Google Cloud APIs from GKE workloads](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) in the GKE documentation._
* _[Access control with IAM](https://cloud.google.com/secret-manager/docs/access-control) in the Secret Manager documentation._

Once the Core Controller Pod can access the Secret Manager secret(s) through WIF via its Kubernetes service account, you can create `SecretStore` or `ClusterSecretStore` instances that only specify the GCP project ID, omitting the `auth` section entirely:

```yaml
{% include 'gcpsm-wif-core-controller-secret-store.yaml' %}
```

#### Explicitly specifying the GKE cluster's name and location

When creating a `SecretStore` or `ClusterSecretStore` that uses WIF, the GKE cluster's project ID, name, and location are automatically determined through the [GCP metadata server](https://cloud.google.com/compute/docs/metadata/overview).
Alternatively, you can explicitly specify some or all of these values.

For a fully specified configuration, you'll need to know the following three values:

* `CLUSTER_PROJECT_ID`: The ID of GCP project that contains the GKE cluster.
* `CLUSTER_NAME`: The name of the GKE cluster.
* `CLUSTER_LOCATION`: The location of the GKE cluster. For a regional cluster, this is the region. For a zonal cluster, this is the zone.

You can optionally verify these values through the CLI:

```shell
gcloud container clusters describe $CLUSTER_NAME \
  --project=$CLUSTER_PROJECT_ID --location=$CLUSTER_LOCATION
```

If the three values are correct, this returns information about your GKE cluster.

Then, you can create a `SecretStore` or `ClusterSecretStore` that explicitly specifies the cluster's project ID, name, and location:

```yaml
{% include 'gcpsm-wif-sa-secret-store-with-explicit-name-and-location.yaml' %}
```

### Authenticating with a GCP service account

The `SecretStore` (or `ClusterSecretStore`) uses a long-lived, static [GCP service account key](https://cloud.google.com/iam/docs/service-account-creds#key-types) to authenticate with GCP.
This approach can be used on any Kubernetes cluster.

To demonstrate this approach, we'll create a `SecretStore` in the `demo` namespace.

First, create a GCP service account and grant it the `secretmanager.secretAccessor` role on the Secret Manager secret(s) you want to access.

_For details and further information on managing service account permissions and Secret Manager roles, refer to:_

* _[Attach service accounts to resources](https://cloud.google.com/iam/docs/attach-service-accounts) in the IAM documentation._
* _[Access control with IAM](https://cloud.google.com/secret-manager/docs/access-control) in the Secret Manager documentation._

Then, create a service account key pair using one of the methods described on the page [Create and delete service account keys](https://cloud.google.com/iam/docs/keys-create-delete) in the Google Cloud IAM documentation and store the JSON file with the private key in a Kubernetes `Secret`:

```yaml
{% include 'gcpsm-sa-credentials-secret.yaml' %}
```

Finally, reference this secret in the `SecretStore` manifest:

```yaml
{% include 'gcpsm-sa-secret-store.yaml' %}
```

In the case of a `ClusterSecretStore`, you additionally have to specify the service account's `namespace` under `auth.secretRef.secretAccessKeySecretRef`.

## Using PushSecret with an existing Google Secret Manager secret

There are some use cases where you want to use PushSecret for an existing Google Secret Manager Secret that already has labels defined. For example when the creation of the secret is managed by another controller like Kubernetes Config Connector (KCC) and the updating of the secret is managed by ESO.

To allow ESO to take ownership of the existing Google Secret Manager Secret, you need to add the label `"managed-by": "external-secrets"`.

By default, the PushSecret spec will replace any existing labels on the existing GCP Secret Manager Secret. To prevent this, a new field was added to the `spec.data.metadata` object called `mergePolicy` which defaults to `Replace` to ensure that there are no breaking changes and is backward compatible. The other option for this field is `Merge` which will merge the existing labels on the Google Secret Manager Secret with the labels defined in the PushSecret spec. This ensures that the existing labels defined on the Google Secret Manager Secret are retained.

Example of using the `mergePolicy` field:

```yaml
{% raw %}
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
  namespace: default
spec:
  updatePolicy: Replace
  deletionPolicy: None
  refreshInterval: 1h
  secretStoreRefs:
    - name: gcp-secretstore
      kind: SecretStore
  selector:
    secret:
      name: bestpokemon
  template:
    data:
      bestpokemon: "{{ .bestpokemon }}"
  data:
    - conversionStrategy: None
      metadata:
        mergePolicy: Merge
        labels:
          anotherLabel: anotherValue
      match:
        secretKey: bestpokemon
        remoteRef:
          remoteKey: best-pokemon
{% endraw %}
```

## Secret Replication and Encryption Configuration

### Location and Replication

By default, secrets are automatically replicated across multiple regions. You can specify a single location for your secrets by setting the `replicationLocation` field:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ... other fields ...
  data:
    - match:
        secretKey: mykey
        remoteRef:
          remoteKey: my-secret
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata`
        spec:
          replicationLocation: "us-east1"
```

### Customer-Managed Encryption Keys (CMEK)

You can use your own encryption keys to encrypt secrets at rest. To use Customer-Managed Encryption Keys (CMEK), you need to:

1. Create a Cloud KMS key
2. Grant the service account the `roles/cloudkms.cryptoKeyEncrypterDecrypter` role on the key
3. Specify the key in the PushSecret metadata

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ... other fields ...
  data:
    - match:
        secretKey: mykey
        remoteRef:
          remoteKey: my-secret
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          cmekKeyName: "projects/my-project/locations/us-east1/keyRings/my-keyring/cryptoKeys/my-key"
```

Note: When using CMEK, you must specify a location in the SecretStore as customer-managed encryption keys are region-specific.

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: gcp-secret-store
spec:
  provider:
    gcpsm:
      projectID: my-project
      location: us-east1  # Required when using CMEK
```

## Regional Secrets
GCP Secret Manager Regional Secrets are available to be used with both ExternalSecrets and PushSecrets.

In order to achieve so, add a `location` to your SecretStore definition:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcp-secret-store
spec:
  provider:
    gcpsm:
      projectID: my-project
      location: us-east1 # uses regional secrets on us-east1
```

## Secret Version Management

### Secret Version Selection Policy

The Google Secret Manager provider includes a `secretVersionSelectionPolicy` field that controls how the provider handles secret version selection when the default "latest" version is unavailable.

By default, when you request a secret without specifying a version, the provider attempts to fetch the "latest" version. The `secretVersionSelectionPolicy` determines what happens if that version is in a DESTROYED or DISABLED state.

#### Available Policies

- **`LatestOrFail`** (default): The provider always uses "latest", or fails if that version is disabled/destroyed.
- **`LatestOrFetch`**: The provider falls back to fetching the latest enabled version if the "latest" version is DESTROYED or DISABLED.

#### Configuration Example

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcp-secret-store
spec:
  provider:
    gcpsm:
      projectID: my-project
      location: us-east1
      secretVersionSelectionPolicy: LatestOrFetch  # or LatestOrFail (default)
```

**Note**: When using `secretVersionSelectionPolicy: LatestOrFetch`, the service account requires additional permissions to list secret versions. You'll need to grant the `roles/secretmanager.viewer` role (which includes `secretmanager.versions.list`) or the specific `secretmanager.versions.list` permission in addition to the standard `secretmanager.secretAccessor` role.

```
