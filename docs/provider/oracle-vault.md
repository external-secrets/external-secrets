## Oracle Vault

External Secrets Operator integrates with [OCI API](https://github.com/oracle/oci-go-sdk) to sync secret on the Oracle Vault to secrets held on the Kubernetes cluster.

### Authentication

Specify the authenticating principal with `principalType`, using `UserPrincipal`, `InstancePrincipal`, or `Workload` as values.
If `principalType` or `auth` are not set, the operator defaults to instance principal for authentication.

For user principal, userOCID, tenancyOCID, fingerprint and private key are required.
The fingerprint and key file should be supplied in the secret with the rest being provided in the secret store.

See url for what region you you are accessing.
![userOCID-details](../pictures/screenshot_region.png)

Select tenancy in the top right to see your user OCID as shown below.
![tenancyOCID-details](../pictures/screenshot_tenancy_OCID.png)

Select your user in the top right to see your user OCID as shown below.
![region-details](../pictures/screenshot_user_OCID.png)


#### Service account key authentication

Create a secret containing your private key and fingerprint:

```yaml
{% include 'oracle-credentials-secret.yaml' %}
```

Your fingerprint will be attatched to your API key, once it has been generated. Found on the same page as the user OCID.
![fingerprint-details](../pictures/screenshot_fingerprint.png)

Once you click "Add API Key" you will be shown the following, where you can download the RSA key in the necessary PEM format for API requests.
This will automatically generate a fingerprint.
![API-key-details](../pictures/screenshot_API_key.png)

### Update secret store
Be sure the `oracle` provider is listed in the `Kind=SecretStore`.

```yaml
{% include 'oracle-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, Be sure to provide `namespace` in `privatekey` and `fingerprint` with the namespaces where the secrets reside.
### Creating external secret

To create a kubernetes secret from the Oracle Cloud Interface secret a`Kind=ExternalSecret` is needed.

```yaml
{% include 'oracle-external-secret.yaml' %}
```


### Getting the Kubernetes secret
The operator will fetch the project variable and inject it as a `Kind=Secret`.
```
kubectl get secret oracle-secret-to-create -o jsonpath='{.data.dev-secret-test}' | base64 -d
```

### PushSecrets and retrieving multiple secrets.
When using [PushSecrets](https://external-secrets.io/latest/guides/pushsecrets/), the compartment OCID and encryption key OCID must be specified in the
Oracle SecretStore. You can find your compartment and encrpytion key OCIDs in the OCI console.

If [retrieving multiple secrets](https://external-secrets.io/latest/guides/getallsecrets/) by tag or regex, only the compartment OCID must be specified.

```yaml
{% include 'oracle-secret-store-pushsecret.yaml' %}
```

