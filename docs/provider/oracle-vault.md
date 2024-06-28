## Oracle Vault

External Secrets Operator integrates with the [Oracle Cloud Infrastructure (OCI) REST API](https://docs.oracle.com/en-us/iaas/api/) to manage secrets in Oracle Vault. All secret operations exposed by External Secrets Operator are supported by the Oracle provider.

For more information on managing OCI Vaults and OCI Vault Secrets, see the following documentation:

- [Managing Vaults](https://docs.oracle.com/en-us/iaas/Content/KeyManagement/Tasks/managingvaults.htm)
- [Managing Vault Secrets](https://docs.oracle.com/en-us/iaas/Content/KeyManagement/Tasks/managingsecrets.htm)

## Authentication

External Secrets Operator may authenticate to OCI Vault using User Principal, [Instance Principal](https://blogs.oracle.com/developers/post/accessing-the-oracle-cloud-infrastructure-api-using-instance-principals), or [Workload Identity](https://blogs.oracle.com/cloud-infrastructure/post/oke-workload-identity-greater-control-access).

To specify the authenticating principal in a secret store, set the `spec.provider.oracle.principalType` value. Note that the value of `principalType` defaults `InstancePrincipal` if not set.

{% include 'oracle-principal-type.yaml' %}

### User Principal Authentication

For user principal authentication, region, user OCID, tenancy OCID, private key, and fingerprint are required.
The private key and fingerprint must be supplied in a Kubernetes secret, while the user OCID, tenancy OCID, and region should be set in the secret store.

To get your user principal information, find url for the OCI region you are accessing.
![userOCID-details](../pictures/screenshot_region.png)

Select tenancy in the top right to see your tenancy OCID as shown below.
![tenancyOCID-details](../pictures/screenshot_tenancy_OCID.png)

Select your user in the top right to see your user OCID as shown below.
![region-details](../pictures/screenshot_user_OCID.png)

Your fingerprint will be attatched to your API key, once it has been generated. Private keys can be created or uploaded on the same page as the your user OCID.
![fingerprint-details](../pictures/screenshot_fingerprint.png)

Once you click "Add API Key" you will be shown the following, where you can download the key in the necessary PEM format for API requests. Creating a private key will automatically generate a fingerprint.
![API-key-details](../pictures/screenshot_API_key.png)

Next, create a secret containing your private key and fingerprint:

```yaml
{% include 'oracle-credentials-secret.yaml' %}
```

After creating the credentials secret, the secret store can be configured:

```yaml
{% include 'oracle-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `privatekey` and `fingerprint` with the namespaces where the secrets reside.

### Instance Principal Authentication (OCI)

Instance Principal uses a pod's instance principal to authenticate to OCI Vault. Ensure your cluster instances have the appropriate policies to use [Instance Principal](https://blogs.oracle.com/developers/post/accessing-the-oracle-cloud-infrastructure-api-using-instance-principals).

```yaml
{% include 'oracle-instance-principal.yaml' %}
```

### Workload Identity Authentication (OCI/OKE)

[Workload Identity](https://blogs.oracle.com/cloud-infrastructure/post/oke-workload-identity-greater-control-access) can be used to grant the External Secrets Operator pod policy driven access to OCI Vault when running on Oracle Container Engine for Kubernetes (OKE).

Note that if a service account is not provided in the secret store, the Oracle provider will authenticate using the service account token of the External Secrets Operator.

```yaml
{% include 'oracle-workload-identity.yaml' %}
```

## Creating an External Secret

To create a Kubernetes secret from an OCI Vault secret a `Kind=ExternalSecret` is needed. The External Secret will reference an OCI Vault instance containing secrets with either JSON or plaintext data.

#### External Secret targeting JSON data

```yaml
{% include 'oracle-external-secret.yaml' %}
```
#### External Secret targeting plaintext data

```yaml
{% include 'oracle-external-secret-plaintext.yaml' %}
```

### Getting the Kubernetes secret
The operator will fetch the OCI Vault Secret and inject it as a `Kind=Secret`.
```
kubectl get secret oracle-secret-to-create -o jsonpath='{.data.dev-secret-test}' | base64 -d
```

## PushSecrets and retrieving multiple secrets.
When using [PushSecrets](https://external-secrets.io/latest/guides/pushsecrets/), the compartment OCID and encryption key OCID must be specified in the
Oracle SecretStore. You can find your compartment and encrpytion key OCIDs in the OCI console.

If [retrieving multiple secrets](https://external-secrets.io/latest/guides/getallsecrets/) by tag or regex, only the compartment OCID must be specified.

```yaml
{% include 'oracle-secret-store-pushsecret.yaml' %}
```
