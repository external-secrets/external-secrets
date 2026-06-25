External Secrets Operator integrates with the [Google Cloud Secret Manager](https://cloud.google.com/secret-manager).

## Authentication

The Google Secret Manager provider resolves credentials in this order: static service account JSON (`auth.secretRef`), [GKE Workload Identity](#workload-identity-gke) (`auth.workloadIdentity`), [GCP Workload Identity Federation](#workload-identity-federation) (`auth.workloadIdentityFederation`), then [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) from the environment (for example the GKE metadata server when no explicit auth is configured).

Pick the mechanism that matches where the operator runs:

| Mechanism | API field | Typical use |
| --- | --- | --- |
| GKE Workload Identity | `auth.workloadIdentity` | GKE clusters with [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) enabled; uses the GKE metadata server and the identity binding token flow. |
| GCP Workload Identity Federation | `auth.workloadIdentityFederation` | AKS, EKS, self-hosted Kubernetes, or any setup where you configure an IAM workload identity pool and provider per [Google’s federation docs](https://cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes). |
| Static service account key | `auth.secretRef` | Any cluster; long-lived JSON key in a Kubernetes `Secret` (not recommended where federation or GKE WI is available). |

<a id="workload-identity-gke"></a>
### Workload Identity (GKE)

Through [GKE Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity), workloads on **Google Kubernetes Engine** can call Google APIs (including Secret Manager) without storing long-lived keys. In External Secrets Operator this path is implemented as `auth.workloadIdentity` and expects the **GCP metadata server** (available on GKE nodes) so the operator can discover the cluster project, name, and location when those fields are omitted.

Authenticating with GKE Workload Identity is the usual choice when the operator runs on GKE. ESO supports three patterns:

- **Using a Kubernetes service account as a GCP IAM principal**: The `SecretStore` (or `ClusterSecretStore`) references a [Kubernetes service account](https://kubernetes.io/docs/concepts/security/service-accounts) that is authorized to access Secret Manager secrets.
- **Linking a Kubernetes service account to a GCP service account:** The `SecretStore` (or `ClusterSecretStore`) references a Kubernetes service account, which is linked to a [GCP service account](https://cloud.google.com/iam/docs/service-accounts) that is authorized to access Secret Manager secrets. This requires that the Kubernetes service account is annotated correctly and granted the `iam.workloadIdentityUser` role on the GCP service account.
- **Authorizing the Core Controller Pod:** The ESO Core Controller Pod's service account is authorized to access Secret Manager secrets. No authentication is required for `SecretStore` and `ClusterSecretStore` instances.

In the following, we will describe each of these options in detail.

#### Prerequisites

* Enable and use [Workload Identity on the GKE cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity).

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

_For more information about GKE Workload Identity and Secret Manager permissions, refer to:_

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
  --role="roles/secretmanager.secretAccessor" \
  --member "serviceAccount:${GCP_SA}@${PROJECT_ID}.iam.gserviceaccount.com"
```

You can also grant the GCP service account access to _all_ secrets in a GCP project:

```shell
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --role="roles/secretmanager.secretAccessor" \
  --member "serviceAccount:${GCP_SA}@${PROJECT_ID}.iam.gserviceaccount.com"
```

Note that this allows anyone who can create `ExternalSecret` resources referencing a `SecretStore` instance using this service account access to all secrets in the project.

_For more information about GKE Workload Identity and Secret Manager permissions, refer to:_

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

Instead of managing authentication at the `SecretStore` and `ClusterSecretStore` level, you can give the [Core Controller](../api/components.md) Pod's service account access to Secret Manager secrets using one of the two GKE Workload Identity approaches described in the previous sections.

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

Use GKE Workload Identity to grant this Kubernetes service account access to the Secret Manager secrets.
You can use either of the approaches described in the previous two sections.

_For details and further information on GKE Workload Identity and Secret Manager permissions, refer to:_

* _[Authenticate to Google Cloud APIs from GKE workloads](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) in the GKE documentation._
* _[Access control with IAM](https://cloud.google.com/secret-manager/docs/access-control) in the Secret Manager documentation._

Once the Core Controller Pod can access the Secret Manager secret(s) through GKE Workload Identity via its Kubernetes service account, you can create `SecretStore` or `ClusterSecretStore` instances without authentication configuration. You can optionally specify the GCP project ID, or omit it to use auto-detection from the GCP metadata server:

```yaml
{% include 'gcpsm-wif-core-controller-secret-store.yaml' %}
```

Alternatively, with projectID auto-detection (GKE only):

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcp-secret-store
  namespace: demo
spec:
  provider:
    gcpsm: {} # Both projectID and auth are optional when using Core Controller authentication in GKE
```

#### Auto-detection of GCP project ID

When creating a `SecretStore` or `ClusterSecretStore`, the `projectID` field is optional only if the provider can infer the Google Cloud project another way. The implementation resolves a fallback project from the [GCP metadata server](https://cloud.google.com/compute/docs/metadata/overview) when **no** `auth.secretRef` is set and the controller runs on **GKE** (metadata is not available on most non-GKE clusters).

In practice:

- With **`auth.workloadIdentity`** or ADC on **GKE**, omitting `projectID` is supported when Secret Manager secrets live in the **same** project as the cluster (or when `clusterProjectID` / explicit `projectID` disambiguates cross-project cases; see below).
- With **`auth.workloadIdentityFederation`** on clusters **without** GCP metadata, set **`projectID`** explicitly to the project that owns your secrets.
- With **`auth.secretRef`**, `projectID` is **required** (no metadata fallback).

This allows portable `SecretStore` configurations on GKE without hard-coding the project when the above conditions hold:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcp-secret-store
spec:
  provider:
    gcpsm:
      # projectID optional on GKE when metadata resolves the secrets project
      auth:
        workloadIdentity:
          serviceAccountRef:
            name: demo-secrets-sa
```

You must set `projectID` explicitly when using static service account credentials (`auth.secretRef`), when the metadata server is unavailable or points at the wrong project, or when accessing secrets in a different project than the one inferred for the client.

#### projectID vs clusterProjectID

`projectID` (`spec.provider.gcpsm.projectID`) tells the provider which GCP project holds the secrets. It is used in secret resource paths like `projects/{projectID}/secrets/{name}`. For **GKE Workload Identity** (`auth.workloadIdentity`), it also feeds cluster-side resolution when `clusterProjectID` is not set.

`clusterProjectID` (`spec.provider.gcpsm.auth.workloadIdentity.clusterProjectID`) identifies the project hosting the GKE cluster. It is **only** used by **`auth.workloadIdentity`** to build the identity pool and provider URL. When either field is omitted on GKE, the provider can query the [GCP metadata server](https://cloud.google.com/compute/docs/metadata/overview) for the project ID. This field does not apply to `auth.workloadIdentityFederation`.

For cross-project access, set both fields explicitly:

```yaml
spec:
  provider:
    gcpsm:
      projectID: "secrets-project-456"
      auth:
        workloadIdentity:
          clusterProjectID: "cluster-project-123"
          serviceAccountRef:
            name: demo-sa
```

#### Explicitly specifying the GKE cluster's name and location

When creating a `SecretStore` or `ClusterSecretStore` that uses **`auth.workloadIdentity`**, the GKE cluster's name and location are automatically determined through the [GCP metadata server](https://cloud.google.com/compute/docs/metadata/overview).
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

<a id="workload-identity-federation"></a>
### Workload Identity Federation

[GCP Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation) lets workloads use **short-lived tokens from an external identity provider** (for example a Kubernetes API server or AWS) that Google trusts through an IAM **workload identity pool** and **provider**. This is different from [GKE Workload Identity](#workload-identity-gke): federation uses the **external account** OAuth flow (STS token exchange via `golang.org/x/oauth2/google/externalaccount`) and does **not** rely on the GKE identity binding token or the default `.svc.id.goog` pool on the cluster project.

Use `auth.workloadIdentityFederation` when you follow Google’s guide to [configure Workload Identity Federation with Kubernetes](https://cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes) on AKS, EKS, self-hosted clusters, and OpenShift, or when you [configure an AWS workload identity pool provider and credential file](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds#create-cred-config) for AWS-based subject tokens.

#### Configuration rules

Under `auth.workloadIdentityFederation` you must set **exactly one** of `serviceAccountRef`, `credConfig`, or `awsSecurityCredentials`. The provider rejects any other combination.

| Field | Purpose |
| --- | --- |
| `serviceAccountRef` | Request a bound token for the named Kubernetes `ServiceAccount` and use it as the STS subject token (`urn:ietf:params:oauth:token-type:jwt`). **Requires `audience`.** |
| `credConfig` | Load an `external_account` JSON document from a `ConfigMap` key ([external identity ADC JSON](https://cloud.google.com/docs/authentication/application-default-credentials#external-identities)). `audience` may come from the JSON or be overridden by the spec field; it must be non-empty after merge. |
| `awsSecurityCredentials` | Supply static AWS credentials in a Kubernetes `Secret` plus `region` so the subject token type is `urn:ietf:params:aws:token-type:aws4_request` without using the instance metadata service from inside the pod. **Requires `audience`.** |

**`audience`:** Required on the spec when `serviceAccountRef` or `awsSecurityCredentials` is set. It must be the full workload identity **provider** resource name, for example `//iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID`. When only `credConfig` is used, `audience` can be supplied in the JSON; a non-empty `audience` on the spec overrides the file value.

**`projectID`:** Set `spec.provider.gcpsm.projectID` to the project that contains your Secret Manager secrets whenever the controller cannot rely on GKE metadata (typical for federation off GCP nodes).

#### Kubernetes subject token (`serviceAccountRef`)

ESO uses the Kubernetes `TokenRequest` API to mint a token for `serviceAccountRef` with `aud` equal to `spec.provider.gcpsm.auth.workloadIdentityFederation.audience`, optionally appending entries from `serviceAccountRef.audiences`. That token is exchanged at Google STS for a Google access token.

Grant access on the secret (or project) to the **federated principal** for that Kubernetes identity:

```shell
gcloud secrets add-iam-policy-binding "${SECRET_NAME}" \
  --project="${PROJECT_ID}" \
  --role="roles/secretmanager.secretAccessor" \
  --member="principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${WIF_POOL_NAME}/subject/system:serviceaccount:${K8S_NAMESPACE}:${K8S_SA}"
```

If the principal does **not** have `secretmanager.secrets.get` / accessor on a secret, sync fails with `PermissionDenied` on `secretmanager.versions.access` even when the `SecretStore` is `Ready`—bind IAM to the identity that actually reaches Secret Manager after impersonation (see below).

Example `SecretStore` when Kubernetes is the external identity provider (see the [WorkloadIdentityFederation API](https://external-secrets.io/latest/api/spec/#external-secrets.io/v1.GCPWorkloadIdentityFederation)):

```yaml
{% include 'gcpsm-wif-non-native-iam-secret-store.yaml' %}
```

For `ClusterSecretStore`, set `serviceAccountRef.namespace` when the `ServiceAccount` lives outside the referent namespace.

#### Google service account impersonation

After STS returns a federated identity, ESO may call the [IAM Credentials API](https://cloud.google.com/iam/docs/reference/credentials/rest) to **impersonate** a Google service account (GSA) and obtain an access token with Secret Manager scopes.

Impersonation is resolved as follows (see `updateServiceAccountImpersonationURL` in the provider):

1. **`gcpServiceAccountEmail`** on `workloadIdentityFederation` — if set, it always sets impersonation for that GSA and overrides any other impersonation hint.
2. With **`credConfig` only** (no `serviceAccountRef`): use **`service_account_impersonation_url`** from the `external_account` JSON when present (unless step 1 already applied).
3. With **`serviceAccountRef`**: if step 1 did not apply, use the **`iam.gke.io/gcp-service-account`** annotation on that `ServiceAccount` when present.

The implementation only allows impersonation URLs that match Google’s `generateAccessToken` endpoint pattern (see validation in the provider).

Typical patterns:

- **Direct access:** bind `roles/secretmanager.secretAccessor` on secrets to the **workload identity principal** (`principal://…/subject/system:serviceaccount:…`), as in the previous section. No impersonation.
- **Access via a GSA:** bind `roles/secretmanager.secretAccessor` on secrets to the **GSA** (`serviceAccount:my-gsa@project.iam.gserviceaccount.com`). Grant the federated principal **`roles/iam.workloadIdentityUser`** on that GSA ([grant access to service accounts](https://cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes#kubernetes-sa)) so it may impersonate it, and set `gcpServiceAccountEmail` (or the `iam.gke.io/gcp-service-account` annotation) so ESO uses impersonation. If the federated principal lacks secret access but the GSA has it, sync fails with `PermissionDenied` until impersonation is configured—see [impersonating a service account](https://cloud.google.com/iam/docs/using-workload-identity-federation#impersonation) and [creating short-lived credentials](https://cloud.google.com/iam/docs/create-short-lived-credentials-direct#sa-credentials-oauth).

#### External account JSON (`credConfig`)

Point `credConfig` at a `ConfigMap` key whose value is JSON with `"type": "external_account"` and the usual fields (`audience`, `subject_token_type`, `token_url`, `token_info_url`, `credential_source`, optional `service_account_impersonation_url`, etc.). Generate a starting file with [`gcloud iam workload-identity-pools create-cred-config`](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds#create-cred-config) as described in Google’s documentation.

Security and validation notes enforced by the provider:

- **`credential_source.executable`** is **not allowed**.
- After merge, **`token_url`** must look like `https://sts.<universe>/v1/token` and **`token_info_url`** like `https://sts.<universe>/v1/introspect` (defaults are filled for `googleapis.com` when omitted).
- If `credential_source` uses a **non-AWS** HTTP **`url`**, set **`externalTokenEndpoint`** on the spec to the **same** URL; the provider verifies they match.
- If `credential_source` uses the **AWS** metadata layout (`environment_id` starting with `aws`), URLs must match the expected IMDS patterns (metadata host or `169.254.169.254`, etc.).
- If the JSON sets `credential_source.file` to the operator pod’s automounted path (`/var/run/secrets/kubernetes.io/serviceaccount/token`), that source is **ignored** so the ESO controller does not accidentally use its own service account token; use **`serviceAccountRef`** instead to select which Kubernetes identity supplies the subject token.

#### AWS subject token (`awsSecurityCredentials`)

For an **AWS** workload identity provider, a `credConfig` file produced by [`gcloud iam workload-identity-pools create-cred-config`](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds#create-cred-config) typically reads credentials from the EC2 instance metadata service (IMDS). Pods usually **cannot** reach `169.254.169.254` from the container network, so that approach often fails with `connection refused` inside the ESO pod even when the node can reach IMDS. In that situation use **`awsSecurityCredentials`**: put **`aws_access_key_id`**, **`aws_secret_access_key`**, and optionally **`aws_session_token`** in a Kubernetes `Secret`, set **`region`**, and reference that secret from `awsSecurityCredentials.awsCredentialsSecretRef` (namespace may be set on `ClusterSecretStore`). On **Amazon EKS**, Google recommends [federation with Kubernetes](https://cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes) and `serviceAccountRef` when your cluster exposes an OIDC issuer.

Grant Secret Manager access to the **AWS principal** in the pool using a `principalSet` on the mapped account attribute, for example:

```shell
gcloud secrets add-iam-policy-binding "${SECRET_NAME}" \
  --project="${PROJECT_ID}" \
  --role="roles/secretmanager.secretAccessor" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${WIF_POOL_NAME}/attribute.account/${AWS_ACCOUNT_ID}"
```

See [Manage workload identity pools and providers](https://cloud.google.com/iam/docs/manage-workload-identity-pools-providers) for creating an AWS provider and attribute mapping, and [Configure Workload Identity Federation with AWS or Azure VMs](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds) for the full AWS setup guide.

#### Other API surfaces

The same `workloadIdentityFederation` block (including `serviceAccountRef`, `credConfig`, `awsSecurityCredentials`, `audience`, and `gcpServiceAccountEmail`) is available on **`GCRAccessToken`** and **`ClusterGenerator`** resources that talk to Google APIs; see the [API spec](https://external-secrets.io/latest/api/spec/#external-secrets.io/v1.GCPWorkloadIdentityFederation).

#### References

- [Workload Identity Federation overview](https://cloud.google.com/iam/docs/workload-identity-federation)
- [Federation with Kubernetes](https://cloud.google.com/iam/docs/workload-identity-federation-with-kubernetes)
- [Federation with AWS or Azure VMs](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds)
- [Manage workload identity pools and providers](https://cloud.google.com/iam/docs/manage-workload-identity-pools-providers)
- [Create credential configuration files](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds#create-cred-config)
- [Use Workload Identity Federation (including impersonation)](https://cloud.google.com/iam/docs/using-workload-identity-federation)
- [External credentials for client libraries](https://cloud.google.com/docs/authentication/client-libraries#external-identities)
- [Secret Manager access control](https://cloud.google.com/secret-manager/docs/access-control)

### Authenticating with a GCP service account (static key)

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
  refreshInterval: 1h0m0s
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
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
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

By default, secrets are automatically replicated across multiple regions. You can specify one or more replication locations for your secrets by setting the `replicationLocations` field:

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
          replicationLocations:
            - "us-east1"
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
