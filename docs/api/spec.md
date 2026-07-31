# API Reference

## Packages
- [external-secrets.io/externalsecrets](#external-secretsioexternalsecrets)
- [external-secrets.io/v1](#external-secretsiov1)
- [external-secrets.io/v1alpha1](#external-secretsiov1alpha1)
- [external-secrets.io/v1beta1](#external-secretsiov1beta1)
- [generators.external-secrets.io/v1alpha1](#generatorsexternal-secretsiov1alpha1)


## external-secrets.io/externalsecrets

Package externalsecrets contains API Schema definitions for the externalsecrets API groups.
Currently, we have v1, v1alpha1 and v1beta1 versions.




## external-secrets.io/v1

Package v1 contains resources for external-secrets

### Resource Types
- [ClusterExternalSecret](#clusterexternalsecret)
- [ClusterSecretStore](#clustersecretstore)
- [ExternalSecret](#externalsecret)
- [GenericStore](#genericstore)
- [Provider](#provider)
- [PushSecretData](#pushsecretdata)
- [PushSecretRemoteRef](#pushsecretremoteref)
- [SecretStore](#secretstore)
- [SecretsClient](#secretsclient)



#### AWSAuth



AWSAuth tells the controller how to do authentication with aws.
Only one of secretRef or jwt can be specified.
if none is specified the controller will load credentials using the aws sdk defaults.



_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AWSAuthSecretRef](#awsauthsecretref)_ |  |  | Optional: \{\} <br /> |
| `jwt` _[AWSJWTAuth](#awsjwtauth)_ |  |  | Optional: \{\} <br /> |


#### AWSAuthSecretRef



AWSAuthSecretRef holds secret references for AWS credentials
both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.



_Appears in:_
- [AWSAuth](#awsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessKeyIDSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeyID is used for authentication |  |  |
| `secretAccessKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  |  |
| `sessionTokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SessionToken used for authentication<br />This must be defined if AccessKeyID and SecretAccessKey are temporary credentials<br />see: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html |  |  |


#### AWSJWTAuth



AWSJWTAuth stores reference to Authenticate against AWS using service account tokens.



_Appears in:_
- [AWSAuth](#awsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ |  |  |  |


#### AWSProvider



AWSProvider configures a store to sync secrets with AWS.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `service` _[AWSServiceType](#awsservicetype)_ | Service defines which service should be used to fetch the secrets |  | Enum: [SecretsManager ParameterStore CertificateManager] <br /> |
| `auth` _[AWSAuth](#awsauth)_ | Auth defines the information necessary to authenticate against AWS<br />if not set aws sdk will infer credentials from your environment<br />see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials |  | Optional: \{\} <br /> |
| `role` _string_ | Role is a Role ARN which the provider will assume |  | Optional: \{\} <br /> |
| `region` _string_ | AWS Region to be used for the provider |  |  |
| `additionalRoles` _string array_ | AdditionalRoles is a chained list of Role ARNs which the provider will sequentially assume before assuming the Role |  | Optional: \{\} <br /> |
| `externalID` _string_ | AWS External ID set on assumed IAM roles |  |  |
| `sessionTags` _[Tag](#tag) array_ | AWS STS assume role session tags |  | Optional: \{\} <br /> |
| `secretsManager` _[SecretsManager](#secretsmanager)_ | SecretsManager defines how the provider behaves when interacting with AWS SecretsManager |  | Optional: \{\} <br /> |
| `transitiveTagKeys` _string array_ | AWS STS assume role transitive session tags. Required when multiple rules are used with the provider |  | Optional: \{\} <br /> |
| `sessionTagsPolicy` _[SessionTagsPolicy](#sessiontagspolicy)_ | SessionTagsPolicy controls whether and how STS session tags are added when assuming roles.<br />None (default): no tags are added.<br />Simple: automatically adds esoNamespace (from the ExternalSecret), esoStoreName, and esoStoreKind tags.<br />Custom: adds esoNamespace, esoStoreName, and esoStoreKind plus any tags defined in CustomSessionTags.<br />Note: the IAM role must have sts:TagSession permission when using Simple or Custom. | None | Enum: [None Simple Custom] <br />Optional: \{\} <br /> |
| `customSessionTags` _object (keys:string, values:string)_ | CustomSessionTags defines additional STS session tags to include when SessionTagsPolicy is Custom.<br />These are merged with the automatically injected esoNamespace, esoStoreName, and esoStoreKind tags. |  | Optional: \{\} <br /> |
| `prefix` _string_ | Prefix adds a prefix to all retrieved values. |  | Optional: \{\} <br /> |


#### AWSServiceType

_Underlying type:_ _string_

AWSServiceType is a enum that defines the service/API that is used to fetch the secrets.

_Validation:_
- Enum: [SecretsManager ParameterStore CertificateManager]

_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description |
| --- | --- |
| `SecretsManager` | AWSServiceSecretsManager is the AWS SecretsManager service.<br />see: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html<br /> |
| `ParameterStore` | AWSServiceParameterStore is the AWS SystemsManager ParameterStore service.<br />see: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html<br /> |
| `CertificateManager` | AWSServiceCertificateManager is the AWS Certificate Manager service.<br />see: https://docs.aws.amazon.com/acm/latest/userguide/acm-overview.html<br /> |


#### AkeylessAuth



AkeylessAuth configures how the operator authenticates with Akeyless.



_Appears in:_
- [AkeylessProvider](#akeylessprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AkeylessAuthSecretRef](#akeylessauthsecretref)_ | Reference to a Secret that contains the details<br />to authenticate with Akeyless. |  | Optional: \{\} <br /> |
| `kubernetesAuth` _[AkeylessKubernetesAuth](#akeylesskubernetesauth)_ | Kubernetes authenticates with Akeyless by passing the ServiceAccount<br />token stored in the named Secret resource. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specifies a Kubernetes ServiceAccount used for azure_ad<br />authentication on AKS Workload Identity. The operator obtains a federated<br />identity token from this ServiceAccount via the TokenRequest API instead<br />of using the ESO controller pod identity. Ignored for other access types. |  | Optional: \{\} <br /> |


#### AkeylessAuthSecretRef



AkeylessAuthSecretRef references a Secret that contains the details
to authenticate with Akeyless.
AKEYLESS_ACCESS_TYPE_PARAM: AZURE_OBJ_ID OR GCP_AUDIENCE OR ACCESS_KEY OR KUB_CONFIG_NAME.



_Appears in:_
- [AkeylessAuth](#akeylessauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessID` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessID is used for authentication |  |  |
| `accessType` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `accessTypeParam` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### AkeylessKubernetesAuth



AkeylessKubernetesAuth configures Kubernetes authentication with Akeyless.
It authenticates with Kubernetes ServiceAccount token stored.



_Appears in:_
- [AkeylessAuth](#akeylessauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessID` _string_ | the Akeyless Kubernetes auth-method access-id |  |  |
| `k8sConfName` _string_ | Kubernetes-auth configuration name in Akeyless-Gateway |  |  |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Optional service account field containing the name of a kubernetes ServiceAccount.<br />If the service account is specified, the service account secret token JWT will be used<br />for authenticating with Akeyless. If the service account selector is not supplied,<br />the secretRef will be used instead. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional secret field containing a Kubernetes ServiceAccount JWT used<br />for authenticating with Akeyless. If a name is specified without a key,<br />`token` is the default. If one is not specified, the one bound to<br />the controller will be used. |  | Optional: \{\} <br /> |


#### AkeylessProvider



AkeylessProvider Configures an store to sync secrets using Akeyless KV.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `akeylessGWApiURL` _string_ | Akeyless GW API Url from which the secrets to be fetched from. |  |  |
| `ignoreCache` _boolean_ | IgnoreCache bypasses the Gateway cache for secret reads when true.<br />Only relevant when akeylessGWApiURL points to an Akeyless Gateway. |  | Optional: \{\} <br /> |
| `authSecretRef` _[AkeylessAuth](#akeylessauth)_ | Auth configures how the operator authenticates with Akeyless. |  |  |
| `caBundle` _integer array_ | PEM/base64 encoded CA bundle used to validate Akeyless Gateway certificate. Only used<br />if the AkeylessGWApiURL URL is using HTTPS protocol. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | The provider for the CA bundle to use to validate Akeyless Gateway certificate. |  | Optional: \{\} <br /> |


#### AuthorizationProtocol



AuthorizationProtocol contains the protocol-specific configuration

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ntlm` _[NTLMProtocol](#ntlmprotocol)_ | NTLMProtocol configures the store to use NTLM for auth |  | Optional: \{\} <br /> |


#### AwsAuthCredentials



AwsAuthCredentials represents the credentials for AWS authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### AwsCredentialsConfig



AwsCredentialsConfig holds the region and the Secret reference which contains the AWS credentials.



_Appears in:_
- [GCPWorkloadIdentityFederation](#gcpworkloadidentityfederation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `region` _string_ | region is for configuring the AWS region to be used. |  | MaxLength: 50 <br />MinLength: 1 <br />Pattern: `^[a-z0-9-]+$` <br />Required: \{\} <br /> |
| `awsCredentialsSecretRef` _[SecretReference](#secretreference)_ | awsCredentialsSecretRef is the reference to the secret which holds the AWS credentials.<br />Secret should be created with below names for keys<br />- aws_access_key_id: Access Key ID, which is the unique identifier for the AWS account or the IAM user.<br />- aws_secret_access_key: Secret Access Key, which is used to authenticate requests made to AWS services.<br />- aws_session_token: Session Token, is the short-lived token to authenticate requests made to AWS services. |  | Required: \{\} <br /> |


#### AzureAuthCredentials



AzureAuthCredentials represents the credentials for Azure authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `resource` _[SecretKeySelector](#secretkeyselector)_ |  |  | Optional: \{\} <br /> |


#### AzureAuthType

_Underlying type:_ _string_

AzureAuthType describes how to authenticate to the Azure Keyvault
Only one of the following auth types may be specified.
If none of the following auth type is specified, the default one
is ServicePrincipal.

_Validation:_
- Enum: [ServicePrincipal ManagedIdentity WorkloadIdentity]

_Appears in:_
- [AzureKVProvider](#azurekvprovider)

| Field | Description |
| --- | --- |
| `ServicePrincipal` | AzureServicePrincipal uses service principal to authenticate, which needs a tenantId, a clientId and a clientSecret.<br /> |
| `ManagedIdentity` | AzureManagedIdentity uses Managed Identity to authenticate. Used with aad-pod-identity installed in the cluster.<br /> |
| `WorkloadIdentity` | AzureWorkloadIdentity uses Workload Identity service accounts to authenticate.<br /> |


#### AzureCustomCloudConfig



AzureCustomCloudConfig specifies custom cloud configuration for private Azure environments
IMPORTANT: Custom cloud configuration is ONLY supported when UseAzureSDK is true.
The legacy go-autorest SDK does not support custom cloud endpoints.



_Appears in:_
- [AzureKVProvider](#azurekvprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `activeDirectoryEndpoint` _string_ | ActiveDirectoryEndpoint is the AAD endpoint for authentication<br />Required when using custom cloud configuration |  | Required: \{\} <br /> |
| `keyVaultEndpoint` _string_ | KeyVaultEndpoint is the Key Vault service endpoint |  | Optional: \{\} <br /> |
| `keyVaultDNSSuffix` _string_ | KeyVaultDNSSuffix is the DNS suffix for Key Vault URLs |  | Optional: \{\} <br /> |
| `resourceManagerEndpoint` _string_ | ResourceManagerEndpoint is the Azure Resource Manager endpoint |  | Optional: \{\} <br /> |


#### AzureEnvironmentType

_Underlying type:_ _string_

AzureEnvironmentType specifies the Azure cloud environment endpoints to use for
connecting and authenticating with Azure. By default, it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud, AzureStackCloud

_Validation:_
- Enum: [PublicCloud USGovernmentCloud ChinaCloud GermanCloud AzureStackCloud]

_Appears in:_
- [ACRAccessTokenSpec](#acraccesstokenspec)
- [AzureKVProvider](#azurekvprovider)

| Field | Description |
| --- | --- |
| `PublicCloud` |  |
| `USGovernmentCloud` |  |
| `ChinaCloud` |  |
| `GermanCloud` |  |
| `AzureStackCloud` |  |


#### AzureKVAuth



AzureKVAuth is the configuration used to authenticate with Azure.



_Appears in:_
- [AzureKVProvider](#azurekvprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _[SecretKeySelector](#secretkeyselector)_ | The Azure clientId of the service principle or managed identity used for authentication. |  | Optional: \{\} <br /> |
| `tenantId` _[SecretKeySelector](#secretkeyselector)_ | The Azure tenantId of the managed identity used for authentication. |  | Optional: \{\} <br /> |
| `clientSecret` _[SecretKeySelector](#secretkeyselector)_ | The Azure ClientSecret of the service principle used for authentication. |  | Optional: \{\} <br /> |
| `clientCertificate` _[SecretKeySelector](#secretkeyselector)_ | The Azure ClientCertificate of the service principle used for authentication. |  | Optional: \{\} <br /> |


#### AzureKVProvider



AzureKVProvider configures a store to sync secrets using Azure KV.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authType` _[AzureAuthType](#azureauthtype)_ | Auth type defines how to authenticate to the keyvault service.<br />Valid values are:<br />- "ServicePrincipal" (default): Using a service principal (tenantId, clientId, clientSecret)<br />- "ManagedIdentity": Using Managed Identity assigned to the pod (see aad-pod-identity)<br />- "WorkloadIdentity": Using a Kubernetes ServiceAccount federated with Entra ID | ServicePrincipal | Enum: [ServicePrincipal ManagedIdentity WorkloadIdentity] <br />Optional: \{\} <br /> |
| `vaultUrl` _string_ | Vault Url from which the secrets to be fetched from. |  |  |
| `tenantId` _string_ | TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type. Optional for WorkloadIdentity. |  | Optional: \{\} <br /> |
| `environmentType` _[AzureEnvironmentType](#azureenvironmenttype)_ | EnvironmentType specifies the Azure cloud environment endpoints to use for<br />connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.<br />The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152<br />PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud, AzureStackCloud<br />Use AzureStackCloud when you need to configure custom Azure Stack Hub or Azure Stack Edge endpoints. | PublicCloud | Enum: [PublicCloud USGovernmentCloud ChinaCloud GermanCloud AzureStackCloud] <br /> |
| `authSecretRef` _[AzureKVAuth](#azurekvauth)_ | Auth configures how the operator authenticates with Azure. Required for ServicePrincipal auth type. Optional for WorkloadIdentity. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specified the service account<br />that should be used when authenticating with WorkloadIdentity. |  | Optional: \{\} <br /> |
| `identityId` _string_ | If multiple Managed Identity is assigned to the pod, you can select the one to be used |  | Optional: \{\} <br /> |
| `useAzureSDK` _boolean_ | UseAzureSDK enables the use of the new Azure SDK for Go (azcore-based) instead of the legacy go-autorest SDK.<br />This is experimental and may have behavioral differences. Defaults to false (legacy SDK). | false | Optional: \{\} <br /> |
| `customCloudConfig` _[AzureCustomCloudConfig](#azurecustomcloudconfig)_ | CustomCloudConfig defines custom Azure endpoints for non-standard clouds.<br />Required when EnvironmentType is AzureStackCloud.<br />Optional for other environment types - useful for Azure China when using Workload Identity<br />with AKS, where the OIDC issuer (login.partner.microsoftonline.cn) differs from the<br />standard China Cloud endpoint (login.chinacloudapi.cn).<br />IMPORTANT: This feature REQUIRES UseAzureSDK to be set to true. Custom cloud<br />configuration is not supported with the legacy go-autorest SDK. |  | Optional: \{\} <br /> |


#### BarbicanAuth



BarbicanAuth contains the authentication information for Barbican.



_Appears in:_
- [BarbicanProvider](#barbicanprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `username` _[BarbicanProviderUsernameRef](#barbicanproviderusernameref)_ |  |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `password` _[BarbicanProviderPasswordRef](#barbicanproviderpasswordref)_ |  |  |  |


#### BarbicanProvider



BarbicanProvider setup a store to sync secrets with barbican.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authURL` _string_ |  |  |  |
| `tenantName` _string_ |  |  |  |
| `domainName` _string_ |  |  |  |
| `region` _string_ |  |  |  |
| `auth` _[BarbicanAuth](#barbicanauth)_ |  |  |  |


#### BarbicanProviderPasswordRef



BarbicanProviderPasswordRef defines a reference to a secret containing password for the Barbican provider.



_Appears in:_
- [BarbicanAuth](#barbicanauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### BarbicanProviderUsernameRef



BarbicanProviderUsernameRef defines a reference to a secret containing username for the Barbican provider.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [BarbicanAuth](#barbicanauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ |  |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### BeyondTrustProviderSecretRef



BeyondTrustProviderSecretRef references a value that can be specified directly or via a secret
for a BeyondTrustProvider.



_Appears in:_
- [BeyondtrustAuth](#beyondtrustauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### BeyondtrustAuth



BeyondtrustAuth provides different ways to authenticate to a BeyondtrustProvider server.



_Appears in:_
- [BeyondtrustProvider](#beyondtrustprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKey` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | APIKey If not provided then ClientID/ClientSecret become required. |  |  |
| `clientId` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | ClientID is the API OAuth Client ID. |  |  |
| `clientSecret` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | ClientSecret is the API OAuth Client Secret. |  |  |
| `certificate` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | Certificate (cert.pem) for use when authenticating with an OAuth client Id using a Client Certificate. |  |  |
| `certificateKey` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | Certificate private key (key.pem). For use when authenticating with an OAuth client Id |  |  |


#### BeyondtrustProvider



BeyondtrustProvider provides access to a BeyondTrust secrets provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[BeyondtrustAuth](#beyondtrustauth)_ | Auth configures how the operator authenticates with Beyondtrust. |  |  |
| `server` _[BeyondtrustServer](#beyondtrustserver)_ | Auth configures how API server works. |  |  |


#### BeyondtrustServer



BeyondtrustServer configures a store to sync secrets using BeyondTrust Password Safe.



_Appears in:_
- [BeyondtrustProvider](#beyondtrustprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ |  |  |  |
| `apiVersion` _string_ |  |  |  |
| `retrievalType` _string_ | The secret retrieval type. SECRET = Secrets Safe (credential, text, file). MANAGED_ACCOUNT = Password Safe account associated with a system. |  |  |
| `separator` _string_ | A character that separates the folder names. |  |  |
| `decrypt` _boolean_ | When true, the response includes the decrypted password. When false, the password field is omitted. This option only applies to the SECRET retrieval type. Default: true. | true | Optional: \{\} <br /> |
| `verifyCA` _boolean_ |  |  |  |
| `clientTimeOutSeconds` _integer_ | Timeout specifies a time limit for requests made by this Client. The timeout includes connection time, any redirects, and reading the response body. Defaults to 45 seconds. |  |  |


#### BeyondtrustWorkloadCredentialsAuth



BeyondtrustWorkloadCredentialsAuth defines the authentication method for the BeyondTrust Workload Credentials provider.
Currently supports API key authentication via Kubernetes secret reference.
For authentication documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication



_Appears in:_
- [BeyondtrustWorkloadCredentialsProvider](#beyondtrustworkloadcredentialsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apikey` _[BeyondtrustWorkloadCredentialsAuthSecretRef](#beyondtrustworkloadcredentialsauthsecretref)_ | APIKey configures API token authentication for BeyondTrust Workload Credentials.<br />The token is retrieved from a Kubernetes secret and used as a Bearer token for API requests. |  |  |


#### BeyondtrustWorkloadCredentialsAuthSecretRef



BeyondtrustWorkloadCredentialsAuthSecretRef defines a reference to a secret containing credentials for the BeyondTrust Workload Credentials provider.
The nested structure supports multiple authentication methods (currently only API token is supported).
For more information on authentication, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication



_Appears in:_
- [BeyondtrustWorkloadCredentialsAuth](#beyondtrustworkloadcredentialsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `token` _[SecretKeySelector](#secretkeyselector)_ | Token references the Kubernetes secret containing the BeyondTrust Workload Credentials API token.<br />The secret should contain the API key used to authenticate with BeyondTrust Workload Credentials.<br />Create an API token in your BeyondTrust Workload Credentials console and store it in a Kubernetes secret.<br />For details on creating API tokens, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication |  |  |


#### BeyondtrustWorkloadCredentialsProvider



BeyondtrustWorkloadCredentialsProvider configures a store to sync secrets using the BeyondTrust Workload Credentials provider.
BeyondTrust Workload Credentials provides secure storage for static secrets and dynamic credential generation.
This provider supports reading secrets and generating dynamic credentials (e.g., temporary AWS credentials).
For complete documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api



_Appears in:_
- [BeyondtrustWorkloadCredentialsDynamicSecretSpec](#beyondtrustworkloadcredentialsdynamicsecretspec)
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[BeyondtrustWorkloadCredentialsAuth](#beyondtrustworkloadcredentialsauth)_ | Auth configures how the Operator authenticates with the BeyondTrust Workload Credentials API.<br />Currently supports API key authentication via Kubernetes secret reference.<br />For authentication setup, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#authentication |  | Required: \{\} <br /> |
| `server` _[BeyondtrustWorkloadCredentialsServer](#beyondtrustworkloadcredentialsserver)_ | Server configures the BeyondTrust Workload Credentials server connection details.<br />Includes the API URL and Site ID for your BeyondTrust instance.<br />For API reference, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api |  | Required: \{\} <br /> |
| `folderPath` _string_ | FolderPath specifies the default folder path for secret retrieval.<br />Secrets will be fetched from this folder unless overridden in the ExternalSecret spec.<br />Example: "production/database" or "dev/api-keys"<br />Leave empty to retrieve secrets from the root folder.<br />For folder organization, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#folders |  | Optional: \{\} <br /> |
| `caBundle` _integer array_ | CABundle is a base64-encoded CA certificate used to validate the BeyondTrust Workload Credentials API TLS certificate.<br />Use this when your BeyondTrust instance uses a self-signed certificate or internal CA.<br />If not set, the system's trusted root certificates are used. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | CAProvider points to a Secret or ConfigMap containing a PEM-encoded CA certificate.<br />This is used to validate the BeyondTrust Workload Credentials API TLS certificate.<br />Use this as an alternative to CABundle when you want to reference an existing Kubernetes resource. |  | Optional: \{\} <br /> |


#### BeyondtrustWorkloadCredentialsServer



BeyondtrustWorkloadCredentialsServer defines connection configuration for BeyondTrust Workload Credentials.
For API reference documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api



_Appears in:_
- [BeyondtrustWorkloadCredentialsProvider](#beyondtrustworkloadcredentialsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the base URL of your BeyondTrust Workload Credentials API server.<br />This should be the full URL to your BeyondTrust instance.<br />Example: https://api.beyondtrust.io/siie<br />For more information, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api#base-url |  | Required: \{\} <br /> |
| `siteId` _string_ | SiteID is your BeyondTrust Workload Credentials site identifier (UUID format).<br />This identifier is unique to your BeyondTrust Workload Credentials instance.<br />You can find your Site ID in the BeyondTrust Workload Credentials admin console.<br />Example: a1b2c3d4-e5f6-4890-abcd-ef1234567890<br />For more information, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api |  | Required: \{\} <br /> |


#### BitwardenSecretsManagerAuth



BitwardenSecretsManagerAuth contains the ref to the secret that contains the machine account token.



_Appears in:_
- [BitwardenSecretsManagerProvider](#bitwardensecretsmanagerprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[BitwardenSecretsManagerSecretRef](#bitwardensecretsmanagersecretref)_ |  |  |  |


#### BitwardenSecretsManagerProvider



BitwardenSecretsManagerProvider configures a store to sync secrets with a Bitwarden Secrets Manager instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiURL` _string_ |  |  |  |
| `identityURL` _string_ |  |  |  |
| `bitwardenServerSDKURL` _string_ |  |  |  |
| `caBundle` _string_ | Base64 encoded certificate for the bitwarden server sdk. The sdk MUST run with HTTPS to make sure no MITM attack<br />can be performed. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | see: https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider |  | Optional: \{\} <br /> |
| `organizationID` _string_ | OrganizationID determines which organization this secret store manages. |  |  |
| `projectID` _string_ | ProjectID determines which project this secret store manages. |  |  |
| `auth` _[BitwardenSecretsManagerAuth](#bitwardensecretsmanagerauth)_ | Auth configures how secret-manager authenticates with a bitwarden machine account instance.<br />Make sure that the token being used has permissions on the given secret. |  |  |


#### BitwardenSecretsManagerSecretRef



BitwardenSecretsManagerSecretRef contains the credential ref to the bitwarden instance.



_Appears in:_
- [BitwardenSecretsManagerAuth](#bitwardensecretsmanagerauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `credentials` _[SecretKeySelector](#secretkeyselector)_ | AccessToken used for the bitwarden instance. |  | Required: \{\} <br /> |


#### ByID



ByID configures the provider to interpret the `data.secretKey.remoteRef.key` field in ExternalSecret as secret ID.



_Appears in:_
- [FetchingPolicy](#fetchingpolicy)



#### ByName



ByName configures the provider to interpret the `data.secretKey.remoteRef.key` field in ExternalSecret as secret name.



_Appears in:_
- [FetchingPolicy](#fetchingpolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `folderID` _string_ | The folder to fetch secrets from |  |  |


#### CAProvider



CAProvider provides a custom certificate authority for accessing the provider's store.
The CAProvider points to a Secret or ConfigMap resource that contains a PEM-encoded certificate.



_Appears in:_
- [AkeylessProvider](#akeylessprovider)
- [BeyondtrustWorkloadCredentialsProvider](#beyondtrustworkloadcredentialsprovider)
- [BitwardenSecretsManagerProvider](#bitwardensecretsmanagerprovider)
- [ConjurProvider](#conjurprovider)
- [GitlabProvider](#gitlabprovider)
- [InfisicalProvider](#infisicalprovider)
- [KubernetesServer](#kubernetesserver)
- [OpenBaoProvider](#openbaoprovider)
- [OvhClientMTLS](#ovhclientmtls)
- [PassboltProvider](#passboltprovider)
- [SecretServerProvider](#secretserverprovider)
- [VaultProvider](#vaultprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[CAProviderType](#caprovidertype)_ | The type of provider to use such as "Secret", or "ConfigMap". |  | Enum: [Secret ConfigMap] <br /> |
| `name` _string_ | The name of the object located at the provider type. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `key` _string_ | The key where the CA certificate can be found in the Secret or ConfigMap. |  | MaxLength: 253 <br />MinLength: 1 <br />Optional: \{\} <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |
| `namespace` _string_ | The namespace the Provider type is in.<br />Can only be defined when used in a ClusterSecretStore. |  | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


#### CAProviderType

_Underlying type:_ _string_

CAProviderType defines the type of provider for certificate authority.



_Appears in:_
- [CAProvider](#caprovider)

| Field | Description |
| --- | --- |
| `Secret` | CAProviderTypeSecret indicates that the CA certificate is stored in a Secret resource.<br /> |
| `ConfigMap` | CAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap resource.<br /> |


#### CRDProvider



CRDProvider configures a store to fetch data from arbitrary Kubernetes
resources, including both custom resources (CRDs) and core API resources
(e.g. ConfigMap, addressed by setting resource.group to ""). Kubernetes
Secrets are intentionally blocked; use the Kubernetes provider for those.

Authentication modes:

In-cluster: set auth.serviceAccount and omit server. The server URL defaults
to the in-cluster API (kubernetes.default) and the controller mints a
short-lived token for the referenced ServiceAccount to read CRDs locally.

Remote cluster: set server plus auth (serviceAccount, token, or cert) or
authRef (a kubeconfig Secret), exactly like the Kubernetes provider.

Remote reference keys:

  - SecretStore: the key is the object name only; '/' is not allowed. The API
    namespace is always the store namespace, never part of the key.
  - ClusterSecretStore: use "namespace/objectName" to read a namespaced CR;
    a key without '/' addresses a cluster-scoped CR by object name. For
    dataFrom Find with a namespaced kind, listing spans all namespaces and
    result keys are "namespace/objectName".

_Validation:_
- AtMostOneOf: [auth authRef]

_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `server` _[KubernetesServer](#kubernetesserver)_ | Server configures the Kubernetes API address and TLS trust, same as the<br />Kubernetes provider. When omitted, the URL defaults to the in-cluster API. |  | Optional: \{\} <br /> |
| `auth` _[KubernetesAuth](#kubernetesauth)_ | Auth configures authentication to the Kubernetes API, same as the<br />Kubernetes provider. Required when Server.URL is set (unless using AuthRef). |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `authRef` _[SecretKeySelector](#secretkeyselector)_ | AuthRef references a Secret containing a kubeconfig. Same semantics as the<br />Kubernetes provider. |  | Optional: \{\} <br /> |
| `resource` _[CRDProviderResource](#crdproviderresource)_ | Resource identifies the CRD by its API group, version and kind. |  | Required: \{\} <br /> |
| `whitelist` _[CRDProviderWhitelist](#crdproviderwhitelist)_ | Whitelist optionally restricts which object names and requested properties<br />are allowed to be read. |  | Optional: \{\} <br /> |


#### CRDProviderResource



CRDProviderResource identifies a Kubernetes resource (CRD or core) by its
full API coordinates: group, version and kind.



_Appears in:_
- [CRDProvider](#crdprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `group` _string_ | Group is the API group of the resource. Use "" (empty string) for core<br />Kubernetes resources such as ConfigMap; use e.g. "config.example.io"<br />for a CRD. The field is required to be present in the manifest — write<br />`group: ""` explicitly for core resources so typos fail at admission<br />time rather than later at discovery. |  | Required: \{\} <br /> |
| `version` _string_ | Version is the API version of the resource (e.g. "v1alpha1"). |  | MinLength: 1 <br />Required: \{\} <br /> |
| `kind` _string_ | Kind is the Kubernetes resource kind (e.g. "MyCustomResource"). |  | MinLength: 1 <br />Required: \{\} <br /> |


#### CRDProviderWhitelist



CRDProviderWhitelist configures allow-list rules for CRD reads.
If any rules are present, a request must satisfy ALL non-empty filters of at
least one rule; requests that match no rule are denied.



_Appears in:_
- [CRDProvider](#crdprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `rules` _[CRDProviderWhitelistRule](#crdproviderwhitelistrule) array_ | Rules is a list of allow rules. If rules are set, at least one rule must<br />match for a request to be allowed. |  | Optional: \{\} <br /> |


#### CRDProviderWhitelistRule



CRDProviderWhitelistRule defines a single allow rule for CRD reads.



_Appears in:_
- [CRDProviderWhitelist](#crdproviderwhitelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is an optional regular expression matched against the bare object name.<br />For both SecretStore and ClusterSecretStore this is always the object name<br />without any namespace prefix (e.g. "my-db-spec", not "prod/my-db-spec"). |  | Optional: \{\} <br /> |
| `namespace` _string_ | Namespace is an optional regular expression matched against the namespace of<br />the object. Applies only when a ClusterSecretStore is used; it is ignored<br />for SecretStore (where the namespace is fixed to the store namespace). |  | Optional: \{\} <br /> |
| `properties` _string array_ | Properties is an optional list of regular expressions matched against<br />requested property keys (for example: "spec.secretValue"). |  | Optional: \{\} <br /> |


#### CSMAuth



CSMAuth contains a secretRef for credentials.



_Appears in:_
- [CloudruSMProvider](#cloudrusmprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[CSMAuthSecretRef](#csmauthsecretref)_ |  |  | Optional: \{\} <br /> |


#### CSMAuthSecretRef



CSMAuthSecretRef holds secret references for Cloud.ru credentials.



_Appears in:_
- [CSMAuth](#csmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessKeyIDSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeyID is used for authentication |  |  |
| `accessKeySecretSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeySecret is used for authentication |  |  |


#### CacheConfig



CacheConfig configures client-side caching for read operations.



_Appears in:_
- [OnePasswordSDKProvider](#onepasswordsdkprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ttl` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | TTL is the time-to-live for cached secrets.<br />Format: duration string (e.g., "5m", "1h", "30s") | 5m | Optional: \{\} <br /> |
| `maxSize` _integer_ | MaxSize is the maximum number of secrets to cache.<br />When the cache is full, least-recently-used entries are evicted. | 100 | Minimum: 1 <br />Optional: \{\} <br /> |


#### CertAuth



CertAuth defines certificate-based authentication configuration for Kubernetes.



_Appears in:_
- [KubernetesAuth](#kubernetesauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientCert` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `clientKey` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### ChefAuth



ChefAuth contains a secretRef for credentials.



_Appears in:_
- [ChefProvider](#chefprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[ChefAuthSecretRef](#chefauthsecretref)_ |  |  |  |


#### ChefAuthSecretRef



ChefAuthSecretRef holds secret references for chef server login credentials.



_Appears in:_
- [ChefAuth](#chefauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `privateKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretKey is the Signing Key in PEM format, used for authentication. |  |  |


#### ChefProvider



ChefProvider configures a store to sync secrets using basic chef server connection credentials.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[ChefAuth](#chefauth)_ | Auth defines the information necessary to authenticate against chef Server |  |  |
| `username` _string_ | UserName should be the user ID on the chef server |  |  |
| `serverUrl` _string_ | ServerURL is the chef server URL used to connect to. If using orgs you should include your org in the url and terminate the url with a "/" |  |  |


#### CloudruSMProvider



CloudruSMProvider configures a store to sync secrets using the Cloud.ru Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[CSMAuth](#csmauth)_ |  |  |  |
| `projectID` _string_ | ProjectID is the project, which the secrets are stored in. |  |  |


#### ClusterExternalSecret



ClusterExternalSecret is the Schema for the clusterexternalsecrets API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1` | | |
| `kind` _string_ | `ClusterExternalSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ClusterExternalSecretSpec](#clusterexternalsecretspec)_ |  |  |  |
| `status` _[ClusterExternalSecretStatus](#clusterexternalsecretstatus)_ |  |  |  |


#### ClusterExternalSecretConditionType

_Underlying type:_ _string_

ClusterExternalSecretConditionType defines a value type for ClusterExternalSecret conditions.



_Appears in:_
- [ClusterExternalSecretStatusCondition](#clusterexternalsecretstatuscondition)

| Field | Description |
| --- | --- |
| `Ready` |  |


#### ClusterExternalSecretNamespaceFailure



ClusterExternalSecretNamespaceFailure represents a failed namespace deployment and it's reason.



_Appears in:_
- [ClusterExternalSecretStatus](#clusterexternalsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespace` _string_ | Namespace is the namespace that failed when trying to apply an ExternalSecret |  |  |
| `reason` _string_ | Reason is why the ExternalSecret failed to apply to the namespace |  | Optional: \{\} <br /> |


#### ClusterExternalSecretSpec



ClusterExternalSecretSpec defines the desired state of ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecret](#clusterexternalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalSecretSpec` _[ExternalSecretSpec](#externalsecretspec)_ | The spec for the ExternalSecrets to be created |  |  |
| `externalSecretName` _string_ | The name of the external secrets to be created.<br />Defaults to the name of the ClusterExternalSecret |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Optional: \{\} <br /> |
| `externalSecretMetadata` _[ExternalSecretMetadata](#externalsecretmetadata)_ | The metadata of the external secrets to be created |  | Optional: \{\} <br /> |
| `namespaceSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | The labels to select by to find the Namespaces to create the ExternalSecrets in.<br />Deprecated: Use NamespaceSelectors instead. |  | Optional: \{\} <br /> |
| `namespaceSelectors` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta) array_ | A list of labels to select by to find the Namespaces to create the ExternalSecrets in. The selectors are ORed. |  | Optional: \{\} <br /> |
| `namespaces` _string array_ | Choose namespaces by name. This field is ORed with anything that NamespaceSelectors ends up choosing.<br />Deprecated: Use NamespaceSelectors instead. |  | items:MaxLength: 63 <br />items:MinLength: 1 <br />items:Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |
| `refreshTime` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | The time in which the controller should reconcile its objects and recheck namespaces for labels. |  |  |


#### ClusterExternalSecretStatus



ClusterExternalSecretStatus defines the observed state of ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecret](#clusterexternalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalSecretName` _string_ | ExternalSecretName is the name of the ExternalSecrets created by the ClusterExternalSecret |  |  |
| `failedNamespaces` _[ClusterExternalSecretNamespaceFailure](#clusterexternalsecretnamespacefailure) array_ | Failed namespaces are the namespaces that failed to apply an ExternalSecret |  | Optional: \{\} <br /> |
| `provisionedNamespaces` _string array_ | ProvisionedNamespaces are the namespaces where the ClusterExternalSecret has secrets |  | Optional: \{\} <br /> |
| `conditions` _[ClusterExternalSecretStatusCondition](#clusterexternalsecretstatuscondition) array_ |  |  | Optional: \{\} <br /> |


#### ClusterExternalSecretStatusCondition



ClusterExternalSecretStatusCondition defines the observed state of a ClusterExternalSecret resource.



_Appears in:_
- [ClusterExternalSecretStatus](#clusterexternalsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ClusterExternalSecretConditionType](#clusterexternalsecretconditiontype)_ |  |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `message` _string_ |  |  | Optional: \{\} <br /> |


#### ClusterSecretStore



ClusterSecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1` | | |
| `kind` _string_ | `ClusterSecretStore` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SecretStoreSpec](#secretstorespec)_ |  |  |  |
| `status` _[SecretStoreStatus](#secretstorestatus)_ |  |  |  |


#### ClusterSecretStoreCondition



ClusterSecretStoreCondition describes a condition by which to choose namespaces to process ExternalSecrets in
for a ClusterSecretStore instance.



_Appears in:_
- [SecretStoreSpec](#secretstorespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespaceSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | Choose namespace using a labelSelector |  | Optional: \{\} <br /> |
| `namespaces` _string array_ | Choose namespaces by name |  | items:MaxLength: 63 <br />items:MinLength: 1 <br />items:Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |
| `namespaceRegexes` _string array_ | Choose namespaces by using regex matching |  | Optional: \{\} <br /> |


#### ConfigMapReference



ConfigMapReference holds the details of a configmap.



_Appears in:_
- [GCPWorkloadIdentityFederation](#gcpworkloadidentityfederation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name of the configmap. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Required: \{\} <br /> |
| `namespace` _string_ | namespace in which the configmap exists. If empty, configmap will looked up in local namespace. |  | MaxLength: 63 <br />MinLength: 1 <br />Optional: \{\} <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br /> |
| `key` _string_ | key name holding the external account credential config. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[-._a-zA-Z0-9]+$` <br />Required: \{\} <br /> |


#### ConjurAPIKey



ConjurAPIKey contains references to a Secret resource that holds
the Conjur username and API key.



_Appears in:_
- [ConjurAuth](#conjurauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `account` _string_ | Account is the Conjur organization account name. |  | Required: \{\} <br /> |
| `userRef` _[SecretKeySelector](#secretkeyselector)_ | A reference to a specific 'key' containing the Conjur username<br />within a Secret resource. In some instances, `key` is a required field. |  | Required: \{\} <br /> |
| `apiKeyRef` _[SecretKeySelector](#secretkeyselector)_ | A reference to a specific 'key' containing the Conjur API key<br />within a Secret resource. In some instances, `key` is a required field. |  | Required: \{\} <br /> |


#### ConjurAuth



ConjurAuth is the way to provide authentication credentials to the ConjurProvider.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ConjurProvider](#conjurprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apikey` _[ConjurAPIKey](#conjurapikey)_ | Authenticates with Conjur using an API key. |  | Optional: \{\} <br /> |
| `jwt` _[ConjurJWT](#conjurjwt)_ | Jwt enables JWT authentication using Kubernetes service account tokens. |  | Optional: \{\} <br /> |
| `cert` _[ConjurCert](#conjurcert)_ | Cert enables certificate-based authentication using a client certificate and key. |  | Optional: \{\} <br /> |


#### ConjurCert



ConjurCert defines the Cert authentication configuration for Conjur provider.



_Appears in:_
- [ConjurAuth](#conjurauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `account` _string_ | Account is the Conjur organization account name. |  | Required: \{\} <br /> |
| `serviceID` _string_ | The conjur authn cert webservice id |  | Required: \{\} <br /> |
| `hostId` _string_ | Optional HostID for cert authentication (can be omitted when using 'spiffe' mode). |  | Optional: \{\} <br /> |
| `clientCertRef` _[SecretKeySelector](#secretkeyselector)_ | ClientCertRef is a reference to a specific 'key' containing the client certificate<br />within a Secret resource. The certificate must be PEM-encoded. |  | Required: \{\} <br /> |
| `clientKeyRef` _[SecretKeySelector](#secretkeyselector)_ | ClientKeyRef is a reference to a specific 'key' containing the private RSA client key<br />within a Secret resource. The key must be PEM-encoded. |  | Required: \{\} <br /> |


#### ConjurJWT



ConjurJWT defines the JWT authentication configuration for Conjur provider.



_Appears in:_
- [ConjurAuth](#conjurauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `account` _string_ | Account is the Conjur organization account name. |  | Required: \{\} <br /> |
| `serviceID` _string_ | The conjur authn jwt webservice id |  | Required: \{\} <br /> |
| `hostId` _string_ | Optional HostID for JWT authentication. This may be used depending<br />on how the Conjur JWT authenticator policy is configured. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional SecretRef that refers to a key in a Secret resource containing JWT token to<br />authenticate with Conjur using the JWT authentication method. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Optional ServiceAccountRef specifies the Kubernetes service account for which to request<br />a token for with the `TokenRequest` API. |  | Optional: \{\} <br /> |


#### ConjurProvider



ConjurProvider provides access to a Conjur provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the endpoint of the Conjur instance. |  | Required: \{\} <br /> |
| `caBundle` _string_ | CABundle is a PEM encoded CA bundle that will be used to validate the Conjur server certificate. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | Used to provide custom certificate authority (CA) certificates<br />for a secret store. The CAProvider points to a Secret or ConfigMap resource<br />that contains a PEM-encoded certificate. |  | Optional: \{\} <br /> |
| `auth` _[ConjurAuth](#conjurauth)_ | Defines authentication settings for connecting to Conjur. |  | MaxProperties: 1 <br />MinProperties: 1 <br />Required: \{\} <br /> |


#### DVLSAuth



DVLSAuth defines the authentication method for the DVLS provider.



_Appears in:_
- [DVLSProvider](#dvlsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[DVLSAuthSecretRef](#dvlsauthsecretref)_ | SecretRef contains the Application ID and Application Secret for authentication. |  | Required: \{\} <br /> |


#### DVLSAuthSecretRef



DVLSAuthSecretRef defines the secret references for DVLS authentication credentials.



_Appears in:_
- [DVLSAuth](#dvlsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `appId` _[SecretKeySelector](#secretkeyselector)_ | AppID is the reference to the secret containing the Application ID. |  | Required: \{\} <br /> |
| `appSecret` _[SecretKeySelector](#secretkeyselector)_ | AppSecret is the reference to the secret containing the Application Secret. |  | Required: \{\} <br /> |


#### DVLSProvider



DVLSProvider configures a store to sync secrets using Devolutions Server.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serverUrl` _string_ | ServerURL is the DVLS instance URL (e.g., https://dvls.example.com). |  | Required: \{\} <br /> |
| `vault` _string_ | Vault is the name or UUID of the vault to fetch secrets from.<br />When omitted, the vault must be specified in the secret key using the legacy format "<vault-id>/<entry-id>". |  | Optional: \{\} <br /> |
| `insecure` _boolean_ | Insecure allows connecting to DVLS over plain HTTP.<br />This is NOT RECOMMENDED for production use.<br />Set to true only if you understand the security implications. |  | Optional: \{\} <br /> |
| `auth` _[DVLSAuth](#dvlsauth)_ | Auth defines the authentication method to use. |  | Required: \{\} <br /> |


#### DelineaProvider



DelineaProvider provides access to Delinea secrets vault Server.
See: https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _[DelineaProviderSecretRef](#delineaprovidersecretref)_ | ClientID is the non-secret part of the credential. |  |  |
| `clientSecret` _[DelineaProviderSecretRef](#delineaprovidersecretref)_ | ClientSecret is the secret part of the credential. |  |  |
| `tenant` _string_ | Tenant is the chosen hostname / site name. |  |  |
| `urlTemplate` _string_ | URLTemplate<br />If unset, defaults to "https://%s.secretsvaultcloud.%s/v1/%s%s". |  | Optional: \{\} <br /> |
| `tld` _string_ | TLD is based on the server location that was chosen during provisioning.<br />If unset, defaults to "com". |  | Optional: \{\} <br /> |


#### DelineaProviderSecretRef



DelineaProviderSecretRef is a secret reference containing either a direct value or a reference to a secret key.



_Appears in:_
- [DelineaProvider](#delineaprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### DopplerAuth



DopplerAuth configures authentication with the Doppler API.
Exactly one of secretRef or oidcConfig must be specified.



_Appears in:_
- [DopplerProvider](#dopplerprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[DopplerAuthSecretRef](#dopplerauthsecretref)_ | SecretRef authenticates using a Doppler service token stored in a Kubernetes Secret. |  | Optional: \{\} <br /> |
| `oidcConfig` _[DopplerOIDCAuth](#doppleroidcauth)_ | OIDCConfig authenticates using Kubernetes ServiceAccount tokens via OIDC. |  | Optional: \{\} <br /> |


#### DopplerAuthSecretRef



DopplerAuthSecretRef contains the secret reference for accessing the Doppler API.



_Appears in:_
- [DopplerAuth](#dopplerauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dopplerToken` _[SecretKeySelector](#secretkeyselector)_ | The DopplerToken is used for authentication.<br />See https://docs.doppler.com/reference/api#authentication for auth token types.<br />The Key attribute defaults to dopplerToken if not specified. |  |  |


#### DopplerOIDCAuth



DopplerOIDCAuth configures OIDC authentication with Doppler using Kubernetes ServiceAccount tokens.



_Appears in:_
- [DopplerAuth](#dopplerauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identity` _string_ | Identity is the Doppler Service Account Identity ID configured for OIDC authentication. |  |  |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specifies the Kubernetes ServiceAccount to use for authentication. |  |  |
| `expirationSeconds` _integer_ | ExpirationSeconds sets the ServiceAccount token validity duration.<br />Defaults to 10 minutes. | 600 | Optional: \{\} <br /> |


#### DopplerProvider



DopplerProvider configures a store to sync secrets using the Doppler provider.
Project and Config are required if not using a Service Token.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[DopplerAuth](#dopplerauth)_ | Auth configures how the Operator authenticates with the Doppler API |  |  |
| `project` _string_ | Doppler project (required if not using a Service Token) |  | Optional: \{\} <br /> |
| `config` _string_ | Doppler config (required if not using a Service Token) |  | Optional: \{\} <br /> |
| `nameTransformer` _string_ | Environment variable compatible name transforms that change secret names to a different format |  | Enum: [upper-camel camel lower-snake tf-var dotnet-env lower-kebab] <br />Optional: \{\} <br /> |
| `format` _string_ | Format enables the downloading of secrets as a file (string) |  | Enum: [json dotnet-json env yaml docker] <br />Optional: \{\} <br /> |


#### ExternalSecret



ExternalSecret is the Schema for the external-secrets API.
It defines how to fetch data from external APIs and make it available as Kubernetes Secrets.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1` | | |
| `kind` _string_ | `ExternalSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ExternalSecretSpec](#externalsecretspec)_ |  |  |  |
| `status` _[ExternalSecretStatus](#externalsecretstatus)_ |  |  |  |


#### ExternalSecretConditionType

_Underlying type:_ _string_

ExternalSecretConditionType defines a value type for ExternalSecret conditions.

_Validation:_
- Enum: [Ready Deleted]

_Appears in:_
- [ExternalSecretStatusCondition](#externalsecretstatuscondition)

| Field | Description |
| --- | --- |
| `Ready` | ExternalSecretReady indicates that the external secret is ready and synced.<br /> |
| `Deleted` | ExternalSecretDeleted indicates that the external secret has been deleted.<br /> |


#### ExternalSecretConversionStrategy

_Underlying type:_ _string_

ExternalSecretConversionStrategy defines strategies for converting secret values.

_Validation:_
- Enum: [Default Unicode]

_Appears in:_
- [ExternalSecretDataRemoteRef](#externalsecretdataremoteref)
- [ExternalSecretFind](#externalsecretfind)

| Field | Description |
| --- | --- |
| `Default` | ExternalSecretConversionDefault specifies the default conversion strategy.<br /> |
| `Unicode` | ExternalSecretConversionUnicode specifies that values should be treated as Unicode.<br /> |


#### ExternalSecretCreationPolicy

_Underlying type:_ _string_

ExternalSecretCreationPolicy defines rules on how to create the resulting Secret.

_Validation:_
- Enum: [Owner Orphan Merge None CreateOrMerge]

_Appears in:_
- [ExternalSecretTarget](#externalsecrettarget)

| Field | Description |
| --- | --- |
| `Owner` | CreatePolicyOwner creates the Secret and sets .metadata.ownerReferences to the ExternalSecret resource.<br /> |
| `Orphan` | CreatePolicyOrphan creates the Secret and does not set the ownerReference.<br />I.e. it will be orphaned after the deletion of the ExternalSecret.<br /> |
| `Merge` | CreatePolicyMerge does not create the Secret, but merges the data fields to the Secret.<br /> |
| `None` | CreatePolicyNone does not create a Secret (future use with injector).<br /> |
| `CreateOrMerge` | CreatePolicyCreateOrMerge creates the Secret if it is missing and merges<br />data fields into it if it exists, without an ownerReference. A deleted<br />target is recreated while the ExternalSecret exists, and the Secret is<br />retained when the ExternalSecret is deleted.<br /> |


#### ExternalSecretData



ExternalSecretData defines the connection between the Kubernetes Secret key (spec.data.<key>) and the Provider data.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretKey` _string_ | The key in the Kubernetes Secret to store the value. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |
| `remoteRef` _[ExternalSecretDataRemoteRef](#externalsecretdataremoteref)_ | RemoteRef points to the remote secret and defines<br />which secret (version/property/..) to fetch. |  |  |
| `sourceRef` _[StoreSourceRef](#storesourceref)_ | SourceRef allows you to override the source<br />from which the value will be pulled. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |


#### ExternalSecretDataFromRemoteRef



ExternalSecretDataFromRemoteRef defines the connection between the Kubernetes Secret keys and the Provider data
when using DataFrom to fetch multiple values from a Provider.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `extract` _[ExternalSecretDataRemoteRef](#externalsecretdataremoteref)_ | Used to extract multiple key/value pairs from one secret<br />Note: Extract does not support sourceRef.Generator or sourceRef.GeneratorRef. |  | Optional: \{\} <br /> |
| `find` _[ExternalSecretFind](#externalsecretfind)_ | Used to find secrets based on tags or regular expressions<br />Note: Find does not support sourceRef.Generator or sourceRef.GeneratorRef. |  | Optional: \{\} <br /> |
| `rewrite` _[ExternalSecretRewrite](#externalsecretrewrite) array_ | Used to rewrite secret Keys after getting them from the secret Provider<br />Multiple Rewrite operations can be provided. They are applied in a layered order (first to last) |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `sourceRef` _[StoreGeneratorSourceRef](#storegeneratorsourceref)_ | SourceRef points to a store or generator<br />which contains secret values ready to use.<br />Use this in combination with Extract or Find pull values out of<br />a specific SecretStore.<br />When sourceRef points to a generator Extract or Find is not supported.<br />The generator returns a static map of values |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |


#### ExternalSecretDataRemoteRef



ExternalSecretDataRemoteRef defines Provider data location.



_Appears in:_
- [ExternalSecretData](#externalsecretdata)
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ | Key is the key used in the Provider, mandatory |  |  |
| `metadataPolicy` _[ExternalSecretMetadataPolicy](#externalsecretmetadatapolicy)_ | Policy for fetching tags/labels from provider secrets, possible options are Fetch, None. Defaults to None | None | Enum: [None Fetch] <br />Optional: \{\} <br /> |
| `property` _string_ | Used to select a specific property of the Provider value (if a map), if supported |  | Optional: \{\} <br /> |
| `version` _string_ | Used to select a specific version of the Provider value, if supported |  | Optional: \{\} <br /> |
| `conversionStrategy` _[ExternalSecretConversionStrategy](#externalsecretconversionstrategy)_ | Used to define a conversion Strategy | Default | Enum: [Default Unicode] <br />Optional: \{\} <br /> |
| `decodingStrategy` _[ExternalSecretDecodingStrategy](#externalsecretdecodingstrategy)_ | Used to define a decoding Strategy | None | Enum: [Auto Base64 Base64URL None] <br />Optional: \{\} <br /> |
| `nullBytePolicy` _[ExternalSecretNullBytePolicy](#externalsecretnullbytepolicy)_ | Controls how ESO handles fetched secret data containing NUL bytes for this source. |  | Enum: [Ignore Fail] <br />Optional: \{\} <br /> |


#### ExternalSecretDecodingStrategy

_Underlying type:_ _string_

ExternalSecretDecodingStrategy defines strategies for decoding secret values.

_Validation:_
- Enum: [Auto Base64 Base64URL None]

_Appears in:_
- [ExternalSecretDataRemoteRef](#externalsecretdataremoteref)
- [ExternalSecretFind](#externalsecretfind)
- [TemplateFrom](#templatefrom)

| Field | Description |
| --- | --- |
| `Auto` | ExternalSecretDecodeAuto specifies automatic detection of the decoding method.<br /> |
| `Base64` | ExternalSecretDecodeBase64 specifies that values should be decoded using Base64.<br /> |
| `Base64URL` | ExternalSecretDecodeBase64URL specifies that values should be decoded using Base64URL.<br /> |
| `None` | ExternalSecretDecodeNone specifies that no decoding should be performed.<br /> |


#### ExternalSecretDeletionPolicy

_Underlying type:_ _string_

ExternalSecretDeletionPolicy defines rules on how to delete the resulting Secret.

_Validation:_
- Enum: [Delete Merge Retain]

_Appears in:_
- [ExternalSecretTarget](#externalsecrettarget)

| Field | Description |
| --- | --- |
| `Delete` | DeletionPolicyDelete deletes the secret if all provider secrets are deleted.<br />If a secret gets deleted on the provider side and is not accessible<br />anymore this is not considered an error and the ExternalSecret<br />does not go into SecretSyncedError status.<br /> |
| `Merge` | DeletionPolicyMerge removes keys in the secret, but not the secret itself.<br />If a secret gets deleted on the provider side and is not accessible<br />anymore this is not considered an error and the ExternalSecret<br />does not go into SecretSyncedError status.<br /> |
| `Retain` | DeletionPolicyRetain will retain the secret if all provider secrets have been deleted.<br />If a provider secret does not exist the ExternalSecret gets into the<br />SecretSyncedError status.<br /> |


#### ExternalSecretFind



ExternalSecretFind defines configuration for finding secrets in the provider.



_Appears in:_
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | A root path to start the find operations. |  | Optional: \{\} <br /> |
| `name` _[FindName](#findname)_ | Finds secrets based on the name. |  | Optional: \{\} <br /> |
| `tags` _object (keys:string, values:string)_ | Find secrets based on tags. |  | Optional: \{\} <br /> |
| `conversionStrategy` _[ExternalSecretConversionStrategy](#externalsecretconversionstrategy)_ | Used to define a conversion Strategy | Default | Enum: [Default Unicode] <br />Optional: \{\} <br /> |
| `decodingStrategy` _[ExternalSecretDecodingStrategy](#externalsecretdecodingstrategy)_ | Used to define a decoding Strategy | None | Enum: [Auto Base64 Base64URL None] <br />Optional: \{\} <br /> |
| `nullBytePolicy` _[ExternalSecretNullBytePolicy](#externalsecretnullbytepolicy)_ | Controls how ESO handles fetched secret data containing NUL bytes for this find source. |  | Enum: [Ignore Fail] <br />Optional: \{\} <br /> |


#### ExternalSecretMetadata



ExternalSecretMetadata defines metadata fields for the ExternalSecret generated by the ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecretSpec](#clusterexternalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `annotations` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |


#### ExternalSecretMetadataPolicy

_Underlying type:_ _string_

ExternalSecretMetadataPolicy defines policies for fetching metadata from provider secrets.

_Validation:_
- Enum: [None Fetch]

_Appears in:_
- [ExternalSecretDataRemoteRef](#externalsecretdataremoteref)

| Field | Description |
| --- | --- |
| `None` | ExternalSecretMetadataPolicyNone specifies that no metadata should be fetched from the provider.<br /> |
| `Fetch` | ExternalSecretMetadataPolicyFetch specifies that metadata should be fetched from the provider.<br /> |


#### ExternalSecretNullBytePolicy

_Underlying type:_ _string_

ExternalSecretNullBytePolicy defines how fetched secret data containing NUL bytes should be handled.

_Validation:_
- Enum: [Ignore Fail]

_Appears in:_
- [ExternalSecretDataRemoteRef](#externalsecretdataremoteref)
- [ExternalSecretFind](#externalsecretfind)

| Field | Description |
| --- | --- |
| `Ignore` | ExternalSecretNullBytePolicyIgnore allows fetched secret data to contain NUL bytes.<br /> |
| `Fail` | ExternalSecretNullBytePolicyFail fails reconciliation if fetched secret data contains NUL bytes.<br /> |


#### ExternalSecretRefreshPolicy

_Underlying type:_ _string_

ExternalSecretRefreshPolicy defines how and when the ExternalSecret should be refreshed.

_Validation:_
- Enum: [CreatedOnce Periodic OnChange]

_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description |
| --- | --- |
| `CreatedOnce` | RefreshPolicyCreatedOnce creates the Secret once and does not update it thereafter.<br /> |
| `Periodic` | RefreshPolicyPeriodic synchronizes the Secret from the provider at regular intervals.<br /> |
| `OnChange` | RefreshPolicyOnChange only synchronizes when the ExternalSecret's metadata or spec changes.<br /> |


#### ExternalSecretRewrite



ExternalSecretRewrite defines how to rewrite secret data values before they are written to the Secret.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `merge` _[ExternalSecretRewriteMerge](#externalsecretrewritemerge)_ | Used to merge key/values in one single Secret<br />The resulting key will contain all values from the specified secrets |  | Optional: \{\} <br /> |
| `regexp` _[ExternalSecretRewriteRegexp](#externalsecretrewriteregexp)_ | Used to rewrite with regular expressions.<br />The resulting key will be the output of a regexp.ReplaceAll operation. |  | Optional: \{\} <br /> |
| `transform` _[ExternalSecretRewriteTransform](#externalsecretrewritetransform)_ | Used to apply string transformation on the secrets.<br />The resulting key will be the output of the template applied by the operation. |  | Optional: \{\} <br /> |


#### ExternalSecretRewriteMerge



ExternalSecretRewriteMerge defines configuration for merging secret values.



_Appears in:_
- [ExternalSecretRewrite](#externalsecretrewrite)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `into` _string_ | Used to define the target key of the merge operation.<br />Required if strategy is JSON. Ignored otherwise. |  | Optional: \{\} <br /> |
| `priority` _string array_ | Used to define key priority in conflict resolution. |  | Optional: \{\} <br /> |
| `priorityPolicy` _[ExternalSecretRewriteMergePriorityPolicy](#externalsecretrewritemergeprioritypolicy)_ | Used to define the policy when a key in the priority list does not exist in the input. | Strict | Enum: [IgnoreNotFound Strict] <br />Optional: \{\} <br /> |
| `conflictPolicy` _[ExternalSecretRewriteMergeConflictPolicy](#externalsecretrewritemergeconflictpolicy)_ | Used to define the policy to use in conflict resolution. | Error | Enum: [Ignore Error] <br />Optional: \{\} <br /> |
| `strategy` _[ExternalSecretRewriteMergeStrategy](#externalsecretrewritemergestrategy)_ | Used to define the strategy to use in the merge operation. | Extract | Enum: [Extract JSON] <br />Optional: \{\} <br /> |


#### ExternalSecretRewriteMergeConflictPolicy

_Underlying type:_ _string_

ExternalSecretRewriteMergeConflictPolicy defines the policy for resolving conflicts when merging secrets.

_Validation:_
- Enum: [Ignore Error]

_Appears in:_
- [ExternalSecretRewriteMerge](#externalsecretrewritemerge)

| Field | Description |
| --- | --- |
| `Ignore` | ExternalSecretRewriteMergeConflictPolicyIgnore ignores conflicts when merging secret values.<br /> |
| `Error` | ExternalSecretRewriteMergeConflictPolicyError returns an error when conflicts occur during merge.<br /> |


#### ExternalSecretRewriteMergePriorityPolicy

_Underlying type:_ _string_

ExternalSecretRewriteMergePriorityPolicy defines the policy for handling missing keys in the priority
list during merge operations.

_Validation:_
- Enum: [IgnoreNotFound Strict]

_Appears in:_
- [ExternalSecretRewriteMerge](#externalsecretrewritemerge)

| Field | Description |
| --- | --- |
| `IgnoreNotFound` |  |
| `Strict` |  |


#### ExternalSecretRewriteMergeStrategy

_Underlying type:_ _string_

ExternalSecretRewriteMergeStrategy defines the strategy for merging secrets.

_Validation:_
- Enum: [Extract JSON]

_Appears in:_
- [ExternalSecretRewriteMerge](#externalsecretrewritemerge)

| Field | Description |
| --- | --- |
| `Extract` | ExternalSecretRewriteMergeStrategyExtract merges secrets by extracting values.<br /> |
| `JSON` | ExternalSecretRewriteMergeStrategyJSON merges secrets using JSON merge strategy.<br /> |


#### ExternalSecretRewriteRegexp



ExternalSecretRewriteRegexp defines configuration for rewriting secrets using regular expressions.



_Appears in:_
- [ExternalSecretRewrite](#externalsecretrewrite)
- [PushSecretRewrite](#pushsecretrewrite)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `source` _string_ | Used to define the regular expression of a re.Compiler. |  |  |
| `target` _string_ | Used to define the target pattern of a ReplaceAll operation. |  |  |


#### ExternalSecretRewriteTransform



ExternalSecretRewriteTransform defines configuration for transforming secrets using templates.



_Appears in:_
- [ExternalSecretRewrite](#externalsecretrewrite)
- [PushSecretRewrite](#pushsecretrewrite)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `template` _string_ | Used to define the template to apply on the secret name.<br />`.value ` will specify the secret name in the template. |  |  |


#### ExternalSecretSpec



ExternalSecretSpec defines the desired state of ExternalSecret.



_Appears in:_
- [ClusterExternalSecretSpec](#clusterexternalsecretspec)
- [ExternalSecret](#externalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretStoreRef` _[SecretStoreRef](#secretstoreref)_ |  |  | Optional: \{\} <br /> |
| `target` _[ExternalSecretTarget](#externalsecrettarget)_ |  | \{ creationPolicy:Owner deletionPolicy:Retain \} | Optional: \{\} <br /> |
| `refreshPolicy` _[ExternalSecretRefreshPolicy](#externalsecretrefreshpolicy)_ | RefreshPolicy determines how the ExternalSecret should be refreshed:<br />- CreatedOnce: Creates the Secret only if it does not exist and does not update it thereafter<br />- Periodic: Synchronizes the Secret from the external source at regular intervals specified by refreshInterval.<br />  No periodic updates occur if refreshInterval is 0.<br />- OnChange: Only synchronizes the Secret when the ExternalSecret's metadata or specification changes |  | Enum: [CreatedOnce Periodic OnChange] <br />Optional: \{\} <br /> |
| `refreshInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | RefreshInterval is the amount of time before the values are read again from the SecretStore provider,<br />specified as Golang Duration strings.<br />Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"<br />Example values: "1h0m0s", "2h30m0s", "10m0s"<br />May be set to "0s" to fetch and create it once. Defaults to 1h0m0s. | 1h0m0s |  |
| `syncWindows` _[ExternalSecretSyncWindows](#externalsecretsyncwindows)_ | SyncWindows optionally restricts when periodic refreshes may occur.<br />Evaluated in UTC, only for Periodic refresh policy (or when refreshPolicy is unset). |  | Optional: \{\} <br /> |
| `data` _[ExternalSecretData](#externalsecretdata) array_ | Data defines the connection between the Kubernetes Secret keys and the Provider data |  | Optional: \{\} <br /> |
| `dataFrom` _[ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref) array_ | DataFrom is used to fetch all properties from a specific Provider data<br />If multiple entries are specified, the Secret keys are merged in the specified order |  | Optional: \{\} <br /> |


#### ExternalSecretStatus



ExternalSecretStatus defines the observed state of ExternalSecret.



_Appears in:_
- [ExternalSecret](#externalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `refreshTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ | refreshTime is the time and date the external secret was fetched and<br />the target secret updated |  |  |
| `syncedResourceVersion` _string_ | SyncedResourceVersion keeps track of the last synced version |  |  |
| `conditions` _[ExternalSecretStatusCondition](#externalsecretstatuscondition) array_ |  |  | Optional: \{\} <br /> |
| `binding` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#localobjectreference-v1-core)_ | Binding represents a servicebinding.io Provisioned Service reference to the secret |  |  |


#### ExternalSecretStatusCondition



ExternalSecretStatusCondition defines a status condition of an ExternalSecret resource.



_Appears in:_
- [ExternalSecretStatus](#externalsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ExternalSecretConditionType](#externalsecretconditiontype)_ |  |  | Enum: [Ready Deleted] <br /> |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `reason` _string_ |  |  | Optional: \{\} <br /> |
| `message` _string_ |  |  | Optional: \{\} <br /> |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ |  |  | Optional: \{\} <br /> |


#### ExternalSecretSyncWindowEntry



ExternalSecretSyncWindowEntry defines a single cron-schedule + duration pair
within a SyncWindows block.



_Appears in:_
- [ExternalSecretSyncWindows](#externalsecretsyncwindows)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `schedule` _string_ | Schedule is a standard 5-field cron expression evaluated in UTC, or a<br />named shorthand such as @daily or @every 1h. It marks the start time of<br />each window occurrence.<br />Example: "0 22 * * 1-5" opens a window every weekday at 22:00 UTC. |  | MinLength: 1 <br />Pattern: `^(@(annually\|yearly\|monthly\|weekly\|daily\|midnight\|hourly)\|@every [^\s]+.*\|[^\s]+( [^\s]+)\{4\})$` <br /> |
| `duration` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | Duration specifies how long the window stays open after each Schedule<br />firing. Example: "8h". |  |  |


#### ExternalSecretSyncWindowKind

_Underlying type:_ _string_

ExternalSecretSyncWindowKind defines whether a SyncWindow permits or
blocks periodic refreshes.

_Validation:_
- Enum: [allow deny]

_Appears in:_
- [ExternalSecretSyncWindows](#externalsecretsyncwindows)

| Field | Description |
| --- | --- |
| `allow` | SyncWindowAllow allows periodic refreshes only while at least one window<br />in the list is active. Refreshes are blocked at all other times.<br /> |
| `deny` | SyncWindowDeny blocks periodic refreshes while any window in the list is<br />active. Refreshes proceed normally at all other times.<br /> |


#### ExternalSecretSyncWindows



ExternalSecretSyncWindows optionally restricts when periodic syncs may occur.
All windows in the list share the same Kind.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _[ExternalSecretSyncWindowKind](#externalsecretsyncwindowkind)_ | Kind applies to every window in the list.<br />"allow" -- syncs are permitted only while at least one window is active;<br />           all other times are blocked.<br />"deny"  -- syncs are blocked while any window is active;<br />           all other times are permitted. |  | Enum: [allow deny] <br /> |
| `windows` _[ExternalSecretSyncWindowEntry](#externalsecretsyncwindowentry) array_ | Windows is the list of schedule+duration pairs. |  | MinItems: 1 <br /> |


#### ExternalSecretTarget



ExternalSecretTarget defines the Kubernetes Secret to be created,
there can be only one target per ExternalSecret.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The name of the Secret resource to be managed.<br />Defaults to the .metadata.name of the ExternalSecret resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Optional: \{\} <br /> |
| `creationPolicy` _[ExternalSecretCreationPolicy](#externalsecretcreationpolicy)_ | CreationPolicy defines rules on how to create the resulting Secret.<br />Defaults to "Owner" | Owner | Enum: [Owner Orphan Merge None CreateOrMerge] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[ExternalSecretDeletionPolicy](#externalsecretdeletionpolicy)_ | DeletionPolicy defines rules on how to delete the resulting Secret.<br />Defaults to "Retain" | Retain | Enum: [Delete Merge Retain] <br />Optional: \{\} <br /> |
| `template` _[ExternalSecretTemplate](#externalsecrettemplate)_ | Template defines a blueprint for the created Secret resource. |  | Optional: \{\} <br /> |
| `manifest` _[ManifestReference](#manifestreference)_ | Manifest defines a custom Kubernetes resource to create instead of a Secret.<br />When specified, ExternalSecret will create the resource type defined here<br />(e.g., ConfigMap, Custom Resource) instead of a Secret.<br />Warning: Using Generic target. Make sure access policies and encryption are properly configured. |  | Optional: \{\} <br /> |
| `immutable` _boolean_ | Immutable defines if the final secret will be immutable |  | Optional: \{\} <br /> |


#### ExternalSecretTemplate



ExternalSecretTemplate defines a blueprint for the created Secret resource.
we can not use native corev1.Secret, it will have empty ObjectMeta values: https://github.com/kubernetes-sigs/controller-tools/issues/448



_Appears in:_
- [ExternalSecretTarget](#externalsecrettarget)
- [PushSecretSpec](#pushsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[SecretType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#secrettype-v1-core)_ |  |  | Optional: \{\} <br /> |
| `engineVersion` _[TemplateEngineVersion](#templateengineversion)_ | EngineVersion specifies the template engine version<br />that should be used to compile/execute the<br />template specified in .data and .templateFrom[]. | v2 | Enum: [v2] <br /> |
| `metadata` _[ExternalSecretTemplateMetadata](#externalsecrettemplatemetadata)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `mergePolicy` _[TemplateMergePolicy](#templatemergepolicy)_ |  | Replace | Enum: [Replace Merge] <br /> |
| `data` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `templateFrom` _[TemplateFrom](#templatefrom) array_ |  |  | Optional: \{\} <br /> |


#### ExternalSecretTemplateMetadata



ExternalSecretTemplateMetadata defines metadata fields for the Secret blueprint.



_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `annotations` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `finalizers` _string array_ |  |  | Optional: \{\} <br /> |




#### FakeProvider



FakeProvider configures a fake provider that returns static values.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _[FakeProviderData](#fakeproviderdata) array_ |  |  |  |
| `validationResult` _[ValidationResult](#validationresult)_ |  |  |  |


#### FakeProviderData



FakeProviderData defines a key-value pair with optional version for the fake provider.



_Appears in:_
- [FakeProvider](#fakeprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  |  |
| `value` _string_ |  |  |  |
| `version` _string_ |  |  |  |


#### FetchingPolicy



FetchingPolicy configures how the provider interprets the `data.secretKey.remoteRef.key` field in ExternalSecret.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [YandexCertificateManagerProvider](#yandexcertificatemanagerprovider)
- [YandexLockboxProvider](#yandexlockboxprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `byID` _[ByID](#byid)_ |  |  |  |
| `byName` _[ByName](#byname)_ |  |  |  |


#### FindName



FindName defines criteria for finding secrets by name patterns.



_Appears in:_
- [ExternalSecretFind](#externalsecretfind)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `regexp` _string_ | Finds secrets base |  | Optional: \{\} <br /> |


#### FortanixProvider



FortanixProvider provides access to Fortanix SDKMS API using the provided credentials.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the URL of SDKMS API. Defaults to `sdkms.fortanix.com`. |  |  |
| `apiKey` _[FortanixProviderSecretRef](#fortanixprovidersecretref)_ | APIKey is the API token to access SDKMS Applications. |  |  |


#### FortanixProviderSecretRef



FortanixProviderSecretRef is a secret reference containing the SDKMS API Key.



_Appears in:_
- [FortanixProvider](#fortanixprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef is a reference to a secret containing the SDKMS API Key. |  |  |


#### GCPSMAuth



GCPSMAuth defines the authentication methods for Google Cloud Platform Secret Manager.



_Appears in:_
- [GCPSMProvider](#gcpsmprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[GCPSMAuthSecretRef](#gcpsmauthsecretref)_ |  |  | Optional: \{\} <br /> |
| `workloadIdentity` _[GCPWorkloadIdentity](#gcpworkloadidentity)_ |  |  | Optional: \{\} <br /> |
| `workloadIdentityFederation` _[GCPWorkloadIdentityFederation](#gcpworkloadidentityfederation)_ |  |  | Optional: \{\} <br /> |


#### GCPSMAuthSecretRef



GCPSMAuthSecretRef contains the secret references for GCP Secret Manager authentication.



_Appears in:_
- [GCPSMAuth](#gcpsmauth)
- [VaultGCPAuth](#vaultgcpauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretAccessKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  | Optional: \{\} <br /> |


#### GCPSMProvider



GCPSMProvider Configures a store to sync secrets using the GCP Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[GCPSMAuth](#gcpsmauth)_ | Auth defines the information necessary to authenticate against GCP |  | Optional: \{\} <br /> |
| `projectID` _string_ | ProjectID project where secret is located |  |  |
| `location` _string_ | Location optionally defines a location for a secret |  |  |
| `secretVersionSelectionPolicy` _[SecretVersionSelectionPolicy](#secretversionselectionpolicy)_ | SecretVersionSelectionPolicy specifies how the provider selects a secret version<br />when "latest" is disabled or destroyed.<br />Possible values are:<br />- LatestOrFail: the provider always uses "latest", or fails if that version is disabled/destroyed.<br />- LatestOrFetch: the provider falls back to fetching the latest version if the version is DESTROYED or DISABLED | LatestOrFail | Optional: \{\} <br /> |


#### GCPWorkloadIdentity



GCPWorkloadIdentity defines configuration for workload identity authentication to GCP.



_Appears in:_
- [GCPSMAuth](#gcpsmauth)
- [VaultGCPAuth](#vaultgcpauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ |  |  | Required: \{\} <br /> |
| `clusterLocation` _string_ | ClusterLocation is the location of the cluster<br />If not specified, it fetches information from the metadata server |  | Optional: \{\} <br /> |
| `clusterName` _string_ | ClusterName is the name of the cluster<br />If not specified, it fetches information from the metadata server |  | Optional: \{\} <br /> |
| `clusterProjectID` _string_ | ClusterProjectID is the project ID of the cluster<br />If not specified, it fetches information from the metadata server |  | Optional: \{\} <br /> |


#### GCPWorkloadIdentityFederation



GCPWorkloadIdentityFederation holds the configurations required for generating federated access tokens.



_Appears in:_
- [GCPSMAuth](#gcpsmauth)
- [GCPSMAuth](#gcpsmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `credConfig` _[ConfigMapReference](#configmapreference)_ | credConfig holds the configmap reference containing the GCP external account credential configuration in JSON format and the key name containing the json data.<br />For using Kubernetes cluster as the identity provider, use serviceAccountRef instead. Operators mounted serviceaccount token cannot be used as the token source, instead<br />serviceAccountRef must be used by providing operators service account details. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | serviceAccountRef is the reference to the kubernetes ServiceAccount to be used for obtaining the tokens,<br />when Kubernetes is configured as provider in workload identity pool. |  | Optional: \{\} <br /> |
| `awsSecurityCredentials` _[AwsCredentialsConfig](#awscredentialsconfig)_ | awsSecurityCredentials is for configuring AWS region and credentials to use for obtaining the access token,<br />when using the AWS metadata server is not an option. |  | Optional: \{\} <br /> |
| `audience` _string_ | audience is the Secure Token Service (STS) audience which contains the resource name for the workload identity pool and the provider identifier in that pool.<br />If specified, Audience found in the external account credential config will be overridden with the configured value.<br />audience must be provided when serviceAccountRef or awsSecurityCredentials is configured. |  | Optional: \{\} <br /> |
| `externalTokenEndpoint` _string_ | externalTokenEndpoint is the endpoint explicitly set up to provide tokens, which will be matched against the<br />credential_source.url in the provided credConfig. This field is merely to double-check the external token source<br />URL is having the expected value. |  | Optional: \{\} <br /> |
| `gcpServiceAccountEmail` _string_ | GCPServiceAccountEmail is the email of the Google Cloud service account to impersonate<br />after Workload Identity Federation. Use this to grant access through the service account's<br />IAM bindings (for example roles/secretmanager.secretAccessor). When set, it overrides<br />service_account_impersonation_url in the external account JSON from credConfig;<br />when serviceAccountRef is set, it also overrides the "iam.gke.io/gcp-service-account" annotation<br />on that ServiceAccount. |  | MinLength: 1 <br />Optional: \{\} <br />Pattern: `^.*@.*\.iam\.gserviceaccount\.com$` <br /> |


#### GcpIDTokenAuthCredentials



GcpIDTokenAuthCredentials represents the credentials for GCP ID token authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### GcpIamAuthCredentials



GcpIamAuthCredentials represents the credentials for GCP IAM authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `serviceAccountKeyFilePath` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### GeneratorRef



GeneratorRef points to a generator custom resource.



_Appears in:_
- [PushSecretSelector](#pushsecretselector)
- [StoreGeneratorSourceRef](#storegeneratorsourceref)
- [StoreSourceRef](#storesourceref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | Specify the apiVersion of the generator resource | generators.external-secrets.io/v1alpha1 |  |
| `kind` _string_ | Specify the Kind of the generator resource |  | Enum: [ACRAccessToken BeyondtrustWorkloadCredentialsDynamicSecret ClusterGenerator CloudsmithAccessToken ECRAuthorizationToken Fake GCRAccessToken GithubAccessToken GitlabDeployToken QuayAccessToken Password SSHKey STSSessionToken UUID VaultDynamicSecret Webhook Grafana MFA] <br /> |
| `name` _string_ | Specify the name of the generator resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |




#### GenericStore

_Underlying type:_ _interface{Copy() GenericStore; GetKind() string; GetNamespacedName() string; GetObjectMeta() *k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta; GetSpec() *SecretStoreSpec; GetStatus() SecretStoreStatus; GetTypeMeta() *k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta; SetStatus(status SecretStoreStatus); k8s.io/apimachinery/pkg/runtime.Object; k8s.io/apimachinery/pkg/apis/meta/v1.Object}_

GenericStore is a common interface for interacting with ClusterSecretStore
or a namespaced SecretStore.









#### GithubAppAuth



GithubAppAuth defines authentication configuration using a GitHub App for accessing GitHub API.



_Appears in:_
- [GithubProvider](#githubprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `privateKey` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### GithubProvider



GithubProvider provides access and authentication to a GitHub instance .



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL configures the Github instance URL. Defaults to https://github.com/. | https://github.com/ |  |
| `uploadURL` _string_ | Upload URL for enterprise instances. Default to URL. |  | Optional: \{\} <br /> |
| `auth` _[GithubAppAuth](#githubappauth)_ | auth configures how secret-manager authenticates with a Github instance. |  |  |
| `appID` _integer_ | appID specifies the Github APP that will be used to authenticate the client |  |  |
| `installationID` _integer_ | installationID specifies the Github APP installation that will be used to authenticate the client |  |  |
| `organization` _string_ | organization will be used to fetch secrets from the Github organization |  |  |
| `repository` _string_ | repository will be used to fetch secrets from the Github repository within an organization |  | Optional: \{\} <br /> |
| `environment` _string_ | environment will be used to fetch secrets from a particular environment within a github repository |  | Optional: \{\} <br /> |
| `orgSecretVisibility` _string_ | orgSecretVisibility controls the visibility of organization secrets pushed via PushSecret.<br />Valid values are "all" or "private".<br />When unset, new secrets are created with visibility "all" and existing secrets preserve<br />whatever visibility they already have in GitHub. |  | Enum: [all private] <br />Optional: \{\} <br /> |


#### GitlabAuth



GitlabAuth defines the authentication method for accessing GitLab API.



_Appears in:_
- [GitlabProvider](#gitlabprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `SecretRef` _[GitlabSecretRef](#gitlabsecretref)_ |  |  |  |


#### GitlabProvider



GitlabProvider configures a store to sync secrets with a GitLab instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL configures the GitLab instance URL. Defaults to https://gitlab.com/. |  |  |
| `auth` _[GitlabAuth](#gitlabauth)_ | Auth configures how secret-manager authenticates with a GitLab instance. |  |  |
| `projectID` _string_ | ProjectID specifies a project where secrets are located. |  |  |
| `inheritFromGroups` _boolean_ | InheritFromGroups specifies whether parent groups should be discovered and checked for secrets. |  |  |
| `groupIDs` _string array_ | GroupIDs specify, which gitlab groups to pull secrets from. Group secrets are read from left to right followed by the project variables. |  |  |
| `environment` _string_ | Environment environment_scope of gitlab CI/CD variables (Please see https://docs.gitlab.com/ee/ci/environments/#create-a-static-environment on how to create environments) |  |  |
| `caBundle` _integer array_ | Base64 encoded certificate for the GitLab server sdk. The sdk MUST run with HTTPS to make sure no MITM attack<br />can be performed. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | see: https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider |  | Optional: \{\} <br /> |


#### GitlabSecretRef



GitlabSecretRef contains the secret reference for GitLab authentication credentials.



_Appears in:_
- [GitlabAuth](#gitlabauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessToken` _[SecretKeySelector](#secretkeyselector)_ | AccessToken is used for authentication. |  |  |


#### IBMAuth



IBMAuth defines authentication options for connecting to IBM Cloud Secrets Manager.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [IBMProvider](#ibmprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[IBMAuthSecretRef](#ibmauthsecretref)_ |  |  |  |
| `containerAuth` _[IBMAuthContainerAuth](#ibmauthcontainerauth)_ |  |  |  |


#### IBMAuthContainerAuth



IBMAuthContainerAuth defines container-based authentication with IAM Trusted Profile.



_Appears in:_
- [IBMAuth](#ibmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `profile` _string_ | the IBM Trusted Profile |  |  |
| `tokenLocation` _string_ | Location the token is mounted on the pod |  |  |
| `iamEndpoint` _string_ |  |  |  |


#### IBMAuthSecretRef



IBMAuthSecretRef contains the secret reference for IBM Cloud API key authentication.



_Appears in:_
- [IBMAuth](#ibmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretApiKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  |  |
| `iamEndpoint` _string_ | The IAM endpoint used to obain a token |  |  |


#### IBMProvider



IBMProvider configures a store to sync secrets using a IBM Cloud Secrets Manager
backend.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[IBMAuth](#ibmauth)_ | Auth configures how secret-manager authenticates with the IBM secrets manager. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `serviceUrl` _string_ | ServiceURL is the Endpoint URL that is specific to the Secrets Manager service instance |  |  |


#### InfisicalAuth



InfisicalAuth specifies the authentication configuration for Infisical.



_Appears in:_
- [InfisicalProvider](#infisicalprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `universalAuthCredentials` _[UniversalAuthCredentials](#universalauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `azureAuthCredentials` _[AzureAuthCredentials](#azureauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `gcpIdTokenAuthCredentials` _[GcpIDTokenAuthCredentials](#gcpidtokenauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `gcpIamAuthCredentials` _[GcpIamAuthCredentials](#gcpiamauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `jwtAuthCredentials` _[JwtAuthCredentials](#jwtauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `ldapAuthCredentials` _[LdapAuthCredentials](#ldapauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `ociAuthCredentials` _[OciAuthCredentials](#ociauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `kubernetesAuthCredentials` _[KubernetesAuthCredentials](#kubernetesauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `awsAuthCredentials` _[AwsAuthCredentials](#awsauthcredentials)_ |  |  | Optional: \{\} <br /> |
| `tokenAuthCredentials` _[TokenAuthCredentials](#tokenauthcredentials)_ |  |  | Optional: \{\} <br /> |


#### InfisicalProvider



InfisicalProvider configures a store to sync secrets using the Infisical provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[InfisicalAuth](#infisicalauth)_ | Auth configures how the Operator authenticates with the Infisical API |  | Required: \{\} <br /> |
| `secretsScope` _[MachineIdentityScopeInWorkspace](#machineidentityscopeinworkspace)_ | SecretsScope defines the scope of the secrets within the workspace |  | Required: \{\} <br /> |
| `hostAPI` _string_ | HostAPI specifies the base URL of the Infisical API. If not provided, it defaults to "https://app.infisical.com/api". | https://app.infisical.com/api | Optional: \{\} <br /> |
| `caBundle` _integer array_ | CABundle is a PEM-encoded CA certificate bundle used to validate<br />the Infisical server's TLS certificate. Mutually exclusive with CAProvider. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | CAProvider is a reference to a Secret or ConfigMap that contains a CA certificate.<br />The certificate is used to validate the Infisical server's TLS certificate.<br />Mutually exclusive with CABundle. |  | Optional: \{\} <br /> |


#### IntegrationInfo



IntegrationInfo specifies the name and version of the integration built using the 1Password Go SDK.



_Appears in:_
- [OnePasswordSDKProvider](#onepasswordsdkprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name defaults to "1Password SDK". | 1Password SDK |  |
| `version` _string_ | Version defaults to "v1.0.0". | v1.0.0 |  |


#### JwtAuthCredentials



JwtAuthCredentials represents the credentials for JWT authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `jwt` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### KeeperSecurityProvider



KeeperSecurityProvider Configures a store to sync secrets using Keeper Security.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `folderID` _string_ |  |  |  |
| `getByTitleFallback` _boolean_ |  |  |  |


#### KubernetesAuth



KubernetesAuth defines authentication options for connecting to a Kubernetes cluster.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [CRDProvider](#crdprovider)
- [KubernetesProvider](#kubernetesprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cert` _[CertAuth](#certauth)_ | has both clientCert and clientKey as secretKeySelector |  | Optional: \{\} <br /> |
| `token` _[TokenAuth](#tokenauth)_ | use static token to authenticate with |  | Optional: \{\} <br /> |
| `serviceAccount` _[ServiceAccountSelector](#serviceaccountselector)_ | points to a service account that should be used for authentication |  | Optional: \{\} <br /> |


#### KubernetesAuthCredentials



KubernetesAuthCredentials represents the credentials for Kubernetes authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `serviceAccountTokenPath` _[SecretKeySelector](#secretkeyselector)_ |  |  | Optional: \{\} <br /> |


#### KubernetesProvider



KubernetesProvider configures a store to sync secrets with a Kubernetes instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `server` _[KubernetesServer](#kubernetesserver)_ | configures the Kubernetes server Address. |  | Optional: \{\} <br /> |
| `auth` _[KubernetesAuth](#kubernetesauth)_ | Auth configures how secret-manager authenticates with a Kubernetes instance. |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `authRef` _[SecretKeySelector](#secretkeyselector)_ | A reference to a secret that contains the auth information. |  | Optional: \{\} <br /> |
| `remoteNamespace` _string_ | Remote namespace to fetch the secrets from | default | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


#### KubernetesServer



KubernetesServer defines configuration for connecting to a Kubernetes API server.



_Appears in:_
- [CRDProvider](#crdprovider)
- [KubernetesProvider](#kubernetesprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | configures the Kubernetes server Address. | kubernetes.default | Optional: \{\} <br /> |
| `caBundle` _integer array_ | CABundle is a base64-encoded CA certificate |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | see: https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider |  | Optional: \{\} <br /> |


#### LdapAuthCredentials



LdapAuthCredentials represents the credentials for LDAP authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `ldapPassword` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `ldapUsername` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### MachineIdentityScopeInWorkspace



MachineIdentityScopeInWorkspace defines the scope for machine identity within a workspace.



_Appears in:_
- [InfisicalProvider](#infisicalprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretsPath` _string_ | SecretsPath specifies the path to the secrets within the workspace. Defaults to "/" if not provided. | / | Optional: \{\} <br /> |
| `recursive` _boolean_ | Recursive indicates whether the secrets should be fetched recursively. Defaults to false if not provided. | false | Optional: \{\} <br /> |
| `environmentSlug` _string_ | EnvironmentSlug is the required slug identifier for the environment. |  | Required: \{\} <br /> |
| `projectSlug` _string_ | ProjectSlug is the required slug identifier for the project. |  | Required: \{\} <br /> |
| `organizationSlug` _string_ | OrganizationSlug is the optional slug that identifies the organization that will be used<br />during authentication. Useful for sub-organization setups |  | Optional: \{\} <br /> |
| `expandSecretReferences` _boolean_ | ExpandSecretReferences indicates whether secret references should be expanded. Defaults to true if not provided. | true | Optional: \{\} <br /> |




#### ManifestReference



ManifestReference defines a custom Kubernetes resource type to be created
instead of a Secret. This allows ExternalSecret to create ConfigMaps,
Custom Resources, or any other Kubernetes resource type.



_Appears in:_
- [ExternalSecretTarget](#externalsecrettarget)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | APIVersion of the target resource (e.g., "v1" for ConfigMap, "argoproj.io/v1alpha1" for ArgoCD Application) |  | MinLength: 1 <br />Required: \{\} <br /> |
| `kind` _string_ | Kind of the target resource (e.g., "ConfigMap", "Application") |  | MinLength: 1 <br />Required: \{\} <br /> |


#### NTLMProtocol



NTLMProtocol contains the NTLM-specific configuration.



_Appears in:_
- [AuthorizationProtocol](#authorizationprotocol)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `usernameSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `passwordSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### NebiusAuth



NebiusAuth defines the authentication method for the Nebius provider.



_Appears in:_
- [NebiusMysteryboxProvider](#nebiusmysteryboxprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountCredsSecretRef` _[SecretKeySelector](#secretkeyselector)_ | ServiceAccountCreds references a Kubernetes Secret key that contains a JSON<br />document with service account credentials used to get an IAM token.<br />Expected JSON structure:<br />\{<br />  "subject-credentials": \{<br />    "alg": "RS256",<br />    "private-key": "-----BEGIN PRIVATE KEY-----\n<private-key>\n-----END PRIVATE KEY-----\n",<br />    "kid": "<public-key-id>",<br />    "iss": "<issuer-service-account-id>",<br />    "sub": "<subject-service-account-id>"<br />  \}<br />\} |  | Optional: \{\} <br /> |
| `tokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | Token authenticates with Nebius Mysterybox by presenting a token. |  | Optional: \{\} <br /> |


#### NebiusCAProvider



NebiusCAProvider The provider for the CA bundle to use to validate Nebius server certificate.



_Appears in:_
- [NebiusMysteryboxProvider](#nebiusmysteryboxprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  | Optional: \{\} <br /> |


#### NebiusMysteryboxProvider



NebiusMysteryboxProvider Configures a store to sync secrets using the Nebius Mysterybox provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiDomain` _string_ | NebiusMysterybox API endpoint |  |  |
| `auth` _[NebiusAuth](#nebiusauth)_ | Auth defines parameters to authenticate in MysteryBox |  |  |
| `caProvider` _[NebiusCAProvider](#nebiuscaprovider)_ | The provider for the CA bundle to use to validate NebiusMysterybox server certificate. |  | Optional: \{\} <br /> |


#### NgrokAuth



NgrokAuth configures the authentication method for the ngrok provider.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [NgrokProvider](#ngrokprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKey` _[NgrokProviderSecretRef](#ngrokprovidersecretref)_ | APIKey is the API Key used to authenticate with ngrok. See https://ngrok.com/docs/api/#authentication |  | Optional: \{\} <br /> |


#### NgrokProvider



NgrokProvider configures a store to sync secrets with a ngrok vault to use in traffic policies.
See: https://ngrok.com/blog-post/secrets-for-traffic-policy



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the URL of the ngrok API. | https://api.ngrok.com |  |
| `auth` _[NgrokAuth](#ngrokauth)_ | Auth configures how the ngrok provider authenticates with the ngrok API. |  | MaxProperties: 1 <br />MinProperties: 1 <br />Required: \{\} <br /> |
| `vault` _[NgrokVault](#ngrokvault)_ | Vault configures the ngrok vault to sync secrets with. |  | Required: \{\} <br /> |


#### NgrokProviderSecretRef



NgrokProviderSecretRef contains the secret reference for the ngrok provider.



_Appears in:_
- [NgrokAuth](#ngrokauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef is a reference to a secret containing the ngrok API key. |  | Optional: \{\} <br /> |


#### NgrokVault



NgrokVault configures the ngrok vault to sync secrets with.



_Appears in:_
- [NgrokProvider](#ngrokprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the ngrok vault to sync secrets with. |  | Required: \{\} <br /> |






#### OciAuthCredentials



OciAuthCredentials represents the credentials for OCI authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `privateKey` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `privateKeyPassphrase` _[SecretKeySelector](#secretkeyselector)_ |  |  | Optional: \{\} <br /> |
| `fingerprint` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `userId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `tenancyId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `region` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### OnboardbaseAuthSecretRef



OnboardbaseAuthSecretRef holds secret references for onboardbase API Key credentials.



_Appears in:_
- [OnboardbaseProvider](#onboardbaseprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKeyRef` _[SecretKeySelector](#secretkeyselector)_ | OnboardbaseAPIKey is the APIKey generated by an admin account.<br />It is used to recognize and authorize access to a project and environment within onboardbase |  | Required: \{\} <br /> |
| `passcodeRef` _[SecretKeySelector](#secretkeyselector)_ | OnboardbasePasscode is the passcode attached to the API Key |  | Required: \{\} <br /> |


#### OnboardbaseProvider



OnboardbaseProvider configures a store to sync secrets using the Onboardbase provider.
Project and Config are required if not using a Service Token.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[OnboardbaseAuthSecretRef](#onboardbaseauthsecretref)_ | Auth configures how the Operator authenticates with the Onboardbase API |  |  |
| `apiHost` _string_ | APIHost use this to configure the host url for the API for selfhosted installation, default is https://public.onboardbase.com/api/v1/ | https://public.onboardbase.com/api/v1/ |  |
| `project` _string_ | Project is an onboardbase project that the secrets should be pulled from | development | Required: \{\} <br /> |
| `environment` _string_ | Environment is the name of an environmnent within a project to pull the secrets from | development | Required: \{\} <br /> |


#### OnePasswordAuth



OnePasswordAuth contains a secretRef for credentials.



_Appears in:_
- [OnePasswordProvider](#onepasswordprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[OnePasswordAuthSecretRef](#onepasswordauthsecretref)_ |  |  |  |


#### OnePasswordAuthSecretRef



OnePasswordAuthSecretRef holds secret references for 1Password credentials.



_Appears in:_
- [OnePasswordAuth](#onepasswordauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectTokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The ConnectToken is used for authentication to a 1Password Connect Server. |  |  |


#### OnePasswordProvider



OnePasswordProvider configures a store to sync secrets using the 1Password Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[OnePasswordAuth](#onepasswordauth)_ | Auth defines the information necessary to authenticate against OnePassword Connect Server |  |  |
| `connectHost` _string_ | ConnectHost defines the OnePassword Connect Server to connect to |  |  |
| `vaults` _object (keys:string, values:integer)_ | Vaults defines which OnePassword vaults to search in which order |  |  |


#### OnePasswordSDKAuth



OnePasswordSDKAuth contains a secretRef for the service account token.



_Appears in:_
- [OnePasswordSDKProvider](#onepasswordsdkprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountSecretRef` _[SecretKeySelector](#secretkeyselector)_ | ServiceAccountSecretRef points to the secret containing the token to access 1Password vault. |  |  |


#### OnePasswordSDKProvider



OnePasswordSDKProvider configures a store to sync secrets using the 1Password sdk.
Exactly one of Vault or Environment must be set.

_Validation:_
- AtMostOneOf: [vault environment]

_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vault` _string_ | Vault defines the vault's name or uuid to access. Do NOT add op:// prefix. This will be done automatically.<br />Mutually exclusive with Environment. |  | Optional: \{\} <br /> |
| `environment` _string_ | Environment defines the 1Password Environment ID to read variables from.<br />Environments are read-only: PushSecret, DeleteSecret, and SecretExists return an error when set.<br />Mutually exclusive with Vault. |  | Optional: \{\} <br /> |
| `integrationInfo` _[IntegrationInfo](#integrationinfo)_ | IntegrationInfo specifies the name and version of the integration built using the 1Password Go SDK.<br />If you don't know which name and version to use, use `DefaultIntegrationName` and `DefaultIntegrationVersion`, respectively. |  | Optional: \{\} <br /> |
| `auth` _[OnePasswordSDKAuth](#onepasswordsdkauth)_ | Auth defines the information necessary to authenticate against OnePassword API. |  |  |
| `cache` _[CacheConfig](#cacheconfig)_ | Cache configures client-side caching for read operations (GetSecret, GetSecretMap).<br />When enabled, secrets are cached with the specified TTL.<br />Write operations (PushSecret, DeleteSecret) automatically invalidate relevant cache entries.<br />If omitted, caching is disabled (default).<br />cache: \{\} is a valid option to set. |  | Optional: \{\} <br /> |


#### OpenBaoAppRole



OpenBaoAppRole authenticates with OpenBao using the [App Role auth
mechanism], with the role and secret stored in a Kubernetes Secret resource.
The role ID has to be specified either inline via `roleId` or by referencing
a secret via `roleRef`.

[App Role auth mechanism]: https://openbao.org/docs/auth/approle/

_Validation:_
- ExactlyOneOf: [roleId roleRef]

_Appears in:_
- [OpenBaoAuth](#openbaoauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the App Role authentication backend is mounted<br />in OpenBao, e.g: "approle" | approle |  |
| `roleId` _string_ | RoleID configured in the App Role authentication backend when setting<br />up the authentication backend in OpenBao. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `roleRef` _[SecretKeySelector](#secretkeyselector)_ | Reference to a key in a Secret that contains the App Role ID used<br />to authenticate with OpenBao.<br />The `key` field must be specified and denotes which entry within the Secret<br />resource is used as the app role id. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Reference to a key in a Secret that contains the App Role secret used<br />to authenticate with OpenBao.<br />The `key` field must be specified and denotes which entry within the Secret<br />resource is used as the app role secret. |  |  |


#### OpenBaoAuth



OpenBaoAuth is the configuration used to authenticate with an OpenBao server.
Currently the following authentication methods are supported: [AppRole],
[Token] and [UserPass]

Additional authentication methods are planned for future releases.

[AppRole]: https://openbao.org/docs/auth/approle/
[Token]: https://openbao.org/docs/auth/token/
[UserPass]: https://openbao.org/docs/auth/userpass/

_Validation:_
- ExactlyOneOf: [appRole tokenSecretRef userPass]

_Appears in:_
- [OpenBaoProvider](#openbaoprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `appRole` _[OpenBaoAppRole](#openbaoapprole)_ | AppRole authenticates with OpenBao using the [App Role auth mechanism],<br />with the role and secret stored in a Kubernetes Secret resource.<br />[App Role auth mechanism]: https://openbao.org/docs/auth/approle/ |  | ExactlyOneOf: [roleId roleRef] <br />Optional: \{\} <br /> |
| `namespace` _string_ | Name of the [OpenBao Namespace] to authenticate to. This can be different<br />than the namespace your secret is in. Namespaces is a set of features<br />within OpenBao that allows OpenBao environments to support secure<br />multi-tenancy. e.g: "ns1". This will default to OpenBao.Namespace field<br />if set, or empty otherwise<br />[OpenBao Namespace]: https://openbao.org/docs/concepts/namespaces/ |  | Optional: \{\} <br /> |
| `tokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | TokenSecretRef authenticates with OpenBao by presenting a token. |  | Optional: \{\} <br /> |
| `userPass` _[OpenBaoUserPassAuth](#openbaouserpassauth)_ | UserPass authenticates with OpenBao by passing a username/password pair |  | Optional: \{\} <br /> |


#### OpenBaoKVStoreVersion

_Underlying type:_ _string_

OpenBaoKVStoreVersion represents the version of the OpenBao KV secret engine.



_Appears in:_
- [OpenBaoProvider](#openbaoprovider)

| Field | Description |
| --- | --- |
| `v1` |  |
| `v2` |  |


#### OpenBaoProvider



OpenBaoProvider configures a store to sync secrets using an OpenBao KV backend.

_Validation:_
- AtMostOneOf: [caBundle caProvider]

_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[OpenBaoAuth](#openbaoauth)_ | Auth configures how secret-manager authenticates with the OpenBao server. |  | ExactlyOneOf: [appRole tokenSecretRef userPass] <br />Optional: \{\} <br /> |
| `caBundle` _integer array_ | PEM encoded CA bundle used to validate the OpenBao server certificate. If<br />this and `caProvider` are not set the system root certificates are used<br />to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | The provider for the CA bundle to use to validate OpenBao server<br />certificate. If this and `caBundle` are not set the system root<br />certificates are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `namespace` _string_ | Name of the [OpenBao Namespace]. Namespaces is a set of features within<br />OpenBao that allows OpenBao environments to support secure multi-tenancy.<br />e.g: "ns1".<br />[OpenBao Namespace]: https://openbao.org/docs/concepts/namespaces/ |  | Optional: \{\} <br /> |
| `server` _string_ | Server is the connection address for the OpenBao server, e.g: `https://openbao.example.com:8200`. |  |  |
| `path` _string_ | Path is the mount path of the OpenBao KV backend endpoint, e.g:<br />"secret". The v2 KV secret engine version specific "/data" path suffix<br />for fetching secrets from OpenBao is optional and will be appended<br />if not present in specified path. |  | Optional: \{\} <br /> |
| `version` _[OpenBaoKVStoreVersion](#openbaokvstoreversion)_ | Version is the OpenBao KV secret engine version. This can be either "v1" or<br />"v2". Version defaults to "v2". | v2 | Enum: [v1 v2] <br />Optional: \{\} <br /> |


#### OpenBaoUserPassAuth



OpenBaoUserPassAuth authenticates with OpenBao using [UserPass authentication
method], with the username and password stored in a Kubernetes Secret
resource.

[UserPass authentication method]: https://openbao.org/docs/auth/userpass/



_Appears in:_
- [OpenBaoAuth](#openbaoauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the UserPassword authentication backend is mounted<br />in OpenBao, e.g: "userpass" | userpass |  |
| `username` _string_ | Username is a username used to authenticate using the [UserPass<br />authentication method]<br />[UserPass authentication method]: https://openbao.org/docs/auth/userpass/ |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef to a key in a Secret resource containing password for the user<br />used to authenticate with OpenBao using the [UserPass authentication<br />method]<br />[UserPass authentication method]: https://openbao.org/docs/auth/userpass/ |  |  |


#### OracleAuth



OracleAuth defines the authentication method for the Oracle Vault provider.



_Appears in:_
- [OracleProvider](#oracleprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tenancy` _string_ | Tenancy is the tenancy OCID where user is located. |  |  |
| `user` _string_ | User is an access OCID specific to the account. |  |  |
| `secretRef` _[OracleSecretRef](#oraclesecretref)_ | SecretRef to pass through sensitive information. |  |  |


#### OraclePrincipalType

_Underlying type:_ _string_

OraclePrincipalType defines the type of principal used for authentication with Oracle Vault.

_Validation:_
- Enum: [ UserPrincipal InstancePrincipal Workload]

_Appears in:_
- [OracleProvider](#oracleprovider)

| Field | Description |
| --- | --- |
| `UserPrincipal` | UserPrincipal represents a user principal.<br /> |
| `InstancePrincipal` | InstancePrincipal represents a instance principal.<br /> |
| `Workload` | WorkloadPrincipal represents a workload principal.<br /> |


#### OracleProvider



OracleProvider configures a store to sync secrets using an Oracle Vault
backend.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `region` _string_ | Region is the region where vault is located. |  |  |
| `vault` _string_ | Vault is the vault's OCID of the specific vault where secret is located. |  |  |
| `compartment` _string_ | Compartment is the vault compartment OCID.<br />Required for PushSecret |  | Optional: \{\} <br /> |
| `encryptionKey` _string_ | EncryptionKey is the OCID of the encryption key within the vault.<br />Required for PushSecret |  | Optional: \{\} <br /> |
| `principalType` _[OraclePrincipalType](#oracleprincipaltype)_ | The type of principal to use for authentication. If left blank, the Auth struct will<br />determine the principal type. This optional field must be specified if using<br />workload identity. |  | Enum: [ UserPrincipal InstancePrincipal Workload] <br />Optional: \{\} <br /> |
| `auth` _[OracleAuth](#oracleauth)_ | Auth configures how secret-manager authenticates with the Oracle Vault.<br />If empty, use the instance principal, otherwise the user credentials specified in Auth. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specified the service account<br />that should be used when authenticating with WorkloadIdentity. |  | Optional: \{\} <br /> |


#### OracleSecretRef



OracleSecretRef contains the secret reference for Oracle Vault authentication credentials.



_Appears in:_
- [OracleAuth](#oracleauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `privatekey` _[SecretKeySelector](#secretkeyselector)_ | PrivateKey is the user's API Signing Key in PEM format, used for authentication. |  |  |
| `fingerprint` _[SecretKeySelector](#secretkeyselector)_ | Fingerprint is the fingerprint of the API private key. |  |  |


#### OvhAuth



OvhAuth tells the controller how to authenticate to OVHcloud's Secret Manager, either using mTLS or a token.



_Appears in:_
- [OvhProvider](#ovhprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mtls` _[OvhClientMTLS](#ovhclientmtls)_ |  |  | Optional: \{\} <br /> |
| `token` _[OvhClientToken](#ovhclienttoken)_ |  |  | Optional: \{\} <br /> |


#### OvhClientMTLS



OvhClientMTLS defines the configuration required to authenticate to OVHcloud's Secret Manager using mTLS.



_Appears in:_
- [OvhAuth](#ovhauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `keySecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `caBundle` _integer array_ |  |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ |  |  | Optional: \{\} <br /> |


#### OvhClientToken



OvhClientToken defines the configuration required to authenticate to OVHcloud's Secret Manager using a token.



_Appears in:_
- [OvhAuth](#ovhauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### OvhProvider



OvhProvider holds the configuration to synchronize secrets with OVHcloud's Secret Manager.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `server` _string_ | specifies the OKMS server endpoint. |  | Required: \{\} <br /> |
| `okmsid` _string_ | specifies the OKMS ID. |  | Required: \{\} <br /> |
| `casRequired` _boolean_ | Enables or disables check-and-set (CAS) (default: false). |  | Optional: \{\} <br /> |
| `okmsTimeout` _integer_ | Setup a timeout in seconds when requests to the KMS are made (default: 30). | 30 | Minimum: 1 <br />Optional: \{\} <br /> |
| `auth` _[OvhAuth](#ovhauth)_ | Authentication method (mtls or token). |  | Required: \{\} <br /> |


#### PassboltAuth



PassboltAuth contains a secretRef for the passbolt credentials.



_Appears in:_
- [PassboltProvider](#passboltprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `passwordSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `privateKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### PassboltProvider



PassboltProvider provides access to Passbolt secrets manager.
See: https://www.passbolt.com.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[PassboltAuth](#passboltauth)_ | Auth defines the information necessary to authenticate against Passbolt Server |  |  |
| `host` _string_ | Host defines the Passbolt Server to connect to |  |  |
| `caBundle` _integer array_ | PEM encoded CA bundle used to validate Passbolt server certificate. Only used<br />if the Host URL is using HTTPS protocol. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | The provider for the CA bundle to use to validate Passbolt server certificate. |  | Optional: \{\} <br /> |


#### PasswordDepotAuth



PasswordDepotAuth defines the authentication method for the Password Depot provider.



_Appears in:_
- [PasswordDepotProvider](#passworddepotprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[PasswordDepotSecretRef](#passworddepotsecretref)_ |  |  |  |


#### PasswordDepotProvider



PasswordDepotProvider configures a store to sync secrets with a Password Depot instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ | URL configures the Password Depot instance URL. |  |  |
| `database` _string_ | Database to use as source |  |  |
| `auth` _[PasswordDepotAuth](#passworddepotauth)_ | Auth configures how secret-manager authenticates with a Password Depot instance. |  |  |


#### PasswordDepotSecretRef



PasswordDepotSecretRef contains the secret reference for Password Depot authentication.



_Appears in:_
- [PasswordDepotAuth](#passworddepotauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `credentials` _[SecretKeySelector](#secretkeyselector)_ | Username / Password is used for authentication. |  | Optional: \{\} <br /> |


#### PreviderAuth



PreviderAuth contains a secretRef for credentials.



_Appears in:_
- [PreviderProvider](#previderprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[PreviderAuthSecretRef](#previderauthsecretref)_ |  |  | Optional: \{\} <br /> |


#### PreviderAuthSecretRef



PreviderAuthSecretRef holds secret references for Previder Vault credentials.



_Appears in:_
- [PreviderAuth](#previderauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessToken` _[SecretKeySelector](#secretkeyselector)_ | The AccessToken is used for authentication |  |  |


#### PreviderProvider



PreviderProvider configures a store to sync secrets using the Previder Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[PreviderAuth](#previderauth)_ |  |  |  |
| `baseUri` _string_ |  |  | Optional: \{\} <br /> |


#### Provider

_Underlying type:_ _interface{Capabilities() SecretStoreCapabilities; NewClient(ctx context.Context, store GenericStore, kube sigs.k8s.io/controller-runtime/pkg/client.Client, namespace string) (SecretsClient, error); ValidateStore(store GenericStore) (sigs.k8s.io/controller-runtime/pkg/webhook/admission.Warnings, error)}_

Provider is a common interface for interacting with secret backends.







#### PulumiAuth



PulumiAuth configures authentication with the Pulumi API.
Exactly one of accessToken or oidcConfig must be specified.



_Appears in:_
- [PulumiProvider](#pulumiprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessToken` _[PulumiProviderSecretRef](#pulumiprovidersecretref)_ | AccessToken authenticates using a Pulumi access token stored in a Kubernetes Secret. |  | Optional: \{\} <br /> |
| `oidcConfig` _[PulumiOIDCAuth](#pulumioidcauth)_ | OIDCConfig authenticates using Kubernetes ServiceAccount tokens via OIDC. |  | Optional: \{\} <br /> |


#### PulumiOIDCAuth



PulumiOIDCAuth configures OIDC authentication with Pulumi using Kubernetes ServiceAccount tokens.



_Appears in:_
- [PulumiAuth](#pulumiauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `organization` _string_ | Organization is the name of the Pulumi organization configured for OIDC authentication. |  |  |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specifies the Kubernetes ServiceAccount to use for authentication. |  |  |
| `expirationSeconds` _integer_ | ExpirationSeconds sets the token validity duration for service account and OIDC token.<br />Defaults to 10 minutes. | 600 | Minimum: 600 <br />Optional: \{\} <br /> |


#### PulumiProvider



PulumiProvider defines configuration for accessing secrets from Pulumi ESC.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the URL of the Pulumi API. | https://api.pulumi.com/api/esc |  |
| `auth` _[PulumiAuth](#pulumiauth)_ | Auth configures how the Operator authenticates with the Pulumi API.<br />Either auth or the deprecated accessToken field must be specified. |  | Optional: \{\} <br /> |
| `organization` _string_ | Organization are a space to collaborate on shared projects and stacks.<br />To create a new organization, visit https://app.pulumi.com/ and click "New Organization". |  |  |
| `project` _string_ | Project is the name of the Pulumi ESC project the environment belongs to. |  |  |
| `environment` _string_ | Environment are YAML documents composed of static key-value pairs, programmatic expressions,<br />dynamically retrieved values from supported providers including all major clouds,<br />and other Pulumi ESC environments.<br />To create a new environment, visit https://www.pulumi.com/docs/esc/environments/ for more information. |  |  |
| `accessToken` _[PulumiProviderSecretRef](#pulumiprovidersecretref)_ | AccessToken is the access tokens to sign in to the Pulumi Cloud Console.<br />Deprecated: Use auth.accessToken instead. |  | Optional: \{\} <br /> |


#### PulumiProviderSecretRef



PulumiProviderSecretRef contains the secret reference for Pulumi authentication.



_Appears in:_
- [PulumiAuth](#pulumiauth)
- [PulumiProvider](#pulumiprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef is a reference to a secret containing the Pulumi API token. |  |  |


#### PushSecretData

_Underlying type:_ _interface{GetMetadata() *k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON; GetProperty() string; GetRemoteKey() string; GetSecretKey() string}_

PushSecretData is an interface to allow using v1alpha1.PushSecretData content in Provider registered in v1.







#### PushSecretRemoteRef

_Underlying type:_ _interface{GetProperty() string; GetRemoteKey() string}_

PushSecretRemoteRef is an interface to allow using v1alpha1.PushSecretRemoteRef in Provider registered in v1.







#### ScalewayProvider



ScalewayProvider defines the configuration for the Scaleway Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the url of the api to use. Defaults to https://api.scaleway.com |  | Optional: \{\} <br /> |
| `region` _string_ | Region where your secrets are located: https://developers.scaleway.com/en/quickstart/#region-and-zone |  |  |
| `projectId` _string_ | ProjectID is the id of your project, which you can find in the console: https://console.scaleway.com/project/settings |  |  |
| `accessKey` _[ScalewayProviderSecretRef](#scalewayprovidersecretref)_ | AccessKey is the non-secret part of the api key. |  |  |
| `secretKey` _[ScalewayProviderSecretRef](#scalewayprovidersecretref)_ | SecretKey is the non-secret part of the api key. |  |  |


#### ScalewayProviderSecretRef



ScalewayProviderSecretRef defines the configuration for Scaleway secret references.



_Appears in:_
- [ScalewayProvider](#scalewayprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### SecretReference



SecretReference holds the details of a secret.



_Appears in:_
- [AwsCredentialsConfig](#awscredentialsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | name of the secret. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Required: \{\} <br /> |
| `namespace` _string_ | namespace in which the secret exists. If empty, secret will looked up in local namespace. |  | MaxLength: 63 <br />MinLength: 1 <br />Optional: \{\} <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br /> |


#### SecretServerProvider



SecretServerProvider provides access to authenticate to a secrets provider server.
See: https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go.
Authentication requires either Token, or both Username and Password. If Token is
set it takes precedence and Username/Password are ignored.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `username` _[SecretServerProviderRef](#secretserverproviderref)_ | Username is the secret server account username.<br />Required unless Token is set. |  | Optional: \{\} <br /> |
| `password` _[SecretServerProviderRef](#secretserverproviderref)_ | Password is the secret server account password.<br />Required unless Token is set. |  | Optional: \{\} <br /> |
| `token` _[SecretServerProviderRef](#secretserverproviderref)_ | Token is an access token used to authenticate to the secret server,<br />as an alternative to Username and Password. When set, Username and<br />Password are not required and are ignored. |  | Optional: \{\} <br /> |
| `domain` _string_ | Domain is the secret server domain. |  | Optional: \{\} <br /> |
| `serverURL` _string_ | ServerURL<br />URL to your secret server installation |  | Required: \{\} <br /> |
| `caBundle` _integer array_ | PEM/base64 encoded CA bundle used to validate Secret ServerURL. Only used<br />if the ServerURL URL is using HTTPS protocol. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | The provider for the CA bundle to use to validate Secret ServerURL certificate. |  | Optional: \{\} <br /> |


#### SecretServerProviderRef



SecretServerProviderRef references a value that can be specified directly or via a secret
for a SecretServerProvider. Exactly one of Value or SecretRef must be set.



_Appears in:_
- [SecretServerProvider](#secretserverprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### SecretStore



SecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1` | | |
| `kind` _string_ | `SecretStore` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SecretStoreSpec](#secretstorespec)_ |  |  |  |
| `status` _[SecretStoreStatus](#secretstorestatus)_ |  |  |  |


#### SecretStoreCapabilities

_Underlying type:_ _string_

SecretStoreCapabilities defines the possible operations a SecretStore can do.



_Appears in:_
- [SecretStoreStatus](#secretstorestatus)

| Field | Description |
| --- | --- |
| `ReadOnly` | SecretStoreReadOnly indicates that the store can only read secrets.<br /> |
| `WriteOnly` | SecretStoreWriteOnly indicates that the store can only write secrets.<br /> |
| `ReadWrite` | SecretStoreReadWrite indicates that the store can both read and write secrets.<br /> |


#### SecretStoreConditionType

_Underlying type:_ _string_

SecretStoreConditionType represents the condition of the SecretStore.



_Appears in:_
- [SecretStoreStatusCondition](#secretstorestatuscondition)

| Field | Description |
| --- | --- |
| `Ready` | SecretStoreReady indicates that the store is ready and able to serve requests.<br /> |


#### SecretStoreProvider



SecretStoreProvider contains the provider-specific configuration.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [SecretStoreSpec](#secretstorespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `aws` _[AWSProvider](#awsprovider)_ | AWS configures this store to sync secrets using AWS Secret Manager provider |  | Optional: \{\} <br /> |
| `azurekv` _[AzureKVProvider](#azurekvprovider)_ | AzureKV configures this store to sync secrets using Azure Key Vault provider |  | Optional: \{\} <br /> |
| `akeyless` _[AkeylessProvider](#akeylessprovider)_ | Akeyless configures this store to sync secrets using Akeyless Vault provider |  | Optional: \{\} <br /> |
| `bitwardensecretsmanager` _[BitwardenSecretsManagerProvider](#bitwardensecretsmanagerprovider)_ | BitwardenSecretsManager configures this store to sync secrets using BitwardenSecretsManager provider |  | Optional: \{\} <br /> |
| `vault` _[VaultProvider](#vaultprovider)_ | Vault configures this store to sync secrets using the HashiCorp Vault provider. |  | Optional: \{\} <br /> |
| `ovh` _[OvhProvider](#ovhprovider)_ | OVHcloud configures this store to sync secrets using the OVHcloud provider. |  | Optional: \{\} <br /> |
| `gcpsm` _[GCPSMProvider](#gcpsmprovider)_ | GCPSM configures this store to sync secrets using Google Cloud Platform Secret Manager provider |  | Optional: \{\} <br /> |
| `oracle` _[OracleProvider](#oracleprovider)_ | Oracle configures this store to sync secrets using Oracle Vault provider |  | Optional: \{\} <br /> |
| `ibm` _[IBMProvider](#ibmprovider)_ | IBM configures this store to sync secrets using IBM Cloud provider |  | Optional: \{\} <br /> |
| `yandexcertificatemanager` _[YandexCertificateManagerProvider](#yandexcertificatemanagerprovider)_ | YandexCertificateManager configures this store to sync secrets using Yandex Certificate Manager provider |  | Optional: \{\} <br /> |
| `yandexlockbox` _[YandexLockboxProvider](#yandexlockboxprovider)_ | YandexLockbox configures this store to sync secrets using Yandex Lockbox provider |  | Optional: \{\} <br /> |
| `github` _[GithubProvider](#githubprovider)_ | Github configures this store to push GitHub Actions secrets using the GitHub API provider.<br />Note: This provider only supports write operations (PushSecret) and cannot fetch secrets from GitHub |  | Optional: \{\} <br /> |
| `gitlab` _[GitlabProvider](#gitlabprovider)_ | GitLab configures this store to sync secrets using GitLab Variables provider |  | Optional: \{\} <br /> |
| `onepassword` _[OnePasswordProvider](#onepasswordprovider)_ | OnePassword configures this store to sync secrets using the 1Password Cloud provider |  | Optional: \{\} <br /> |
| `onepasswordSDK` _[OnePasswordSDKProvider](#onepasswordsdkprovider)_ | OnePasswordSDK configures this store to use 1Password's new Go SDK to sync secrets. |  | AtMostOneOf: [vault environment] <br />Optional: \{\} <br /> |
| `webhook` _[WebhookProvider](#webhookprovider)_ | Webhook configures this store to sync secrets using a generic templated webhook |  | Optional: \{\} <br /> |
| `kubernetes` _[KubernetesProvider](#kubernetesprovider)_ | Kubernetes configures this store to sync secrets using a Kubernetes cluster provider |  | Optional: \{\} <br /> |
| `crd` _[CRDProvider](#crdprovider)_ | CRD configures this store to sync secrets from arbitrary Kubernetes resources,<br />including both custom resources (CRDs) and core API resources. Resources are<br />selected by API group, version and kind, where group can be "" (empty string)<br />for core resources such as ConfigMap. Reading the core v1 Secret is<br />intentionally blocked — use the Kubernetes provider for that. |  | AtMostOneOf: [auth authRef] <br />Optional: \{\} <br /> |
| `fake` _[FakeProvider](#fakeprovider)_ | Fake configures a store with static key/value pairs |  | Optional: \{\} <br /> |
| `senhasegura` _[SenhaseguraProvider](#senhaseguraprovider)_ | Senhasegura configures this store to sync secrets using senhasegura provider |  | Optional: \{\} <br /> |
| `scaleway` _[ScalewayProvider](#scalewayprovider)_ | Scaleway configures this store to sync secrets using the Scaleway provider. |  | Optional: \{\} <br /> |
| `doppler` _[DopplerProvider](#dopplerprovider)_ | Doppler configures this store to sync secrets using the Doppler provider |  | Optional: \{\} <br /> |
| `previder` _[PreviderProvider](#previderprovider)_ | Previder configures this store to sync secrets using the Previder provider |  | Optional: \{\} <br /> |
| `onboardbase` _[OnboardbaseProvider](#onboardbaseprovider)_ | Onboardbase configures this store to sync secrets using the Onboardbase provider |  | Optional: \{\} <br /> |
| `keepersecurity` _[KeeperSecurityProvider](#keepersecurityprovider)_ | KeeperSecurity configures this store to sync secrets using the KeeperSecurity provider |  | Optional: \{\} <br /> |
| `conjur` _[ConjurProvider](#conjurprovider)_ | Conjur configures this store to sync secrets using conjur provider |  | Optional: \{\} <br /> |
| `delinea` _[DelineaProvider](#delineaprovider)_ | Delinea DevOps Secrets Vault<br />https://docs.delinea.com/online-help/products/devops-secrets-vault/current |  | Optional: \{\} <br /> |
| `secretserver` _[SecretServerProvider](#secretserverprovider)_ | SecretServer configures this store to sync secrets using SecretServer provider<br />https://docs.delinea.com/online-help/secret-server/start.htm |  | Optional: \{\} <br /> |
| `chef` _[ChefProvider](#chefprovider)_ | Chef configures this store to sync secrets with chef server |  | Optional: \{\} <br /> |
| `pulumi` _[PulumiProvider](#pulumiprovider)_ | Pulumi configures this store to sync secrets using the Pulumi provider |  | Optional: \{\} <br /> |
| `fortanix` _[FortanixProvider](#fortanixprovider)_ | Fortanix configures this store to sync secrets using the Fortanix provider |  | Optional: \{\} <br /> |
| `passworddepot` _[PasswordDepotProvider](#passworddepotprovider)_ |  |  | Optional: \{\} <br /> |
| `passbolt` _[PassboltProvider](#passboltprovider)_ |  |  | Optional: \{\} <br /> |
| `dvls` _[DVLSProvider](#dvlsprovider)_ | DVLS configures this store to sync secrets using Devolutions Server provider |  | Optional: \{\} <br /> |
| `infisical` _[InfisicalProvider](#infisicalprovider)_ | Infisical configures this store to sync secrets using the Infisical provider |  | Optional: \{\} <br /> |
| `beyondtrust` _[BeyondtrustProvider](#beyondtrustprovider)_ | Beyondtrust configures this store to sync secrets using Password Safe provider. |  | Optional: \{\} <br /> |
| `beyondtrustworkloadcredentials` _[BeyondtrustWorkloadCredentialsProvider](#beyondtrustworkloadcredentialsprovider)_ | BeyondtrustWorkloadCredentials configures this store to sync secrets using the BeyondTrust Workload Credentials provider. |  | Optional: \{\} <br /> |
| `cloudrusm` _[CloudruSMProvider](#cloudrusmprovider)_ | CloudruSM configures this store to sync secrets using the Cloud.ru Secret Manager provider |  | Optional: \{\} <br /> |
| `volcengine` _[VolcengineProvider](#volcengineprovider)_ | Volcengine configures this store to sync secrets using the Volcengine provider |  | Optional: \{\} <br /> |
| `ngrok` _[NgrokProvider](#ngrokprovider)_ | Ngrok configures this store to sync secrets using the ngrok provider. |  | Optional: \{\} <br /> |
| `barbican` _[BarbicanProvider](#barbicanprovider)_ | Barbican configures this store to sync secrets using the OpenStack Barbican provider |  | Optional: \{\} <br /> |
| `nebiusmysterybox` _[NebiusMysteryboxProvider](#nebiusmysteryboxprovider)_ | NebiusMysterybox configures this store to sync secrets using NebiusMysterybox provider |  | Optional: \{\} <br /> |
| `openBao` _[OpenBaoProvider](#openbaoprovider)_ | OpenBao configures this store to sync secrets using the OpenBao provider. |  | AtMostOneOf: [caBundle caProvider] <br />Optional: \{\} <br /> |


#### SecretStoreRef



SecretStoreRef defines which SecretStore to fetch the ExternalSecret data.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)
- [StoreGeneratorSourceRef](#storegeneratorsourceref)
- [StoreSourceRef](#storesourceref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the SecretStore resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `kind` _string_ | Kind of the SecretStore resource (SecretStore or ClusterSecretStore)<br />Defaults to `SecretStore` |  | Enum: [SecretStore ClusterSecretStore] <br />Optional: \{\} <br /> |


#### SecretStoreRetrySettings



SecretStoreRetrySettings defines the retry settings for accessing external secrets manager stores.



_Appears in:_
- [BeyondtrustWorkloadCredentialsDynamicSecretSpec](#beyondtrustworkloadcredentialsdynamicsecretspec)
- [SecretStoreSpec](#secretstorespec)
- [VaultDynamicSecretSpec](#vaultdynamicsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxRetries` _integer_ |  |  |  |
| `retryInterval` _string_ |  |  |  |


#### SecretStoreSpec



SecretStoreSpec defines the desired state of SecretStore.



_Appears in:_
- [ClusterSecretStore](#clustersecretstore)
- [SecretStore](#secretstore)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `controller` _string_ | Used to select the correct ESO controller (think: ingress.ingressClassName)<br />The ESO controller is instantiated with a specific controller name and filters ES based on this property |  | Optional: \{\} <br /> |
| `provider` _[SecretStoreProvider](#secretstoreprovider)_ | Used to configure the provider. Only one provider may be set |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `retrySettings` _[SecretStoreRetrySettings](#secretstoreretrysettings)_ | Used to configure HTTP retries on failures. |  | Optional: \{\} <br /> |
| `refreshInterval` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#intorstring-intstr-util)_ | Used to configure store refresh interval. Accepts either an integer number<br />of seconds (legacy) or a Go duration string such as "1h" or "5m". Empty or<br />0 will default to the controller config. |  | XIntOrString: \{\} <br />Optional: \{\} <br /> |
| `conditions` _[ClusterSecretStoreCondition](#clustersecretstorecondition) array_ | Used to constrain a ClusterSecretStore to specific namespaces. Relevant only to ClusterSecretStore. |  | Optional: \{\} <br /> |


#### SecretStoreStatus



SecretStoreStatus defines the observed state of the SecretStore.



_Appears in:_
- [ClusterSecretStore](#clustersecretstore)
- [SecretStore](#secretstore)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[SecretStoreStatusCondition](#secretstorestatuscondition) array_ |  |  | Optional: \{\} <br /> |
| `capabilities` _[SecretStoreCapabilities](#secretstorecapabilities)_ |  |  | Optional: \{\} <br /> |


#### SecretStoreStatusCondition



SecretStoreStatusCondition contains condition information for a SecretStore.



_Appears in:_
- [SecretStoreStatus](#secretstorestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[SecretStoreConditionType](#secretstoreconditiontype)_ |  |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `reason` _string_ |  |  | Optional: \{\} <br /> |
| `message` _string_ |  |  | Optional: \{\} <br /> |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ |  |  | Optional: \{\} <br /> |


#### SecretVersionSelectionPolicy

_Underlying type:_ _string_

SecretVersionSelectionPolicy defines the policy for selecting secret versions in GCP Secret Manager.



_Appears in:_
- [GCPSMProvider](#gcpsmprovider)

| Field | Description |
| --- | --- |
| `LatestOrFail` | SecretVersionSelectionPolicyLatestOrFail means the provider always uses "latest", or fails if that version is disabled/destroyed.<br /> |
| `LatestOrFetch` | SecretVersionSelectionPolicyLatestOrFetch behaves like SecretVersionSelectionPolicyLatestOrFail but falls back to fetching the latest version if the version is DESTROYED or DISABLED.<br /> |


#### SecretsClient

_Underlying type:_ _interface{Close(ctx context.Context) error; DeleteSecret(ctx context.Context, remoteRef PushSecretRemoteRef) error; GetAllSecrets(ctx context.Context, ref ExternalSecretFind) (map[string][]byte, error); GetSecret(ctx context.Context, ref ExternalSecretDataRemoteRef) ([]byte, error); GetSecretMap(ctx context.Context, ref ExternalSecretDataRemoteRef) (map[string][]byte, error); PushSecret(ctx context.Context, secret *k8s.io/api/core/v1.Secret, data PushSecretData) error; SecretExists(ctx context.Context, remoteRef PushSecretRemoteRef) (bool, error); Validate() (ValidationResult, error)}_

SecretsClient provides access to secrets.







#### SecretsManager



SecretsManager defines how the provider behaves when interacting with AWS
SecretsManager. Some of these settings are only applicable to controlling how
secrets are deleted, and hence only apply to PushSecret (and only when
deletionPolicy is set to Delete).



_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `forceDeleteWithoutRecovery` _boolean_ | Specifies whether to delete the secret without any recovery window. You<br />can't use both this parameter and RecoveryWindowInDays in the same call.<br />If you don't use either, then by default Secrets Manager uses a 30 day<br />recovery window.<br />see: https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-ForceDeleteWithoutRecovery |  | Optional: \{\} <br /> |
| `recoveryWindowInDays` _integer_ | The number of days from 7 to 30 that Secrets Manager waits before<br />permanently deleting the secret. You can't use both this parameter and<br />ForceDeleteWithoutRecovery in the same call. If you don't use either,<br />then by default Secrets Manager uses a 30-day recovery window.<br />see: https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-RecoveryWindowInDays |  | Optional: \{\} <br /> |


#### SenhaseguraAuth



SenhaseguraAuth tells the controller how to do auth in senhasegura.



_Appears in:_
- [SenhaseguraProvider](#senhaseguraprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _string_ |  |  |  |
| `clientSecretSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### SenhaseguraModuleType

_Underlying type:_ _string_

SenhaseguraModuleType enum defines senhasegura target module to fetch secrets
+kubebuilder:validation:Enum=DSM



_Appears in:_
- [SenhaseguraProvider](#senhaseguraprovider)

| Field | Description |
| --- | --- |
| `DSM` | 		SenhaseguraModuleDSM is the senhasegura DevOps Secrets Management module<br />		see: https://senhasegura.com/devops<br /> |


#### SenhaseguraProvider



SenhaseguraProvider setup a store to sync secrets with senhasegura.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL of senhasegura |  |  |
| `module` _[SenhaseguraModuleType](#senhaseguramoduletype)_ | Module defines which senhasegura module should be used to get secrets |  |  |
| `auth` _[SenhaseguraAuth](#senhaseguraauth)_ | Auth defines parameters to authenticate in senhasegura |  |  |
| `ignoreSslCertificate` _boolean_ | IgnoreSslCertificate defines if SSL certificate must be ignored | false |  |


#### SessionTagsPolicy

_Underlying type:_ _string_

SessionTagsPolicy defines how STS session tags are handled.

_Validation:_
- Enum: [None Simple Custom]

_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description |
| --- | --- |
| `None` | SessionTagsPolicyNone is the default behavior - no session tags are added.<br /> |
| `Simple` | SessionTagsPolicySimple automatically adds esoNamespace, esoStoreName, and esoStoreKind<br />session tags.<br /> |
| `Custom` | SessionTagsPolicyCustom adds the tags defined in CustomSessionTags in addition to<br />the esoNamespace, esoStoreName, and esoStoreKind tags.<br /> |


#### StoreGeneratorSourceRef



StoreGeneratorSourceRef allows you to override the source
from which the secret will be pulled from.
You can define at maximum one property.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storeRef` _[SecretStoreRef](#secretstoreref)_ |  |  | Optional: \{\} <br /> |
| `generatorRef` _[GeneratorRef](#generatorref)_ | GeneratorRef points to a generator custom resource. |  | Optional: \{\} <br /> |


#### StoreSourceRef



StoreSourceRef allows you to override the SecretStore source
from which the secret will be pulled from.
You can define at maximum one property.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ExternalSecretData](#externalsecretdata)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storeRef` _[SecretStoreRef](#secretstoreref)_ |  |  | Optional: \{\} <br /> |
| `generatorRef` _[GeneratorRef](#generatorref)_ | GeneratorRef points to a generator custom resource.<br />Deprecated: The generatorRef is not implemented in .data[].<br />this will be removed with v1. |  |  |


#### Tag



Tag is a key-value pair that can be attached to an AWS resource.
see: https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html



_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  |  |
| `value` _string_ |  |  |  |


#### TemplateEngineVersion

_Underlying type:_ _string_

TemplateEngineVersion specifies the template engine version that should be used to
compile/execute the template.

_Validation:_
- Enum: [v2]

_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description |
| --- | --- |
| `v2` | TemplateEngineV2 is the currently supported template engine version.<br /> |


#### TemplateFrom



TemplateFrom specifies a source for templates.
Each item in the list can either reference a ConfigMap or a Secret resource.



_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `configMap` _[TemplateRef](#templateref)_ |  |  |  |
| `secret` _[TemplateRef](#templateref)_ |  |  |  |
| `target` _string_ | Target specifies where to place the template result.<br />For Secret resources the accepted values are empty, "Data", "Annotations" and "Labels";<br />any other value is rejected because it would allow writes to privileged Secret fields.<br />For custom resources (when spec.target.manifest is set), this supports<br />nested paths like "spec.database.config" or "data". | Data | Optional: \{\} <br /> |
| `literal` _string_ |  |  | Optional: \{\} <br /> |
| `valuesDecodingStrategy` _[ExternalSecretDecodingStrategy](#externalsecretdecodingstrategy)_ | Used to define a decoding Strategy for the rendered template values. | None | Enum: [Auto Base64 Base64URL None] <br />Optional: \{\} <br /> |


#### TemplateMergePolicy

_Underlying type:_ _string_

TemplateMergePolicy defines how the rendered template should be merged with the existing Secret data.

_Validation:_
- Enum: [Replace Merge]

_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description |
| --- | --- |
| `Replace` |  |
| `Merge` |  |


#### TemplateRef



TemplateRef specifies a reference to either a ConfigMap or a Secret resource.



_Appears in:_
- [TemplateFrom](#templatefrom)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The name of the ConfigMap/Secret resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `items` _[TemplateRefItem](#templaterefitem) array_ | A list of keys in the ConfigMap/Secret to use as templates for Secret data |  |  |


#### TemplateRefItem

_Underlying type:_ _[struct{Key string "json:\"key\""; TemplateAs TemplateScope "json:\"templateAs,omitempty\""}](#struct{key-string-"json:\"key\"";-templateas-templatescope-"json:\"templateas,omitempty\""})_

TemplateRefItem specifies a key in the ConfigMap/Secret to use as a template for Secret data.



_Appears in:_
- [TemplateRef](#templateref)





#### TokenAuth



TokenAuth defines token-based authentication configuration for Kubernetes.



_Appears in:_
- [KubernetesAuth](#kubernetesauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `bearerToken` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### TokenAuthCredentials



TokenAuthCredentials represents the credentials for access token-based authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessToken` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### UniversalAuthCredentials



UniversalAuthCredentials represents the client credentials for universal authentication.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `clientSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |


#### ValidationResult

_Underlying type:_ _integer_

ValidationResult is defined type for the number of validation results.



_Appears in:_
- [FakeProvider](#fakeprovider)



#### VaultAppRole



VaultAppRole authenticates with Vault using the App Role auth mechanism,
with the role and secret stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the App Role authentication backend is mounted<br />in Vault, e.g: "approle" | approle |  |
| `roleId` _string_ | RoleID configured in the App Role authentication backend when setting<br />up the authentication backend in Vault. |  | Optional: \{\} <br /> |
| `roleRef` _[SecretKeySelector](#secretkeyselector)_ | Reference to a key in a Secret that contains the App Role ID used<br />to authenticate with Vault.<br />The `key` field must be specified and denotes which entry within the Secret<br />resource is used as the app role id. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Reference to a key in a Secret that contains the App Role secret used<br />to authenticate with Vault.<br />The `key` field must be specified and denotes which entry within the Secret<br />resource is used as the app role secret. |  |  |


#### VaultAuth



VaultAuth is the configuration used to authenticate with a Vault server.
Only one of `tokenSecretRef`, `appRole`,  `kubernetes`, `ldap`, `userPass`, `jwt`, `cert`, `iam` or `gcp`
can be specified. A namespace to authenticate against can optionally be specified.



_Appears in:_
- [VaultProvider](#vaultprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespace` _string_ | Name of the vault namespace to authenticate to. This can be different than the namespace your secret is in.<br />Namespaces is a set of features within Vault Enterprise that allows<br />Vault environments to support Secure Multi-tenancy. e.g: "ns1".<br />More about namespaces can be found here https://www.vaultproject.io/docs/enterprise/namespaces<br />This will default to Vault.Namespace field if set, or empty otherwise |  | Optional: \{\} <br /> |
| `tokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | TokenSecretRef authenticates with Vault by presenting a token. |  | Optional: \{\} <br /> |
| `appRole` _[VaultAppRole](#vaultapprole)_ | AppRole authenticates with Vault using the App Role auth mechanism,<br />with the role and secret stored in a Kubernetes Secret resource. |  | Optional: \{\} <br /> |
| `kubernetes` _[VaultKubernetesAuth](#vaultkubernetesauth)_ | Kubernetes authenticates with Vault by passing the ServiceAccount<br />token stored in the named Secret resource to the Vault server. |  | Optional: \{\} <br /> |
| `ldap` _[VaultLdapAuth](#vaultldapauth)_ | Ldap authenticates with Vault by passing username/password pair using<br />the LDAP authentication method |  | Optional: \{\} <br /> |
| `jwt` _[VaultJwtAuth](#vaultjwtauth)_ | Jwt authenticates with Vault by passing role and JWT token using the<br />JWT/OIDC authentication method |  | Optional: \{\} <br /> |
| `cert` _[VaultCertAuth](#vaultcertauth)_ | Cert authenticates with TLS Certificates by passing client certificate, private key and ca certificate<br />Cert authentication method |  | Optional: \{\} <br /> |
| `iam` _[VaultIamAuth](#vaultiamauth)_ | Iam authenticates with vault by passing a special AWS request signed with AWS IAM credentials<br />AWS IAM authentication method |  | Optional: \{\} <br /> |
| `userPass` _[VaultUserPassAuth](#vaultuserpassauth)_ | UserPass authenticates with Vault by passing username/password pair |  | Optional: \{\} <br /> |
| `gcp` _[VaultGCPAuth](#vaultgcpauth)_ | Gcp authenticates with Vault using Google Cloud Platform authentication method<br />GCP authentication method |  | Optional: \{\} <br /> |




#### VaultAwsAuthSecretRef

_Underlying type:_ _[struct{AccessKeyID github.com/external-secrets/external-secrets/apis/meta/v1.SecretKeySelector "json:\"accessKeyIDSecretRef,omitempty\""; SecretAccessKey github.com/external-secrets/external-secrets/apis/meta/v1.SecretKeySelector "json:\"secretAccessKeySecretRef,omitempty\""; SessionToken *github.com/external-secrets/external-secrets/apis/meta/v1.SecretKeySelector "json:\"sessionTokenSecretRef,omitempty\""}](#struct{accesskeyid-githubcomexternal-secretsexternal-secretsapismetav1secretkeyselector-"json:\"accesskeyidsecretref,omitempty\"";-secretaccesskey-githubcomexternal-secretsexternal-secretsapismetav1secretkeyselector-"json:\"secretaccesskeysecretref,omitempty\"";-sessiontoken-*githubcomexternal-secretsexternal-secretsapismetav1secretkeyselector-"json:\"sessiontokensecretref,omitempty\""})_

VaultAwsAuthSecretRef holds secret references for AWS credentials
both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.



_Appears in:_
- [VaultAwsAuth](#vaultawsauth)
- [VaultIamAuth](#vaultiamauth)



#### VaultAwsJWTAuth

_Underlying type:_ _[struct{ServiceAccountRef *github.com/external-secrets/external-secrets/apis/meta/v1.ServiceAccountSelector "json:\"serviceAccountRef,omitempty\""}](#struct{serviceaccountref-*githubcomexternal-secretsexternal-secretsapismetav1serviceaccountselector-"json:\"serviceaccountref,omitempty\""})_

VaultAwsJWTAuth Authenticate against AWS using service account tokens.



_Appears in:_
- [VaultAwsAuth](#vaultawsauth)
- [VaultIamAuth](#vaultiamauth)



#### VaultCertAuth



VaultCertAuth authenticates with Vault using the JWT/OIDC authentication
method, with the role name and token stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the Certificate authentication backend is mounted<br />in Vault, e.g: "cert" | cert | Optional: \{\} <br /> |
| `vaultRole` _string_ | VaultRole specifies the Vault role to use for TLS certificate authentication. |  | Optional: \{\} <br /> |
| `clientCert` _[SecretKeySelector](#secretkeyselector)_ | ClientCert is a certificate to authenticate using the Cert Vault<br />authentication method |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef to a key in a Secret resource containing client private key to<br />authenticate with Vault using the Cert authentication method |  | Optional: \{\} <br /> |


#### VaultCheckAndSet



VaultCheckAndSet defines the Check-And-Set (CAS) settings for Vault KV v2 PushSecret operations.



_Appears in:_
- [VaultProvider](#vaultprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `required` _boolean_ | Required when true, all write operations must include a check-and-set parameter.<br />This helps prevent unintentional overwrites of secrets. |  | Optional: \{\} <br /> |


#### VaultClientTLS



VaultClientTLS is the configuration used for client side related TLS communication,
when the Vault server requires mutual authentication.



_Appears in:_
- [VaultProvider](#vaultprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certSecretRef` _[SecretKeySelector](#secretkeyselector)_ | CertSecretRef is a certificate added to the transport layer<br />when communicating with the Vault server.<br />If no key for the Secret is specified, external-secret will default to 'tls.crt'. |  | Optional: \{\} <br /> |
| `keySecretRef` _[SecretKeySelector](#secretkeyselector)_ | KeySecretRef to a key in a Secret resource containing client private key<br />added to the transport layer when communicating with the Vault server.<br />If no key for the Secret is specified, external-secret will default to 'tls.key'. |  | Optional: \{\} <br /> |


#### VaultGCPAuth



VaultGCPAuth authenticates with Vault using Google Cloud Platform authentication method.
Refer: https://developer.hashicorp.com/vault/docs/auth/gcp

When ServiceAccountRef, SecretRef and WorkloadIdentity are not specified, the provider will use the controller pod's
identity to authenticate with GCP. This supports both GKE Workload Identity and service account keys.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the GCP auth method is enabled in Vault, e.g: "gcp" | gcp | Optional: \{\} <br /> |
| `role` _string_ | Vault Role. In Vault, a role describes an identity with a set of permissions, groups, or policies you want to attach to a user of the secrets engine. |  | Required: \{\} <br /> |
| `projectID` _string_ | Project ID of the Google Cloud Platform project |  | Optional: \{\} <br /> |
| `location` _string_ | Location optionally defines a location/region for the secret |  | Optional: \{\} <br /> |
| `secretRef` _[GCPSMAuthSecretRef](#gcpsmauthsecretref)_ | Specify credentials in a Secret object |  | Optional: \{\} <br /> |
| `workloadIdentity` _[GCPWorkloadIdentity](#gcpworkloadidentity)_ | Specify a service account with Workload Identity |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef to a service account for impersonation |  | Optional: \{\} <br /> |


#### VaultIamAuth



VaultIamAuth authenticates with Vault using the Vault's AWS IAM authentication method. Refer: https://developer.hashicorp.com/vault/docs/auth/aws

When JWTAuth and SecretRef are not specified, the provider will use the controller pod's
identity to authenticate with AWS. This supports both IRSA and EKS Pod Identity.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the AWS auth method is enabled in Vault, e.g: "aws" |  | Optional: \{\} <br /> |
| `region` _string_ | AWS region |  | Optional: \{\} <br /> |
| `role` _string_ | This is the AWS role to be assumed before talking to vault |  | Optional: \{\} <br /> |
| `vaultRole` _string_ | Vault Role. In vault, a role describes an identity with a set of permissions, groups, or policies you want to attach a user of the secrets engine |  |  |
| `externalID` _string_ | AWS External ID set on assumed IAM roles |  |  |
| `vaultAwsIamServerID` _string_ | X-Vault-AWS-IAM-Server-ID is an additional header used by Vault IAM auth method to mitigate against different types of replay attacks. More details here: https://developer.hashicorp.com/vault/docs/auth/aws |  | Optional: \{\} <br /> |
| `secretRef` _[VaultAwsAuthSecretRef](#vaultawsauthsecretref)_ | Specify credentials in a Secret object |  | Optional: \{\} <br /> |
| `jwt` _[VaultAwsJWTAuth](#vaultawsjwtauth)_ | Specify a service account with IRSA enabled |  | Optional: \{\} <br /> |


#### VaultJwtAuth



VaultJwtAuth authenticates with Vault using the JWT/OIDC authentication
method, with the role name and a token stored in a Kubernetes Secret resource or
a Kubernetes service account token retrieved via `TokenRequest`.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the JWT authentication backend is mounted<br />in Vault, e.g: "jwt" | jwt |  |
| `role` _string_ | Role is a JWT role to authenticate using the JWT/OIDC Vault<br />authentication method |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional SecretRef that refers to a key in a Secret resource containing JWT token to<br />authenticate with Vault using the JWT/OIDC authentication method. |  | Optional: \{\} <br /> |
| `kubernetesServiceAccountToken` _[VaultKubernetesServiceAccountTokenAuth](#vaultkubernetesserviceaccounttokenauth)_ | Optional ServiceAccountToken specifies the Kubernetes service account for which to request<br />a token for with the `TokenRequest` API. |  | Optional: \{\} <br /> |


#### VaultKVStoreVersion

_Underlying type:_ _string_

VaultKVStoreVersion represents the version of the Vault KV secret engine.



_Appears in:_
- [VaultProvider](#vaultprovider)

| Field | Description |
| --- | --- |
| `v1` |  |
| `v2` |  |


#### VaultKubernetesAuth



VaultKubernetesAuth authenticates against Vault using a Kubernetes ServiceAccount token stored in
a Secret.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mountPath` _string_ | Path where the Kubernetes authentication backend is mounted in Vault, e.g:<br />"kubernetes" | kubernetes |  |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Optional service account field containing the name of a kubernetes ServiceAccount.<br />If the service account is specified, the service account secret token JWT will be used<br />for authenticating with Vault. If the service account selector is not supplied,<br />the secretRef will be used instead. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional secret field containing a Kubernetes ServiceAccount JWT used<br />for authenticating with Vault. If a name is specified without a key,<br />`token` is the default. If one is not specified, the one bound to<br />the controller will be used. |  | Optional: \{\} <br /> |
| `role` _string_ | A required field containing the Vault Role to assume. A Role binds a<br />Kubernetes ServiceAccount with a set of Vault policies. |  |  |


#### VaultKubernetesServiceAccountTokenAuth

_Underlying type:_ _[struct{ServiceAccountRef github.com/external-secrets/external-secrets/apis/meta/v1.ServiceAccountSelector "json:\"serviceAccountRef\""; Audiences *[]string "json:\"audiences,omitempty\""; ExpirationSeconds *int64 "json:\"expirationSeconds,omitempty\""}](#struct{serviceaccountref-githubcomexternal-secretsexternal-secretsapismetav1serviceaccountselector-"json:\"serviceaccountref\"";-audiences-*[]string-"json:\"audiences,omitempty\"";-expirationseconds-*int64-"json:\"expirationseconds,omitempty\""})_

VaultKubernetesServiceAccountTokenAuth authenticates with Vault using a temporary
Kubernetes service account token retrieved by the `TokenRequest` API.



_Appears in:_
- [VaultJwtAuth](#vaultjwtauth)



#### VaultLdapAuth



VaultLdapAuth authenticates with Vault using the LDAP authentication method,
with the username and password stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the LDAP authentication backend is mounted<br />in Vault, e.g: "ldap" | ldap |  |
| `username` _string_ | Username is an LDAP username used to authenticate using the LDAP Vault<br />authentication method |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef to a key in a Secret resource containing password for the LDAP<br />user used to authenticate with Vault using the LDAP authentication<br />method |  | Optional: \{\} <br /> |


#### VaultProvider



VaultProvider configures a store to sync secrets using a Hashicorp Vault KV backend.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)
- [VaultDynamicSecretSpec](#vaultdynamicsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[VaultAuth](#vaultauth)_ | Auth configures how secret-manager authenticates with the Vault server. |  |  |
| `server` _string_ | Server is the connection address for the Vault server, e.g: "https://vault.example.com:8200". |  |  |
| `path` _string_ | Path is the mount path of the Vault KV backend endpoint, e.g:<br />"secret". The v2 KV secret engine version specific "/data" path suffix<br />for fetching secrets from Vault is optional and will be appended<br />if not present in specified path. |  | Optional: \{\} <br /> |
| `version` _[VaultKVStoreVersion](#vaultkvstoreversion)_ | Version is the Vault KV secret engine version. This can be either "v1" or<br />"v2". Version defaults to "v2". | v2 | Enum: [v1 v2] <br />Optional: \{\} <br /> |
| `namespace` _string_ | Name of the vault namespace. Namespaces is a set of features within Vault Enterprise that allows<br />Vault environments to support Secure Multi-tenancy. e.g: "ns1".<br />More about namespaces can be found here https://www.vaultproject.io/docs/enterprise/namespaces |  | Optional: \{\} <br /> |
| `caBundle` _integer array_ | PEM encoded CA bundle used to validate Vault server certificate. Only used<br />if the Server URL is using HTTPS protocol. This parameter is ignored for<br />plain HTTP protocol connection. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `tls` _[VaultClientTLS](#vaultclienttls)_ | The configuration used for client side related TLS communication, when the Vault server<br />requires mutual authentication. Only used if the Server URL is using HTTPS protocol.<br />This parameter is ignored for plain HTTP protocol connection.<br />It's worth noting this configuration is different from the "TLS certificates auth method",<br />which is available under the `auth.cert` section. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | The provider for the CA bundle to use to validate Vault server certificate. |  | Optional: \{\} <br /> |
| `readYourWrites` _boolean_ | ReadYourWrites ensures isolated read-after-write semantics by<br />providing discovered cluster replication states in each request.<br />More information about eventual consistency in Vault can be found here<br />https://www.vaultproject.io/docs/enterprise/consistency |  | Optional: \{\} <br /> |
| `forwardInconsistent` _boolean_ | ForwardInconsistent tells Vault to forward read-after-write requests to the Vault<br />leader instead of simply retrying within a loop. This can increase performance if<br />the option is enabled serverside.<br />https://www.vaultproject.io/docs/configuration/replication#allow_forwarding_via_header |  | Optional: \{\} <br /> |
| `headers` _object (keys:string, values:string)_ | Headers to be added in Vault request |  | Optional: \{\} <br /> |
| `checkAndSet` _[VaultCheckAndSet](#vaultcheckandset)_ | CheckAndSet defines the Check-And-Set (CAS) settings for PushSecret operations.<br />Only applies to Vault KV v2 stores. When enabled, write operations must include<br />the current version of the secret to prevent unintentional overwrites. |  | Optional: \{\} <br /> |


#### VaultUserPassAuth



VaultUserPassAuth authenticates with Vault using UserPass authentication method,
with the username and password stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the UserPassword authentication backend is mounted<br />in Vault, e.g: "userpass" | userpass |  |
| `username` _string_ | Username is a username used to authenticate using the UserPass Vault<br />authentication method |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef to a key in a Secret resource containing password for the<br />user used to authenticate with Vault using the UserPass authentication<br />method |  | Optional: \{\} <br /> |


#### VolcengineAuth



VolcengineAuth defines the authentication method for the Volcengine provider.
Only one of the fields should be set.



_Appears in:_
- [VolcengineProvider](#volcengineprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[VolcengineAuthSecretRef](#volcengineauthsecretref)_ | SecretRef defines the static credentials to use for authentication.<br />If not set, IRSA is used. |  | Optional: \{\} <br /> |


#### VolcengineAuthSecretRef



VolcengineAuthSecretRef defines the secret reference for static credentials.



_Appears in:_
- [VolcengineAuth](#volcengineauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessKeyID` _[SecretKeySelector](#secretkeyselector)_ | AccessKeyID is the reference to the secret containing the Access Key ID. |  |  |
| `secretAccessKey` _[SecretKeySelector](#secretkeyselector)_ | SecretAccessKey is the reference to the secret containing the Secret Access Key. |  |  |
| `token` _[SecretKeySelector](#secretkeyselector)_ | Token is the reference to the secret containing the STS(Security Token Service) Token. |  | Optional: \{\} <br /> |


#### VolcengineProvider



VolcengineProvider defines the configuration for the Volcengine provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `region` _string_ | Region specifies the Volcengine region to connect to. |  |  |
| `auth` _[VolcengineAuth](#volcengineauth)_ | Auth defines the authentication method to use.<br />If not specified, the provider will try to use IRSA (IAM Role for Service Account). |  | Optional: \{\} <br /> |


#### WebhookCAProvider



WebhookCAProvider defines a location to fetch the cert for the webhook provider from.



_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[WebhookCAProviderType](#webhookcaprovidertype)_ | The type of provider to use such as "Secret", or "ConfigMap". |  | Enum: [Secret ConfigMap] <br /> |
| `name` _string_ | The name of the object located at the provider type. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `key` _string_ | The key where the CA certificate can be found in the Secret or ConfigMap. |  | MaxLength: 253 <br />MinLength: 1 <br />Optional: \{\} <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |
| `namespace` _string_ | The namespace the Provider type is in. |  | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


#### WebhookCAProviderType

_Underlying type:_ _string_

WebhookCAProviderType defines the type of provider for certificate authority in webhook connections.



_Appears in:_
- [WebhookCAProvider](#webhookcaprovider)

| Field | Description |
| --- | --- |
| `Secret` | WebhookCAProviderTypeSecret indicates that the CA certificate is stored in a Secret resource.<br /> |
| `ConfigMap` | WebhookCAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap resource.<br /> |


#### WebhookProvider



WebhookProvider configures a store to sync secrets from simple web APIs.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `method` _string_ | Webhook Method |  |  |
| `url` _string_ | Webhook url to call |  |  |
| `headers` _object (keys:string, values:string)_ | Headers |  | Optional: \{\} <br /> |
| `auth` _[AuthorizationProtocol](#authorizationprotocol)_ | Auth specifies a authorization protocol. Only one protocol may be set. |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `body` _string_ | Body |  | Optional: \{\} <br /> |
| `timeout` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | Timeout |  | Optional: \{\} <br /> |
| `result` _[WebhookResult](#webhookresult)_ | Result formatting |  | Optional: \{\} <br /> |
| `secrets` _[WebhookSecret](#webhooksecret) array_ | Secrets to fill in templates<br />These secrets will be passed to the templating function as key value pairs under the given name |  | Optional: \{\} <br /> |
| `caBundle` _integer array_ | PEM encoded CA bundle used to validate webhook server certificate. Only used<br />if the Server URL is using HTTPS protocol. This parameter is ignored for<br />plain HTTP protocol connection. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[WebhookCAProvider](#webhookcaprovider)_ | The provider for the CA bundle to use to validate webhook server certificate. |  | Optional: \{\} <br /> |


#### WebhookResult



WebhookResult defines how to process and extract secrets from the webhook response.



_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `jsonPath` _string_ | Json path of return value |  | Optional: \{\} <br /> |


#### WebhookSecret



WebhookSecret defines a secret that will be passed to the webhook request.



_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of this secret in templates |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Secret ref to fill in credentials |  |  |


#### YandexAuth



YandexAuth defines the authentication method for the Yandex provider.



_Appears in:_
- [YandexCertificateManagerProvider](#yandexcertificatemanagerprovider)
- [YandexLockboxProvider](#yandexlockboxprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authorizedKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The authorized key used for authentication |  | Optional: \{\} <br /> |


#### YandexCAProvider



YandexCAProvider defines the configuration for Yandex custom certificate authority.



_Appears in:_
- [YandexCertificateManagerProvider](#yandexcertificatemanagerprovider)
- [YandexLockboxProvider](#yandexlockboxprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### YandexCertificateManagerProvider



YandexCertificateManagerProvider Configures a store to sync secrets using the Yandex Certificate Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiEndpoint` _string_ | Yandex.Cloud API endpoint (e.g. 'api.cloud.yandex.net:443') |  | Optional: \{\} <br /> |
| `auth` _[YandexAuth](#yandexauth)_ | Auth defines the information necessary to authenticate against Yandex.Cloud |  |  |
| `caProvider` _[YandexCAProvider](#yandexcaprovider)_ | The provider for the CA bundle to use to validate Yandex.Cloud server certificate. |  | Optional: \{\} <br /> |
| `fetching` _[FetchingPolicy](#fetchingpolicy)_ | FetchingPolicy configures the provider to interpret the `data.secretKey.remoteRef.key` field in ExternalSecret as certificate ID or certificate name |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |


#### YandexLockboxProvider



YandexLockboxProvider Configures a store to sync secrets using the Yandex Lockbox provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiEndpoint` _string_ | Yandex.Cloud API endpoint (e.g. 'api.cloud.yandex.net:443') |  | Optional: \{\} <br /> |
| `auth` _[YandexAuth](#yandexauth)_ | Auth defines the information necessary to authenticate against Yandex.Cloud |  |  |
| `caProvider` _[YandexCAProvider](#yandexcaprovider)_ | The provider for the CA bundle to use to validate Yandex.Cloud server certificate. |  | Optional: \{\} <br /> |
| `fetching` _[FetchingPolicy](#fetchingpolicy)_ | FetchingPolicy configures the provider to interpret the `data.secretKey.remoteRef.key` field in ExternalSecret as secret ID or secret name |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |



## external-secrets.io/v1alpha1

Package v1alpha1 contains resources for external-secrets

### Resource Types
- [ClusterPushSecret](#clusterpushsecret)
- [PushSecret](#pushsecret)



#### ClusterPushSecret



ClusterPushSecret is the Schema for the ClusterPushSecrets API that enables cluster-wide management of pushing Kubernetes secrets to external providers.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `ClusterPushSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ClusterPushSecretSpec](#clusterpushsecretspec)_ |  |  |  |
| `status` _[ClusterPushSecretStatus](#clusterpushsecretstatus)_ |  |  |  |




#### ClusterPushSecretNamespaceFailure



ClusterPushSecretNamespaceFailure represents a failed namespace deployment and it's reason.



_Appears in:_
- [ClusterPushSecretStatus](#clusterpushsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespace` _string_ | Namespace is the namespace that failed when trying to apply an PushSecret |  |  |
| `reason` _string_ | Reason is why the PushSecret failed to apply to the namespace |  | Optional: \{\} <br /> |


#### ClusterPushSecretSpec



ClusterPushSecretSpec defines the configuration for a ClusterPushSecret resource.



_Appears in:_
- [ClusterPushSecret](#clusterpushsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `pushSecretSpec` _[PushSecretSpec](#pushsecretspec)_ | PushSecretSpec defines what to do with the secrets. |  |  |
| `refreshTime` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | The time in which the controller should reconcile its objects and recheck namespaces for labels. |  |  |
| `pushSecretName` _string_ | The name of the push secrets to be created.<br />Defaults to the name of the ClusterPushSecret |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Optional: \{\} <br /> |
| `pushSecretMetadata` _[PushSecretMetadata](#pushsecretmetadata)_ | The metadata of the external secrets to be created |  | Optional: \{\} <br /> |
| `namespaceSelectors` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta) array_ | A list of labels to select by to find the Namespaces to create the ExternalSecrets in. The selectors are ORed. |  | Optional: \{\} <br /> |


#### ClusterPushSecretStatus



ClusterPushSecretStatus contains the status information for the ClusterPushSecret resource.



_Appears in:_
- [ClusterPushSecret](#clusterpushsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `failedNamespaces` _[ClusterPushSecretNamespaceFailure](#clusterpushsecretnamespacefailure) array_ | Failed namespaces are the namespaces that failed to apply an PushSecret |  | Optional: \{\} <br /> |
| `provisionedNamespaces` _string array_ | ProvisionedNamespaces are the namespaces where the ClusterPushSecret has secrets |  | Optional: \{\} <br /> |
| `pushSecretName` _string_ |  |  |  |
| `conditions` _[PushSecretStatusCondition](#pushsecretstatuscondition) array_ |  |  | Optional: \{\} <br /> |


#### PushSecret



PushSecret is the Schema for the PushSecrets API that enables pushing Kubernetes secrets to external secret providers.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `PushSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PushSecretSpec](#pushsecretspec)_ |  |  |  |
| `status` _[PushSecretStatus](#pushsecretstatus)_ |  |  |  |


#### PushSecretConditionType

_Underlying type:_ _string_

PushSecretConditionType indicates the condition of the PushSecret.



_Appears in:_
- [PushSecretStatusCondition](#pushsecretstatuscondition)

| Field | Description |
| --- | --- |
| `Ready` | PushSecretReady indicates the PushSecret resource is ready.<br /> |


#### PushSecretConversionStrategy

_Underlying type:_ _string_

PushSecretConversionStrategy defines how secret values are converted when pushed to providers.

_Validation:_
- Enum: [None ReverseUnicode]

_Appears in:_
- [PushSecretData](#pushsecretdata)
- [PushSecretDataTo](#pushsecretdatato)

| Field | Description |
| --- | --- |
| `None` | PushSecretConversionNone indicates no conversion will be performed on the secret value.<br /> |
| `ReverseUnicode` | PushSecretConversionReverseUnicode indicates that unicode escape sequences will be reversed.<br /> |


#### PushSecretData



PushSecretData defines data to be pushed to the provider and associated metadata.



_Appears in:_
- [PushSecretSpec](#pushsecretspec)
- [SyncedPushSecretsMap](#syncedpushsecretsmap)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `match` _[PushSecretMatch](#pushsecretmatch)_ | Match a given Secret Key to be pushed to the provider. |  |  |
| `metadata` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#json-v1-apiextensions-k8s-io)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `conversionStrategy` _[PushSecretConversionStrategy](#pushsecretconversionstrategy)_ | Used to define a conversion Strategy for the secret keys | None | Enum: [None ReverseUnicode] <br />Optional: \{\} <br /> |


#### PushSecretDataTo



PushSecretDataTo defines how to bulk-push secrets to providers without explicit per-key mappings.



_Appears in:_
- [PushSecretSpec](#pushsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storeRef` _[PushSecretStoreRef](#pushsecretstoreref)_ | StoreRef specifies which SecretStore to push to. Required. |  |  |
| `remoteKey` _string_ | RemoteKey is the name of the single provider secret that will receive ALL<br />matched keys bundled as a JSON object (e.g. \{"DB_HOST":"...","DB_USER":"..."\}).<br />When set, per-key expansion is skipped and a single push is performed.<br />The provider's store prefix (if any) is still prepended to this value.<br />When not set, each matched key is pushed as its own individual provider secret. |  | Optional: \{\} <br /> |
| `match` _[PushSecretDataToMatch](#pushsecretdatatomatch)_ | Match pattern for selecting keys from the source Secret.<br />If not specified, all keys are selected. |  | Optional: \{\} <br /> |
| `rewrite` _[PushSecretRewrite](#pushsecretrewrite) array_ | Rewrite operations to transform keys before pushing to the provider.<br />Operations are applied sequentially. |  | Optional: \{\} <br /> |
| `metadata` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#json-v1-apiextensions-k8s-io)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `conversionStrategy` _[PushSecretConversionStrategy](#pushsecretconversionstrategy)_ | Used to define a conversion Strategy for the secret keys | None | Enum: [None ReverseUnicode] <br />Optional: \{\} <br /> |


#### PushSecretDataToMatch



PushSecretDataToMatch defines pattern matching for key selection.



_Appears in:_
- [PushSecretDataTo](#pushsecretdatato)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `regexp` _string_ | Regexp matches keys by regular expression.<br />If not specified, all keys are matched. |  | Optional: \{\} <br /> |


#### PushSecretDeletionPolicy

_Underlying type:_ _string_

PushSecretDeletionPolicy defines how push secrets are deleted in the provider.

_Validation:_
- Enum: [Delete None]

_Appears in:_
- [PushSecretSpec](#pushsecretspec)

| Field | Description |
| --- | --- |
| `Delete` | PushSecretDeletionPolicyDelete deletes secrets from the provider when the PushSecret is deleted.<br /> |
| `None` | PushSecretDeletionPolicyNone keeps secrets in the provider when the PushSecret is deleted.<br /> |


#### PushSecretMatch



PushSecretMatch defines how a source Secret key maps to a destination in the provider.



_Appears in:_
- [PushSecretData](#pushsecretdata)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretKey` _string_ | Secret Key to be pushed |  | Optional: \{\} <br /> |
| `remoteRef` _[PushSecretRemoteRef](#pushsecretremoteref)_ | Remote Refs to push to providers. |  |  |


#### PushSecretMetadata



PushSecretMetadata defines metadata fields for the PushSecret generated by the ClusterPushSecret.



_Appears in:_
- [ClusterPushSecretSpec](#clusterpushsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `annotations` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |


#### PushSecretRemoteRef



PushSecretRemoteRef defines the location of the secret in the provider.



_Appears in:_
- [PushSecretMatch](#pushsecretmatch)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `remoteKey` _string_ | Name of the resulting provider secret. |  |  |
| `property` _string_ | Name of the property in the resulting secret |  | Optional: \{\} <br /> |


#### PushSecretRewrite



PushSecretRewrite defines how to transform secret keys before pushing.



_Appears in:_
- [PushSecretDataTo](#pushsecretdatato)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `regexp` _[ExternalSecretRewriteRegexp](#externalsecretrewriteregexp)_ | Used to rewrite with regular expressions. |  | Optional: \{\} <br /> |
| `transform` _[ExternalSecretRewriteTransform](#externalsecretrewritetransform)_ | Used to apply string transformation on the secrets. |  | Optional: \{\} <br /> |


#### PushSecretSecret



PushSecretSecret defines a Secret that will be used as a source for pushing to providers.



_Appears in:_
- [PushSecretSelector](#pushsecretselector)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Secret.<br />The Secret must exist in the same namespace as the PushSecret manifest. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Optional: \{\} <br /> |
| `selector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | Selector chooses secrets using a labelSelector. |  | Optional: \{\} <br /> |


#### PushSecretSelector



PushSecretSelector defines criteria for selecting the source Secret for pushing to providers.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [PushSecretSpec](#pushsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secret` _[PushSecretSecret](#pushsecretsecret)_ | Select a Secret to Push. |  | Optional: \{\} <br /> |
| `generatorRef` _[GeneratorRef](#generatorref)_ | Point to a generator to create a Secret. |  | Optional: \{\} <br /> |


#### PushSecretSpec



PushSecretSpec configures the behavior of the PushSecret.



_Appears in:_
- [ClusterPushSecretSpec](#clusterpushsecretspec)
- [PushSecret](#pushsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `refreshInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | The Interval to which External Secrets will try to push a secret definition | 1h0m0s |  |
| `secretStoreRefs` _[PushSecretStoreRef](#pushsecretstoreref) array_ |  |  |  |
| `updatePolicy` _[PushSecretUpdatePolicy](#pushsecretupdatepolicy)_ | UpdatePolicy to handle Secrets in the provider. | Replace | Enum: [Replace IfNotExists] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[PushSecretDeletionPolicy](#pushsecretdeletionpolicy)_ | Deletion Policy to handle Secrets in the provider. | None | Enum: [Delete None] <br />Optional: \{\} <br /> |
| `selector` _[PushSecretSelector](#pushsecretselector)_ | The Secret Selector (k8s source) for the Push Secret |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `data` _[PushSecretData](#pushsecretdata) array_ | Secret Data that should be pushed to providers |  | Optional: \{\} <br /> |
| `dataTo` _[PushSecretDataTo](#pushsecretdatato) array_ | DataTo defines bulk push rules that expand source Secret keys into provider entries. |  | Optional: \{\} <br /> |
| `template` _[ExternalSecretTemplate](#externalsecrettemplate)_ | Template defines a blueprint for the created Secret resource. |  | Optional: \{\} <br /> |


#### PushSecretStatus



PushSecretStatus indicates the history of the status of PushSecret.



_Appears in:_
- [PushSecret](#pushsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `refreshTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ | refreshTime is the time and date the external secret was fetched and<br />the target secret updated |  |  |
| `syncedResourceVersion` _string_ | SyncedResourceVersion keeps track of the last synced version. |  |  |
| `syncedPushSecrets` _[SyncedPushSecretsMap](#syncedpushsecretsmap)_ | Synced PushSecrets, including secrets that already exist in provider.<br />Matches secret stores to PushSecretData that was stored to that secret store. |  | Optional: \{\} <br /> |
| `conditions` _[PushSecretStatusCondition](#pushsecretstatuscondition) array_ |  |  | Optional: \{\} <br /> |


#### PushSecretStatusCondition



PushSecretStatusCondition indicates the status of the PushSecret.



_Appears in:_
- [ClusterPushSecretStatus](#clusterpushsecretstatus)
- [PushSecretStatus](#pushsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[PushSecretConditionType](#pushsecretconditiontype)_ |  |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `reason` _string_ |  |  | Optional: \{\} <br /> |
| `message` _string_ |  |  | Optional: \{\} <br /> |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ |  |  | Optional: \{\} <br /> |


#### PushSecretStoreRef



PushSecretStoreRef contains a reference on how to sync to a SecretStore.



_Appears in:_
- [PushSecretDataTo](#pushsecretdatato)
- [PushSecretSpec](#pushsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Optionally, sync to the SecretStore of the given name |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Optional: \{\} <br /> |
| `labelSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | Optionally, sync to secret stores with label selector |  | Optional: \{\} <br /> |
| `kind` _string_ | Kind of the SecretStore resource (SecretStore or ClusterSecretStore) | SecretStore | Enum: [SecretStore ClusterSecretStore] <br />Optional: \{\} <br /> |


#### PushSecretUpdatePolicy

_Underlying type:_ _string_

PushSecretUpdatePolicy defines how push secrets are updated in the provider.

_Validation:_
- Enum: [Replace IfNotExists]

_Appears in:_
- [PushSecretSpec](#pushsecretspec)

| Field | Description |
| --- | --- |
| `Replace` | PushSecretUpdatePolicyReplace replaces existing secrets in the provider.<br /> |
| `IfNotExists` | PushSecretUpdatePolicyIfNotExists only creates secrets that don't exist in the provider.<br /> |


#### SyncedPushSecretsMap

_Underlying type:_ _[map[string]map[string]PushSecretData](#map[string]map[string]pushsecretdata)_

SyncedPushSecretsMap is a map that tracks which PushSecretData was stored to which secret store.
The outer map's key is the secret store name, and the inner map's key is the remote key name.



_Appears in:_
- [PushSecretStatus](#pushsecretstatus)




## external-secrets.io/v1beta1

Package v1beta1 contains resources for external-secrets

### Resource Types
- [ClusterExternalSecret](#clusterexternalsecret)
- [ClusterSecretStore](#clustersecretstore)
- [ExternalSecret](#externalsecret)
- [GenericStore](#genericstore)
- [Provider](#provider)
- [PushSecretData](#pushsecretdata)
- [PushSecretRemoteRef](#pushsecretremoteref)
- [SecretStore](#secretstore)
- [SecretsClient](#secretsclient)



#### AWSAuth



AWSAuth tells the controller how to do authentication with aws.
Only one of secretRef or jwt can be specified.
if none is specified the controller will load credentials using the aws sdk defaults.



_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AWSAuthSecretRef](#awsauthsecretref)_ |  |  | Optional: \{\} <br /> |
| `jwt` _[AWSJWTAuth](#awsjwtauth)_ |  |  | Optional: \{\} <br /> |


#### AWSAuthSecretRef



AWSAuthSecretRef holds secret references for AWS credentials
both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.



_Appears in:_
- [AWSAuth](#awsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessKeyIDSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeyID is used for authentication |  |  |
| `secretAccessKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  |  |
| `sessionTokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SessionToken used for authentication<br />This must be defined if AccessKeyID and SecretAccessKey are temporary credentials<br />see: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html |  |  |


#### AWSJWTAuth



AWSJWTAuth authenticates against AWS using service account tokens from the Kubernetes cluster.



_Appears in:_
- [AWSAuth](#awsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ |  |  |  |


#### AWSProvider



AWSProvider configures a store to sync secrets with AWS.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `service` _[AWSServiceType](#awsservicetype)_ | Service defines which service should be used to fetch the secrets |  | Enum: [SecretsManager ParameterStore] <br /> |
| `auth` _[AWSAuth](#awsauth)_ | Auth defines the information necessary to authenticate against AWS<br />if not set aws sdk will infer credentials from your environment<br />see: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials |  | Optional: \{\} <br /> |
| `role` _string_ | Role is a Role ARN which the provider will assume |  | Optional: \{\} <br /> |
| `region` _string_ | AWS Region to be used for the provider |  |  |
| `additionalRoles` _string array_ | AdditionalRoles is a chained list of Role ARNs which the provider will sequentially assume before assuming the Role |  | Optional: \{\} <br /> |
| `externalID` _string_ | AWS External ID set on assumed IAM roles |  |  |
| `sessionTags` _[Tag](#tag) array_ | AWS STS assume role session tags |  | Optional: \{\} <br /> |
| `secretsManager` _[SecretsManager](#secretsmanager)_ | SecretsManager defines how the provider behaves when interacting with AWS SecretsManager |  | Optional: \{\} <br /> |
| `transitiveTagKeys` _string array_ | AWS STS assume role transitive session tags. Required when multiple rules are used with the provider |  | Optional: \{\} <br /> |
| `prefix` _string_ | Prefix adds a prefix to all retrieved values. |  | Optional: \{\} <br /> |


#### AWSServiceType

_Underlying type:_ _string_

AWSServiceType is an enum that defines the service/API that is used to fetch the secrets.

_Validation:_
- Enum: [SecretsManager ParameterStore]

_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description |
| --- | --- |
| `SecretsManager` | AWSServiceSecretsManager is the AWS SecretsManager service.<br />see: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html<br /> |
| `ParameterStore` | AWSServiceParameterStore is the AWS SystemsManager ParameterStore service.<br />see: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html<br /> |


#### AkeylessAuth



AkeylessAuth defines methods of authentication with Akeyless Vault.



_Appears in:_
- [AkeylessProvider](#akeylessprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AkeylessAuthSecretRef](#akeylessauthsecretref)_ | Reference to a Secret that contains the details<br />to authenticate with Akeyless. |  | Optional: \{\} <br /> |
| `kubernetesAuth` _[AkeylessKubernetesAuth](#akeylesskubernetesauth)_ | Kubernetes authenticates with Akeyless by passing the ServiceAccount<br />token stored in the named Secret resource. |  | Optional: \{\} <br /> |


#### AkeylessAuthSecretRef



AkeylessAuthSecretRef defines how to authenticate using a secret reference.
AKEYLESS_ACCESS_TYPE_PARAM: AZURE_OBJ_ID OR GCP_AUDIENCE OR ACCESS_KEY OR KUB_CONFIG_NAME.



_Appears in:_
- [AkeylessAuth](#akeylessauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessID` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessID is used for authentication |  |  |
| `accessType` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `accessTypeParam` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### AkeylessKubernetesAuth



AkeylessKubernetesAuth authenticates with Akeyless using a Kubernetes ServiceAccount token.



_Appears in:_
- [AkeylessAuth](#akeylessauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessID` _string_ | the Akeyless Kubernetes auth-method access-id |  |  |
| `k8sConfName` _string_ | Kubernetes-auth configuration name in Akeyless-Gateway |  |  |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Optional service account field containing the name of a kubernetes ServiceAccount.<br />If the service account is specified, the service account secret token JWT will be used<br />for authenticating with Akeyless. If the service account selector is not supplied,<br />the secretRef will be used instead. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional secret field containing a Kubernetes ServiceAccount JWT used<br />for authenticating with Akeyless. If a name is specified without a key,<br />`token` is the default. If one is not specified, the one bound to<br />the controller will be used. |  | Optional: \{\} <br /> |


#### AkeylessProvider



AkeylessProvider Configures an store to sync secrets using Akeyless KV.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `akeylessGWApiURL` _string_ | Akeyless GW API Url from which the secrets to be fetched from. |  |  |
| `authSecretRef` _[AkeylessAuth](#akeylessauth)_ | Auth configures how the operator authenticates with Akeyless. |  |  |
| `caBundle` _integer array_ | PEM/base64 encoded CA bundle used to validate Akeyless Gateway certificate. Only used<br />if the AkeylessGWApiURL URL is using HTTPS protocol. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | The provider for the CA bundle to use to validate Akeyless Gateway certificate. |  | Optional: \{\} <br /> |


#### AlibabaAuth



AlibabaAuth contains a secretRef for credentials.



_Appears in:_
- [AlibabaProvider](#alibabaprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AlibabaAuthSecretRef](#alibabaauthsecretref)_ |  |  | Optional: \{\} <br /> |
| `rrsa` _[AlibabaRRSAAuth](#alibabarrsaauth)_ |  |  | Optional: \{\} <br /> |


#### AlibabaAuthSecretRef



AlibabaAuthSecretRef holds secret references for Alibaba credentials.



_Appears in:_
- [AlibabaAuth](#alibabaauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessKeyIDSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeyID is used for authentication |  |  |
| `accessKeySecretSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeySecret is used for authentication |  |  |


#### AlibabaProvider



AlibabaProvider configures a store to sync secrets using the Alibaba Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[AlibabaAuth](#alibabaauth)_ |  |  |  |
| `regionID` _string_ | Alibaba Region to be used for the provider |  |  |


#### AlibabaRRSAAuth



AlibabaRRSAAuth authenticates against Alibaba using RRSA (Resource-oriented RAM-based Service Authentication).



_Appears in:_
- [AlibabaAuth](#alibabaauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `oidcProviderArn` _string_ |  |  |  |
| `oidcTokenFilePath` _string_ |  |  |  |
| `roleArn` _string_ |  |  |  |
| `sessionName` _string_ |  |  |  |


#### AuthorizationProtocol



AuthorizationProtocol contains the protocol-specific configuration

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ntlm` _[NTLMProtocol](#ntlmprotocol)_ | NTLMProtocol configures the store to use NTLM for auth |  | Optional: \{\} <br /> |


#### AzureAuthType

_Underlying type:_ _string_

AzureAuthType describes how to authenticate to the Azure Keyvault.
Only one of the following auth types may be specified.
If none of the following auth type is specified, the default one
is ServicePrincipal.

_Validation:_
- Enum: [ServicePrincipal ManagedIdentity WorkloadIdentity]

_Appears in:_
- [AzureKVProvider](#azurekvprovider)

| Field | Description |
| --- | --- |
| `ServicePrincipal` | AzureServicePrincipal uses service principal to authenticate, which needs a tenantId, a clientId and a clientSecret.<br /> |
| `ManagedIdentity` | AzureManagedIdentity uses Managed Identity to authenticate. Used with aad-pod-identity installed in the cluster.<br /> |
| `WorkloadIdentity` | AzureWorkloadIdentity uses Workload Identity service accounts to authenticate.<br /> |


#### AzureEnvironmentType

_Underlying type:_ _string_

AzureEnvironmentType specifies the Azure cloud environment endpoints to use for
connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud

_Validation:_
- Enum: [PublicCloud USGovernmentCloud ChinaCloud GermanCloud]

_Appears in:_
- [AzureKVProvider](#azurekvprovider)

| Field | Description |
| --- | --- |
| `PublicCloud` | AzureEnvironmentPublicCloud represents the Azure public cloud environment.<br /> |
| `USGovernmentCloud` | AzureEnvironmentUSGovernmentCloud represents the Azure US government cloud environment.<br /> |
| `ChinaCloud` | AzureEnvironmentChinaCloud represents the Azure China cloud environment.<br /> |
| `GermanCloud` | AzureEnvironmentGermanCloud represents the Azure German cloud environment.<br /> |


#### AzureKVAuth



AzureKVAuth defines configuration for authentication with Azure Key Vault.



_Appears in:_
- [AzureKVProvider](#azurekvprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _[SecretKeySelector](#secretkeyselector)_ | The Azure clientId of the service principle or managed identity used for authentication. |  | Optional: \{\} <br /> |
| `tenantId` _[SecretKeySelector](#secretkeyselector)_ | The Azure tenantId of the managed identity used for authentication. |  | Optional: \{\} <br /> |
| `clientSecret` _[SecretKeySelector](#secretkeyselector)_ | The Azure ClientSecret of the service principle used for authentication. |  | Optional: \{\} <br /> |
| `clientCertificate` _[SecretKeySelector](#secretkeyselector)_ | The Azure ClientCertificate of the service principle used for authentication. |  | Optional: \{\} <br /> |


#### AzureKVProvider



AzureKVProvider configures a store to sync secrets using Azure Key Vault.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authType` _[AzureAuthType](#azureauthtype)_ | Auth type defines how to authenticate to the keyvault service.<br />Valid values are:<br />- "ServicePrincipal" (default): Using a service principal (tenantId, clientId, clientSecret)<br />- "ManagedIdentity": Using Managed Identity assigned to the pod (see aad-pod-identity) | ServicePrincipal | Enum: [ServicePrincipal ManagedIdentity WorkloadIdentity] <br />Optional: \{\} <br /> |
| `vaultUrl` _string_ | Vault Url from which the secrets to be fetched from. |  |  |
| `tenantId` _string_ | TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type. Optional for WorkloadIdentity. |  | Optional: \{\} <br /> |
| `environmentType` _[AzureEnvironmentType](#azureenvironmenttype)_ | EnvironmentType specifies the Azure cloud environment endpoints to use for<br />connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.<br />The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152<br />PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud | PublicCloud | Enum: [PublicCloud USGovernmentCloud ChinaCloud GermanCloud] <br /> |
| `authSecretRef` _[AzureKVAuth](#azurekvauth)_ | Auth configures how the operator authenticates with Azure. Required for ServicePrincipal auth type. Optional for WorkloadIdentity. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specified the service account<br />that should be used when authenticating with WorkloadIdentity. |  | Optional: \{\} <br /> |
| `identityId` _string_ | If multiple Managed Identity is assigned to the pod, you can select the one to be used |  | Optional: \{\} <br /> |


#### BeyondTrustProviderSecretRef



BeyondTrustProviderSecretRef defines a reference to a secret containing credentials for the BeyondTrust provider.



_Appears in:_
- [BeyondtrustAuth](#beyondtrustauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### BeyondtrustAuth



BeyondtrustAuth configures authentication for BeyondTrust Password Safe.



_Appears in:_
- [BeyondtrustProvider](#beyondtrustprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKey` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | APIKey If not provided then ClientID/ClientSecret become required. |  |  |
| `clientId` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | ClientID is the API OAuth Client ID. |  |  |
| `clientSecret` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | ClientSecret is the API OAuth Client Secret. |  |  |
| `certificate` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | Certificate (cert.pem) for use when authenticating with an OAuth client Id using a Client Certificate. |  |  |
| `certificateKey` _[BeyondTrustProviderSecretRef](#beyondtrustprovidersecretref)_ | Certificate private key (key.pem). For use when authenticating with an OAuth client Id |  |  |


#### BeyondtrustProvider



BeyondtrustProvider defines configuration for the BeyondTrust Password Safe provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[BeyondtrustAuth](#beyondtrustauth)_ | Auth configures how the operator authenticates with Beyondtrust. |  |  |
| `server` _[BeyondtrustServer](#beyondtrustserver)_ | Auth configures how API server works. |  |  |


#### BeyondtrustServer



BeyondtrustServer defines configuration for connecting to BeyondTrust Password Safe server.



_Appears in:_
- [BeyondtrustProvider](#beyondtrustprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ |  |  |  |
| `apiVersion` _string_ |  |  |  |
| `retrievalType` _string_ | The secret retrieval type. SECRET = Secrets Safe (credential, text, file). MANAGED_ACCOUNT = Password Safe account associated with a system. |  |  |
| `separator` _string_ | A character that separates the folder names. |  |  |
| `decrypt` _boolean_ | When true, the response includes the decrypted password. When false, the password field is omitted. This option only applies to the SECRET retrieval type. Default: true. | true | Optional: \{\} <br /> |
| `verifyCA` _boolean_ |  |  |  |
| `clientTimeOutSeconds` _integer_ | Timeout specifies a time limit for requests made by this Client. The timeout includes connection time, any redirects, and reading the response body. Defaults to 45 seconds. |  |  |


#### BitwardenSecretsManagerAuth



BitwardenSecretsManagerAuth contains the ref to the secret that contains the machine account token.



_Appears in:_
- [BitwardenSecretsManagerProvider](#bitwardensecretsmanagerprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[BitwardenSecretsManagerSecretRef](#bitwardensecretsmanagersecretref)_ |  |  |  |


#### BitwardenSecretsManagerProvider



BitwardenSecretsManagerProvider configures a store to sync secrets with a Bitwarden Secrets Manager instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiURL` _string_ |  |  |  |
| `identityURL` _string_ |  |  |  |
| `bitwardenServerSDKURL` _string_ |  |  |  |
| `caBundle` _string_ | Base64 encoded certificate for the bitwarden server sdk. The sdk MUST run with HTTPS to make sure no MITM attack<br />can be performed. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | see: https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider |  | Optional: \{\} <br /> |
| `organizationID` _string_ | OrganizationID determines which organization this secret store manages. |  |  |
| `projectID` _string_ | ProjectID determines which project this secret store manages. |  |  |
| `auth` _[BitwardenSecretsManagerAuth](#bitwardensecretsmanagerauth)_ | Auth configures how secret-manager authenticates with a bitwarden machine account instance.<br />Make sure that the token being used has permissions on the given secret. |  |  |


#### BitwardenSecretsManagerSecretRef



BitwardenSecretsManagerSecretRef contains the credential ref to the bitwarden instance.



_Appears in:_
- [BitwardenSecretsManagerAuth](#bitwardensecretsmanagerauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `credentials` _[SecretKeySelector](#secretkeyselector)_ | AccessToken used for the bitwarden instance. |  | Required: \{\} <br /> |


#### CAProvider



CAProvider provides custom certificate authority (CA) certificates
for a secret store. The CAProvider points to a Secret or ConfigMap resource
that contains a PEM-encoded certificate.



_Appears in:_
- [AkeylessProvider](#akeylessprovider)
- [BitwardenSecretsManagerProvider](#bitwardensecretsmanagerprovider)
- [ConjurProvider](#conjurprovider)
- [GitlabProvider](#gitlabprovider)
- [KubernetesServer](#kubernetesserver)
- [VaultProvider](#vaultprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[CAProviderType](#caprovidertype)_ | The type of provider to use such as "Secret", or "ConfigMap". |  | Enum: [Secret ConfigMap] <br /> |
| `name` _string_ | The name of the object located at the provider type. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `key` _string_ | The key where the CA certificate can be found in the Secret or ConfigMap. |  | MaxLength: 253 <br />MinLength: 1 <br />Optional: \{\} <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |
| `namespace` _string_ | The namespace the Provider type is in.<br />Can only be defined when used in a ClusterSecretStore. |  | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


#### CAProviderType

_Underlying type:_ _string_

CAProviderType defines the type of provider to use for CA certificates.



_Appears in:_
- [CAProvider](#caprovider)

| Field | Description |
| --- | --- |
| `Secret` | CAProviderTypeSecret indicates that the CA certificate is stored in a Secret.<br /> |
| `ConfigMap` | CAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap.<br /> |


#### CSMAuth



CSMAuth contains a secretRef for credentials.



_Appears in:_
- [CloudruSMProvider](#cloudrusmprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[CSMAuthSecretRef](#csmauthsecretref)_ |  |  | Optional: \{\} <br /> |


#### CSMAuthSecretRef



CSMAuthSecretRef holds secret references for Cloud.ru credentials.



_Appears in:_
- [CSMAuth](#csmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessKeyIDSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeyID is used for authentication |  |  |
| `accessKeySecretSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeySecret is used for authentication |  |  |


#### CertAuth



CertAuth defines certificate-based authentication for the Kubernetes provider.



_Appears in:_
- [KubernetesAuth](#kubernetesauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientCert` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `clientKey` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### ChefAuth



ChefAuth contains a secretRef for credentials.



_Appears in:_
- [ChefProvider](#chefprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[ChefAuthSecretRef](#chefauthsecretref)_ |  |  |  |


#### ChefAuthSecretRef



ChefAuthSecretRef holds secret references for chef server login credentials.



_Appears in:_
- [ChefAuth](#chefauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `privateKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretKey is the Signing Key in PEM format, used for authentication. |  |  |


#### ChefProvider



ChefProvider configures a store to sync secrets using basic chef server connection credentials.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[ChefAuth](#chefauth)_ | Auth defines the information necessary to authenticate against chef Server |  |  |
| `username` _string_ | UserName should be the user ID on the chef server |  |  |
| `serverUrl` _string_ | ServerURL is the chef server URL used to connect to. If using orgs you should include your org in the url and terminate the url with a "/" |  |  |


#### CloudruSMProvider



CloudruSMProvider configures a store to sync secrets using the Cloud.ru Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[CSMAuth](#csmauth)_ |  |  |  |
| `projectID` _string_ | ProjectID is the project, which the secrets are stored in. |  |  |


#### ClusterExternalSecret



ClusterExternalSecret is the schema for the clusterexternalsecrets API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1beta1` | | |
| `kind` _string_ | `ClusterExternalSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ClusterExternalSecretSpec](#clusterexternalsecretspec)_ |  |  |  |
| `status` _[ClusterExternalSecretStatus](#clusterexternalsecretstatus)_ |  |  |  |


#### ClusterExternalSecretConditionType

_Underlying type:_ _string_

ClusterExternalSecretConditionType indicates the condition of the ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecretStatusCondition](#clusterexternalsecretstatuscondition)

| Field | Description |
| --- | --- |
| `Ready` |  |


#### ClusterExternalSecretNamespaceFailure



ClusterExternalSecretNamespaceFailure represents a failed namespace deployment and it's reason.



_Appears in:_
- [ClusterExternalSecretStatus](#clusterexternalsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespace` _string_ | Namespace is the namespace that failed when trying to apply an ExternalSecret |  |  |
| `reason` _string_ | Reason is why the ExternalSecret failed to apply to the namespace |  | Optional: \{\} <br /> |


#### ClusterExternalSecretSpec



ClusterExternalSecretSpec defines the desired state of ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecret](#clusterexternalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalSecretSpec` _[ExternalSecretSpec](#externalsecretspec)_ | The spec for the ExternalSecrets to be created |  |  |
| `externalSecretName` _string_ | The name of the external secrets to be created.<br />Defaults to the name of the ClusterExternalSecret |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Optional: \{\} <br /> |
| `externalSecretMetadata` _[ExternalSecretMetadata](#externalsecretmetadata)_ | The metadata of the external secrets to be created |  | Optional: \{\} <br /> |
| `namespaceSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | The labels to select by to find the Namespaces to create the ExternalSecrets in |  |  |
| `namespaceSelectors` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta) array_ | A list of labels to select by to find the Namespaces to create the ExternalSecrets in. The selectors are ORed. |  | Optional: \{\} <br /> |
| `namespaces` _string array_ | Choose namespaces by name. This field is ORed with anything that NamespaceSelectors ends up choosing.<br />Deprecated: Use NamespaceSelectors instead. |  | items:MaxLength: 63 <br />items:MinLength: 1 <br />items:Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |
| `refreshTime` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | The time in which the controller should reconcile its objects and recheck namespaces for labels. |  |  |


#### ClusterExternalSecretStatus



ClusterExternalSecretStatus defines the observed state of ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecret](#clusterexternalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `externalSecretName` _string_ | ExternalSecretName is the name of the ExternalSecrets created by the ClusterExternalSecret |  |  |
| `failedNamespaces` _[ClusterExternalSecretNamespaceFailure](#clusterexternalsecretnamespacefailure) array_ | Failed namespaces are the namespaces that failed to apply an ExternalSecret |  | Optional: \{\} <br /> |
| `provisionedNamespaces` _string array_ | ProvisionedNamespaces are the namespaces where the ClusterExternalSecret has secrets |  | Optional: \{\} <br /> |
| `conditions` _[ClusterExternalSecretStatusCondition](#clusterexternalsecretstatuscondition) array_ |  |  | Optional: \{\} <br /> |


#### ClusterExternalSecretStatusCondition



ClusterExternalSecretStatusCondition indicates the status of the ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecretStatus](#clusterexternalsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ClusterExternalSecretConditionType](#clusterexternalsecretconditiontype)_ |  |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `message` _string_ |  |  | Optional: \{\} <br /> |


#### ClusterSecretStore



ClusterSecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1beta1` | | |
| `kind` _string_ | `ClusterSecretStore` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SecretStoreSpec](#secretstorespec)_ |  |  |  |
| `status` _[SecretStoreStatus](#secretstorestatus)_ |  |  |  |


#### ClusterSecretStoreCondition



ClusterSecretStoreCondition describes a condition by which to choose namespaces to process ExternalSecrets in
for a ClusterSecretStore instance.



_Appears in:_
- [SecretStoreSpec](#secretstorespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespaceSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | Choose namespace using a labelSelector |  | Optional: \{\} <br /> |
| `namespaces` _string array_ | Choose namespaces by name |  | items:MaxLength: 63 <br />items:MinLength: 1 <br />items:Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |
| `namespaceRegexes` _string array_ | Choose namespaces by using regex matching |  | Optional: \{\} <br /> |


#### ConjurAPIKey



ConjurAPIKey defines authentication using a Conjur API key.



_Appears in:_
- [ConjurAuth](#conjurauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `account` _string_ | Account is the Conjur organization account name. |  |  |
| `userRef` _[SecretKeySelector](#secretkeyselector)_ | A reference to a specific 'key' containing the Conjur username<br />within a Secret resource. In some instances, `key` is a required field. |  |  |
| `apiKeyRef` _[SecretKeySelector](#secretkeyselector)_ | A reference to a specific 'key' containing the Conjur API key<br />within a Secret resource. In some instances, `key` is a required field. |  |  |


#### ConjurAuth



ConjurAuth defines the methods of authentication with Conjur.



_Appears in:_
- [ConjurProvider](#conjurprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apikey` _[ConjurAPIKey](#conjurapikey)_ | Authenticates with Conjur using an API key. |  | Optional: \{\} <br /> |
| `jwt` _[ConjurJWT](#conjurjwt)_ | Jwt enables JWT authentication using Kubernetes service account tokens. |  | Optional: \{\} <br /> |


#### ConjurJWT



ConjurJWT defines authentication using a JWT service account token.



_Appears in:_
- [ConjurAuth](#conjurauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `account` _string_ | Account is the Conjur organization account name. |  |  |
| `serviceID` _string_ | The conjur authn jwt webservice id |  |  |
| `hostId` _string_ | Optional HostID for JWT authentication. This may be used depending<br />on how the Conjur JWT authenticator policy is configured. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional SecretRef that refers to a key in a Secret resource containing JWT token to<br />authenticate with Conjur using the JWT authentication method. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Optional ServiceAccountRef specifies the Kubernetes service account for which to request<br />a token for with the `TokenRequest` API. |  | Optional: \{\} <br /> |


#### ConjurProvider



ConjurProvider defines configuration for the CyberArk Conjur provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the endpoint of the Conjur instance. |  |  |
| `caBundle` _string_ | CABundle is a PEM encoded CA bundle that will be used to validate the Conjur server certificate. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | Used to provide custom certificate authority (CA) certificates<br />for a secret store. The CAProvider points to a Secret or ConfigMap resource<br />that contains a PEM-encoded certificate. |  | Optional: \{\} <br /> |
| `auth` _[ConjurAuth](#conjurauth)_ | Defines authentication settings for connecting to Conjur. |  |  |


#### DelineaProvider



DelineaProvider defines configuration for the Delinea DevOps Secrets Vault provider.
See https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _[DelineaProviderSecretRef](#delineaprovidersecretref)_ | ClientID is the non-secret part of the credential. |  |  |
| `clientSecret` _[DelineaProviderSecretRef](#delineaprovidersecretref)_ | ClientSecret is the secret part of the credential. |  |  |
| `tenant` _string_ | Tenant is the chosen hostname / site name. |  |  |
| `urlTemplate` _string_ | URLTemplate<br />If unset, defaults to "https://%s.secretsvaultcloud.%s/v1/%s%s". |  | Optional: \{\} <br /> |
| `tld` _string_ | TLD is based on the server location that was chosen during provisioning.<br />If unset, defaults to "com". |  | Optional: \{\} <br /> |


#### DelineaProviderSecretRef



DelineaProviderSecretRef defines a reference to a secret containing credentials for the Delinea provider.



_Appears in:_
- [DelineaProvider](#delineaprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### Device42Auth



Device42Auth defines the authentication method for the Device42 provider.



_Appears in:_
- [Device42Provider](#device42provider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[Device42SecretRef](#device42secretref)_ |  |  |  |


#### Device42Provider



Device42Provider configures a store to sync secrets with a Device42 instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ | URL configures the Device42 instance URL. |  |  |
| `auth` _[Device42Auth](#device42auth)_ | Auth configures how secret-manager authenticates with a Device42 instance. |  |  |


#### Device42SecretRef



Device42SecretRef defines a reference to a secret containing credentials for the Device42 provider.



_Appears in:_
- [Device42Auth](#device42auth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `credentials` _[SecretKeySelector](#secretkeyselector)_ | Username / Password is used for authentication. |  | Optional: \{\} <br /> |


#### DopplerAuth



DopplerAuth defines the authentication method for the Doppler provider.



_Appears in:_
- [DopplerProvider](#dopplerprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[DopplerAuthSecretRef](#dopplerauthsecretref)_ |  |  |  |


#### DopplerAuthSecretRef



DopplerAuthSecretRef defines a reference to a secret containing credentials for the Doppler provider.



_Appears in:_
- [DopplerAuth](#dopplerauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dopplerToken` _[SecretKeySelector](#secretkeyselector)_ | The DopplerToken is used for authentication.<br />See https://docs.doppler.com/reference/api#authentication for auth token types.<br />The Key attribute defaults to dopplerToken if not specified. |  |  |


#### DopplerProvider



DopplerProvider configures a store to sync secrets using the Doppler provider.
Project and Config are required if not using a Service Token.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[DopplerAuth](#dopplerauth)_ | Auth configures how the Operator authenticates with the Doppler API |  |  |
| `project` _string_ | Doppler project (required if not using a Service Token) |  | Optional: \{\} <br /> |
| `config` _string_ | Doppler config (required if not using a Service Token) |  | Optional: \{\} <br /> |
| `nameTransformer` _string_ | Environment variable compatible name transforms that change secret names to a different format |  | Enum: [upper-camel camel lower-snake tf-var dotnet-env lower-kebab] <br />Optional: \{\} <br /> |
| `format` _string_ | Format enables the downloading of secrets as a file (string) |  | Enum: [json dotnet-json env yaml docker] <br />Optional: \{\} <br /> |


#### ExternalSecret



ExternalSecret is the schema for the external-secrets API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1beta1` | | |
| `kind` _string_ | `ExternalSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ExternalSecretSpec](#externalsecretspec)_ |  |  |  |
| `status` _[ExternalSecretStatus](#externalsecretstatus)_ |  |  |  |


#### ExternalSecretConditionType

_Underlying type:_ _string_

ExternalSecretConditionType defines the condition type for an ExternalSecret.



_Appears in:_
- [ExternalSecretStatusCondition](#externalsecretstatuscondition)

| Field | Description |
| --- | --- |
| `Ready` | ExternalSecretReady indicates the ExternalSecret has been successfully reconciled.<br /> |
| `Deleted` | ExternalSecretDeleted indicates the ExternalSecret has been deleted.<br /> |


#### ExternalSecretConversionStrategy

_Underlying type:_ _string_

ExternalSecretConversionStrategy defines how secret values are converted.

_Validation:_
- Enum: [Default Unicode]

_Appears in:_
- [ExternalSecretDataRemoteRef](#externalsecretdataremoteref)
- [ExternalSecretFind](#externalsecretfind)

| Field | Description |
| --- | --- |
| `Default` | ExternalSecretConversionDefault indicates the default conversion strategy.<br /> |
| `Unicode` | ExternalSecretConversionUnicode indicates that unicode conversion will be performed.<br /> |


#### ExternalSecretCreationPolicy

_Underlying type:_ _string_

ExternalSecretCreationPolicy defines rules on how to create the resulting Secret.

_Validation:_
- Enum: [Owner Orphan Merge None]

_Appears in:_
- [ExternalSecretTarget](#externalsecrettarget)

| Field | Description |
| --- | --- |
| `Owner` | CreatePolicyOwner creates the Secret and sets .metadata.ownerReferences to the ExternalSecret resource.<br /> |
| `Orphan` | CreatePolicyOrphan creates the Secret and does not set the ownerReference.<br />I.e. it will be orphaned after the deletion of the ExternalSecret.<br /> |
| `Merge` | CreatePolicyMerge does not create the Secret, but merges the data fields to the Secret.<br /> |
| `None` | CreatePolicyNone does not create a Secret (future use with injector).<br /> |


#### ExternalSecretData



ExternalSecretData defines the connection between the Kubernetes Secret key (spec.data.<key>) and the Provider data.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretKey` _string_ | The key in the Kubernetes Secret to store the value. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |
| `remoteRef` _[ExternalSecretDataRemoteRef](#externalsecretdataremoteref)_ | RemoteRef points to the remote secret and defines<br />which secret (version/property/..) to fetch. |  |  |
| `sourceRef` _[StoreSourceRef](#storesourceref)_ | SourceRef allows you to override the source<br />from which the value will be pulled. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |


#### ExternalSecretDataFromRemoteRef



ExternalSecretDataFromRemoteRef defines a reference to multiple secrets in the provider to be fetched using options.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `extract` _[ExternalSecretDataRemoteRef](#externalsecretdataremoteref)_ | Used to extract multiple key/value pairs from one secret<br />Note: Extract does not support sourceRef.Generator or sourceRef.GeneratorRef. |  | Optional: \{\} <br /> |
| `find` _[ExternalSecretFind](#externalsecretfind)_ | Used to find secrets based on tags or regular expressions<br />Note: Find does not support sourceRef.Generator or sourceRef.GeneratorRef. |  | Optional: \{\} <br /> |
| `rewrite` _[ExternalSecretRewrite](#externalsecretrewrite) array_ | Used to rewrite secret Keys after getting them from the secret Provider<br />Multiple Rewrite operations can be provided. They are applied in a layered order (first to last) |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `sourceRef` _[StoreGeneratorSourceRef](#storegeneratorsourceref)_ | SourceRef points to a store or generator<br />which contains secret values ready to use.<br />Use this in combination with Extract or Find pull values out of<br />a specific SecretStore.<br />When sourceRef points to a generator Extract or Find is not supported.<br />The generator returns a static map of values |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |


#### ExternalSecretDataRemoteRef



ExternalSecretDataRemoteRef defines Provider data location.



_Appears in:_
- [ExternalSecretData](#externalsecretdata)
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ | Key is the key used in the Provider, mandatory |  |  |
| `metadataPolicy` _[ExternalSecretMetadataPolicy](#externalsecretmetadatapolicy)_ | Policy for fetching tags/labels from provider secrets, possible options are Fetch, None. Defaults to None | None | Enum: [None Fetch] <br />Optional: \{\} <br /> |
| `property` _string_ | Used to select a specific property of the Provider value (if a map), if supported |  | Optional: \{\} <br /> |
| `version` _string_ | Used to select a specific version of the Provider value, if supported |  | Optional: \{\} <br /> |
| `conversionStrategy` _[ExternalSecretConversionStrategy](#externalsecretconversionstrategy)_ | Used to define a conversion Strategy | Default | Enum: [Default Unicode] <br />Optional: \{\} <br /> |
| `decodingStrategy` _[ExternalSecretDecodingStrategy](#externalsecretdecodingstrategy)_ | Used to define a decoding Strategy | None | Enum: [Auto Base64 Base64URL None] <br />Optional: \{\} <br /> |


#### ExternalSecretDecodingStrategy

_Underlying type:_ _string_

ExternalSecretDecodingStrategy defines how secret values are decoded.

_Validation:_
- Enum: [Auto Base64 Base64URL None]

_Appears in:_
- [ExternalSecretDataRemoteRef](#externalsecretdataremoteref)
- [ExternalSecretFind](#externalsecretfind)

| Field | Description |
| --- | --- |
| `Auto` | ExternalSecretDecodeAuto indicates that the decoding strategy will be automatically determined.<br /> |
| `Base64` | ExternalSecretDecodeBase64 indicates that base64 decoding will be used.<br /> |
| `Base64URL` | ExternalSecretDecodeBase64URL indicates that base64url decoding will be used.<br /> |
| `None` | ExternalSecretDecodeNone indicates that no decoding will be performed.<br /> |


#### ExternalSecretDeletionPolicy

_Underlying type:_ _string_

ExternalSecretDeletionPolicy defines rules on how to delete the resulting Secret.

_Validation:_
- Enum: [Delete Merge Retain]

_Appears in:_
- [ExternalSecretTarget](#externalsecrettarget)

| Field | Description |
| --- | --- |
| `Delete` | DeletionPolicyDelete deletes the secret if all provider secrets are deleted.<br />If a secret gets deleted on the provider side and is not accessible<br />anymore this is not considered an error and the ExternalSecret<br />does not go into SecretSyncedError status.<br /> |
| `Merge` | DeletionPolicyMerge removes keys in the secret, but not the secret itself.<br />If a secret gets deleted on the provider side and is not accessible<br />anymore this is not considered an error and the ExternalSecret<br />does not go into SecretSyncedError status.<br /> |
| `Retain` | DeletionPolicyRetain will retain the secret if all provider secrets have been deleted.<br />If a provider secret does not exist the ExternalSecret gets into the<br />SecretSyncedError status.<br /> |


#### ExternalSecretFind



ExternalSecretFind defines criteria for finding secrets in the provider.



_Appears in:_
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | A root path to start the find operations. |  | Optional: \{\} <br /> |
| `name` _[FindName](#findname)_ | Finds secrets based on the name. |  | Optional: \{\} <br /> |
| `tags` _object (keys:string, values:string)_ | Find secrets based on tags. |  | Optional: \{\} <br /> |
| `conversionStrategy` _[ExternalSecretConversionStrategy](#externalsecretconversionstrategy)_ | Used to define a conversion Strategy | Default | Enum: [Default Unicode] <br />Optional: \{\} <br /> |
| `decodingStrategy` _[ExternalSecretDecodingStrategy](#externalsecretdecodingstrategy)_ | Used to define a decoding Strategy | None | Enum: [Auto Base64 Base64URL None] <br />Optional: \{\} <br /> |


#### ExternalSecretMetadata



ExternalSecretMetadata defines metadata fields for the ExternalSecret generated by the ClusterExternalSecret.



_Appears in:_
- [ClusterExternalSecretSpec](#clusterexternalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `annotations` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |


#### ExternalSecretMetadataPolicy

_Underlying type:_ _string_

ExternalSecretMetadataPolicy defines the policy for fetching tags/labels from provider secrets.

_Validation:_
- Enum: [None Fetch]

_Appears in:_
- [ExternalSecretDataRemoteRef](#externalsecretdataremoteref)

| Field | Description |
| --- | --- |
| `None` | ExternalSecretMetadataPolicyNone indicates that no metadata will be fetched.<br /> |
| `Fetch` | ExternalSecretMetadataPolicyFetch indicates that metadata will be fetched from the provider.<br /> |


#### ExternalSecretRefreshPolicy

_Underlying type:_ _string_

ExternalSecretRefreshPolicy defines how and when the ExternalSecret should be refreshed.

_Validation:_
- Enum: [CreatedOnce Periodic OnChange]

_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description |
| --- | --- |
| `CreatedOnce` | RefreshPolicyCreatedOnce creates the Secret only if it does not exist and does not update it thereafter.<br /> |
| `Periodic` | RefreshPolicyPeriodic synchronizes the Secret from the external source at regular intervals.<br /> |
| `OnChange` | RefreshPolicyOnChange only synchronizes the Secret when the ExternalSecret's metadata or specification changes.<br /> |


#### ExternalSecretRewrite



ExternalSecretRewrite defines rules on how to rewrite secret keys.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `regexp` _[ExternalSecretRewriteRegexp](#externalsecretrewriteregexp)_ | Used to rewrite with regular expressions.<br />The resulting key will be the output of a regexp.ReplaceAll operation. |  | Optional: \{\} <br /> |
| `transform` _[ExternalSecretRewriteTransform](#externalsecretrewritetransform)_ | Used to apply string transformation on the secrets.<br />The resulting key will be the output of the template applied by the operation. |  | Optional: \{\} <br /> |


#### ExternalSecretRewriteRegexp



ExternalSecretRewriteRegexp defines how to use regular expressions for rewriting secret keys.



_Appears in:_
- [ExternalSecretRewrite](#externalsecretrewrite)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `source` _string_ | Used to define the regular expression of a re.Compiler. |  |  |
| `target` _string_ | Used to define the target pattern of a ReplaceAll operation. |  |  |


#### ExternalSecretRewriteTransform



ExternalSecretRewriteTransform defines how to use string templates for transforming secret keys.



_Appears in:_
- [ExternalSecretRewrite](#externalsecretrewrite)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `template` _string_ | Used to define the template to apply on the secret name.<br />`.value ` will specify the secret name in the template. |  |  |


#### ExternalSecretSpec



ExternalSecretSpec defines the desired state of ExternalSecret.



_Appears in:_
- [ClusterExternalSecretSpec](#clusterexternalsecretspec)
- [ExternalSecret](#externalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretStoreRef` _[SecretStoreRef](#secretstoreref)_ |  |  | Optional: \{\} <br /> |
| `target` _[ExternalSecretTarget](#externalsecrettarget)_ |  | \{ creationPolicy:Owner deletionPolicy:Retain \} | Optional: \{\} <br /> |
| `refreshPolicy` _[ExternalSecretRefreshPolicy](#externalsecretrefreshpolicy)_ | RefreshPolicy determines how the ExternalSecret should be refreshed:<br />- CreatedOnce: Creates the Secret only if it does not exist and does not update it thereafter<br />- Periodic: Synchronizes the Secret from the external source at regular intervals specified by refreshInterval.<br />  No periodic updates occur if refreshInterval is 0.<br />- OnChange: Only synchronizes the Secret when the ExternalSecret's metadata or specification changes |  | Enum: [CreatedOnce Periodic OnChange] <br />Optional: \{\} <br /> |
| `refreshInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | RefreshInterval is the amount of time before the values are read again from the SecretStore provider,<br />specified as Golang Duration strings.<br />Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"<br />Example values: "1h0m0s", "2h30m0s", "10m0s"<br />May be set to "0s" to fetch and create it once. Defaults to 1h0m0s. | 1h0m0s |  |
| `data` _[ExternalSecretData](#externalsecretdata) array_ | Data defines the connection between the Kubernetes Secret keys and the Provider data |  | Optional: \{\} <br /> |
| `dataFrom` _[ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref) array_ | DataFrom is used to fetch all properties from a specific Provider data<br />If multiple entries are specified, the Secret keys are merged in the specified order |  | Optional: \{\} <br /> |


#### ExternalSecretStatus



ExternalSecretStatus defines the observed state of ExternalSecret.



_Appears in:_
- [ExternalSecret](#externalsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `refreshTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ | refreshTime is the time and date the external secret was fetched and<br />the target secret updated |  |  |
| `syncedResourceVersion` _string_ | SyncedResourceVersion keeps track of the last synced version |  |  |
| `conditions` _[ExternalSecretStatusCondition](#externalsecretstatuscondition) array_ |  |  | Optional: \{\} <br /> |
| `binding` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#localobjectreference-v1-core)_ | Binding represents a servicebinding.io Provisioned Service reference to the secret |  |  |


#### ExternalSecretStatusCondition



ExternalSecretStatusCondition contains condition information for an ExternalSecret.



_Appears in:_
- [ExternalSecretStatus](#externalsecretstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[ExternalSecretConditionType](#externalsecretconditiontype)_ |  |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `reason` _string_ |  |  | Optional: \{\} <br /> |
| `message` _string_ |  |  | Optional: \{\} <br /> |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ |  |  | Optional: \{\} <br /> |


#### ExternalSecretTarget



ExternalSecretTarget defines the Kubernetes Secret to be created
There can be only one target per ExternalSecret.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The name of the Secret resource to be managed.<br />Defaults to the .metadata.name of the ExternalSecret resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br />Optional: \{\} <br /> |
| `creationPolicy` _[ExternalSecretCreationPolicy](#externalsecretcreationpolicy)_ | CreationPolicy defines rules on how to create the resulting Secret.<br />Defaults to "Owner" | Owner | Enum: [Owner Orphan Merge None] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[ExternalSecretDeletionPolicy](#externalsecretdeletionpolicy)_ | DeletionPolicy defines rules on how to delete the resulting Secret.<br />Defaults to "Retain" | Retain | Enum: [Delete Merge Retain] <br />Optional: \{\} <br /> |
| `template` _[ExternalSecretTemplate](#externalsecrettemplate)_ | Template defines a blueprint for the created Secret resource. |  | Optional: \{\} <br /> |
| `immutable` _boolean_ | Immutable defines if the final secret will be immutable |  | Optional: \{\} <br /> |


#### ExternalSecretTemplate



ExternalSecretTemplate defines a blueprint for the created Secret resource.
we can not use native corev1.Secret, it will have empty ObjectMeta values: https://github.com/kubernetes-sigs/controller-tools/issues/448



_Appears in:_
- [ExternalSecretTarget](#externalsecrettarget)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[SecretType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#secrettype-v1-core)_ |  |  | Optional: \{\} <br /> |
| `engineVersion` _[TemplateEngineVersion](#templateengineversion)_ | EngineVersion specifies the template engine version<br />that should be used to compile/execute the<br />template specified in .data and .templateFrom[]. | v2 | Enum: [v2] <br /> |
| `metadata` _[ExternalSecretTemplateMetadata](#externalsecrettemplatemetadata)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `mergePolicy` _[TemplateMergePolicy](#templatemergepolicy)_ |  | Replace | Enum: [Replace Merge] <br /> |
| `data` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `templateFrom` _[TemplateFrom](#templatefrom) array_ |  |  | Optional: \{\} <br /> |


#### ExternalSecretTemplateMetadata



ExternalSecretTemplateMetadata defines metadata fields for the Secret blueprint.



_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `annotations` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |
| `labels` _object (keys:string, values:string)_ |  |  | Optional: \{\} <br /> |




#### FakeProvider



FakeProvider configures a fake provider that returns static values.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _[FakeProviderData](#fakeproviderdata) array_ |  |  |  |


#### FakeProviderData



FakeProviderData defines a key-value pair for the fake provider used in testing.



_Appears in:_
- [FakeProvider](#fakeprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  |  |
| `value` _string_ |  |  |  |
| `version` _string_ |  |  |  |


#### FindName



FindName defines name matching criteria for finding secrets.



_Appears in:_
- [ExternalSecretFind](#externalsecretfind)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `regexp` _string_ | Finds secrets base |  | Optional: \{\} <br /> |


#### FortanixProvider



FortanixProvider configures a store to sync secrets using the Fortanix SDKMS provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the URL of SDKMS API. Defaults to `sdkms.fortanix.com`. |  |  |
| `apiKey` _[FortanixProviderSecretRef](#fortanixprovidersecretref)_ | APIKey is the API token to access SDKMS Applications. |  |  |


#### FortanixProviderSecretRef



FortanixProviderSecretRef defines a reference to a secret containing credentials for the Fortanix provider.



_Appears in:_
- [FortanixProvider](#fortanixprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef is a reference to a secret containing the SDKMS API Key. |  |  |


#### GCPSMAuth



GCPSMAuth defines the authentication methods for the GCP Secret Manager provider.



_Appears in:_
- [GCPSMProvider](#gcpsmprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[GCPSMAuthSecretRef](#gcpsmauthsecretref)_ |  |  | Optional: \{\} <br /> |
| `workloadIdentity` _[GCPWorkloadIdentity](#gcpworkloadidentity)_ |  |  | Optional: \{\} <br /> |


#### GCPSMAuthSecretRef



GCPSMAuthSecretRef defines a reference to a secret containing credentials for the GCP Secret Manager provider.



_Appears in:_
- [GCPSMAuth](#gcpsmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretAccessKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  | Optional: \{\} <br /> |


#### GCPSMProvider



GCPSMProvider Configures a store to sync secrets using the GCP Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[GCPSMAuth](#gcpsmauth)_ | Auth defines the information necessary to authenticate against GCP |  | Optional: \{\} <br /> |
| `projectID` _string_ | ProjectID project where secret is located |  |  |
| `location` _string_ | Location optionally defines a location for a secret |  |  |


#### GCPWorkloadIdentity



GCPWorkloadIdentity defines configuration for using GCP Workload Identity authentication.



_Appears in:_
- [GCPSMAuth](#gcpsmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ |  |  | Required: \{\} <br /> |
| `clusterLocation` _string_ | ClusterLocation is the location of the cluster<br />If not specified, it fetches information from the metadata server |  | Optional: \{\} <br /> |
| `clusterName` _string_ | ClusterName is the name of the cluster<br />If not specified, it fetches information from the metadata server |  | Optional: \{\} <br /> |
| `clusterProjectID` _string_ | ClusterProjectID is the project ID of the cluster<br />If not specified, it fetches information from the metadata server |  | Optional: \{\} <br /> |


#### GeneratorRef



GeneratorRef points to a generator custom resource.



_Appears in:_
- [StoreGeneratorSourceRef](#storegeneratorsourceref)
- [StoreSourceRef](#storesourceref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | Specify the apiVersion of the generator resource | generators.external-secrets.io/v1alpha1 |  |
| `kind` _string_ | Specify the Kind of the generator resource |  | Enum: [ACRAccessToken ClusterGenerator ECRAuthorizationToken Fake GCRAccessToken GithubAccessToken QuayAccessToken Password SSHKey STSSessionToken UUID VaultDynamicSecret Webhook Grafana] <br /> |
| `name` _string_ | Specify the name of the generator resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |




#### GenericStore

_Underlying type:_ _interface{Copy() GenericStore; GetKind() string; GetNamespacedName() string; GetObjectMeta() *k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta; GetSpec() *SecretStoreSpec; GetStatus() SecretStoreStatus; GetTypeMeta() *k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta; SetStatus(status SecretStoreStatus); k8s.io/apimachinery/pkg/runtime.Object; k8s.io/apimachinery/pkg/apis/meta/v1.Object}_

GenericStore is a common interface for interacting with ClusterSecretStore
or a namespaced SecretStore.









#### GithubAppAuth



GithubAppAuth defines the GitHub App authentication mechanism for the GitHub provider.



_Appears in:_
- [GithubProvider](#githubprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `privateKey` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### GithubProvider



GithubProvider configures a store to push secrets to Github Actions.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL configures the Github instance URL. Defaults to https://github.com/. | https://github.com/ |  |
| `uploadURL` _string_ | Upload URL for enterprise instances. Default to URL. |  | Optional: \{\} <br /> |
| `auth` _[GithubAppAuth](#githubappauth)_ | auth configures how secret-manager authenticates with a Github instance. |  |  |
| `appID` _integer_ | appID specifies the Github APP that will be used to authenticate the client |  |  |
| `installationID` _integer_ | installationID specifies the Github APP installation that will be used to authenticate the client |  |  |
| `organization` _string_ | organization will be used to fetch secrets from the Github organization |  |  |
| `repository` _string_ | repository will be used to fetch secrets from the Github repository within an organization |  | Optional: \{\} <br /> |
| `environment` _string_ | environment will be used to fetch secrets from a particular environment within a github repository |  | Optional: \{\} <br /> |


#### GitlabAuth



GitlabAuth defines the authentication method for the GitLab provider.



_Appears in:_
- [GitlabProvider](#gitlabprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `SecretRef` _[GitlabSecretRef](#gitlabsecretref)_ |  |  |  |


#### GitlabProvider



GitlabProvider configures a store to sync secrets with a GitLab instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL configures the GitLab instance URL. Defaults to https://gitlab.com/. |  |  |
| `auth` _[GitlabAuth](#gitlabauth)_ | Auth configures how secret-manager authenticates with a GitLab instance. |  |  |
| `projectID` _string_ | ProjectID specifies a project where secrets are located. |  |  |
| `inheritFromGroups` _boolean_ | InheritFromGroups specifies whether parent groups should be discovered and checked for secrets. |  |  |
| `groupIDs` _string array_ | GroupIDs specify, which gitlab groups to pull secrets from. Group secrets are read from left to right followed by the project variables. |  |  |
| `environment` _string_ | Environment environment_scope of gitlab CI/CD variables (Please see https://docs.gitlab.com/ee/ci/environments/#create-a-static-environment on how to create environments) |  |  |
| `caBundle` _integer array_ | Base64 encoded certificate for the GitLab server sdk. The sdk MUST run with HTTPS to make sure no MITM attack<br />can be performed. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | see: https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider |  | Optional: \{\} <br /> |


#### GitlabSecretRef



GitlabSecretRef defines a reference to a secret containing credentials for the GitLab provider.



_Appears in:_
- [GitlabAuth](#gitlabauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessToken` _[SecretKeySelector](#secretkeyselector)_ | AccessToken is used for authentication. |  |  |


#### IBMAuth



IBMAuth defines the authentication methods for the IBM Cloud Secrets Manager provider.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [IBMProvider](#ibmprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[IBMAuthSecretRef](#ibmauthsecretref)_ |  |  |  |
| `containerAuth` _[IBMAuthContainerAuth](#ibmauthcontainerauth)_ |  |  |  |


#### IBMAuthContainerAuth



IBMAuthContainerAuth defines authentication using IBM Container-based auth with IAM Trusted Profile.



_Appears in:_
- [IBMAuth](#ibmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `profile` _string_ | the IBM Trusted Profile |  |  |
| `tokenLocation` _string_ | Location the token is mounted on the pod |  |  |
| `iamEndpoint` _string_ |  |  |  |


#### IBMAuthSecretRef



IBMAuthSecretRef defines a reference to a secret containing credentials for the IBM provider.



_Appears in:_
- [IBMAuth](#ibmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretApiKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  |  |


#### IBMProvider



IBMProvider configures a store to sync secrets using a IBM Cloud Secrets Manager backend.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[IBMAuth](#ibmauth)_ | Auth configures how secret-manager authenticates with the IBM secrets manager. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `serviceUrl` _string_ | ServiceURL is the Endpoint URL that is specific to the Secrets Manager service instance |  |  |


#### InfisicalAuth



InfisicalAuth defines the authentication methods for the Infisical provider.



_Appears in:_
- [InfisicalProvider](#infisicalprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `universalAuthCredentials` _[UniversalAuthCredentials](#universalauthcredentials)_ |  |  | Optional: \{\} <br /> |


#### InfisicalProvider



InfisicalProvider configures a store to sync secrets using the Infisical provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[InfisicalAuth](#infisicalauth)_ | Auth configures how the Operator authenticates with the Infisical API |  | Required: \{\} <br /> |
| `secretsScope` _[MachineIdentityScopeInWorkspace](#machineidentityscopeinworkspace)_ | SecretsScope defines the scope of the secrets within the workspace |  | Required: \{\} <br /> |
| `hostAPI` _string_ | HostAPI specifies the base URL of the Infisical API. If not provided, it defaults to "https://app.infisical.com/api". | https://app.infisical.com/api | Optional: \{\} <br /> |


#### KeeperSecurityProvider



KeeperSecurityProvider Configures a store to sync secrets using Keeper Security.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `folderID` _string_ |  |  |  |


#### KubernetesAuth



KubernetesAuth defines authentication methods for the Kubernetes provider.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [KubernetesProvider](#kubernetesprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cert` _[CertAuth](#certauth)_ | has both clientCert and clientKey as secretKeySelector |  | Optional: \{\} <br /> |
| `token` _[TokenAuth](#tokenauth)_ | use static token to authenticate with |  | Optional: \{\} <br /> |
| `serviceAccount` _[ServiceAccountSelector](#serviceaccountselector)_ | points to a service account that should be used for authentication |  | Optional: \{\} <br /> |


#### KubernetesProvider



KubernetesProvider configures a store to sync secrets with a Kubernetes instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `server` _[KubernetesServer](#kubernetesserver)_ | configures the Kubernetes server Address. |  | Optional: \{\} <br /> |
| `auth` _[KubernetesAuth](#kubernetesauth)_ | Auth configures how secret-manager authenticates with a Kubernetes instance. |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `authRef` _[SecretKeySelector](#secretkeyselector)_ | A reference to a secret that contains the auth information. |  | Optional: \{\} <br /> |
| `remoteNamespace` _string_ | Remote namespace to fetch the secrets from | default | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


#### KubernetesServer



KubernetesServer defines the Kubernetes server connection configuration.



_Appears in:_
- [KubernetesProvider](#kubernetesprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | configures the Kubernetes server Address. | kubernetes.default | Optional: \{\} <br /> |
| `caBundle` _integer array_ | CABundle is a base64-encoded CA certificate |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | see: https://external-secrets.io/v0.4.1/spec/#external-secrets.io/v1alpha1.CAProvider |  | Optional: \{\} <br /> |


#### MachineIdentityScopeInWorkspace



MachineIdentityScopeInWorkspace defines the scope of a machine identity in an Infisical workspace.



_Appears in:_
- [InfisicalProvider](#infisicalprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretsPath` _string_ | SecretsPath specifies the path to the secrets within the workspace. Defaults to "/" if not provided. | / | Optional: \{\} <br /> |
| `recursive` _boolean_ | Recursive indicates whether the secrets should be fetched recursively. Defaults to false if not provided. | false | Optional: \{\} <br /> |
| `environmentSlug` _string_ | EnvironmentSlug is the required slug identifier for the environment. |  | Required: \{\} <br /> |
| `projectSlug` _string_ | ProjectSlug is the required slug identifier for the project. |  | Required: \{\} <br /> |
| `expandSecretReferences` _boolean_ | ExpandSecretReferences indicates whether secret references should be expanded. Defaults to true if not provided. | true | Optional: \{\} <br /> |


#### NTLMProtocol



NTLMProtocol contains the NTLM-specific configuration.



_Appears in:_
- [AuthorizationProtocol](#authorizationprotocol)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `usernameSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `passwordSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |






#### OnboardbaseAuthSecretRef



OnboardbaseAuthSecretRef holds secret references for onboardbase API Key credentials.



_Appears in:_
- [OnboardbaseProvider](#onboardbaseprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKeyRef` _[SecretKeySelector](#secretkeyselector)_ | OnboardbaseAPIKey is the APIKey generated by an admin account.<br />It is used to recognize and authorize access to a project and environment within onboardbase |  | Required: \{\} <br /> |
| `passcodeRef` _[SecretKeySelector](#secretkeyselector)_ | OnboardbasePasscode is the passcode attached to the API Key |  | Required: \{\} <br /> |


#### OnboardbaseProvider



OnboardbaseProvider configures a store to sync secrets using the Onboardbase provider.
Project and Config are required if not using a Service Token.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[OnboardbaseAuthSecretRef](#onboardbaseauthsecretref)_ | Auth configures how the Operator authenticates with the Onboardbase API |  |  |
| `apiHost` _string_ | APIHost use this to configure the host url for the API for selfhosted installation, default is https://public.onboardbase.com/api/v1/ | https://public.onboardbase.com/api/v1/ |  |
| `project` _string_ | Project is an onboardbase project that the secrets should be pulled from | development | Required: \{\} <br /> |
| `environment` _string_ | Environment is the name of an environmnent within a project to pull the secrets from | development | Required: \{\} <br /> |


#### OnePasswordAuth



OnePasswordAuth contains a secretRef for credentials.



_Appears in:_
- [OnePasswordProvider](#onepasswordprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[OnePasswordAuthSecretRef](#onepasswordauthsecretref)_ |  |  |  |


#### OnePasswordAuthSecretRef



OnePasswordAuthSecretRef holds secret references for 1Password credentials.



_Appears in:_
- [OnePasswordAuth](#onepasswordauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectTokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The ConnectToken is used for authentication to a 1Password Connect Server. |  |  |


#### OnePasswordProvider



OnePasswordProvider configures a store to sync secrets using the 1Password Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[OnePasswordAuth](#onepasswordauth)_ | Auth defines the information necessary to authenticate against OnePassword Connect Server |  |  |
| `connectHost` _string_ | ConnectHost defines the OnePassword Connect Server to connect to |  |  |
| `vaults` _object (keys:string, values:integer)_ | Vaults defines which OnePassword vaults to search in which order |  |  |


#### OracleAuth



OracleAuth defines authentication configuration for the Oracle Vault provider.



_Appears in:_
- [OracleProvider](#oracleprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tenancy` _string_ | Tenancy is the tenancy OCID where user is located. |  |  |
| `user` _string_ | User is an access OCID specific to the account. |  |  |
| `secretRef` _[OracleSecretRef](#oraclesecretref)_ | SecretRef to pass through sensitive information. |  |  |


#### OraclePrincipalType

_Underlying type:_ _string_

OraclePrincipalType defines the type of principal used for authentication to Oracle Vault.

_Validation:_
- Enum: [ UserPrincipal InstancePrincipal Workload]

_Appears in:_
- [OracleProvider](#oracleprovider)

| Field | Description |
| --- | --- |
| `UserPrincipal` | UserPrincipal represents a user principal.<br /> |
| `InstancePrincipal` | InstancePrincipal represents a instance principal.<br /> |
| `Workload` | WorkloadPrincipal represents a workload principal.<br /> |


#### OracleProvider



OracleProvider configures a store to sync secrets using an Oracle Vault backend.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `region` _string_ | Region is the region where vault is located. |  |  |
| `vault` _string_ | Vault is the vault's OCID of the specific vault where secret is located. |  |  |
| `compartment` _string_ | Compartment is the vault compartment OCID.<br />Required for PushSecret |  | Optional: \{\} <br /> |
| `encryptionKey` _string_ | EncryptionKey is the OCID of the encryption key within the vault.<br />Required for PushSecret |  | Optional: \{\} <br /> |
| `principalType` _[OraclePrincipalType](#oracleprincipaltype)_ | The type of principal to use for authentication. If left blank, the Auth struct will<br />determine the principal type. This optional field must be specified if using<br />workload identity. |  | Enum: [ UserPrincipal InstancePrincipal Workload] <br />Optional: \{\} <br /> |
| `auth` _[OracleAuth](#oracleauth)_ | Auth configures how secret-manager authenticates with the Oracle Vault.<br />If empty, use the instance principal, otherwise the user credentials specified in Auth. |  | Optional: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specified the service account<br />that should be used when authenticating with WorkloadIdentity. |  | Optional: \{\} <br /> |


#### OracleSecretRef



OracleSecretRef defines references to secrets containing Oracle credentials.



_Appears in:_
- [OracleAuth](#oracleauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `privatekey` _[SecretKeySelector](#secretkeyselector)_ | PrivateKey is the user's API Signing Key in PEM format, used for authentication. |  |  |
| `fingerprint` _[SecretKeySelector](#secretkeyselector)_ | Fingerprint is the fingerprint of the API private key. |  |  |


#### PassboltAuth



PassboltAuth contains credentials and configuration for authenticating with the Passbolt server.



_Appears in:_
- [PassboltProvider](#passboltprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `passwordSecretRef` _[SecretKeySelector](#secretkeyselector)_ | PasswordSecretRef is a reference to the secret containing the Passbolt password |  |  |
| `privateKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | PrivateKeySecretRef is a reference to the secret containing the Passbolt private key |  |  |


#### PassboltProvider



PassboltProvider defines configuration for the Passbolt provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[PassboltAuth](#passboltauth)_ | Auth defines the information necessary to authenticate against Passbolt Server |  |  |
| `host` _string_ | Host defines the Passbolt Server to connect to |  |  |


#### PasswordDepotAuth



PasswordDepotAuth defines the authentication method for the Password Depot provider.



_Appears in:_
- [PasswordDepotProvider](#passworddepotprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[PasswordDepotSecretRef](#passworddepotsecretref)_ |  |  |  |


#### PasswordDepotProvider



PasswordDepotProvider configures a store to sync secrets with a Password Depot instance.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ | URL configures the Password Depot instance URL. |  |  |
| `database` _string_ | Database to use as source |  |  |
| `auth` _[PasswordDepotAuth](#passworddepotauth)_ | Auth configures how secret-manager authenticates with a Password Depot instance. |  |  |


#### PasswordDepotSecretRef



PasswordDepotSecretRef defines a reference to a secret containing credentials for the Password Depot provider.



_Appears in:_
- [PasswordDepotAuth](#passworddepotauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `credentials` _[SecretKeySelector](#secretkeyselector)_ | Username / Password is used for authentication. |  | Optional: \{\} <br /> |


#### PreviderAuth



PreviderAuth contains a secretRef for credentials.



_Appears in:_
- [PreviderProvider](#previderprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[PreviderAuthSecretRef](#previderauthsecretref)_ |  |  | Optional: \{\} <br /> |


#### PreviderAuthSecretRef



PreviderAuthSecretRef holds secret references for Previder Vault credentials.



_Appears in:_
- [PreviderAuth](#previderauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessToken` _[SecretKeySelector](#secretkeyselector)_ | The AccessToken is used for authentication |  |  |


#### PreviderProvider



PreviderProvider configures a store to sync secrets using the Previder Secret Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[PreviderAuth](#previderauth)_ |  |  |  |
| `baseUri` _string_ |  |  | Optional: \{\} <br /> |


#### Provider

_Underlying type:_ _interface{Capabilities() SecretStoreCapabilities; NewClient(ctx context.Context, store GenericStore, kube sigs.k8s.io/controller-runtime/pkg/client.Client, namespace string) (SecretsClient, error); ValidateStore(store GenericStore) (sigs.k8s.io/controller-runtime/pkg/webhook/admission.Warnings, error)}_

Provider is a common interface for interacting with secret backends.







#### PulumiProvider



PulumiProvider defines configuration for the Pulumi provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the URL of the Pulumi API. | https://api.pulumi.com/api/esc |  |
| `accessToken` _[PulumiProviderSecretRef](#pulumiprovidersecretref)_ | AccessToken is the access tokens to sign in to the Pulumi Cloud Console. |  |  |
| `organization` _string_ | Organization are a space to collaborate on shared projects and stacks.<br />To create a new organization, visit https://app.pulumi.com/ and click "New Organization". |  |  |
| `project` _string_ | Project is the name of the Pulumi ESC project the environment belongs to. |  |  |
| `environment` _string_ | Environment are YAML documents composed of static key-value pairs, programmatic expressions,<br />dynamically retrieved values from supported providers including all major clouds,<br />and other Pulumi ESC environments.<br />To create a new environment, visit https://www.pulumi.com/docs/esc/environments/ for more information. |  |  |


#### PulumiProviderSecretRef



PulumiProviderSecretRef defines a reference to a secret containing credentials for the Pulumi provider.



_Appears in:_
- [PulumiProvider](#pulumiprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef is a reference to a secret containing the Pulumi API token. |  |  |


#### PushSecretData

_Underlying type:_ _interface{GetMetadata() *k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON; GetProperty() string; GetRemoteKey() string; GetSecretKey() string}_

PushSecretData is an interface to allow using v1alpha1.PushSecretData content in Provider registered in v1beta1.







#### PushSecretRemoteRef

_Underlying type:_ _interface{GetProperty() string; GetRemoteKey() string}_

PushSecretRemoteRef is an interface to allow using v1alpha1.PushSecretRemoteRef in Provider registered in v1beta1.







#### ScalewayProvider



ScalewayProvider defines configuration for the Scaleway provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL is the url of the api to use. Defaults to https://api.scaleway.com |  | Optional: \{\} <br /> |
| `region` _string_ | Region where your secrets are located: https://developers.scaleway.com/en/quickstart/#region-and-zone |  |  |
| `projectId` _string_ | ProjectID is the id of your project, which you can find in the console: https://console.scaleway.com/project/settings |  |  |
| `accessKey` _[ScalewayProviderSecretRef](#scalewayprovidersecretref)_ | AccessKey is the non-secret part of the api key. |  |  |
| `secretKey` _[ScalewayProviderSecretRef](#scalewayprovidersecretref)_ | SecretKey is the non-secret part of the api key. |  |  |


#### ScalewayProviderSecretRef



ScalewayProviderSecretRef defines a reference to a secret containing credentials for the Scaleway provider.



_Appears in:_
- [ScalewayProvider](#scalewayprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### SecretServerProvider



SecretServerProvider defines configuration for the Delinea Secret Server provider.
See https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `username` _[SecretServerProviderRef](#secretserverproviderref)_ | Username is the secret server account username. |  | Required: \{\} <br /> |
| `password` _[SecretServerProviderRef](#secretserverproviderref)_ | Password is the secret server account password. |  | Required: \{\} <br /> |
| `serverURL` _string_ | ServerURL<br />URL to your secret server installation |  | Required: \{\} <br /> |


#### SecretServerProviderRef



SecretServerProviderRef defines a reference to a secret containing credentials for the Secret Server provider.



_Appears in:_
- [SecretServerProvider](#secretserverprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value can be specified directly to set a value without using a secret. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef references a key in a secret that will be used as value. |  | Optional: \{\} <br /> |


#### SecretStore



SecretStore represents a secure external location for storing secrets, which can be referenced as part of `storeRef` fields.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `external-secrets.io/v1beta1` | | |
| `kind` _string_ | `SecretStore` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SecretStoreSpec](#secretstorespec)_ |  |  |  |
| `status` _[SecretStoreStatus](#secretstorestatus)_ |  |  |  |


#### SecretStoreCapabilities

_Underlying type:_ _string_

SecretStoreCapabilities defines the possible operations a SecretStore can do.



_Appears in:_
- [SecretStoreStatus](#secretstorestatus)

| Field | Description |
| --- | --- |
| `ReadOnly` | SecretStoreReadOnly indicates that the SecretStore only supports reading secrets.<br /> |
| `WriteOnly` | SecretStoreWriteOnly indicates that the SecretStore only supports writing secrets.<br /> |
| `ReadWrite` | SecretStoreReadWrite indicates that the SecretStore supports both reading and writing secrets.<br /> |


#### SecretStoreConditionType

_Underlying type:_ _string_

SecretStoreConditionType represents the condition type of the SecretStore.



_Appears in:_
- [SecretStoreStatusCondition](#secretstorestatuscondition)

| Field | Description |
| --- | --- |
| `Ready` | SecretStoreReady indicates that the SecretStore has been successfully configured.<br /> |


#### SecretStoreProvider



SecretStoreProvider contains the provider-specific configuration.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [SecretStoreSpec](#secretstorespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `aws` _[AWSProvider](#awsprovider)_ | AWS configures this store to sync secrets using AWS Secret Manager provider |  | Optional: \{\} <br /> |
| `azurekv` _[AzureKVProvider](#azurekvprovider)_ | AzureKV configures this store to sync secrets using Azure Key Vault provider |  | Optional: \{\} <br /> |
| `akeyless` _[AkeylessProvider](#akeylessprovider)_ | Akeyless configures this store to sync secrets using Akeyless Vault provider |  | Optional: \{\} <br /> |
| `bitwardensecretsmanager` _[BitwardenSecretsManagerProvider](#bitwardensecretsmanagerprovider)_ | BitwardenSecretsManager configures this store to sync secrets using BitwardenSecretsManager provider |  | Optional: \{\} <br /> |
| `vault` _[VaultProvider](#vaultprovider)_ | Vault configures this store to sync secrets using the HashiCorp Vault provider. |  | Optional: \{\} <br /> |
| `gcpsm` _[GCPSMProvider](#gcpsmprovider)_ | GCPSM configures this store to sync secrets using Google Cloud Platform Secret Manager provider |  | Optional: \{\} <br /> |
| `oracle` _[OracleProvider](#oracleprovider)_ | Oracle configures this store to sync secrets using Oracle Vault provider |  | Optional: \{\} <br /> |
| `ibm` _[IBMProvider](#ibmprovider)_ | IBM configures this store to sync secrets using IBM Cloud provider |  | Optional: \{\} <br /> |
| `yandexcertificatemanager` _[YandexCertificateManagerProvider](#yandexcertificatemanagerprovider)_ | YandexCertificateManager configures this store to sync secrets using Yandex Certificate Manager provider |  | Optional: \{\} <br /> |
| `yandexlockbox` _[YandexLockboxProvider](#yandexlockboxprovider)_ | YandexLockbox configures this store to sync secrets using Yandex Lockbox provider |  | Optional: \{\} <br /> |
| `github` _[GithubProvider](#githubprovider)_ | Github configures this store to push GitHub Actions secrets using the GitHub API provider. |  | Optional: \{\} <br /> |
| `gitlab` _[GitlabProvider](#gitlabprovider)_ | GitLab configures this store to sync secrets using GitLab Variables provider |  | Optional: \{\} <br /> |
| `alibaba` _[AlibabaProvider](#alibabaprovider)_ | Alibaba configures this store to sync secrets using Alibaba Cloud provider |  | Optional: \{\} <br /> |
| `onepassword` _[OnePasswordProvider](#onepasswordprovider)_ | OnePassword configures this store to sync secrets using the 1Password Cloud provider |  | Optional: \{\} <br /> |
| `webhook` _[WebhookProvider](#webhookprovider)_ | Webhook configures this store to sync secrets using a generic templated webhook |  | Optional: \{\} <br /> |
| `kubernetes` _[KubernetesProvider](#kubernetesprovider)_ | Kubernetes configures this store to sync secrets using a Kubernetes cluster provider |  | Optional: \{\} <br /> |
| `fake` _[FakeProvider](#fakeprovider)_ | Fake configures a store with static key/value pairs |  | Optional: \{\} <br /> |
| `senhasegura` _[SenhaseguraProvider](#senhaseguraprovider)_ | Senhasegura configures this store to sync secrets using senhasegura provider |  | Optional: \{\} <br /> |
| `scaleway` _[ScalewayProvider](#scalewayprovider)_ | Scaleway configures this store to sync secrets using the Scaleway provider. |  | Optional: \{\} <br /> |
| `doppler` _[DopplerProvider](#dopplerprovider)_ | Doppler configures this store to sync secrets using the Doppler provider |  | Optional: \{\} <br /> |
| `previder` _[PreviderProvider](#previderprovider)_ | Previder configures this store to sync secrets using the Previder provider |  | Optional: \{\} <br /> |
| `onboardbase` _[OnboardbaseProvider](#onboardbaseprovider)_ | Onboardbase configures this store to sync secrets using the Onboardbase provider |  | Optional: \{\} <br /> |
| `keepersecurity` _[KeeperSecurityProvider](#keepersecurityprovider)_ | KeeperSecurity configures this store to sync secrets using the KeeperSecurity provider |  | Optional: \{\} <br /> |
| `conjur` _[ConjurProvider](#conjurprovider)_ | Conjur configures this store to sync secrets using conjur provider |  | Optional: \{\} <br /> |
| `delinea` _[DelineaProvider](#delineaprovider)_ | Delinea DevOps Secrets Vault<br />https://docs.delinea.com/online-help/products/devops-secrets-vault/current |  | Optional: \{\} <br /> |
| `secretserver` _[SecretServerProvider](#secretserverprovider)_ | SecretServer configures this store to sync secrets using SecretServer provider<br />https://docs.delinea.com/online-help/secret-server/start.htm |  | Optional: \{\} <br /> |
| `chef` _[ChefProvider](#chefprovider)_ | Chef configures this store to sync secrets with chef server |  | Optional: \{\} <br /> |
| `pulumi` _[PulumiProvider](#pulumiprovider)_ | Pulumi configures this store to sync secrets using the Pulumi provider |  | Optional: \{\} <br /> |
| `fortanix` _[FortanixProvider](#fortanixprovider)_ | Fortanix configures this store to sync secrets using the Fortanix provider |  | Optional: \{\} <br /> |
| `passworddepot` _[PasswordDepotProvider](#passworddepotprovider)_ |  |  | Optional: \{\} <br /> |
| `passbolt` _[PassboltProvider](#passboltprovider)_ |  |  | Optional: \{\} <br /> |
| `device42` _[Device42Provider](#device42provider)_ | Device42 configures this store to sync secrets using the Device42 provider |  | Optional: \{\} <br /> |
| `infisical` _[InfisicalProvider](#infisicalprovider)_ | Infisical configures this store to sync secrets using the Infisical provider |  | Optional: \{\} <br /> |
| `beyondtrust` _[BeyondtrustProvider](#beyondtrustprovider)_ | Beyondtrust configures this store to sync secrets using Password Safe provider. |  | Optional: \{\} <br /> |
| `cloudrusm` _[CloudruSMProvider](#cloudrusmprovider)_ | CloudruSM configures this store to sync secrets using the Cloud.ru Secret Manager provider |  | Optional: \{\} <br /> |


#### SecretStoreRef



SecretStoreRef defines which SecretStore to fetch the ExternalSecret data.



_Appears in:_
- [ExternalSecretSpec](#externalsecretspec)
- [StoreGeneratorSourceRef](#storegeneratorsourceref)
- [StoreSourceRef](#storesourceref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the SecretStore resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `kind` _string_ | Kind of the SecretStore resource (SecretStore or ClusterSecretStore)<br />Defaults to `SecretStore` |  | Enum: [SecretStore ClusterSecretStore] <br />Optional: \{\} <br /> |


#### SecretStoreRetrySettings



SecretStoreRetrySettings defines configuration for retrying failed requests to the provider.



_Appears in:_
- [SecretStoreSpec](#secretstorespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxRetries` _integer_ | MaxRetries is the maximum number of retry attempts. |  |  |
| `retryInterval` _string_ | RetryInterval is the interval between retry attempts. |  |  |


#### SecretStoreSpec



SecretStoreSpec defines the desired state of SecretStore.



_Appears in:_
- [ClusterSecretStore](#clustersecretstore)
- [SecretStore](#secretstore)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `controller` _string_ | Used to select the correct ESO controller (think: ingress.ingressClassName)<br />The ESO controller is instantiated with a specific controller name and filters ES based on this property |  | Optional: \{\} <br /> |
| `provider` _[SecretStoreProvider](#secretstoreprovider)_ | Used to configure the provider. Only one provider may be set |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |
| `retrySettings` _[SecretStoreRetrySettings](#secretstoreretrysettings)_ | Used to configure HTTP retries on failures. |  | Optional: \{\} <br /> |
| `refreshInterval` _integer_ | Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config. |  | Optional: \{\} <br /> |
| `conditions` _[ClusterSecretStoreCondition](#clustersecretstorecondition) array_ | Used to constrain a ClusterSecretStore to specific namespaces. Relevant only to ClusterSecretStore. |  | Optional: \{\} <br /> |


#### SecretStoreStatus



SecretStoreStatus defines the observed state of the SecretStore.



_Appears in:_
- [ClusterSecretStore](#clustersecretstore)
- [SecretStore](#secretstore)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[SecretStoreStatusCondition](#secretstorestatuscondition) array_ |  |  | Optional: \{\} <br /> |
| `capabilities` _[SecretStoreCapabilities](#secretstorecapabilities)_ |  |  | Optional: \{\} <br /> |


#### SecretStoreStatusCondition



SecretStoreStatusCondition defines the observed condition of the SecretStore.



_Appears in:_
- [SecretStoreStatus](#secretstorestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[SecretStoreConditionType](#secretstoreconditiontype)_ |  |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `reason` _string_ |  |  | Optional: \{\} <br /> |
| `message` _string_ |  |  | Optional: \{\} <br /> |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ |  |  | Optional: \{\} <br /> |


#### SecretsClient

_Underlying type:_ _interface{Close(ctx context.Context) error; DeleteSecret(ctx context.Context, remoteRef PushSecretRemoteRef) error; GetAllSecrets(ctx context.Context, ref ExternalSecretFind) (map[string][]byte, error); GetSecret(ctx context.Context, ref ExternalSecretDataRemoteRef) ([]byte, error); GetSecretMap(ctx context.Context, ref ExternalSecretDataRemoteRef) (map[string][]byte, error); PushSecret(ctx context.Context, secret *k8s.io/api/core/v1.Secret, data PushSecretData) error; SecretExists(ctx context.Context, remoteRef PushSecretRemoteRef) (bool, error); Validate() (ValidationResult, error)}_

SecretsClient provides access to secrets.







#### SecretsManager



SecretsManager defines how the provider behaves when interacting with AWS
SecretsManager. Some of these settings are only applicable to controlling how
secrets are deleted, and hence only apply to PushSecret (and only when
deletionPolicy is set to Delete).



_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `forceDeleteWithoutRecovery` _boolean_ | Specifies whether to delete the secret without any recovery window. You<br />can't use both this parameter and RecoveryWindowInDays in the same call.<br />If you don't use either, then by default Secrets Manager uses a 30 day<br />recovery window.<br />see: https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-ForceDeleteWithoutRecovery |  | Optional: \{\} <br /> |
| `recoveryWindowInDays` _integer_ | The number of days from 7 to 30 that Secrets Manager waits before<br />permanently deleting the secret. You can't use both this parameter and<br />ForceDeleteWithoutRecovery in the same call. If you don't use either,<br />then by default Secrets Manager uses a 30 day recovery window.<br />see: https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-RecoveryWindowInDays |  | Optional: \{\} <br /> |


#### SenhaseguraAuth



SenhaseguraAuth tells the controller how to do auth in senhasegura.



_Appears in:_
- [SenhaseguraProvider](#senhaseguraprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _string_ |  |  |  |
| `clientSecretSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### SenhaseguraModuleType

_Underlying type:_ _string_

SenhaseguraModuleType enum defines senhasegura target module to fetch secrets
+kubebuilder:validation:Enum=DSM



_Appears in:_
- [SenhaseguraProvider](#senhaseguraprovider)

| Field | Description |
| --- | --- |
| `DSM` | 		SenhaseguraModuleDSM is the senhasegura DevOps Secrets Management module<br />		see: https://senhasegura.com/devops<br /> |


#### SenhaseguraProvider



SenhaseguraProvider setup a store to sync secrets with senhasegura.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL of senhasegura |  |  |
| `module` _[SenhaseguraModuleType](#senhaseguramoduletype)_ | Module defines which senhasegura module should be used to get secrets |  |  |
| `auth` _[SenhaseguraAuth](#senhaseguraauth)_ | Auth defines parameters to authenticate in senhasegura |  |  |
| `ignoreSslCertificate` _boolean_ | IgnoreSslCertificate defines if SSL certificate must be ignored | false |  |


#### StoreGeneratorSourceRef



StoreGeneratorSourceRef allows you to override the source
from which the secret will be pulled from.
You can define at maximum one property.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ExternalSecretDataFromRemoteRef](#externalsecretdatafromremoteref)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storeRef` _[SecretStoreRef](#secretstoreref)_ |  |  | Optional: \{\} <br /> |
| `generatorRef` _[GeneratorRef](#generatorref)_ | GeneratorRef points to a generator custom resource. |  | Optional: \{\} <br /> |


#### StoreSourceRef



StoreSourceRef allows you to override the SecretStore source
from which the secret will be pulled from.
You can define at maximum one property.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ExternalSecretData](#externalsecretdata)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storeRef` _[SecretStoreRef](#secretstoreref)_ |  |  | Optional: \{\} <br /> |
| `generatorRef` _[GeneratorRef](#generatorref)_ | GeneratorRef points to a generator custom resource.<br />Deprecated: The generatorRef is not implemented in .data[].<br />this will be removed with v1. |  |  |


#### Tag



Tag defines a tag key and value for AWS resources.



_Appears in:_
- [AWSProvider](#awsprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  |  |
| `value` _string_ |  |  |  |


#### TemplateEngineVersion

_Underlying type:_ _string_

TemplateEngineVersion defines the version of the template engine to use.

_Validation:_
- Enum: [v2]

_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description |
| --- | --- |
| `v2` | TemplateEngineV2 specifies the v2 template engine version.<br /> |


#### TemplateFrom



TemplateFrom defines a source for template data.



_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `configMap` _[TemplateRef](#templateref)_ |  |  |  |
| `secret` _[TemplateRef](#templateref)_ |  |  |  |
| `target` _[TemplateTarget](#templatetarget)_ |  | Data | Enum: [Data Annotations Labels] <br />Optional: \{\} <br /> |
| `literal` _string_ |  |  | Optional: \{\} <br /> |


#### TemplateMergePolicy

_Underlying type:_ _string_

TemplateMergePolicy defines how template values should be merged when generating a secret.

_Validation:_
- Enum: [Replace Merge]

_Appears in:_
- [ExternalSecretTemplate](#externalsecrettemplate)

| Field | Description |
| --- | --- |
| `Replace` | MergePolicyReplace replaces the entire template content during merge operations.<br /> |
| `Merge` | MergePolicyMerge merges the template content with existing values.<br /> |


#### TemplateRef



TemplateRef defines a reference to a template source in a ConfigMap or Secret.



_Appears in:_
- [TemplateFrom](#templatefrom)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The name of the ConfigMap/Secret resource |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `items` _[TemplateRefItem](#templaterefitem) array_ | A list of keys in the ConfigMap/Secret to use as templates for Secret data |  |  |


#### TemplateRefItem

_Underlying type:_ _[struct{Key string "json:\"key\""; TemplateAs TemplateScope "json:\"templateAs,omitempty\""}](#struct{key-string-"json:\"key\"";-templateas-templatescope-"json:\"templateas,omitempty\""})_

TemplateRefItem defines which key in the referenced ConfigMap or Secret to use as a template.



_Appears in:_
- [TemplateRef](#templateref)





#### TemplateTarget

_Underlying type:_ _string_

TemplateTarget defines the target field where the template result will be stored.

_Validation:_
- Enum: [Data Annotations Labels]

_Appears in:_
- [TemplateFrom](#templatefrom)

| Field | Description |
| --- | --- |
| `Data` | TemplateTargetData stores template results in the data field of the secret.<br /> |
| `Annotations` | TemplateTargetAnnotations stores template results in the annotations field of the secret.<br /> |
| `Labels` | TemplateTargetLabels stores template results in the labels field of the secret.<br /> |


#### TokenAuth



TokenAuth defines token-based authentication for the Kubernetes provider.



_Appears in:_
- [KubernetesAuth](#kubernetesauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `bearerToken` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### UniversalAuthCredentials



UniversalAuthCredentials defines the credentials for Infisical Universal Auth.



_Appears in:_
- [InfisicalAuth](#infisicalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |
| `clientSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  | Required: \{\} <br /> |




#### VaultAppRole



VaultAppRole authenticates with Vault using the App Role auth mechanism,
with the role and secret stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the App Role authentication backend is mounted<br />in Vault, e.g: "approle" | approle |  |
| `roleId` _string_ | RoleID configured in the App Role authentication backend when setting<br />up the authentication backend in Vault. |  | Optional: \{\} <br /> |
| `roleRef` _[SecretKeySelector](#secretkeyselector)_ | Reference to a key in a Secret that contains the App Role ID used<br />to authenticate with Vault.<br />The `key` field must be specified and denotes which entry within the Secret<br />resource is used as the app role id. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Reference to a key in a Secret that contains the App Role secret used<br />to authenticate with Vault.<br />The `key` field must be specified and denotes which entry within the Secret<br />resource is used as the app role secret. |  |  |


#### VaultAuth



VaultAuth is the configuration used to authenticate with a Vault server.
Only one of `tokenSecretRef`, `appRole`,  `kubernetes`, `ldap`, `userPass`, `jwt` or `cert`
can be specified. A namespace to authenticate against can optionally be specified.



_Appears in:_
- [VaultProvider](#vaultprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespace` _string_ | Name of the vault namespace to authenticate to. This can be different than the namespace your secret is in.<br />Namespaces is a set of features within Vault Enterprise that allows<br />Vault environments to support Secure Multi-tenancy. e.g: "ns1".<br />More about namespaces can be found here https://www.vaultproject.io/docs/enterprise/namespaces<br />This will default to Vault.Namespace field if set, or empty otherwise |  | Optional: \{\} <br /> |
| `tokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | TokenSecretRef authenticates with Vault by presenting a token. |  | Optional: \{\} <br /> |
| `appRole` _[VaultAppRole](#vaultapprole)_ | AppRole authenticates with Vault using the App Role auth mechanism,<br />with the role and secret stored in a Kubernetes Secret resource. |  | Optional: \{\} <br /> |
| `kubernetes` _[VaultKubernetesAuth](#vaultkubernetesauth)_ | Kubernetes authenticates with Vault by passing the ServiceAccount<br />token stored in the named Secret resource to the Vault server. |  | Optional: \{\} <br /> |
| `ldap` _[VaultLdapAuth](#vaultldapauth)_ | Ldap authenticates with Vault by passing username/password pair using<br />the LDAP authentication method |  | Optional: \{\} <br /> |
| `jwt` _[VaultJwtAuth](#vaultjwtauth)_ | Jwt authenticates with Vault by passing role and JWT token using the<br />JWT/OIDC authentication method |  | Optional: \{\} <br /> |
| `cert` _[VaultCertAuth](#vaultcertauth)_ | Cert authenticates with TLS Certificates by passing client certificate, private key and ca certificate<br />Cert authentication method |  | Optional: \{\} <br /> |
| `iam` _[VaultIamAuth](#vaultiamauth)_ | Iam authenticates with vault by passing a special AWS request signed with AWS IAM credentials<br />AWS IAM authentication method |  | Optional: \{\} <br /> |
| `userPass` _[VaultUserPassAuth](#vaultuserpassauth)_ | UserPass authenticates with Vault by passing username/password pair |  | Optional: \{\} <br /> |




#### VaultAwsAuthSecretRef

_Underlying type:_ _[struct{AccessKeyID github.com/external-secrets/external-secrets/apis/meta/v1.SecretKeySelector "json:\"accessKeyIDSecretRef,omitempty\""; SecretAccessKey github.com/external-secrets/external-secrets/apis/meta/v1.SecretKeySelector "json:\"secretAccessKeySecretRef,omitempty\""; SessionToken *github.com/external-secrets/external-secrets/apis/meta/v1.SecretKeySelector "json:\"sessionTokenSecretRef,omitempty\""}](#struct{accesskeyid-githubcomexternal-secretsexternal-secretsapismetav1secretkeyselector-"json:\"accesskeyidsecretref,omitempty\"";-secretaccesskey-githubcomexternal-secretsexternal-secretsapismetav1secretkeyselector-"json:\"secretaccesskeysecretref,omitempty\"";-sessiontoken-*githubcomexternal-secretsexternal-secretsapismetav1secretkeyselector-"json:\"sessiontokensecretref,omitempty\""})_

VaultAwsAuthSecretRef holds secret references for AWS credentials
both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.



_Appears in:_
- [VaultAwsAuth](#vaultawsauth)
- [VaultIamAuth](#vaultiamauth)



#### VaultAwsJWTAuth

_Underlying type:_ _[struct{ServiceAccountRef *github.com/external-secrets/external-secrets/apis/meta/v1.ServiceAccountSelector "json:\"serviceAccountRef,omitempty\""}](#struct{serviceaccountref-*githubcomexternal-secretsexternal-secretsapismetav1serviceaccountselector-"json:\"serviceaccountref,omitempty\""})_

VaultAwsJWTAuth Authenticate against AWS using service account tokens.



_Appears in:_
- [VaultAwsAuth](#vaultawsauth)
- [VaultIamAuth](#vaultiamauth)



#### VaultCertAuth



VaultCertAuth authenticates with Vault using the JWT/OIDC authentication
method, with the role name and token stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientCert` _[SecretKeySelector](#secretkeyselector)_ | ClientCert is a certificate to authenticate using the Cert Vault<br />authentication method |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef to a key in a Secret resource containing client private key to<br />authenticate with Vault using the Cert authentication method |  | Optional: \{\} <br /> |


#### VaultClientTLS



VaultClientTLS is the configuration used for client side related TLS communication,
when the Vault server requires mutual authentication.



_Appears in:_
- [VaultProvider](#vaultprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certSecretRef` _[SecretKeySelector](#secretkeyselector)_ | CertSecretRef is a certificate added to the transport layer<br />when communicating with the Vault server.<br />If no key for the Secret is specified, external-secret will default to 'tls.crt'. |  | Optional: \{\} <br /> |
| `keySecretRef` _[SecretKeySelector](#secretkeyselector)_ | KeySecretRef to a key in a Secret resource containing client private key<br />added to the transport layer when communicating with the Vault server.<br />If no key for the Secret is specified, external-secret will default to 'tls.key'. |  | Optional: \{\} <br /> |


#### VaultIamAuth



VaultIamAuth authenticates with Vault using the Vault's AWS IAM authentication method. Refer: https://developer.hashicorp.com/vault/docs/auth/aws



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the AWS auth method is enabled in Vault, e.g: "aws" |  | Optional: \{\} <br /> |
| `region` _string_ | AWS region |  | Optional: \{\} <br /> |
| `role` _string_ | This is the AWS role to be assumed before talking to vault |  | Optional: \{\} <br /> |
| `vaultRole` _string_ | Vault Role. In vault, a role describes an identity with a set of permissions, groups, or policies you want to attach a user of the secrets engine |  |  |
| `externalID` _string_ | AWS External ID set on assumed IAM roles |  |  |
| `vaultAwsIamServerID` _string_ | X-Vault-AWS-IAM-Server-ID is an additional header used by Vault IAM auth method to mitigate against different types of replay attacks. More details here: https://developer.hashicorp.com/vault/docs/auth/aws |  | Optional: \{\} <br /> |
| `secretRef` _[VaultAwsAuthSecretRef](#vaultawsauthsecretref)_ | Specify credentials in a Secret object |  | Optional: \{\} <br /> |
| `jwt` _[VaultAwsJWTAuth](#vaultawsjwtauth)_ | Specify a service account with IRSA enabled |  | Optional: \{\} <br /> |


#### VaultJwtAuth



VaultJwtAuth authenticates with Vault using the JWT/OIDC authentication
method, with the role name and a token stored in a Kubernetes Secret resource or
a Kubernetes service account token retrieved via `TokenRequest`.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the JWT authentication backend is mounted<br />in Vault, e.g: "jwt" | jwt |  |
| `role` _string_ | Role is a JWT role to authenticate using the JWT/OIDC Vault<br />authentication method |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional SecretRef that refers to a key in a Secret resource containing JWT token to<br />authenticate with Vault using the JWT/OIDC authentication method. |  | Optional: \{\} <br /> |
| `kubernetesServiceAccountToken` _[VaultKubernetesServiceAccountTokenAuth](#vaultkubernetesserviceaccounttokenauth)_ | Optional ServiceAccountToken specifies the Kubernetes service account for which to request<br />a token for with the `TokenRequest` API. |  | Optional: \{\} <br /> |


#### VaultKVStoreVersion

_Underlying type:_ _string_

VaultKVStoreVersion defines the version of the KV store in Vault.



_Appears in:_
- [VaultProvider](#vaultprovider)

| Field | Description |
| --- | --- |
| `v1` | VaultKVStoreV1 represents version 1 of the Vault KV store.<br /> |
| `v2` | VaultKVStoreV2 represents version 2 of the Vault KV store.<br /> |


#### VaultKubernetesAuth



VaultKubernetesAuth authenticates against Vault using a Kubernetes ServiceAccount token stored in a Secret.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mountPath` _string_ | Path where the Kubernetes authentication backend is mounted in Vault, e.g:<br />"kubernetes" | kubernetes |  |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Optional service account field containing the name of a kubernetes ServiceAccount.<br />If the service account is specified, the service account secret token JWT will be used<br />for authenticating with Vault. If the service account selector is not supplied,<br />the secretRef will be used instead. |  | Optional: \{\} <br /> |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Optional secret field containing a Kubernetes ServiceAccount JWT used<br />for authenticating with Vault. If a name is specified without a key,<br />`token` is the default. If one is not specified, the one bound to<br />the controller will be used. |  | Optional: \{\} <br /> |
| `role` _string_ | A required field containing the Vault Role to assume. A Role binds a<br />Kubernetes ServiceAccount with a set of Vault policies. |  |  |


#### VaultKubernetesServiceAccountTokenAuth

_Underlying type:_ _[struct{ServiceAccountRef github.com/external-secrets/external-secrets/apis/meta/v1.ServiceAccountSelector "json:\"serviceAccountRef\""; Audiences *[]string "json:\"audiences,omitempty\""; ExpirationSeconds *int64 "json:\"expirationSeconds,omitempty\""}](#struct{serviceaccountref-githubcomexternal-secretsexternal-secretsapismetav1serviceaccountselector-"json:\"serviceaccountref\"";-audiences-*[]string-"json:\"audiences,omitempty\"";-expirationseconds-*int64-"json:\"expirationseconds,omitempty\""})_

VaultKubernetesServiceAccountTokenAuth authenticates with Vault using a temporary
Kubernetes service account token retrieved by the `TokenRequest` API.



_Appears in:_
- [VaultJwtAuth](#vaultjwtauth)



#### VaultLdapAuth



VaultLdapAuth authenticates with Vault using the LDAP authentication method,
with the username and password stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the LDAP authentication backend is mounted<br />in Vault, e.g: "ldap" | ldap |  |
| `username` _string_ | Username is an LDAP username used to authenticate using the LDAP Vault<br />authentication method |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef to a key in a Secret resource containing password for the LDAP<br />user used to authenticate with Vault using the LDAP authentication<br />method |  | Optional: \{\} <br /> |


#### VaultProvider



VaultProvider configures a store to sync secrets using a HashiCorp Vault KV backend.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[VaultAuth](#vaultauth)_ | Auth configures how secret-manager authenticates with the Vault server. |  |  |
| `server` _string_ | Server is the connection address for the Vault server, e.g: "https://vault.example.com:8200". |  |  |
| `path` _string_ | Path is the mount path of the Vault KV backend endpoint, e.g:<br />"secret". The v2 KV secret engine version specific "/data" path suffix<br />for fetching secrets from Vault is optional and will be appended<br />if not present in specified path. |  | Optional: \{\} <br /> |
| `version` _[VaultKVStoreVersion](#vaultkvstoreversion)_ | Version is the Vault KV secret engine version. This can be either "v1" or<br />"v2". Version defaults to "v2". | v2 | Enum: [v1 v2] <br />Optional: \{\} <br /> |
| `namespace` _string_ | Name of the vault namespace. Namespaces is a set of features within Vault Enterprise that allows<br />Vault environments to support Secure Multi-tenancy. e.g: "ns1".<br />More about namespaces can be found here https://www.vaultproject.io/docs/enterprise/namespaces |  | Optional: \{\} <br /> |
| `caBundle` _integer array_ | PEM encoded CA bundle used to validate Vault server certificate. Only used<br />if the Server URL is using HTTPS protocol. This parameter is ignored for<br />plain HTTP protocol connection. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `tls` _[VaultClientTLS](#vaultclienttls)_ | The configuration used for client side related TLS communication, when the Vault server<br />requires mutual authentication. Only used if the Server URL is using HTTPS protocol.<br />This parameter is ignored for plain HTTP protocol connection.<br />It's worth noting this configuration is different from the "TLS certificates auth method",<br />which is available under the `auth.cert` section. |  | Optional: \{\} <br /> |
| `caProvider` _[CAProvider](#caprovider)_ | The provider for the CA bundle to use to validate Vault server certificate. |  | Optional: \{\} <br /> |
| `readYourWrites` _boolean_ | ReadYourWrites ensures isolated read-after-write semantics by<br />providing discovered cluster replication states in each request.<br />More information about eventual consistency in Vault can be found here<br />https://www.vaultproject.io/docs/enterprise/consistency |  | Optional: \{\} <br /> |
| `forwardInconsistent` _boolean_ | ForwardInconsistent tells Vault to forward read-after-write requests to the Vault<br />leader instead of simply retrying within a loop. This can increase performance if<br />the option is enabled serverside.<br />https://www.vaultproject.io/docs/configuration/replication#allow_forwarding_via_header |  | Optional: \{\} <br /> |
| `headers` _object (keys:string, values:string)_ | Headers to be added in Vault request |  | Optional: \{\} <br /> |


#### VaultUserPassAuth



VaultUserPassAuth authenticates with Vault using UserPass authentication method,
with the username and password stored in a Kubernetes Secret resource.



_Appears in:_
- [VaultAuth](#vaultauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `path` _string_ | Path where the UserPassword authentication backend is mounted<br />in Vault, e.g: "userpass" | userpass |  |
| `username` _string_ | Username is a username used to authenticate using the UserPass Vault<br />authentication method |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | SecretRef to a key in a Secret resource containing password for the<br />user used to authenticate with Vault using the UserPass authentication<br />method |  | Optional: \{\} <br /> |


#### WebhookCAProvider



WebhookCAProvider defines a location to fetch the certificate for the webhook provider.



_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[WebhookCAProviderType](#webhookcaprovidertype)_ | The type of provider to use such as "Secret", or "ConfigMap". |  | Enum: [Secret ConfigMap] <br /> |
| `name` _string_ | The name of the object located at the provider type. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `key` _string_ | The key where the CA certificate can be found in the Secret or ConfigMap. |  | MaxLength: 253 <br />MinLength: 1 <br />Optional: \{\} <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |
| `namespace` _string_ | The namespace the Provider type is in. |  | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


#### WebhookCAProviderType

_Underlying type:_ _string_

WebhookCAProviderType defines the type of provider to use for CA certificates with Webhook providers.



_Appears in:_
- [WebhookCAProvider](#webhookcaprovider)

| Field | Description |
| --- | --- |
| `Secret` | WebhookCAProviderTypeSecret indicates that the CA certificate is stored in a Secret.<br /> |
| `ConfigMap` | WebhookCAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap.<br /> |


#### WebhookProvider



WebhookProvider configures a store to sync secrets from simple web APIs.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `method` _string_ | Webhook Method |  |  |
| `url` _string_ | Webhook url to call |  |  |
| `headers` _object (keys:string, values:string)_ | Headers |  | Optional: \{\} <br /> |
| `auth` _[AuthorizationProtocol](#authorizationprotocol)_ | Auth specifies a authorization protocol. Only one protocol may be set. |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `body` _string_ | Body |  | Optional: \{\} <br /> |
| `timeout` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | Timeout |  | Optional: \{\} <br /> |
| `result` _[WebhookResult](#webhookresult)_ | Result formatting |  |  |
| `secrets` _[WebhookSecret](#webhooksecret) array_ | Secrets to fill in templates<br />These secrets will be passed to the templating function as key value pairs under the given name |  | Optional: \{\} <br /> |
| `caBundle` _integer array_ | PEM encoded CA bundle used to validate webhook server certificate. Only used<br />if the Server URL is using HTTPS protocol. This parameter is ignored for<br />plain HTTP protocol connection. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[WebhookCAProvider](#webhookcaprovider)_ | The provider for the CA bundle to use to validate webhook server certificate. |  | Optional: \{\} <br /> |


#### WebhookResult



WebhookResult defines how to extract and format the result from the webhook response.



_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `jsonPath` _string_ | Json path of return value |  | Optional: \{\} <br /> |


#### WebhookSecret



WebhookSecret defines a secret to be used in webhook templates.



_Appears in:_
- [WebhookProvider](#webhookprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of this secret in templates |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Secret ref to fill in credentials |  |  |


#### YandexCertificateManagerAuth



YandexCertificateManagerAuth defines authentication configuration for the Yandex Certificate Manager provider.



_Appears in:_
- [YandexCertificateManagerProvider](#yandexcertificatemanagerprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authorizedKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The authorized key used for authentication |  | Optional: \{\} <br /> |


#### YandexCertificateManagerCAProvider



YandexCertificateManagerCAProvider defines CA certificate configuration for Yandex Certificate Manager.



_Appears in:_
- [YandexCertificateManagerProvider](#yandexcertificatemanagerprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### YandexCertificateManagerProvider



YandexCertificateManagerProvider configures a store to sync secrets using the Yandex Certificate Manager provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiEndpoint` _string_ | Yandex.Cloud API endpoint (e.g. 'api.cloud.yandex.net:443') |  | Optional: \{\} <br /> |
| `auth` _[YandexCertificateManagerAuth](#yandexcertificatemanagerauth)_ | Auth defines the information necessary to authenticate against Yandex Certificate Manager |  |  |
| `caProvider` _[YandexCertificateManagerCAProvider](#yandexcertificatemanagercaprovider)_ | The provider for the CA bundle to use to validate Yandex.Cloud server certificate. |  | Optional: \{\} <br /> |


#### YandexLockboxAuth



YandexLockboxAuth defines authentication configuration for the Yandex Lockbox provider.



_Appears in:_
- [YandexLockboxProvider](#yandexlockboxprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authorizedKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The authorized key used for authentication |  | Optional: \{\} <br /> |


#### YandexLockboxCAProvider



YandexLockboxCAProvider defines CA certificate configuration for Yandex Lockbox.



_Appears in:_
- [YandexLockboxProvider](#yandexlockboxprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certSecretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### YandexLockboxProvider



YandexLockboxProvider configures a store to sync secrets using the Yandex Lockbox provider.



_Appears in:_
- [SecretStoreProvider](#secretstoreprovider)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiEndpoint` _string_ | Yandex.Cloud API endpoint (e.g. 'api.cloud.yandex.net:443') |  | Optional: \{\} <br /> |
| `auth` _[YandexLockboxAuth](#yandexlockboxauth)_ | Auth defines the information necessary to authenticate against Yandex Lockbox |  |  |
| `caProvider` _[YandexLockboxCAProvider](#yandexlockboxcaprovider)_ | The provider for the CA bundle to use to validate Yandex.Cloud server certificate. |  | Optional: \{\} <br /> |



## generators.external-secrets.io/v1alpha1

Package v1alpha1 contains resources for generators

### Resource Types
- [ACRAccessToken](#acraccesstoken)
- [BeyondtrustWorkloadCredentialsDynamicSecret](#beyondtrustworkloadcredentialsdynamicsecret)
- [CloudsmithAccessToken](#cloudsmithaccesstoken)
- [ClusterGenerator](#clustergenerator)
- [ECRAuthorizationToken](#ecrauthorizationtoken)
- [Fake](#fake)
- [GCRAccessToken](#gcraccesstoken)
- [Generator](#generator)
- [GeneratorState](#generatorstate)
- [GithubAccessToken](#githubaccesstoken)
- [GitlabDeployToken](#gitlabdeploytoken)
- [Grafana](#grafana)
- [MFA](#mfa)
- [Password](#password)
- [QuayAccessToken](#quayaccesstoken)
- [SSHKey](#sshkey)
- [STSSessionToken](#stssessiontoken)
- [StatefulResource](#statefulresource)
- [UUID](#uuid)
- [VaultDynamicSecret](#vaultdynamicsecret)
- [Webhook](#webhook)



#### ACRAccessToken



ACRAccessToken returns an Azure Container Registry token
that can be used for pushing/pulling images.
Note: by default it will return an ACR Refresh Token with full access
(depending on the identity).
This can be scoped down to the repository level using .spec.scope.
In case scope is defined it will return an ACR Access Token.

See docs: https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `ACRAccessToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ACRAccessTokenSpec](#acraccesstokenspec)_ |  |  |  |


#### ACRAccessTokenSpec



ACRAccessTokenSpec defines how to generate the access token
e.g. how to authenticate and which registry to use.
see: https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#overview



_Appears in:_
- [ACRAccessToken](#acraccesstoken)
- [GeneratorSpec](#generatorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[ACRAuth](#acrauth)_ |  |  |  |
| `tenantId` _string_ | TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type. |  |  |
| `registry` _string_ | the domain name of the ACR registry<br />e.g. foobarexample.azurecr.io |  |  |
| `scope` _string_ | Define the scope for the access token, e.g. pull/push access for a repository.<br />if not provided it will return a refresh token that has full scope.<br />Note: you need to pin it down to the repository level, there is no wildcard available.<br />examples:<br />repository:my-repository:pull,push<br />repository:my-repository:pull<br />see docs for details: https://docs.docker.com/registry/spec/auth/scope/ |  | Optional: \{\} <br /> |
| `environmentType` _[AzureEnvironmentType](#azureenvironmenttype)_ | EnvironmentType specifies the Azure cloud environment endpoints to use for<br />connecting and authenticating with Azure. By default, it points to the public cloud AAD endpoint.<br />The following endpoints are available, also see here: https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152<br />PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud | PublicCloud | Enum: [PublicCloud USGovernmentCloud ChinaCloud GermanCloud AzureStackCloud] <br /> |


#### ACRAuth



ACRAuth defines the authentication methods for Azure Container Registry.



_Appears in:_
- [ACRAccessTokenSpec](#acraccesstokenspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `servicePrincipal` _[AzureACRServicePrincipalAuth](#azureacrserviceprincipalauth)_ | ServicePrincipal uses Azure Service Principal credentials to authenticate with Azure. |  | Optional: \{\} <br /> |
| `managedIdentity` _[AzureACRManagedIdentityAuth](#azureacrmanagedidentityauth)_ | ManagedIdentity uses Azure Managed Identity to authenticate with Azure. |  | Optional: \{\} <br /> |
| `workloadIdentity` _[AzureACRWorkloadIdentityAuth](#azureacrworkloadidentityauth)_ | WorkloadIdentity uses Azure Workload Identity to authenticate with Azure. |  | Optional: \{\} <br /> |


#### AWSAuth



AWSAuth tells the controller how to do authentication with aws.
Only one of secretRef or jwt can be specified.
if none is specified the controller will load credentials using the aws sdk defaults.



_Appears in:_
- [ECRAuthorizationTokenSpec](#ecrauthorizationtokenspec)
- [STSSessionTokenSpec](#stssessiontokenspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AWSAuthSecretRef](#awsauthsecretref)_ |  |  | Optional: \{\} <br /> |
| `jwt` _[AWSJWTAuth](#awsjwtauth)_ |  |  | Optional: \{\} <br /> |


#### AWSAuthSecretRef



AWSAuthSecretRef holds secret references for AWS credentials
both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.



_Appears in:_
- [AWSAuth](#awsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `accessKeyIDSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The AccessKeyID is used for authentication |  |  |
| `secretAccessKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  |  |
| `sessionTokenSecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SessionToken used for authentication<br />This must be defined if AccessKeyID and SecretAccessKey are temporary credentials<br />see: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html |  |  |


#### AWSJWTAuth



AWSJWTAuth provides configuration to authenticate against AWS using service account tokens.



_Appears in:_
- [AWSAuth](#awsauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ |  |  |  |


#### AuthorizationProtocol



AuthorizationProtocol contains the protocol-specific configuration

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [WebhookSpec](#webhookspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ntlm` _[NTLMProtocol](#ntlmprotocol)_ | NTLMProtocol configures the store to use NTLM for auth |  | Optional: \{\} <br /> |


#### AzureACRManagedIdentityAuth



AzureACRManagedIdentityAuth defines the configuration for using Azure Managed Identity authentication.



_Appears in:_
- [ACRAuth](#acrauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityId` _string_ | If multiple Managed Identity is assigned to the pod, you can select the one to be used |  |  |


#### AzureACRServicePrincipalAuth



AzureACRServicePrincipalAuth defines the configuration for using Azure Service Principal authentication.



_Appears in:_
- [ACRAuth](#acrauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[AzureACRServicePrincipalAuthSecretRef](#azureacrserviceprincipalauthsecretref)_ |  |  |  |


#### AzureACRServicePrincipalAuthSecretRef



AzureACRServicePrincipalAuthSecretRef defines the secret references for Azure Service Principal authentication.
It uses static credentials stored in a Kind=Secret.



_Appears in:_
- [AzureACRServicePrincipalAuth](#azureacrserviceprincipalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientId` _[SecretKeySelector](#secretkeyselector)_ | The Azure clientId of the service principle used for authentication. |  |  |
| `clientSecret` _[SecretKeySelector](#secretkeyselector)_ | The Azure ClientSecret of the service principle used for authentication. |  |  |


#### AzureACRWorkloadIdentityAuth



AzureACRWorkloadIdentityAuth defines the configuration for using Azure Workload Identity authentication.



_Appears in:_
- [ACRAuth](#acrauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | ServiceAccountRef specified the service account<br />that should be used when authenticating with WorkloadIdentity. |  | Optional: \{\} <br /> |


#### BeyondtrustWorkloadCredentialsDynamicSecret



BeyondtrustWorkloadCredentialsDynamicSecret represents a generator that requests dynamic credentials from BeyondTrust Workload Credentials.
This generator calls the BeyondTrust Workload Credentials API to generate fresh, temporary credentials
(such as AWS STS credentials) each time an ExternalSecret is refreshed.
Dynamic secret definitions must be created in BeyondTrust Workload Credentials before they can be referenced.
For complete documentation, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `BeyondtrustWorkloadCredentialsDynamicSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[BeyondtrustWorkloadCredentialsDynamicSecretSpec](#beyondtrustworkloadcredentialsdynamicsecretspec)_ |  |  |  |


#### BeyondtrustWorkloadCredentialsDynamicSecretSpec



BeyondtrustWorkloadCredentialsDynamicSecretSpec defines the desired spec for BeyondtrustWorkloadCredentials dynamic generator.
This generator enables obtaining temporary, short-lived credentials from BeyondTrust Workload Credentials.
For more information, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api



_Appears in:_
- [BeyondtrustWorkloadCredentialsDynamicSecret](#beyondtrustworkloadcredentialsdynamicsecret)
- [GeneratorSpec](#generatorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `controller` _string_ | Controller selects the controller that should handle this generator.<br />Leave empty to use the default controller. |  | Optional: \{\} <br /> |
| `provider` _[BeyondtrustWorkloadCredentialsProvider](#beyondtrustworkloadcredentialsprovider)_ | Provider contains the BeyondtrustWorkloadCredentials provider configuration including authentication,<br />server connection details, and the folder path to the dynamic secret definition.<br />The folderPath should point to a dynamic secret definition that has been created in<br />BeyondTrust Workload Credentials (e.g., "production/aws-temp").<br />For setup details, see: https://docs.beyondtrust.com/bt-docs/docs/secrets-api |  | Required: \{\} <br /> |
| `retrySettings` _[SecretStoreRetrySettings](#secretstoreretrysettings)_ | RetrySettings configures exponential backoff for failed API requests.<br />If not specified, uses the default retry settings. |  | Optional: \{\} <br /> |


#### CloudsmithAccessToken



CloudsmithAccessToken generates Cloudsmith access token using OIDC authentication





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `CloudsmithAccessToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[CloudsmithAccessTokenSpec](#cloudsmithaccesstokenspec)_ |  |  |  |


#### CloudsmithAccessTokenSpec



CloudsmithAccessTokenSpec defines the configuration for generating a Cloudsmith access token using OIDC authentication.



_Appears in:_
- [CloudsmithAccessToken](#cloudsmithaccesstoken)
- [GeneratorSpec](#generatorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiUrl` _string_ | APIURL configures the Cloudsmith API URL. Defaults to https://api.cloudsmith.io. |  | Optional: \{\} <br /> |
| `orgSlug` _string_ | OrgSlug is the organization slug in Cloudsmith |  | Required: \{\} <br /> |
| `serviceSlug` _string_ | ServiceSlug is the service slug in Cloudsmith for OIDC authentication |  | Required: \{\} <br /> |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Name of the service account you are federating with |  | Required: \{\} <br /> |


#### ClusterGenerator



ClusterGenerator represents a cluster-wide generator which can be referenced as part of `generatorRef` fields.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `ClusterGenerator` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ClusterGeneratorSpec](#clustergeneratorspec)_ |  |  |  |


#### ClusterGeneratorSpec



ClusterGeneratorSpec defines the desired state of a ClusterGenerator.



_Appears in:_
- [ClusterGenerator](#clustergenerator)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _[GeneratorKind](#generatorkind)_ | Kind the kind of this generator. |  | Enum: [ACRAccessToken BeyondtrustWorkloadCredentialsDynamicSecret CloudsmithAccessToken ECRAuthorizationToken Fake GCRAccessToken GithubAccessToken GitlabDeployToken QuayAccessToken Password SSHKey STSSessionToken UUID VaultDynamicSecret Webhook Grafana MFA] <br /> |
| `generator` _[GeneratorSpec](#generatorspec)_ | Generator the spec for this generator, must match the kind. |  | MaxProperties: 1 <br />MinProperties: 1 <br /> |




#### ECRAuthorizationToken



ECRAuthorizationToken uses the GetAuthorizationToken API to retrieve an authorization token.
The authorization token is valid for 12 hours.
The authorizationToken returned is a base64 encoded string that can be decoded
and used in a docker login command to authenticate to a registry.
For more information, see Registry authentication (https://docs.aws.amazon.com/AmazonECR/latest/userguide/Registries.html#registry_auth) in the Amazon Elastic Container Registry User Guide.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `ECRAuthorizationToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ECRAuthorizationTokenSpec](#ecrauthorizationtokenspec)_ |  |  |  |


#### ECRAuthorizationTokenSpec



ECRAuthorizationTokenSpec defines the desired state to generate an AWS ECR authorization token.



_Appears in:_
- [ECRAuthorizationToken](#ecrauthorizationtoken)
- [GeneratorSpec](#generatorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `region` _string_ | Region specifies the region to operate in. |  |  |
| `auth` _[AWSAuth](#awsauth)_ | Auth defines how to authenticate with AWS |  | Optional: \{\} <br /> |
| `role` _string_ | You can assume a role before making calls to the<br />desired AWS service. |  | Optional: \{\} <br /> |
| `scope` _string_ | Scope specifies the ECR service scope.<br />Valid options are private and public. |  | Optional: \{\} <br /> |


#### Fake



Fake generator is used for testing. It lets you define
a static set of credentials that is always returned.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `Fake` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[FakeSpec](#fakespec)_ |  |  |  |


#### FakeSpec



FakeSpec contains the static data.



_Appears in:_
- [Fake](#fake)
- [GeneratorSpec](#generatorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `controller` _string_ | Used to select the correct ESO controller (think: ingress.ingressClassName)<br />The ESO controller is instantiated with a specific controller name and filters VDS based on this property |  | Optional: \{\} <br /> |
| `data` _object (keys:string, values:string)_ | Data defines the static data returned<br />by this generator. |  |  |


#### GCPSMAuth



GCPSMAuth defines the authentication methods for Google Cloud Platform.



_Appears in:_
- [GCRAccessTokenSpec](#gcraccesstokenspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[GCPSMAuthSecretRef](#gcpsmauthsecretref)_ |  |  | Optional: \{\} <br /> |
| `workloadIdentity` _[GCPWorkloadIdentity](#gcpworkloadidentity)_ |  |  | Optional: \{\} <br /> |
| `workloadIdentityFederation` _[GCPWorkloadIdentityFederation](#gcpworkloadidentityfederation)_ |  |  | Optional: \{\} <br /> |


#### GCPSMAuthSecretRef



GCPSMAuthSecretRef defines the reference to a secret containing Google Cloud Platform credentials.



_Appears in:_
- [GCPSMAuth](#gcpsmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretAccessKeySecretRef` _[SecretKeySelector](#secretkeyselector)_ | The SecretAccessKey is used for authentication |  | Optional: \{\} <br /> |


#### GCPWorkloadIdentity



GCPWorkloadIdentity defines the configuration for using GCP Workload Identity authentication.



_Appears in:_
- [GCPSMAuth](#gcpsmauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ |  |  |  |
| `clusterLocation` _string_ |  |  |  |
| `clusterName` _string_ |  |  |  |
| `clusterProjectID` _string_ |  |  |  |


#### GCRAccessToken



GCRAccessToken generates an GCP access token
that can be used to authenticate with GCR.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `GCRAccessToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GCRAccessTokenSpec](#gcraccesstokenspec)_ |  |  |  |


#### GCRAccessTokenSpec



GCRAccessTokenSpec defines the desired state to generate a Google Container Registry access token.



_Appears in:_
- [GCRAccessToken](#gcraccesstoken)
- [GeneratorSpec](#generatorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[GCPSMAuth](#gcpsmauth)_ | Auth defines the means for authenticating with GCP |  |  |
| `projectID` _string_ | ProjectID defines which project to use to authenticate with |  |  |


#### Generator

_Underlying type:_ _interface{Cleanup(ctx context.Context, obj *k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON, status GeneratorProviderState, kube sigs.k8s.io/controller-runtime/pkg/client.Client, namespace string) error; Generate(ctx context.Context, obj *k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON, kube sigs.k8s.io/controller-runtime/pkg/client.Client, namespace string) (map[string][]byte, GeneratorProviderState, error)}_

Generator is the common interface for all generators that is actually used to generate whatever is needed.







#### GeneratorKind

_Underlying type:_ _string_

GeneratorKind represents a kind of generator.

_Validation:_
- Enum: [ACRAccessToken BeyondtrustWorkloadCredentialsDynamicSecret CloudsmithAccessToken ECRAuthorizationToken Fake GCRAccessToken GithubAccessToken GitlabDeployToken QuayAccessToken Password SSHKey STSSessionToken UUID VaultDynamicSecret Webhook Grafana MFA]

_Appears in:_
- [ClusterGeneratorSpec](#clustergeneratorspec)

| Field | Description |
| --- | --- |
| `ACRAccessToken` | GeneratorKindACRAccessToken represents an Azure Container Registry access token generator.<br /> |
| `ECRAuthorizationToken` | GeneratorKindECRAuthorizationToken represents an AWS ECR authorization token generator.<br /> |
| `Fake` | GeneratorKindFake represents a fake generator for testing purposes.<br /> |
| `GCRAccessToken` | GeneratorKindGCRAccessToken represents a Google Container Registry access token generator.<br /> |
| `GithubAccessToken` | GeneratorKindGithubAccessToken represents a GitHub access token generator.<br /> |
| `GitlabDeployToken` | GeneratorKindGitlabDeployToken represents a GitLab deploy token generator.<br /> |
| `QuayAccessToken` | GeneratorKindQuayAccessToken represents a Quay access token generator.<br /> |
| `Password` | GeneratorKindPassword represents a password generator.<br /> |
| `SSHKey` | GeneratorKindSSHKey represents an SSH key generator.<br /> |
| `STSSessionToken` | GeneratorKindSTSSessionToken represents an AWS STS session token generator.<br /> |
| `UUID` | GeneratorKindUUID represents a UUID generator.<br /> |
| `VaultDynamicSecret` | GeneratorKindVaultDynamicSecret represents a HashiCorp Vault dynamic secret generator.<br /> |
| `Webhook` | GeneratorKindWebhook represents a webhook-based generator.<br /> |
| `Grafana` | GeneratorKindGrafana represents a Grafana token generator.<br /> |
| `MFA` | GeneratorKindMFA represents a Multi-Factor Authentication generator.<br /> |
| `CloudsmithAccessToken` | GeneratorKindCloudsmithAccessToken represents a Cloudsmith access token generator.<br /> |
| `BeyondtrustWorkloadCredentialsDynamicSecret` | GeneratorKindBeyondtrustWorkloadCredentialsDynamicSecret represents a BeyondTrust Workload Credentials dynamic secret generator.<br /> |




#### GeneratorSpec



GeneratorSpec defines the configuration for various supported generator types.

_Validation:_
- MaxProperties: 1
- MinProperties: 1

_Appears in:_
- [ClusterGeneratorSpec](#clustergeneratorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `acrAccessTokenSpec` _[ACRAccessTokenSpec](#acraccesstokenspec)_ |  |  |  |
| `beyondtrustWorkloadCredentialsDynamicSecretSpec` _[BeyondtrustWorkloadCredentialsDynamicSecretSpec](#beyondtrustworkloadcredentialsdynamicsecretspec)_ |  |  |  |
| `cloudsmithAccessTokenSpec` _[CloudsmithAccessTokenSpec](#cloudsmithaccesstokenspec)_ |  |  |  |
| `ecrAuthorizationTokenSpec` _[ECRAuthorizationTokenSpec](#ecrauthorizationtokenspec)_ |  |  |  |
| `fakeSpec` _[FakeSpec](#fakespec)_ |  |  |  |
| `gcrAccessTokenSpec` _[GCRAccessTokenSpec](#gcraccesstokenspec)_ |  |  |  |
| `githubAccessTokenSpec` _[GithubAccessTokenSpec](#githubaccesstokenspec)_ |  |  |  |
| `gitlabDeployTokenSpec` _[GitlabDeployTokenSpec](#gitlabdeploytokenspec)_ |  |  |  |
| `quayAccessTokenSpec` _[QuayAccessTokenSpec](#quayaccesstokenspec)_ |  |  |  |
| `passwordSpec` _[PasswordSpec](#passwordspec)_ |  |  |  |
| `sshKeySpec` _[SSHKeySpec](#sshkeyspec)_ |  |  |  |
| `stsSessionTokenSpec` _[STSSessionTokenSpec](#stssessiontokenspec)_ |  |  |  |
| `uuidSpec` _[UUIDSpec](#uuidspec)_ |  |  |  |
| `vaultDynamicSecretSpec` _[VaultDynamicSecretSpec](#vaultdynamicsecretspec)_ |  |  |  |
| `webhookSpec` _[WebhookSpec](#webhookspec)_ |  |  |  |
| `grafanaSpec` _[GrafanaSpec](#grafanaspec)_ |  |  |  |
| `mfaSpec` _[MFASpec](#mfaspec)_ |  |  |  |


#### GeneratorState



GeneratorState represents the state created and managed by a generator resource.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `GeneratorState` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GeneratorStateSpec](#generatorstatespec)_ |  |  |  |
| `status` _[GeneratorStateStatus](#generatorstatestatus)_ |  |  |  |


#### GeneratorStateConditionType

_Underlying type:_ _string_

GeneratorStateConditionType represents the type of condition for a generator state.



_Appears in:_
- [GeneratorStateStatusCondition](#generatorstatestatuscondition)

| Field | Description |
| --- | --- |
| `Ready` | GeneratorStateReady indicates the generator state is ready and available.<br /> |


#### GeneratorStateSpec



GeneratorStateSpec defines the desired state of a generator state resource.



_Appears in:_
- [GeneratorState](#generatorstate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `garbageCollectionDeadline` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ | GarbageCollectionDeadline is the time after which the generator state<br />will be deleted.<br />It is set by the controller which creates the generator state and<br />can be set configured by the user.<br />If the garbage collection deadline is not set the generator state will not be deleted. |  |  |
| `resource` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#json-v1-apiextensions-k8s-io)_ | Resource is the generator manifest that produced the state.<br />It is a snapshot of the generator manifest at the time the state was produced.<br />This manifest will be used to delete the resource. Any configuration that is referenced<br />in the manifest should be available at the time of garbage collection. If that is not the case deletion will<br />be blocked by a finalizer. |  |  |
| `state` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#json-v1-apiextensions-k8s-io)_ | State is the state that was produced by the generator implementation. |  |  |


#### GeneratorStateStatus



GeneratorStateStatus defines the observed state of a generator state resource.



_Appears in:_
- [GeneratorState](#generatorstate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[GeneratorStateStatusCondition](#generatorstatestatuscondition) array_ |  |  |  |


#### GeneratorStateStatusCondition



GeneratorStateStatusCondition represents the observed condition of a generator state.



_Appears in:_
- [GeneratorStateStatus](#generatorstatestatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[GeneratorStateConditionType](#generatorstateconditiontype)_ |  |  |  |
| `status` _[ConditionStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#conditionstatus-v1-core)_ |  |  |  |
| `reason` _string_ |  |  | Optional: \{\} <br /> |
| `message` _string_ |  |  | Optional: \{\} <br /> |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ |  |  | Optional: \{\} <br /> |


#### GithubAccessToken



GithubAccessToken generates ghs_ accessToken





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `GithubAccessToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GithubAccessTokenSpec](#githubaccesstokenspec)_ |  |  |  |


#### GithubAccessTokenSpec



GithubAccessTokenSpec defines the desired state to generate a GitHub access token.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [GithubAccessToken](#githubaccesstoken)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL configures the GitHub instance URL. Defaults to https://github.com/. |  |  |
| `appID` _string_ |  |  |  |
| `installID` _string_ |  |  |  |
| `repositories` _string array_ | List of repositories the token will have access to. If omitted, defaults to all repositories the GitHub App<br />is installed to. |  |  |
| `permissions` _object (keys:string, values:string)_ | Map of permissions the token will have. If omitted, defaults to all permissions the GitHub App has. |  |  |
| `auth` _[GithubAuth](#githubauth)_ | Auth configures how ESO authenticates with a Github instance. |  |  |


#### GithubAuth



GithubAuth defines the authentication configuration for GitHub access.



_Appears in:_
- [GithubAccessTokenSpec](#githubaccesstokenspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `privateKey` _[GithubSecretRef](#githubsecretref)_ |  |  |  |


#### GithubSecretRef



GithubSecretRef references a secret containing GitHub credentials.



_Appears in:_
- [GithubAuth](#githubauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### GitlabDeployToken



GitlabDeployToken generates a GitLab deploy token.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `GitlabDeployToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GitlabDeployTokenSpec](#gitlabdeploytokenspec)_ |  |  |  |


#### GitlabDeployTokenAuth



GitlabDeployTokenAuth defines the authentication configuration for the GitLab API.



_Appears in:_
- [GitlabDeployTokenSpec](#gitlabdeploytokenspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `token` _[GitlabDeployTokenSecretRef](#gitlabdeploytokensecretref)_ | Token references a secret containing a GitLab access token (personal, group, or<br />project) with the api scope and at least the Maintainer role on the target. |  |  |


#### GitlabDeployTokenScope

_Underlying type:_ _string_

GitlabDeployTokenScope is a scope that can be granted to a GitLab deploy token.

_Validation:_
- Enum: [read_repository read_registry write_registry read_package_registry write_package_registry read_virtual_registry write_virtual_registry]

_Appears in:_
- [GitlabDeployTokenSpec](#gitlabdeploytokenspec)

| Field | Description |
| --- | --- |
| `read_repository` | GitlabDeployTokenScopeReadRepository allows read access to the repository.<br /> |
| `read_registry` | GitlabDeployTokenScopeReadRegistry allows read access to the container registry.<br /> |
| `write_registry` | GitlabDeployTokenScopeWriteRegistry allows write access to the container registry.<br /> |
| `read_package_registry` | GitlabDeployTokenScopeReadPackageRegistry allows read access to the package registry.<br /> |
| `write_package_registry` | GitlabDeployTokenScopeWritePackageRegistry allows write access to the package registry.<br /> |
| `read_virtual_registry` | GitlabDeployTokenScopeReadVirtualRegistry allows read access to the virtual registry (projects only).<br /> |
| `write_virtual_registry` | GitlabDeployTokenScopeWriteVirtualRegistry allows write access to the virtual registry (projects only).<br /> |


#### GitlabDeployTokenSecretRef



GitlabDeployTokenSecretRef references a secret containing a GitLab access token.



_Appears in:_
- [GitlabDeployTokenAuth](#gitlabdeploytokenauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### GitlabDeployTokenSpec



GitlabDeployTokenSpec defines the desired state to generate a GitLab deploy token.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [GitlabDeployToken](#gitlabdeploytoken)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL configures the GitLab instance URL. Defaults to https://gitlab.com. |  | Optional: \{\} <br /> |
| `projectID` _string_ | ProjectID is the numeric ID or unescaped path (e.g. group/project) of the<br />project to create the deploy token in. The generator URL-escapes paths before<br />calling the GitLab API, so do not pre-encode. Mutually exclusive with groupID. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `groupID` _string_ | GroupID is the numeric ID or unescaped path (e.g. parent/group) of the group to<br />create the deploy token in. The generator URL-escapes paths before calling the<br />GitLab API, so do not pre-encode. Mutually exclusive with projectID. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `name` _string_ | Name of the deploy token. |  | MinLength: 1 <br /> |
| `scopes` _[GitlabDeployTokenScope](#gitlabdeploytokenscope) array_ | Scopes granted to the deploy token. At least one scope is required. |  | Enum: [read_repository read_registry write_registry read_package_registry write_package_registry read_virtual_registry write_virtual_registry] <br />MinItems: 1 <br /> |
| `expiresAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ | ExpiresAt is an optional expiry for the deploy token. If omitted the token does<br />not expire on the GitLab side and is revoked only when the generator state is<br />cleaned up (on regeneration or when the consuming ExternalSecret is deleted). |  | Optional: \{\} <br /> |
| `username` _string_ | Username is an optional username for the deploy token. GitLab defaults it to<br />gitlab+deploy-token-\{n\} when omitted. |  | Optional: \{\} <br /> |
| `auth` _[GitlabDeployTokenAuth](#gitlabdeploytokenauth)_ | Auth configures how ESO authenticates with the GitLab API. |  |  |


#### Grafana



Grafana represents a generator for Grafana service account tokens.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `Grafana` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GrafanaSpec](#grafanaspec)_ |  |  |  |


#### GrafanaAuth



GrafanaAuth defines the authentication methods for connecting to a Grafana instance.



_Appears in:_
- [GrafanaSpec](#grafanaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `token` _[SecretKeySelector](#secretkeyselector)_ | A service account token used to authenticate against the Grafana instance.<br />Note: you need a token which has elevated permissions to create service accounts.<br />See here for the documentation on basic roles offered by Grafana:<br />https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/ |  | Optional: \{\} <br /> |
| `basic` _[GrafanaBasicAuth](#grafanabasicauth)_ | Basic auth credentials used to authenticate against the Grafana instance.<br />Note: you need a token which has elevated permissions to create service accounts.<br />See here for the documentation on basic roles offered by Grafana:<br />https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/ |  | Optional: \{\} <br /> |


#### GrafanaBasicAuth



GrafanaBasicAuth defines the credentials for basic authentication with Grafana.



_Appears in:_
- [GrafanaAuth](#grafanaauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `username` _string_ | A basic auth username used to authenticate against the Grafana instance. |  |  |
| `password` _[SecretKeySelector](#secretkeyselector)_ | A basic auth password used to authenticate against the Grafana instance. |  |  |


#### GrafanaServiceAccount



GrafanaServiceAccount defines the configuration for a Grafana service account to be created.



_Appears in:_
- [GrafanaSpec](#grafanaspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the service account that will be created by ESO. |  |  |
| `role` _string_ | Role is the role of the service account.<br />See here for the documentation on basic roles offered by Grafana:<br />https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/ |  |  |
| `secondsToLive` _integer_ | SecondsToLive is the number of seconds before the generated service account token will expire.<br />Some Grafana deployments (e.g. AWS Managed Grafana) require this value to be set. |  | Minimum: 1 <br />Optional: \{\} <br /> |




#### GrafanaSpec



GrafanaSpec controls the behavior of the grafana generator.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [Grafana](#grafana)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the URL of the Grafana instance. |  |  |
| `auth` _[GrafanaAuth](#grafanaauth)_ | Auth is the authentication configuration to authenticate<br />against the Grafana instance. |  |  |
| `serviceAccount` _[GrafanaServiceAccount](#grafanaserviceaccount)_ | ServiceAccount is the configuration for the service account that<br />is supposed to be generated by the generator. |  |  |


#### GrafanaStateServiceAccount



GrafanaStateServiceAccount contains the service account ID, login and token ID.



_Appears in:_
- [GrafanaServiceAccountTokenState](#grafanaserviceaccounttokenstate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _integer_ |  |  |  |
| `login` _string_ |  |  |  |
| `tokenID` _integer_ |  |  |  |


#### MFA



MFA generates a new TOTP token that is compliant with RFC 6238.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `MFA` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[MFASpec](#mfaspec)_ |  |  |  |


#### MFASpec



MFASpec controls the behavior of the mfa generator.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [MFA](#mfa)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secret` _[SecretKeySelector](#secretkeyselector)_ | Secret is a secret selector to a secret containing the seed secret to generate the TOTP value from. |  |  |
| `length` _integer_ | Length defines the token length. Defaults to 6 characters. |  |  |
| `timePeriod` _integer_ | TimePeriod defines how long the token can be active. Defaults to 30 seconds. |  |  |
| `algorithm` _string_ | Algorithm to use for encoding. Defaults to SHA1 as per the RFC. |  |  |
| `when` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#time-v1-meta)_ | When defines a time parameter that can be used to pin the origin time of the generated token. |  |  |


#### NTLMProtocol



NTLMProtocol contains the NTLM-specific configuration.



_Appears in:_
- [AuthorizationProtocol](#authorizationprotocol)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `usernameSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |
| `passwordSecret` _[SecretKeySelector](#secretkeyselector)_ |  |  |  |


#### Password



Password generates a random password based on the
configuration parameters in spec.
You can specify the length, characterset and other attributes.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `Password` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PasswordSpec](#passwordspec)_ |  |  |  |


#### PasswordSpec



PasswordSpec controls the behavior of the password generator.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [Password](#password)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `length` _integer_ | Length of the password to be generated.<br />Defaults to 24 | 24 |  |
| `digits` _integer_ | Digits specifies the number of digits in the generated<br />password. If omitted it defaults to 25% of the length of the password |  |  |
| `symbols` _integer_ | Symbols specifies the number of symbol characters in the generated<br />password. If omitted it defaults to 25% of the length of the password |  |  |
| `symbolCharacters` _string_ | SymbolCharacters specifies the special characters that should be used<br />in the generated password. |  |  |
| `noUpper` _boolean_ | Set NoUpper to disable uppercase characters | false |  |
| `allowRepeat` _boolean_ | set AllowRepeat to true to allow repeating characters. | false |  |
| `secretKeys` _string array_ | SecretKeys defines the keys that will be populated with generated passwords.<br />Defaults to "password" when not set. |  | MinItems: 1 <br />Optional: \{\} <br /> |
| `encoding` _string_ | Encoding specifies the encoding of the generated password.<br />Valid values are:<br />- "raw" (default): no encoding<br />- "base64": standard base64 encoding<br />- "base64url": base64url encoding<br />- "base32": base32 encoding<br />- "hex": hexadecimal encoding | raw | Enum: [base64 base64url base32 hex raw] <br /> |


#### QuayAccessToken



QuayAccessToken generates Quay oauth token for pulling/pushing images





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `QuayAccessToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[QuayAccessTokenSpec](#quayaccesstokenspec)_ |  |  |  |


#### QuayAccessTokenSpec



QuayAccessTokenSpec defines the desired state to generate a Quay access token.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [QuayAccessToken](#quayaccesstoken)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL configures the Quay instance URL. Defaults to quay.io. |  |  |
| `robotAccount` _string_ | Name of the robot account you are federating with |  |  |
| `serviceAccountRef` _[ServiceAccountSelector](#serviceaccountselector)_ | Name of the service account you are federating with |  |  |


#### RequestParameters



RequestParameters contains parameters that can be passed to the STS service.



_Appears in:_
- [STSSessionTokenSpec](#stssessiontokenspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `sessionDuration` _integer_ |  |  | Optional: \{\} <br /> |
| `serialNumber` _string_ | SerialNumber is the identification number of the MFA device that is associated with the IAM user who is making<br />the GetSessionToken call.<br />Possible values: hardware device (such as GAHT12345678) or an Amazon Resource Name (ARN) for a virtual device<br />(such as arn:aws:iam::123456789012:mfa/user) |  | Optional: \{\} <br /> |
| `tokenCode` _string_ | TokenCode is the value provided by the MFA device, if MFA is required. |  | Optional: \{\} <br /> |


#### SSHKey



SSHKey generates SSH key pairs.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `SSHKey` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SSHKeySpec](#sshkeyspec)_ |  |  |  |


#### SSHKeySpec



SSHKeySpec controls the behavior of the ssh key generator.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [SSHKey](#sshkey)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `keyType` _string_ | KeyType specifies the SSH key type (rsa, ecdsa, ed25519) | rsa | Enum: [rsa ecdsa ed25519] <br /> |
| `keySize` _integer_ | KeySize specifies the key size for RSA keys (default: 2048) and ECDSA keys (default: 256).<br />For RSA keys: 2048, 3072, 4096<br />For ECDSA keys: 256, 384, 521<br />Ignored for ed25519 keys |  | Maximum: 8192 <br />Minimum: 256 <br /> |
| `comment` _string_ | Comment specifies an optional comment for the SSH key |  |  |


#### STSSessionToken



STSSessionToken uses the GetSessionToken API to retrieve an authorization token.
The authorization token is valid for 12 hours.
The authorizationToken returned is a base64 encoded string that can be decoded.
For more information, see GetSessionToken (https://docs.aws.amazon.com/STS/latest/APIReference/API_GetSessionToken.html).





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `STSSessionToken` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[STSSessionTokenSpec](#stssessiontokenspec)_ |  |  |  |


#### STSSessionTokenSpec



STSSessionTokenSpec defines the desired state to generate an AWS STS session token.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [STSSessionToken](#stssessiontoken)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `region` _string_ | Region specifies the region to operate in. |  |  |
| `auth` _[AWSAuth](#awsauth)_ | Auth defines how to authenticate with AWS |  | Optional: \{\} <br /> |
| `role` _string_ | You can assume a role before making calls to the<br />desired AWS service. |  | Optional: \{\} <br /> |
| `requestParameters` _[RequestParameters](#requestparameters)_ | RequestParameters contains parameters that can be passed to the STS service. |  | Optional: \{\} <br /> |


#### SecretKeySelector



SecretKeySelector defines a reference to a specific key within a Kubernetes Secret.



_Appears in:_
- [GrafanaAuth](#grafanaauth)
- [GrafanaBasicAuth](#grafanabasicauth)
- [WebhookSecret](#webhooksecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The name of the Secret resource being referred to. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `key` _string_ | The key where the token is found. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |


#### StatefulResource

_Underlying type:_ _interface{k8s.io/apimachinery/pkg/runtime.Object; k8s.io/apimachinery/pkg/apis/meta/v1.Object}_

StatefulResource represents a Kubernetes resource that has state which can be tracked.







#### UUID



UUID generates a version 1 UUID (e56657e3-764f-11ef-a397-65231a88c216).





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `UUID` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UUIDSpec](#uuidspec)_ |  |  |  |


#### UUIDSpec



UUIDSpec controls the behavior of the uuid generator.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [UUID](#uuid)



#### VaultDynamicSecret



VaultDynamicSecret represents a generator that can create dynamic secrets from HashiCorp Vault.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `VaultDynamicSecret` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VaultDynamicSecretSpec](#vaultdynamicsecretspec)_ |  |  |  |


#### VaultDynamicSecretResultType

_Underlying type:_ _string_

VaultDynamicSecretResultType defines which part of the Vault API response should be returned.

_Validation:_
- Enum: [Data Auth Raw]

_Appears in:_
- [VaultDynamicSecretSpec](#vaultdynamicsecretspec)

| Field | Description |
| --- | --- |
| `Data` | VaultDynamicSecretResultTypeData specifies to return the "data" section of the Vault API response.<br /> |
| `Auth` | VaultDynamicSecretResultTypeAuth specifies to return the "auth" section of the Vault API response.<br /> |
| `Raw` | VaultDynamicSecretResultTypeRaw specifies to return the raw response from the Vault API.<br /> |


#### VaultDynamicSecretSpec



VaultDynamicSecretSpec defines the desired spec of VaultDynamicSecret.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [VaultDynamicSecret](#vaultdynamicsecret)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `controller` _string_ | Used to select the correct ESO controller (think: ingress.ingressClassName)<br />The ESO controller is instantiated with a specific controller name and filters VDS based on this property |  | Optional: \{\} <br /> |
| `method` _string_ | Vault API method to use (GET/POST/other) |  |  |
| `parameters` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#json-v1-apiextensions-k8s-io)_ | Parameters to pass to Vault write (for non-GET methods) |  |  |
| `getParameters` _object (keys:string, values:string array)_ | GetParameters are query-string parameters passed to Vault on GET calls.<br />Each key may map to multiple values, matching HTTP query-string semantics.<br />Ignored for non-GET methods; use Parameters for write bodies. |  | Optional: \{\} <br /> |
| `resultType` _[VaultDynamicSecretResultType](#vaultdynamicsecretresulttype)_ | Result type defines which data is returned from the generator.<br />By default, it is the "data" section of the Vault API response.<br />When using e.g. /auth/token/create the "data" section is empty but<br />the "auth" section contains the generated token.<br />Please refer to the vault docs regarding the result data structure.<br />Additionally, accessing the raw response is possibly by using "Raw" result type. | Data | Enum: [Data Auth Raw] <br /> |
| `retrySettings` _[SecretStoreRetrySettings](#secretstoreretrysettings)_ | Used to configure http retries if failed |  | Optional: \{\} <br /> |
| `provider` _[VaultProvider](#vaultprovider)_ | Vault provider common spec |  |  |
| `path` _string_ | Vault path to obtain the dynamic secret from |  |  |
| `allowEmptyResponse` _boolean_ | Do not fail if no secrets are found. Useful for requests where no data is expected. | false | Optional: \{\} <br /> |


#### Webhook



Webhook connects to a third party API server to handle the secrets generation
configuration parameters in spec.
You can specify the server, the token, and additional body parameters.
See documentation for the full API specification for requests and responses.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `generators.external-secrets.io/v1alpha1` | | |
| `kind` _string_ | `Webhook` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[WebhookSpec](#webhookspec)_ |  |  |  |


#### WebhookCAProvider



WebhookCAProvider defines a location to fetch the cert for the webhook provider from.



_Appears in:_
- [WebhookSpec](#webhookspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _[WebhookCAProviderType](#webhookcaprovidertype)_ | The type of provider to use such as "Secret", or "ConfigMap". |  | Enum: [Secret ConfigMap] <br /> |
| `name` _string_ | The name of the object located at the provider type. |  | MaxLength: 253 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$` <br /> |
| `key` _string_ | The key where the CA certificate can be found in the Secret or ConfigMap. |  | MaxLength: 253 <br />MinLength: 1 <br />Optional: \{\} <br />Pattern: `^[-._a-zA-Z0-9]+$` <br /> |
| `namespace` _string_ | The namespace the Provider type is in. |  | MaxLength: 63 <br />MinLength: 1 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


#### WebhookCAProviderType

_Underlying type:_ _string_

WebhookCAProviderType defines the type of provider for webhook CA certificates.



_Appears in:_
- [WebhookCAProvider](#webhookcaprovider)

| Field | Description |
| --- | --- |
| `Secret` | WebhookCAProviderTypeSecret indicates the CA provider is a Secret resource.<br /> |
| `ConfigMap` | WebhookCAProviderTypeConfigMap indicates the CA provider is a ConfigMap resource.<br /> |


#### WebhookResult



WebhookResult defines how to format and extract results from the webhook response.



_Appears in:_
- [WebhookSpec](#webhookspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `jsonPath` _string_ | Json path of return value |  | Optional: \{\} <br /> |


#### WebhookSecret



WebhookSecret defines a secret reference that will be used in webhook templates.



_Appears in:_
- [WebhookSpec](#webhookspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of this secret in templates |  |  |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | Secret ref to fill in credentials |  |  |


#### WebhookSpec



WebhookSpec controls the behavior of the external generator. Any body parameters should be passed to the server through the parameters field.



_Appears in:_
- [GeneratorSpec](#generatorspec)
- [Webhook](#webhook)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `method` _string_ | Webhook Method |  |  |
| `url` _string_ | Webhook url to call |  |  |
| `headers` _object (keys:string, values:string)_ | Headers |  | Optional: \{\} <br /> |
| `auth` _[AuthorizationProtocol](#authorizationprotocol)_ | Auth specifies a authorization protocol. Only one protocol may be set. |  | MaxProperties: 1 <br />MinProperties: 1 <br />Optional: \{\} <br /> |
| `body` _string_ | Body |  | Optional: \{\} <br /> |
| `timeout` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#duration-v1-meta)_ | Timeout |  | Optional: \{\} <br /> |
| `result` _[WebhookResult](#webhookresult)_ | Result formatting |  |  |
| `secrets` _[WebhookSecret](#webhooksecret) array_ | Secrets to fill in templates<br />These secrets will be passed to the templating function as key value pairs under the given name |  | Optional: \{\} <br /> |
| `caBundle` _integer array_ | PEM encoded CA bundle used to validate webhook server certificate. Only used<br />if the Server URL is using HTTPS protocol. This parameter is ignored for<br />plain HTTP protocol connection. If not set the system root certificates<br />are used to validate the TLS connection. |  | Optional: \{\} <br /> |
| `caProvider` _[WebhookCAProvider](#webhookcaprovider)_ | The provider for the CA bundle to use to validate webhook server certificate. |  | Optional: \{\} <br /> |
