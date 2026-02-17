
![aws sm](../pictures/eso-az-kv-azure-kv.png)

## Azure Key Vault

External Secrets Operator integrates with [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/). We support both `SecretStore` and `ClusterSecretStore` resources to connect to Azure Key Vault. The operator can fetch secrets, keys and certificates stored in Azure Key Vault and synchronize them as Kubernetes secrets using an `ExternalSecret`. Vice versa, it can also push secrets from Kubernetes into Azure Key Vault as secrets, keys and certificates by using a `PushSecret`.

We support connecting to different cloud flavours azure supports: `PublicCloud`, `USGovernmentCloud`, `ChinaCloud`, `GermanCloud` and `AzureStackCloud` (for Azure Stack Hub/Edge). You have to specify the `environmentType` and point to the correct cloud flavour. This defaults to `PublicCloud`.

For environments with non-standard endpoints (Azure Stack, Azure China with AKS Workload Identity, etc.), you can provide custom cloud configuration to override the default endpoints. See the [Custom Cloud Configuration](#custom-cloud-configuration) section below.

### Configure SecretStore

The Azure Key Vault provider is fully compatible with both namespaced `SecretStore` and cluster-wide `ClusterSecretStore` resources. For simplicity, this article refers only to `SecretStore`, but the same is applicable also for `ClusterSecretStore`. To establish the integration with Azure Key Vault, specify `azurekv` as the provider and provide the vault URL and authentication details. The vault URL is the URL of your Key Vault instance, which typically looks like `https://<your-keyvault-name>.vault.azure.net`. For Azure Stack, it may have a different format, so make sure to use the correct URL for your environment. See below for different authentication methods.

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: azure-store
spec:
  provider:
    azurekv:
      # URL of your Key Vault instance, see: https://docs.microsoft.com/en-us/azure/key-vault/general/about-keys-secrets-certificates
      vaultUrl: "https://xx-xxxx-xx.vault.azure.net"
```

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: azure-store
spec:
  provider:
    azurekv:
      # URL of your Key Vault instance, see: https://docs.microsoft.com/en-us/azure/key-vault/general/about-keys-secrets-certificates
      vaultUrl: "https://xx-xxxx-xx.vault.azure.net"
```

### Authentication

We support multiple authentication methods to connect to Azure Key Vault:

- Service Principal
- Managed Identity (AAD Pod Identity)
- Workload Identity

Regardless which authentication method is used to authenticate to Azure Key Vault, the identity which is assigned to External Secrets Operator needs to have proper permissions to access the Key Vault. We support both [RBAC](https://learn.microsoft.com/en-us/azure/key-vault/general/rbac-guide) and [Access Policy](https://learn.microsoft.com/en-us/azure/key-vault/general/assign-access-policy) based Key Vaults. RBAC is the recommended approach, but Access Policies are still supported for backward compatibility.

The required permissions depend on the type of objects you want to manage (secrets, keys, certificates) and the operations you want to perform (read, write, delete, etc.). For example, to grant External Secrets Operator permissions to synchronize secrets and certificates using an `ExternalSecret`, the minimum required permissions are either the [Key Vault Secrets User](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/security#key-vault-secrets-user) and [Key Vault Certificates User](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/security#key-vault-certificates-user) RBAC roles, alternatively for Access Policy based Key Vaults, the `Get` permission over secrets and certificates.

#### Service Principal

To use a Service Principal's credentials for authentication, you need to create a Service Principal in Entra ID and grant it the necessary permissions to access the Key Vault. Then, you can create a Kubernetes Secret containing the Service Principal's credentials:

- Client ID
- **Either** Client Secret **or** Client Certificate (in PEM format)

Reference this secret in your `SecretStore` configuration.

```yaml
{% include 'azkv-secret-store.yaml' %}
```

#### Managed Identity (AAD Pod Identity)

!!! warning
    This authentication option uses [AAD Pod Identity](https://azure.github.io/aad-pod-identity/docs/) which is deprecated, it is recommended to use the [Workload Identity](https://azure.github.io/azure-workload-identity) authentication instead.

To use a Managed Identity with AAD Pod Identity for authentication, you need to create a Managed Identity in Azure and grant it the necessary permissions to access the Key Vault. Then, assign that Managed Identity to the External Secrets Operator using [AAD Pod Identity](https://azure.github.io/aad-pod-identity/docs/). Finally, you can reference that identity in your `SecretStore` configuration.

If you have multiple identities assigned to External Secrets Operator, you can specify which of them to use by providing its client ID in the field `spec.provider.azurekv.identityId` on the `SecretStore`. If only one identity is assigned, you can omit the `identityId` field.

```yaml
{% include 'azkv-secret-store-mi.yaml' %}
```

#### Workload Identity

Workload Identity is the recommended authentication method for Azure Key Vault. It allows the External Secrets Operator to authenticate to Azure Key Vault using the identity of a Kubernetes Service Account, without needing to manage secrets or credentials. This is achieved by creating a trust relationship between your Kubernetes cluster and the Entra ID identity you want to use. This process is known as [Workload Identity Federation](https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation). The identity in Entra ID can be either an Application, a user-assigned Managed Identity or a Service Principal.

The Workload Identity Federation can be done in various ways, for instance using Terraform, the Azure Portal or Azure CLI. We found the [azwi](https://azure.github.io/azure-workload-identity/docs/installation/azwi.html) CLI very helpful. The Azure [Workload Identity Quick Start Guide](https://azure.github.io/azure-workload-identity/docs/quick-start.html) is a good place to get started.

This is basically a two step process:

1. Create a Kubernetes Service Account ([guide](https://azure.github.io/azure-workload-identity/docs/quick-start.html#5-create-a-kubernetes-service-account))

```sh
azwi serviceaccount create phase sa \
  --aad-application-name "${APPLICATION_NAME}" \
  --service-account-namespace "${SERVICE_ACCOUNT_NAMESPACE}" \
  --service-account-name "${SERVICE_ACCOUNT_NAME}"
```
2. Configure the trust relationship between Azure AD and Kubernetes ([guide](https://azure.github.io/azure-workload-identity/docs/quick-start.html#6-establish-federated-identity-credential-between-the-aad-application-and-the-service-account-issuer--subject))

```sh
azwi serviceaccount create phase federated-identity \
  --aad-application-name "${APPLICATION_NAME}" \
  --service-account-namespace "${SERVICE_ACCOUNT_NAMESPACE}" \
  --service-account-name "${SERVICE_ACCOUNT_NAME}" \
  --service-account-issuer-url "${SERVICE_ACCOUNT_ISSUER}"
```

With these prerequisites met you can configure External Secrets Operator to use that Service Account. You have two options:

##### Mounted Service Account
You run the controller and mount that particular service account into the pod by adding the label `azure.workload.identity/use: "true"`to the pod. That grants _everyone_ who is able to create a secret store or reference a correctly configured one the ability to read secrets. **This approach is usually not recommended**. But may make sense when you want to share an identity with multiple namespaces. Also see our [Multi-Tenancy Guide](../guides/multi-tenancy.md) for design considerations.

```yaml
{% include 'azkv-workload-identity-mounted.yaml' %}
```

##### Referenced Service Account
You run the controller without service account (effectively without azure permissions). Now you have to configure the SecretStore and set the `serviceAccountRef` and point to the service account you have just created. **This is usually the recommended approach**. It makes sense for everyone who wants to run the controller without Azure permissions and delegate authentication via service accounts in particular namespaces. Also see our [Multi-Tenancy Guide](../guides/multi-tenancy.md) for design considerations.

```yaml
{% include 'azkv-workload-identity.yaml' %}
```

In case you don't have the clientId when deploying the SecretStore, such as when deploying a Helm chart that includes instructions for creating a [Managed Identity](https://github.com/Azure/azure-service-operator/blob/main/v2/samples/managedidentity/v1api20181130/v1api20181130_userassignedidentity.yaml) using [Azure Service Operator](https://azure.github.io/azure-service-operator/) next to the SecretStore definition, you may encounter an interpolation problem. Helm lacks dependency management, which means it can create an issue when the clientId is only known after everything is deployed. Although the Service Account can inject `clientId` and `tenantId` into a pod, it doesn't support secretKeyRef/configMapKeyRef. Therefore, you can deliver the clientId and tenantId directly, bypassing the Service Account.

The following example demonstrates using the secretRef field to directly deliver the `clientId` and `tenantId` to the SecretStore while utilizing Workload Identity authentication.

```yaml
{% include 'azkv-workload-identity-secretref.yaml' %}
```

### Custom Cloud Configuration

External Secrets Operator supports custom cloud endpoints for Azure Stack Hub, Azure Stack Edge, and other scenarios where the default cloud endpoints don't match your environment. This feature requires using the new Azure SDK.

#### Azure China Workload Identity

Azure China's AKS uses a different OIDC issuer (`login.partner.microsoftonline.cn`) than the standard China Cloud endpoint (`login.chinacloudapi.cn`). When using Workload Identity with AKS in Azure China, you need to override the Active Directory endpoint:

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: azure-china-workload-identity
spec:
  provider:
    azurekv:
      vaultUrl: "https://my-vault.vault.azure.cn"
      environmentType: ChinaCloud
      authType: WorkloadIdentity
      # REQUIRED: Must be true to use custom cloud configuration
      useAzureSDK: true
      # Override the Active Directory endpoint to match AKS OIDC issuer
      customCloudConfig:
        activeDirectoryEndpoint: "https://login.partner.microsoftonline.cn/"
        keyVaultEndpoint: "https://vault.azure.cn/"
        resourceManagerEndpoint: "https://management.chinacloudapi.cn/"
      serviceAccountRef:
        name: my-service-account
        namespace: default
```

#### Azure Stack Configuration

For Azure Stack Hub or Azure Stack Edge environments:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: azure-stack-backend
spec:
  provider:
    azurekv:
      vaultUrl: "https://my-vault.vault.local.azurestack.external/"
      # REQUIRED: Must be set to AzureStackCloud for custom environments
      environmentType: AzureStackCloud
      # REQUIRED: Must be true for Azure Stack (legacy SDK doesn't support custom clouds)
      useAzureSDK: true
      # REQUIRED: Custom cloud endpoints for your Azure Stack deployment
      customCloudConfig:
        # Azure Active Directory endpoint for authentication
        activeDirectoryEndpoint: "https://login.microsoftonline.com/"
        # Optional: Key Vault endpoint if different from vaultUrl domain
        keyVaultEndpoint: "https://vault.local.azurestack.external/"
        # Optional: Resource Manager endpoint for resource operations
        resourceManagerEndpoint: "https://management.local.azurestack.external/"
      # ... rest of authentication configuration (Service Principal example)
      authType: ServicePrincipal
      tenantId: "your-tenant-id"
      authSecretRef:
        clientId:
          name: azure-secret
          key: client-id
        clientSecret:
          name: azure-secret
          key: client-secret
```

!!! note

    - `useAzureSDK: true` is mandatory when using `customCloudConfig`
    - `customCloudConfig` can be used with any `environmentType` (PublicCloud, ChinaCloud, etc.)
    - For AzureStackCloud, `customCloudConfig` is required
    - Contact your Azure Stack administrator for the correct endpoint URLs

### Object Types

Azure Key Vault has different [object types](https://docs.microsoft.com/en-us/azure/key-vault/general/about-keys-secrets-certificates#object-types); Secrets, Keys and Certificates, all of which are supported. To explicitly select which object type to fetch via an `ExternalSecret` or push via a `PushSecret`, prefix the `spec.data[].remoteRef.key` field with either `key`, `secret` or `cert`. If no prefix is provided, the operator defaults to `secret`.

| Object Type | Prefix | Description
| ----------- | ------ | ------------------------- |
| Secret      | `secret` (default) | Generic secrets, such as passwords and database connection strings. |
| Key         | `key`    | Cryptographic keys represented as JSON Web Key [JWK] objects. Azure Key Vault does **not** export the private key. You may want to use [template functions](../guides/templating.md) to transform this JWK into PEM encoded PKIX ASN.1 DER format. |
| Certificate | `cert`   | X509 certificates, for example TLS certificates. You may want to use [template functions](../guides/templating.md) to transform this into your desired encoding. |

### Creating an ExternalSecret

To synchronize secrets from Azure Key Vault into Kubernetes, you need to create an `ExternalSecret` which references the `SecretStore` or `ClusterSecretStore` you configured. You specify the name of the secret in Azure Key Vault that you want to synchronize and the name of the Kubernetes secret that should be created.

```yaml
{% include 'azkv-external-secret.yaml' %}
```

The operator will fetch the Azure Key Vault secret and inject it as a `Secret`. View the created secret by running the following command:

```sh
kubectl get secret secret-to-be-created -n <namespace> -o jsonpath='{.data.dev-secret-test}' | base64 -d
```

To select all secrets inside the key vault or all tags inside a secret, you can use the `dataFrom` directive:

```yaml
{% include 'azkv-datafrom-external-secret.yaml' %}
```

To fetch a P12 certificate (also known as PKCS12 or PFX) from Azure Key Vault and inject it as a `Secret` of type `kubernetes.io/tls`, you can use the following configuration. Note that template functions are used to transform the certificate from its original format into a PEM encoded format that Kubernetes expects for TLS secrets.

```yaml
{% include 'azkv-pkcs12-cert-external-secret.yaml' %}
```

### Creating a PushSecret
You can push secrets from Kubernetes into Azure Key Vault as secrets, keys or certificates by using a `PushSecret`. A `PushSecret` references a Kubernetes Secret as the source of the data. The operator can create, update or delete the corresponding secret in Azure Key Vault to match the desired state defined in the `PushSecret`.

#### Pushing to a Secret
Pushing to a Secret requires no previous setup. Provided you have a Kubernetes Secret available, you can create a `PushSecret` which references it to have it created on Azure Key Vault. You can optionally set metadata such as content type or tags. The operator will read the data from the Kubernetes Secret and push it to Azure Key Vault as a secret.

```yaml
{% include 'azkv-pushsecret-secret.yaml' %}
```

!!! note
    In order to create a PushSecret targeting Secrets, the [Key Vault Secrets Officer](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/security#key-vault-secrets-officer) role, alternatively Access Policy permissions `Set` and `Delete` for Secrets must be granted to the identity configured on the SecretStore.

#### Pushing to a Key
The first step is to generate a valid Private Key. Supported formats include `PRIVATE KEY`, `RSA PRIVATE KEY` AND `EC PRIVATE KEY` (EC/PKCS1/PKCS8 types). After uploading your key to a Kubernetes Secret, the next step is to create a PushSecret manifest with the following configuration:

```yaml
{% include 'azkv-pushsecret-key.yaml' %}
```

!!! note
    In order to create a PushSecret targeting keys, the [Key Vault Crypto Officer](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/security#key-vault-crypto-officer) role, alternatively Access Policy permissions `Import` and `Delete` for Keys must be granted to the identity configured on the SecretStore.

#### Pushing to a Certificate

The P12 format (also known as PKCS12 or PFX) is the most stable format for importing certificates to Azure Key Vault. The P12 file must contain both the certificate and the private key. Make sure to use PKCS8 format for the private key, as Azure Key Vault does not support PKCS1 format. Additionally, only password-less P12 files are supported.

Provided you have a Kubernetes Secret with a P12 certificate, you can push it to Azure Key Vault by creating a `PushSecret` with the following configuration. The operator will read the P12 file from the Kubernetes Secret and import it into Azure Key Vault as a certificate.

```yaml
{% include 'azkv-pushsecret-certificate-p12.yaml' %}
```

Provided you have a Kubernetes Secret with the certificate and private key in PEM format, you can use the following configuration to transform it into a P12 file and push it to Azure Key Vault. The operator will read the certificate and private key from the Kubernetes Secret, convert them into a P12 file using template functions, and import it into Azure Key Vault as a certificate.

```yaml
{% include 'azkv-pushsecret-certificate-pem.yaml' %}
```

If you are using [cert-manager](https://cert-manager.io/) and its `Certificate` resource to generate the Kubernetes Secret in a PEM format as shown above, make sure to set `spec.privateKey.encoding` to `PKCS8`. By default, cert-manager generates private keys with PKCS1 encoding, which is not supported by Azure Key Vault.

```yaml hl_lines="14"
{% include 'azkv-pushsecret-certificate-cert-manager.yaml' %}
```

!!! note
    In order to create a PushSecret targeting certificates, the [Key Vault Certificates Officer](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/security#key-vault-certificates-officer) role, alternatively Access Policy permissions `Import` and `Delete` for Certificates must be granted to the identity configured on the SecretStore.
