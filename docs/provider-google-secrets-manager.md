## Google Cloud Secret Manager

External Secrets Operator integrates with [GCP Secret Manager](https://cloud.google.com/secret-manager) for secret management.

### Service account key authentication

A service account key is created and the JSON keyfile is stored in a `Kind=Secret`. The `project_id` and `private_key` should be configured for the project.

```yaml
{% include 'gcpsm-credentials-secret.yaml' %}
```

### Update secret store
Be sure the `gcpsm` provider is listed in the `Kind=SecretStore`

```yaml
{% include 'gcpsm-secret-store.yaml' %}
```

### Creating external secret

To create a kubernetes secret from the GCP Secret Manager secret a `Kind=ExternalSecret` is needed.

```yaml
{% include 'gcpsm-external-secret.yaml' %}
```

The operator will fetch the GCP Secret Manager secret and inject it as a `Kind=Secret`
```
kubectl get secret secret-to-be-created -n <namespace> | -o jsonpath='{.data.dev-secret-test}' | base64 -d
```

## Authentication with Workload Identity

This makes it possible for your Google Kubernetes Engine (GKE) applications to consume services provided by Google APIs, namely Secrets Manager service in this case.

Here we will assume that you installed ESO using helm and that you named the chart installation `external-secrets` and the namespace where it lives `es` like:

```sh
helm install external-secrets external-secrets/external-secrets --namespace es
```

Then most of the resources would have this name, the important one here being the k8s service account attached to the external-secrets operator deployment:

```
# ...
      containers:
      - image: ghcr.io/external-secrets/external-secrets:vVERSION
        name: external-secrets
        ports:
        - containerPort: 8080
          protocol: TCP
      restartPolicy: Always
      schedulerName: default-scheduler
      serviceAccount: external-secrets
      serviceAccountName: external-secrets # <--- here
```

### Following the documentation

You can find the documentation for Workload Identity under [this url](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity). We will walk you through how to navigate it here.

#### Changing Values

Search [the documment](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) for this editable values and change them to your values:

- CLUSTER_NAME: The name of your cluster
- PROJECT_ID: Your project ID (not your Project number nor your Project name)
- K8S_NAMESPACE: For us folowing these steps here it will be `es`, but this will be the namespace where you deployed the external-secrets operator
- KSA_NAME: external-secrets (if you are not creating a new one to attach to the deployemnt)
- GSA_NAME: external-secrets for simplicity, or something else if you have to follow different naming convetions for cloud resources
- ROLE_NAME: roles/secretmanager.secretAccessor so you make the pod only be able to access secrets on Secret Manager

#### Following through

You can follow through the documentation and adapt it to your specific use case. If you want to just use the serviceaccount that we deployed with the helm chart, for example, you don't need to create a new service account on 2 of [Authenticating to Google Cloud](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#authenticating_to).

#### SecretStore with WorkloadIdentity

To use workload identity you can just omit the auth field of the secret store and let the operator client fall back to defaults using the roles attached to your service account.

```
apiVersion: external-secrets.io/v1alpha1
kind: SecretStore
metadata:
  name: example
spec:
  provider:
    gcpsm:
      projectID: pid
```