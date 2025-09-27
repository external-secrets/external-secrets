<p>Packages:</p>
<ul>
<li>
<a href="#external-secrets.io%2fv1">external-secrets.io/v1</a>
</li>
</ul>
<h2 id="external-secrets.io/v1">external-secrets.io/v1</h2>
<p>
<p>Package v1 contains resources for external-secrets</p>
</p>
Resource Types:
<ul></ul>
<h3 id="external-secrets.io/v1.AWSAuth">AWSAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AWSProvider">AWSProvider</a>)
</p>
<p>
<p>AWSAuth tells the controller how to do authentication with aws.
Only one of secretRef or jwt can be specified.
if none is specified the controller will load credentials using the aws sdk defaults.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.AWSAuthSecretRef">
AWSAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>jwt</code></br>
<em>
<a href="#external-secrets.io/v1.AWSJWTAuth">
AWSJWTAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AWSAuthSecretRef">AWSAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AWSAuth">AWSAuth</a>)
</p>
<p>
<p>AWSAuthSecretRef holds secret references for AWS credentials
both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessKeyIDSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The AccessKeyID is used for authentication</p>
</td>
</tr>
<tr>
<td>
<code>secretAccessKeySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The SecretAccessKey is used for authentication</p>
</td>
</tr>
<tr>
<td>
<code>sessionTokenSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The SessionToken used for authentication
This must be defined if AccessKeyID and SecretAccessKey are temporary credentials
see: <a href="https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html">https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AWSJWTAuth">AWSJWTAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AWSAuth">AWSAuth</a>)
</p>
<p>
<p>Authenticate against AWS using service account tokens.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AWSProvider">AWSProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>AWSProvider configures a store to sync secrets with AWS.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>service</code></br>
<em>
<a href="#external-secrets.io/v1.AWSServiceType">
AWSServiceType
</a>
</em>
</td>
<td>
<p>Service defines which service should be used to fetch the secrets</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.AWSAuth">
AWSAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines the information necessary to authenticate against AWS
if not set aws sdk will infer credentials from your environment
see: <a href="https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials">https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials</a></p>
</td>
</tr>
<tr>
<td>
<code>role</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Role is a Role ARN which the provider will assume</p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>AWS Region to be used for the provider</p>
</td>
</tr>
<tr>
<td>
<code>additionalRoles</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdditionalRoles is a chained list of Role ARNs which the provider will sequentially assume before assuming the Role</p>
</td>
</tr>
<tr>
<td>
<code>externalID</code></br>
<em>
string
</em>
</td>
<td>
<p>AWS External ID set on assumed IAM roles</p>
</td>
</tr>
<tr>
<td>
<code>sessionTags</code></br>
<em>
<a href="#external-secrets.io/v1.*github.com/external-secrets/external-secrets/apis/externalsecrets/v1.Tag">
[]*github.com/external-secrets/external-secrets/apis/externalsecrets/v1.Tag
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWS STS assume role session tags</p>
</td>
</tr>
<tr>
<td>
<code>secretsManager</code></br>
<em>
<a href="#external-secrets.io/v1.SecretsManager">
SecretsManager
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretsManager defines how the provider behaves when interacting with AWS SecretsManager</p>
</td>
</tr>
<tr>
<td>
<code>transitiveTagKeys</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWS STS assume role transitive session tags. Required when multiple rules are used with the provider</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Prefix adds a prefix to all retrieved values.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AWSServiceType">AWSServiceType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AWSProvider">AWSProvider</a>)
</p>
<p>
<p>AWSServiceType is a enum that defines the service/API that is used to fetch the secrets.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ParameterStore&#34;</p></td>
<td><p>AWSServiceParameterStore is the AWS SystemsManager ParameterStore service.
see: <a href="https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html">https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html</a></p>
</td>
</tr><tr><td><p>&#34;SecretsManager&#34;</p></td>
<td><p>AWSServiceSecretsManager is the AWS SecretsManager service.
see: <a href="https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html">https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html</a></p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.AkeylessAuth">AkeylessAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AkeylessProvider">AkeylessProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.AkeylessAuthSecretRef">
AkeylessAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to a Secret that contains the details
to authenticate with Akeyless.</p>
</td>
</tr>
<tr>
<td>
<code>kubernetesAuth</code></br>
<em>
<a href="#external-secrets.io/v1.AkeylessKubernetesAuth">
AkeylessKubernetesAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Kubernetes authenticates with Akeyless by passing the ServiceAccount
token stored in the named Secret resource.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AkeylessAuthSecretRef">AkeylessAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AkeylessAuth">AkeylessAuth</a>)
</p>
<p>
<p>AkeylessAuthSecretRef
AKEYLESS_ACCESS_TYPE_PARAM: AZURE_OBJ_ID OR GCP_AUDIENCE OR ACCESS_KEY OR KUB_CONFIG_NAME.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessID</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The SecretAccessID is used for authentication</p>
</td>
</tr>
<tr>
<td>
<code>accessType</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>accessTypeParam</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AkeylessKubernetesAuth">AkeylessKubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AkeylessAuth">AkeylessAuth</a>)
</p>
<p>
<p>Authenticate with Kubernetes ServiceAccount token stored.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessID</code></br>
<em>
string
</em>
</td>
<td>
<p>the Akeyless Kubernetes auth-method access-id</p>
</td>
</tr>
<tr>
<td>
<code>k8sConfName</code></br>
<em>
string
</em>
</td>
<td>
<p>Kubernetes-auth configuration name in Akeyless-Gateway</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional service account field containing the name of a kubernetes ServiceAccount.
If the service account is specified, the service account secret token JWT will be used
for authenticating with Akeyless. If the service account selector is not supplied,
the secretRef will be used instead.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional secret field containing a Kubernetes ServiceAccount JWT used
for authenticating with Akeyless. If a name is specified without a key,
<code>token</code> is the default. If one is not specified, the one bound to
the controller will be used.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AkeylessProvider">AkeylessProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>AkeylessProvider Configures an store to sync secrets using Akeyless KV.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>akeylessGWApiURL</code></br>
<em>
string
</em>
</td>
<td>
<p>Akeyless GW API Url from which the secrets to be fetched from.</p>
</td>
</tr>
<tr>
<td>
<code>authSecretRef</code></br>
<em>
<a href="#external-secrets.io/v1.AkeylessAuth">
AkeylessAuth
</a>
</em>
</td>
<td>
<p>Auth configures how the operator authenticates with Akeyless.</p>
</td>
</tr>
<tr>
<td>
<code>caBundle</code></br>
<em>
[]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>PEM/base64 encoded CA bundle used to validate Akeyless Gateway certificate. Only used
if the AkeylessGWApiURL URL is using HTTPS protocol. If not set the system root certificates
are used to validate the TLS connection.</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.CAProvider">
CAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate Akeyless Gateway certificate.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AlibabaAuth">AlibabaAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AlibabaProvider">AlibabaProvider</a>)
</p>
<p>
<p>AlibabaAuth contains a secretRef for credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.AlibabaAuthSecretRef">
AlibabaAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>rrsa</code></br>
<em>
<a href="#external-secrets.io/v1.AlibabaRRSAAuth">
AlibabaRRSAAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AlibabaAuthSecretRef">AlibabaAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AlibabaAuth">AlibabaAuth</a>)
</p>
<p>
<p>AlibabaAuthSecretRef holds secret references for Alibaba credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessKeyIDSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The AccessKeyID is used for authentication</p>
</td>
</tr>
<tr>
<td>
<code>accessKeySecretSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The AccessKeySecret is used for authentication</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AlibabaProvider">AlibabaProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>AlibabaProvider configures a store to sync secrets using the Alibaba Secret Manager provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.AlibabaAuth">
AlibabaAuth
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>regionID</code></br>
<em>
string
</em>
</td>
<td>
<p>Alibaba Region to be used for the provider</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AlibabaRRSAAuth">AlibabaRRSAAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AlibabaAuth">AlibabaAuth</a>)
</p>
<p>
<p>Authenticate against Alibaba using RRSA.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>oidcProviderArn</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>oidcTokenFilePath</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>roleArn</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>sessionName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AuthorizationProtocol">AuthorizationProtocol
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.WebhookProvider">WebhookProvider</a>)
</p>
<p>
<p>AuthorizationProtocol contains the protocol-specific configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ntlm</code></br>
<em>
<a href="#external-secrets.io/v1.NTLMProtocol">
NTLMProtocol
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NTLMProtocol configures the store to use NTLM for auth</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AwsAuthCredentials">AwsAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AwsCredentialsConfig">AwsCredentialsConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GCPWorkloadIdentityFederation">GCPWorkloadIdentityFederation</a>)
</p>
<p>
<p>AwsCredentialsConfig holds the region and the Secret reference which contains the AWS credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>region is for configuring the AWS region to be used.</p>
</td>
</tr>
<tr>
<td>
<code>awsCredentialsSecretRef</code></br>
<em>
<a href="#external-secrets.io/v1.SecretReference">
SecretReference
</a>
</em>
</td>
<td>
<p>awsCredentialsSecretRef is the reference to the secret which holds the AWS credentials.
Secret should be created with below names for keys
- aws_access_key_id: Access Key ID, which is the unique identifier for the AWS account or the IAM user.
- aws_secret_access_key: Secret Access Key, which is used to authenticate requests made to AWS services.
- aws_session_token: Session Token, is the short-lived token to authenticate requests made to AWS services.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AzureAuthCredentials">AzureAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resource</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AzureAuthType">AzureAuthType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AzureKVProvider">AzureKVProvider</a>)
</p>
<p>
<p>AuthType describes how to authenticate to the Azure Keyvault
Only one of the following auth types may be specified.
If none of the following auth type is specified, the default one
is ServicePrincipal.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ManagedIdentity&#34;</p></td>
<td><p>Using Managed Identity to authenticate. Used with aad-pod-identity installed in the cluster.</p>
</td>
</tr><tr><td><p>&#34;ServicePrincipal&#34;</p></td>
<td><p>Using service principal to authenticate, which needs a tenantId, a clientId and a clientSecret.</p>
</td>
</tr><tr><td><p>&#34;WorkloadIdentity&#34;</p></td>
<td><p>Using Workload Identity service accounts to authenticate.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.AzureCustomCloudConfig">AzureCustomCloudConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AzureKVProvider">AzureKVProvider</a>)
</p>
<p>
<p>AzureCustomCloudConfig specifies custom cloud configuration for private Azure environments
IMPORTANT: Custom cloud configuration is ONLY supported when UseAzureSDK is true.
The legacy go-autorest SDK does not support custom cloud endpoints.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>activeDirectoryEndpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>ActiveDirectoryEndpoint is the AAD endpoint for authentication
Required when using custom cloud configuration</p>
</td>
</tr>
<tr>
<td>
<code>keyVaultEndpoint</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>KeyVaultEndpoint is the Key Vault service endpoint</p>
</td>
</tr>
<tr>
<td>
<code>keyVaultDNSSuffix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>KeyVaultDNSSuffix is the DNS suffix for Key Vault URLs</p>
</td>
</tr>
<tr>
<td>
<code>resourceManagerEndpoint</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ResourceManagerEndpoint is the Azure Resource Manager endpoint</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AzureEnvironmentType">AzureEnvironmentType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AzureKVProvider">AzureKVProvider</a>)
</p>
<p>
<p>AzureEnvironmentType specifies the Azure cloud environment endpoints to use for
connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: <a href="https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152">https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152</a>
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud, AzureStackCloud</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;AzureStackCloud&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;ChinaCloud&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;GermanCloud&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;PublicCloud&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;USGovernmentCloud&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.AzureKVAuth">AzureKVAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AzureKVProvider">AzureKVProvider</a>)
</p>
<p>
<p>Configuration used to authenticate with Azure.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The Azure clientId of the service principle or managed identity used for authentication.</p>
</td>
</tr>
<tr>
<td>
<code>tenantId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The Azure tenantId of the managed identity used for authentication.</p>
</td>
</tr>
<tr>
<td>
<code>clientSecret</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The Azure ClientSecret of the service principle used for authentication.</p>
</td>
</tr>
<tr>
<td>
<code>clientCertificate</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The Azure ClientCertificate of the service principle used for authentication.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.AzureKVProvider">AzureKVProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures an store to sync secrets using Azure KV.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>authType</code></br>
<em>
<a href="#external-secrets.io/v1.AzureAuthType">
AzureAuthType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth type defines how to authenticate to the keyvault service.
Valid values are:
- &ldquo;ServicePrincipal&rdquo; (default): Using a service principal (tenantId, clientId, clientSecret)
- &ldquo;ManagedIdentity&rdquo;: Using Managed Identity assigned to the pod (see aad-pod-identity)</p>
</td>
</tr>
<tr>
<td>
<code>vaultUrl</code></br>
<em>
string
</em>
</td>
<td>
<p>Vault Url from which the secrets to be fetched from.</p>
</td>
</tr>
<tr>
<td>
<code>tenantId</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type. Optional for WorkloadIdentity.</p>
</td>
</tr>
<tr>
<td>
<code>environmentType</code></br>
<em>
<a href="#external-secrets.io/v1.AzureEnvironmentType">
AzureEnvironmentType
</a>
</em>
</td>
<td>
<p>EnvironmentType specifies the Azure cloud environment endpoints to use for
connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: <a href="https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152">https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152</a>
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud, AzureStackCloud
Use AzureStackCloud when you need to configure custom Azure Stack Hub or Azure Stack Edge endpoints.</p>
</td>
</tr>
<tr>
<td>
<code>authSecretRef</code></br>
<em>
<a href="#external-secrets.io/v1.AzureKVAuth">
AzureKVAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth configures how the operator authenticates with Azure. Required for ServicePrincipal auth type. Optional for WorkloadIdentity.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountRef specified the service account
that should be used when authenticating with WorkloadIdentity.</p>
</td>
</tr>
<tr>
<td>
<code>identityId</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>If multiple Managed Identity is assigned to the pod, you can select the one to be used</p>
</td>
</tr>
<tr>
<td>
<code>useAzureSDK</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>UseAzureSDK enables the use of the new Azure SDK for Go (azcore-based) instead of the legacy go-autorest SDK.
This is experimental and may have behavioral differences. Defaults to false (legacy SDK).</p>
</td>
</tr>
<tr>
<td>
<code>customCloudConfig</code></br>
<em>
<a href="#external-secrets.io/v1.AzureCustomCloudConfig">
AzureCustomCloudConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CustomCloudConfig defines custom Azure Stack Hub or Azure Stack Edge endpoints.
Required when EnvironmentType is AzureStackCloud.
IMPORTANT: This feature REQUIRES UseAzureSDK to be set to true. Custom cloud
configuration is not supported with the legacy go-autorest SDK.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.BeyondTrustProviderSecretRef">BeyondTrustProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.BeyondtrustAuth">BeyondtrustAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value can be specified directly to set a value without using a secret.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef references a key in a secret that will be used as value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.BeyondtrustAuth">BeyondtrustAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.BeyondtrustProvider">BeyondtrustProvider</a>)
</p>
<p>
<p>Configures a store to sync secrets using BeyondTrust Password Safe.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiKey</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondTrustProviderSecretRef">
BeyondTrustProviderSecretRef
</a>
</em>
</td>
<td>
<p>APIKey If not provided then ClientID/ClientSecret become required.</p>
</td>
</tr>
<tr>
<td>
<code>clientId</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondTrustProviderSecretRef">
BeyondTrustProviderSecretRef
</a>
</em>
</td>
<td>
<p>ClientID is the API OAuth Client ID.</p>
</td>
</tr>
<tr>
<td>
<code>clientSecret</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondTrustProviderSecretRef">
BeyondTrustProviderSecretRef
</a>
</em>
</td>
<td>
<p>ClientSecret is the API OAuth Client Secret.</p>
</td>
</tr>
<tr>
<td>
<code>certificate</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondTrustProviderSecretRef">
BeyondTrustProviderSecretRef
</a>
</em>
</td>
<td>
<p>Certificate (cert.pem) for use when authenticating with an OAuth client Id using a Client Certificate.</p>
</td>
</tr>
<tr>
<td>
<code>certificateKey</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondTrustProviderSecretRef">
BeyondTrustProviderSecretRef
</a>
</em>
</td>
<td>
<p>Certificate private key (key.pem). For use when authenticating with an OAuth client Id</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.BeyondtrustProvider">BeyondtrustProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondtrustAuth">
BeyondtrustAuth
</a>
</em>
</td>
<td>
<p>Auth configures how the operator authenticates with Beyondtrust.</p>
</td>
</tr>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondtrustServer">
BeyondtrustServer
</a>
</em>
</td>
<td>
<p>Auth configures how API server works.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.BeyondtrustServer">BeyondtrustServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.BeyondtrustProvider">BeyondtrustProvider</a>)
</p>
<p>
<p>Configures a store to sync secrets using BeyondTrust Password Safe.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiUrl</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>apiVersion</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>retrievalType</code></br>
<em>
string
</em>
</td>
<td>
<p>The secret retrieval type. SECRET = Secrets Safe (credential, text, file). MANAGED_ACCOUNT = Password Safe account associated with a system.</p>
</td>
</tr>
<tr>
<td>
<code>separator</code></br>
<em>
string
</em>
</td>
<td>
<p>A character that separates the folder names.</p>
</td>
</tr>
<tr>
<td>
<code>verifyCA</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>clientTimeOutSeconds</code></br>
<em>
int
</em>
</td>
<td>
<p>Timeout specifies a time limit for requests made by this Client. The timeout includes connection time, any redirects, and reading the response body. Defaults to 45 seconds.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.BitwardenSecretsManagerAuth">BitwardenSecretsManagerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.BitwardenSecretsManagerProvider">BitwardenSecretsManagerProvider</a>)
</p>
<p>
<p>BitwardenSecretsManagerAuth contains the ref to the secret that contains the machine account token.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.BitwardenSecretsManagerSecretRef">
BitwardenSecretsManagerSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.BitwardenSecretsManagerProvider">BitwardenSecretsManagerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>BitwardenSecretsManagerProvider configures a store to sync secrets with a Bitwarden Secrets Manager instance.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiURL</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>identityURL</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bitwardenServerSDKURL</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>caBundle</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base64 encoded certificate for the bitwarden server sdk. The sdk MUST run with HTTPS to make sure no MITM attack
can be performed.</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.CAProvider">
CAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>see: <a href="https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider">https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider</a></p>
</td>
</tr>
<tr>
<td>
<code>organizationID</code></br>
<em>
string
</em>
</td>
<td>
<p>OrganizationID determines which organization this secret store manages.</p>
</td>
</tr>
<tr>
<td>
<code>projectID</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectID determines which project this secret store manages.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.BitwardenSecretsManagerAuth">
BitwardenSecretsManagerAuth
</a>
</em>
</td>
<td>
<p>Auth configures how secret-manager authenticates with a bitwarden machine account instance.
Make sure that the token being used has permissions on the given secret.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.BitwardenSecretsManagerSecretRef">BitwardenSecretsManagerSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.BitwardenSecretsManagerAuth">BitwardenSecretsManagerAuth</a>)
</p>
<p>
<p>BitwardenSecretsManagerSecretRef contains the credential ref to the bitwarden instance.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>credentials</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>AccessToken used for the bitwarden instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ByID">ByID
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.FetchingPolicy">FetchingPolicy</a>)
</p>
<p>
<p>ByID configures the provider to interpret the <code>data.secretKey.remoteRef.key</code> field in ExternalSecret as secret ID.</p>
</p>
<h3 id="external-secrets.io/v1.ByName">ByName
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.FetchingPolicy">FetchingPolicy</a>)
</p>
<p>
<p>ByName configures the provider to interpret the <code>data.secretKey.remoteRef.key</code> field in ExternalSecret as secret name.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>folderID</code></br>
<em>
string
</em>
</td>
<td>
<p>The folder to fetch secrets from</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.CAProvider">CAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AkeylessProvider">AkeylessProvider</a>, 
<a href="#external-secrets.io/v1.BitwardenSecretsManagerProvider">BitwardenSecretsManagerProvider</a>, 
<a href="#external-secrets.io/v1.ConjurProvider">ConjurProvider</a>, 
<a href="#external-secrets.io/v1.GitlabProvider">GitlabProvider</a>, 
<a href="#external-secrets.io/v1.KubernetesServer">KubernetesServer</a>, 
<a href="#external-secrets.io/v1.VaultProvider">VaultProvider</a>)
</p>
<p>
<p>Used to provide custom certificate authority (CA) certificates
for a secret store. The CAProvider points to a Secret or ConfigMap resource
that contains a PEM-encoded certificate.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#external-secrets.io/v1.CAProviderType">
CAProviderType
</a>
</em>
</td>
<td>
<p>The type of provider to use such as &ldquo;Secret&rdquo;, or &ldquo;ConfigMap&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the object located at the provider type.</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>The key where the CA certificate can be found in the Secret or ConfigMap.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The namespace the Provider type is in.
Can only be defined when used in a ClusterSecretStore.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.CAProviderType">CAProviderType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.CAProvider">CAProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ConfigMap&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Secret&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.CSMAuth">CSMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.CloudruSMProvider">CloudruSMProvider</a>)
</p>
<p>
<p>CSMAuth contains a secretRef for credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.CSMAuthSecretRef">
CSMAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.CSMAuthSecretRef">CSMAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.CSMAuth">CSMAuth</a>)
</p>
<p>
<p>CSMAuthSecretRef holds secret references for Cloud.ru credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessKeyIDSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The AccessKeyID is used for authentication</p>
</td>
</tr>
<tr>
<td>
<code>accessKeySecretSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The AccessKeySecret is used for authentication</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.CertAuth">CertAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.KubernetesAuth">KubernetesAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientCert</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>clientKey</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ChefAuth">ChefAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ChefProvider">ChefProvider</a>)
</p>
<p>
<p>ChefAuth contains a secretRef for credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.ChefAuthSecretRef">
ChefAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ChefAuthSecretRef">ChefAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ChefAuth">ChefAuth</a>)
</p>
<p>
<p>ChefAuthSecretRef holds secret references for chef server login credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>privateKeySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>SecretKey is the Signing Key in PEM format, used for authentication.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ChefProvider">ChefProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>ChefProvider configures a store to sync secrets using basic chef server connection credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.ChefAuth">
ChefAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against chef Server</p>
</td>
</tr>
<tr>
<td>
<code>username</code></br>
<em>
string
</em>
</td>
<td>
<p>UserName should be the user ID on the chef server</p>
</td>
</tr>
<tr>
<td>
<code>serverUrl</code></br>
<em>
string
</em>
</td>
<td>
<p>ServerURL is the chef server URL used to connect to. If using orgs you should include your org in the url and terminate the url with a &ldquo;/&rdquo;</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.CloudruSMProvider">CloudruSMProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>CloudruSMProvider configures a store to sync secrets using the Cloud.ru Secret Manager provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.CSMAuth">
CSMAuth
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>projectID</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectID is the project, which the secrets are stored in.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterExternalSecret">ClusterExternalSecret
</h3>
<p>
<p>ClusterExternalSecret is the Schema for the clusterexternalsecrets API.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterExternalSecretSpec">
ClusterExternalSecretSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>externalSecretSpec</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">
ExternalSecretSpec
</a>
</em>
</td>
<td>
<p>The spec for the ExternalSecrets to be created</p>
</td>
</tr>
<tr>
<td>
<code>externalSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the external secrets to be created.
Defaults to the name of the ClusterExternalSecret</p>
</td>
</tr>
<tr>
<td>
<code>externalSecretMetadata</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretMetadata">
ExternalSecretMetadata
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The metadata of the external secrets to be created</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The labels to select by to find the Namespaces to create the ExternalSecrets in.
Deprecated: Use NamespaceSelectors instead.</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelectors</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#*k8s.io/apimachinery/pkg/apis/meta/v1.labelselector--">
[]*k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A list of labels to select by to find the Namespaces to create the ExternalSecrets in. The selectors are ORed.</p>
</td>
</tr>
<tr>
<td>
<code>namespaces</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Choose namespaces by name. This field is ORed with anything that NamespaceSelectors ends up choosing.
Deprecated: Use NamespaceSelectors instead.</p>
</td>
</tr>
<tr>
<td>
<code>refreshTime</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>The time in which the controller should reconcile its objects and recheck namespaces for labels.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterExternalSecretStatus">
ClusterExternalSecretStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterExternalSecretConditionType">ClusterExternalSecretConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterExternalSecretStatusCondition">ClusterExternalSecretStatusCondition</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterExternalSecretNamespaceFailure">ClusterExternalSecretNamespaceFailure
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterExternalSecretStatus">ClusterExternalSecretStatus</a>)
</p>
<p>
<p>ClusterExternalSecretNamespaceFailure represents a failed namespace deployment and it&rsquo;s reason.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<p>Namespace is the namespace that failed when trying to apply an ExternalSecret</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reason is why the ExternalSecret failed to apply to the namespace</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterExternalSecretSpec">ClusterExternalSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterExternalSecret">ClusterExternalSecret</a>)
</p>
<p>
<p>ClusterExternalSecretSpec defines the desired state of ClusterExternalSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>externalSecretSpec</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">
ExternalSecretSpec
</a>
</em>
</td>
<td>
<p>The spec for the ExternalSecrets to be created</p>
</td>
</tr>
<tr>
<td>
<code>externalSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the external secrets to be created.
Defaults to the name of the ClusterExternalSecret</p>
</td>
</tr>
<tr>
<td>
<code>externalSecretMetadata</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretMetadata">
ExternalSecretMetadata
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The metadata of the external secrets to be created</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The labels to select by to find the Namespaces to create the ExternalSecrets in.
Deprecated: Use NamespaceSelectors instead.</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelectors</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#*k8s.io/apimachinery/pkg/apis/meta/v1.labelselector--">
[]*k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A list of labels to select by to find the Namespaces to create the ExternalSecrets in. The selectors are ORed.</p>
</td>
</tr>
<tr>
<td>
<code>namespaces</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Choose namespaces by name. This field is ORed with anything that NamespaceSelectors ends up choosing.
Deprecated: Use NamespaceSelectors instead.</p>
</td>
</tr>
<tr>
<td>
<code>refreshTime</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>The time in which the controller should reconcile its objects and recheck namespaces for labels.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterExternalSecretStatus">ClusterExternalSecretStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterExternalSecret">ClusterExternalSecret</a>)
</p>
<p>
<p>ClusterExternalSecretStatus defines the observed state of ClusterExternalSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>externalSecretName</code></br>
<em>
string
</em>
</td>
<td>
<p>ExternalSecretName is the name of the ExternalSecrets created by the ClusterExternalSecret</p>
</td>
</tr>
<tr>
<td>
<code>failedNamespaces</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterExternalSecretNamespaceFailure">
[]ClusterExternalSecretNamespaceFailure
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Failed namespaces are the namespaces that failed to apply an ExternalSecret</p>
</td>
</tr>
<tr>
<td>
<code>provisionedNamespaces</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProvisionedNamespaces are the namespaces where the ClusterExternalSecret has secrets</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterExternalSecretStatusCondition">
[]ClusterExternalSecretStatusCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterExternalSecretStatusCondition">ClusterExternalSecretStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterExternalSecretStatus">ClusterExternalSecretStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterExternalSecretConditionType">
ClusterExternalSecretConditionType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterSecretStore">ClusterSecretStore
</h3>
<p>
<p>ClusterSecretStore represents a secure external location for storing secrets, which can be referenced as part of <code>storeRef</code> fields.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreSpec">
SecretStoreSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>controller</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to select the correct ESO controller (think: ingress.ingressClassName)
The ESO controller is instantiated with a specific controller name and filters ES based on this property</p>
</td>
</tr>
<tr>
<td>
<code>provider</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreProvider">
SecretStoreProvider
</a>
</em>
</td>
<td>
<p>Used to configure the provider. Only one provider may be set</p>
</td>
</tr>
<tr>
<td>
<code>retrySettings</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreRetrySettings">
SecretStoreRetrySettings
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to configure http retries if failed</p>
</td>
</tr>
<tr>
<td>
<code>refreshInterval</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterSecretStoreCondition">
[]ClusterSecretStoreCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to constraint a ClusterSecretStore to specific namespaces. Relevant only to ClusterSecretStore</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreStatus">
SecretStoreStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ClusterSecretStoreCondition">ClusterSecretStoreCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreSpec">SecretStoreSpec</a>)
</p>
<p>
<p>ClusterSecretStoreCondition describes a condition by which to choose namespaces to process ExternalSecrets in
for a ClusterSecretStore instance.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespaceSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Choose namespace using a labelSelector</p>
</td>
</tr>
<tr>
<td>
<code>namespaces</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Choose namespaces by name</p>
</td>
</tr>
<tr>
<td>
<code>namespaceRegexes</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Choose namespaces by using regex matching</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ConfigMapReference">ConfigMapReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GCPWorkloadIdentityFederation">GCPWorkloadIdentityFederation</a>)
</p>
<p>
<p>ConfigMapReference holds the details of a configmap.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>name of the configmap.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<p>namespace in which the configmap exists. If empty, configmap will looked up in local namespace.</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>key name holding the external account credential config.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ConjurAPIKey">ConjurAPIKey
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ConjurAuth">ConjurAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>account</code></br>
<em>
string
</em>
</td>
<td>
<p>Account is the Conjur organization account name.</p>
</td>
</tr>
<tr>
<td>
<code>userRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>A reference to a specific &lsquo;key&rsquo; containing the Conjur username
within a Secret resource. In some instances, <code>key</code> is a required field.</p>
</td>
</tr>
<tr>
<td>
<code>apiKeyRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>A reference to a specific &lsquo;key&rsquo; containing the Conjur API key
within a Secret resource. In some instances, <code>key</code> is a required field.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ConjurAuth">ConjurAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ConjurProvider">ConjurProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apikey</code></br>
<em>
<a href="#external-secrets.io/v1.ConjurAPIKey">
ConjurAPIKey
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Authenticates with Conjur using an API key.</p>
</td>
</tr>
<tr>
<td>
<code>jwt</code></br>
<em>
<a href="#external-secrets.io/v1.ConjurJWT">
ConjurJWT
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Jwt enables JWT authentication using Kubernetes service account tokens.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ConjurJWT">ConjurJWT
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ConjurAuth">ConjurAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>account</code></br>
<em>
string
</em>
</td>
<td>
<p>Account is the Conjur organization account name.</p>
</td>
</tr>
<tr>
<td>
<code>serviceID</code></br>
<em>
string
</em>
</td>
<td>
<p>The conjur authn jwt webservice id</p>
</td>
</tr>
<tr>
<td>
<code>hostId</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional HostID for JWT authentication. This may be used depending
on how the Conjur JWT authenticator policy is configured.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional SecretRef that refers to a key in a Secret resource containing JWT token to
authenticate with Conjur using the JWT authentication method.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional ServiceAccountRef specifies the Kubernetes service account for which to request
a token for with the <code>TokenRequest</code> API.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ConjurProvider">ConjurProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL is the endpoint of the Conjur instance.</p>
</td>
</tr>
<tr>
<td>
<code>caBundle</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CABundle is a PEM encoded CA bundle that will be used to validate the Conjur server certificate.</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.CAProvider">
CAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to provide custom certificate authority (CA) certificates
for a secret store. The CAProvider points to a Secret or ConfigMap resource
that contains a PEM-encoded certificate.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.ConjurAuth">
ConjurAuth
</a>
</em>
</td>
<td>
<p>Defines authentication settings for connecting to Conjur.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.DelineaProvider">DelineaProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>See <a href="https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go">https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go</a>.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientId</code></br>
<em>
<a href="#external-secrets.io/v1.DelineaProviderSecretRef">
DelineaProviderSecretRef
</a>
</em>
</td>
<td>
<p>ClientID is the non-secret part of the credential.</p>
</td>
</tr>
<tr>
<td>
<code>clientSecret</code></br>
<em>
<a href="#external-secrets.io/v1.DelineaProviderSecretRef">
DelineaProviderSecretRef
</a>
</em>
</td>
<td>
<p>ClientSecret is the secret part of the credential.</p>
</td>
</tr>
<tr>
<td>
<code>tenant</code></br>
<em>
string
</em>
</td>
<td>
<p>Tenant is the chosen hostname / site name.</p>
</td>
</tr>
<tr>
<td>
<code>urlTemplate</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>URLTemplate
If unset, defaults to &ldquo;https://%s.secretsvaultcloud.%s/v1/%s%s&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>tld</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLD is based on the server location that was chosen during provisioning.
If unset, defaults to &ldquo;com&rdquo;.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.DelineaProviderSecretRef">DelineaProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.DelineaProvider">DelineaProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value can be specified directly to set a value without using a secret.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef references a key in a secret that will be used as value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.Device42Auth">Device42Auth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.Device42Provider">Device42Provider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.Device42SecretRef">
Device42SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.Device42Provider">Device42Provider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Device42Provider configures a store to sync secrets with a Device42 instance.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>host</code></br>
<em>
string
</em>
</td>
<td>
<p>URL configures the Device42 instance URL.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.Device42Auth">
Device42Auth
</a>
</em>
</td>
<td>
<p>Auth configures how secret-manager authenticates with a Device42 instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.Device42SecretRef">Device42SecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.Device42Auth">Device42Auth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>credentials</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Username / Password is used for authentication.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.DopplerAuth">DopplerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.DopplerProvider">DopplerProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.DopplerAuthSecretRef">
DopplerAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.DopplerAuthSecretRef">DopplerAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.DopplerAuth">DopplerAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>dopplerToken</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The DopplerToken is used for authentication.
See <a href="https://docs.doppler.com/reference/api#authentication">https://docs.doppler.com/reference/api#authentication</a> for auth token types.
The Key attribute defaults to dopplerToken if not specified.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.DopplerProvider">DopplerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>DopplerProvider configures a store to sync secrets using the Doppler provider.
Project and Config are required if not using a Service Token.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.DopplerAuth">
DopplerAuth
</a>
</em>
</td>
<td>
<p>Auth configures how the Operator authenticates with the Doppler API</p>
</td>
</tr>
<tr>
<td>
<code>project</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Doppler project (required if not using a Service Token)</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Doppler config (required if not using a Service Token)</p>
</td>
</tr>
<tr>
<td>
<code>nameTransformer</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Environment variable compatible name transforms that change secret names to a different format</p>
</td>
</tr>
<tr>
<td>
<code>format</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Format enables the downloading of secrets as a file (string)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecret">ExternalSecret
</h3>
<p>
<p>ExternalSecret is the Schema for the external-secrets API.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">
ExternalSecretSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>secretStoreRef</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreRef">
SecretStoreRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretTarget">
ExternalSecretTarget
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>refreshPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRefreshPolicy">
ExternalSecretRefreshPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RefreshPolicy determines how the ExternalSecret should be refreshed:
- CreatedOnce: Creates the Secret only if it does not exist and does not update it thereafter
- Periodic: Synchronizes the Secret from the external source at regular intervals specified by refreshInterval.
No periodic updates occur if refreshInterval is 0.
- OnChange: Only synchronizes the Secret when the ExternalSecret&rsquo;s metadata or specification changes</p>
</td>
</tr>
<tr>
<td>
<code>refreshInterval</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>RefreshInterval is the amount of time before the values are read again from the SecretStore provider,
specified as Golang Duration strings.
Valid time units are &ldquo;ns&rdquo;, &ldquo;us&rdquo; (or &ldquo;µs&rdquo;), &ldquo;ms&rdquo;, &ldquo;s&rdquo;, &ldquo;m&rdquo;, &ldquo;h&rdquo;
Example values: &ldquo;1h&rdquo;, &ldquo;2h30m&rdquo;, &ldquo;10s&rdquo;
May be set to zero to fetch and create it once. Defaults to 1h.</p>
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretData">
[]ExternalSecretData
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Data defines the connection between the Kubernetes Secret keys and the Provider data</p>
</td>
</tr>
<tr>
<td>
<code>dataFrom</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretDataFromRemoteRef">
[]ExternalSecretDataFromRemoteRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DataFrom is used to fetch all properties from a specific Provider data
If multiple entries are specified, the Secret keys are merged in the specified order</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretStatus">
ExternalSecretStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretConditionType">ExternalSecretConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretStatusCondition">ExternalSecretStatusCondition</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Deleted&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Ready&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretConversionStrategy">ExternalSecretConversionStrategy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef</a>, 
<a href="#external-secrets.io/v1.ExternalSecretFind">ExternalSecretFind</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Default&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Unicode&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretCreationPolicy">ExternalSecretCreationPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTarget">ExternalSecretTarget</a>)
</p>
<p>
<p>ExternalSecretCreationPolicy defines rules on how to create the resulting Secret.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Merge&#34;</p></td>
<td><p>Merge does not create the Secret, but merges the data fields to the Secret.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>None does not create a Secret (future use with injector).</p>
</td>
</tr><tr><td><p>&#34;Orphan&#34;</p></td>
<td><p>Orphan creates the Secret and does not set the ownerReference.
I.e. it will be orphaned after the deletion of the ExternalSecret.</p>
</td>
</tr><tr><td><p>&#34;Owner&#34;</p></td>
<td><p>Owner creates the Secret and sets .metadata.ownerReferences to the ExternalSecret resource.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretData">ExternalSecretData
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
<p>ExternalSecretData defines the connection between the Kubernetes Secret key (spec.data.<key>) and the Provider data.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretKey</code></br>
<em>
string
</em>
</td>
<td>
<p>The key in the Kubernetes Secret to store the value.</p>
</td>
</tr>
<tr>
<td>
<code>remoteRef</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretDataRemoteRef">
ExternalSecretDataRemoteRef
</a>
</em>
</td>
<td>
<p>RemoteRef points to the remote secret and defines
which secret (version/property/..) to fetch.</p>
</td>
</tr>
<tr>
<td>
<code>sourceRef</code></br>
<em>
<a href="#external-secrets.io/v1.StoreSourceRef">
StoreSourceRef
</a>
</em>
</td>
<td>
<p>SourceRef allows you to override the source
from which the value will be pulled.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>extract</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretDataRemoteRef">
ExternalSecretDataRemoteRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to extract multiple key/value pairs from one secret
Note: Extract does not support sourceRef.Generator or sourceRef.GeneratorRef.</p>
</td>
</tr>
<tr>
<td>
<code>find</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretFind">
ExternalSecretFind
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to find secrets based on tags or regular expressions
Note: Find does not support sourceRef.Generator or sourceRef.GeneratorRef.</p>
</td>
</tr>
<tr>
<td>
<code>rewrite</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRewrite">
[]ExternalSecretRewrite
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to rewrite secret Keys after getting them from the secret Provider
Multiple Rewrite operations can be provided. They are applied in a layered order (first to last)</p>
</td>
</tr>
<tr>
<td>
<code>sourceRef</code></br>
<em>
<a href="#external-secrets.io/v1.StoreGeneratorSourceRef">
StoreGeneratorSourceRef
</a>
</em>
</td>
<td>
<p>SourceRef points to a store or generator
which contains secret values ready to use.
Use this in combination with Extract or Find pull values out of
a specific SecretStore.
When sourceRef points to a generator Extract or Find is not supported.
The generator returns a static map of values</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretData">ExternalSecretData</a>, 
<a href="#external-secrets.io/v1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
</p>
<p>
<p>ExternalSecretDataRemoteRef defines Provider data location.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>Key is the key used in the Provider, mandatory</p>
</td>
</tr>
<tr>
<td>
<code>metadataPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretMetadataPolicy">
ExternalSecretMetadataPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Policy for fetching tags/labels from provider secrets, possible options are Fetch, None. Defaults to None</p>
</td>
</tr>
<tr>
<td>
<code>property</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to select a specific property of the Provider value (if a map), if supported</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to select a specific version of the Provider value, if supported</p>
</td>
</tr>
<tr>
<td>
<code>conversionStrategy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretConversionStrategy">
ExternalSecretConversionStrategy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define a conversion Strategy</p>
</td>
</tr>
<tr>
<td>
<code>decodingStrategy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretDecodingStrategy">
ExternalSecretDecodingStrategy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define a decoding Strategy</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretDecodingStrategy">ExternalSecretDecodingStrategy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef</a>, 
<a href="#external-secrets.io/v1.ExternalSecretFind">ExternalSecretFind</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Auto&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Base64&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Base64URL&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretDeletionPolicy">ExternalSecretDeletionPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTarget">ExternalSecretTarget</a>)
</p>
<p>
<p>ExternalSecretDeletionPolicy defines rules on how to delete the resulting Secret.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Delete&#34;</p></td>
<td><p>Delete deletes the secret if all provider secrets are deleted.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.</p>
</td>
</tr><tr><td><p>&#34;Merge&#34;</p></td>
<td><p>Merge removes keys in the secret, but not the secret itself.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.</p>
</td>
</tr><tr><td><p>&#34;Retain&#34;</p></td>
<td><p>Retain will retain the secret if all provider secrets have been deleted.
If a provider secret does not exist the ExternalSecret gets into the
SecretSyncedError status.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretFind">ExternalSecretFind
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>A root path to start the find operations.</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
<a href="#external-secrets.io/v1.FindName">
FindName
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Finds secrets based on the name.</p>
</td>
</tr>
<tr>
<td>
<code>tags</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Find secrets based on tags.</p>
</td>
</tr>
<tr>
<td>
<code>conversionStrategy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretConversionStrategy">
ExternalSecretConversionStrategy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define a conversion Strategy</p>
</td>
</tr>
<tr>
<td>
<code>decodingStrategy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretDecodingStrategy">
ExternalSecretDecodingStrategy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define a decoding Strategy</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretMetadata">ExternalSecretMetadata
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterExternalSecretSpec">ClusterExternalSecretSpec</a>)
</p>
<p>
<p>ExternalSecretMetadata defines metadata fields for the ExternalSecret generated by the ClusterExternalSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretMetadataPolicy">ExternalSecretMetadataPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Fetch&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRefreshPolicy">ExternalSecretRefreshPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;CreatedOnce&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;OnChange&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Periodic&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewrite">ExternalSecretRewrite
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>merge</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMerge">
ExternalSecretRewriteMerge
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to merge key/values in one single Secret
The resulting key will contain all values from the specified secrets</p>
</td>
</tr>
<tr>
<td>
<code>regexp</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteRegexp">
ExternalSecretRewriteRegexp
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to rewrite with regular expressions.
The resulting key will be the output of a regexp.ReplaceAll operation.</p>
</td>
</tr>
<tr>
<td>
<code>transform</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteTransform">
ExternalSecretRewriteTransform
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to apply string transformation on the secrets.
The resulting key will be the output of the template applied by the operation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteMerge">ExternalSecretRewriteMerge
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewrite">ExternalSecretRewrite</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>into</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define the target key of the merge operation.
Required if strategy is JSON. Ignored otherwise.</p>
</td>
</tr>
<tr>
<td>
<code>priority</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define key priority in conflict resolution.</p>
</td>
</tr>
<tr>
<td>
<code>priorityPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMergePriorityPolicy">
ExternalSecretRewriteMergePriorityPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define the policy when a key in the priority list does not exist in the input.</p>
</td>
</tr>
<tr>
<td>
<code>conflictPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMergeConflictPolicy">
ExternalSecretRewriteMergeConflictPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define the policy to use in conflict resolution.</p>
</td>
</tr>
<tr>
<td>
<code>strategy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMergeStrategy">
ExternalSecretRewriteMergeStrategy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define the strategy to use in the merge operation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteMergeConflictPolicy">ExternalSecretRewriteMergeConflictPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMerge">ExternalSecretRewriteMerge</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Error&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Ignore&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteMergePriorityPolicy">ExternalSecretRewriteMergePriorityPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMerge">ExternalSecretRewriteMerge</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;IgnoreNotFound&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Strict&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteMergeStrategy">ExternalSecretRewriteMergeStrategy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMerge">ExternalSecretRewriteMerge</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Extract&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;JSON&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteRegexp">ExternalSecretRewriteRegexp
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewrite">ExternalSecretRewrite</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>source</code></br>
<em>
string
</em>
</td>
<td>
<p>Used to define the regular expression of a re.Compiler.</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
string
</em>
</td>
<td>
<p>Used to define the target pattern of a ReplaceAll operation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteTransform">ExternalSecretRewriteTransform
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewrite">ExternalSecretRewrite</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>template</code></br>
<em>
string
</em>
</td>
<td>
<p>Used to define the template to apply on the secret name.
<code>.value</code> will specify the secret name in the template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterExternalSecretSpec">ClusterExternalSecretSpec</a>, 
<a href="#external-secrets.io/v1.ExternalSecret">ExternalSecret</a>)
</p>
<p>
<p>ExternalSecretSpec defines the desired state of ExternalSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretStoreRef</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreRef">
SecretStoreRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretTarget">
ExternalSecretTarget
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>refreshPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretRefreshPolicy">
ExternalSecretRefreshPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RefreshPolicy determines how the ExternalSecret should be refreshed:
- CreatedOnce: Creates the Secret only if it does not exist and does not update it thereafter
- Periodic: Synchronizes the Secret from the external source at regular intervals specified by refreshInterval.
No periodic updates occur if refreshInterval is 0.
- OnChange: Only synchronizes the Secret when the ExternalSecret&rsquo;s metadata or specification changes</p>
</td>
</tr>
<tr>
<td>
<code>refreshInterval</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>RefreshInterval is the amount of time before the values are read again from the SecretStore provider,
specified as Golang Duration strings.
Valid time units are &ldquo;ns&rdquo;, &ldquo;us&rdquo; (or &ldquo;µs&rdquo;), &ldquo;ms&rdquo;, &ldquo;s&rdquo;, &ldquo;m&rdquo;, &ldquo;h&rdquo;
Example values: &ldquo;1h&rdquo;, &ldquo;2h30m&rdquo;, &ldquo;10s&rdquo;
May be set to zero to fetch and create it once. Defaults to 1h.</p>
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretData">
[]ExternalSecretData
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Data defines the connection between the Kubernetes Secret keys and the Provider data</p>
</td>
</tr>
<tr>
<td>
<code>dataFrom</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretDataFromRemoteRef">
[]ExternalSecretDataFromRemoteRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DataFrom is used to fetch all properties from a specific Provider data
If multiple entries are specified, the Secret keys are merged in the specified order</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretStatus">ExternalSecretStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecret">ExternalSecret</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>refreshTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>refreshTime is the time and date the external secret was fetched and
the target secret updated</p>
</td>
</tr>
<tr>
<td>
<code>syncedResourceVersion</code></br>
<em>
string
</em>
</td>
<td>
<p>SyncedResourceVersion keeps track of the last synced version</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretStatusCondition">
[]ExternalSecretStatusCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>binding</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>Binding represents a servicebinding.io Provisioned Service reference to the secret</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretStatusCondition">ExternalSecretStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretStatus">ExternalSecretStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretConditionType">
ExternalSecretConditionType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretTarget">ExternalSecretTarget
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
<p>ExternalSecretTarget defines the Kubernetes Secret to be created
There can be only one target per ExternalSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the Secret resource to be managed.
Defaults to the .metadata.name of the ExternalSecret resource</p>
</td>
</tr>
<tr>
<td>
<code>creationPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretCreationPolicy">
ExternalSecretCreationPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CreationPolicy defines rules on how to create the resulting Secret.
Defaults to &ldquo;Owner&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>deletionPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretDeletionPolicy">
ExternalSecretDeletionPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DeletionPolicy defines rules on how to delete the resulting Secret.
Defaults to &ldquo;Retain&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>template</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretTemplate">
ExternalSecretTemplate
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Template defines a blueprint for the created Secret resource.</p>
</td>
</tr>
<tr>
<td>
<code>immutable</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Immutable defines if the final secret will be immutable</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretTemplate">ExternalSecretTemplate
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTarget">ExternalSecretTarget</a>)
</p>
<p>
<p>ExternalSecretTemplate defines a blueprint for the created Secret resource.
we can not use native corev1.Secret, it will have empty ObjectMeta values: <a href="https://github.com/kubernetes-sigs/controller-tools/issues/448">https://github.com/kubernetes-sigs/controller-tools/issues/448</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#secrettype-v1-core">
Kubernetes core/v1.SecretType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>engineVersion</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateEngineVersion">
TemplateEngineVersion
</a>
</em>
</td>
<td>
<p>EngineVersion specifies the template engine version
that should be used to compile/execute the
template specified in .data and .templateFrom[].</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretTemplateMetadata">
ExternalSecretTemplateMetadata
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>mergePolicy</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateMergePolicy">
TemplateMergePolicy
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>templateFrom</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateFrom">
[]TemplateFrom
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretTemplateMetadata">ExternalSecretTemplateMetadata
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
<p>ExternalSecretTemplateMetadata defines metadata fields for the Secret blueprint.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>annotations</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>finalizers</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretValidator">ExternalSecretValidator
</h3>
<p>
</p>
<h3 id="external-secrets.io/v1.FakeProvider">FakeProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>FakeProvider configures a fake provider that returns static values.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#external-secrets.io/v1.FakeProviderData">
[]FakeProviderData
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>validationResult</code></br>
<em>
<a href="#external-secrets.io/v1.ValidationResult">
ValidationResult
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.FakeProviderData">FakeProviderData
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.FakeProvider">FakeProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.FetchingPolicy">FetchingPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.YandexCertificateManagerProvider">YandexCertificateManagerProvider</a>, 
<a href="#external-secrets.io/v1.YandexLockboxProvider">YandexLockboxProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>byID</code></br>
<em>
<a href="#external-secrets.io/v1.ByID">
ByID
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>byName</code></br>
<em>
<a href="#external-secrets.io/v1.ByName">
ByName
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.FindName">FindName
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretFind">ExternalSecretFind</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>regexp</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Finds secrets base</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.FortanixProvider">FortanixProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiUrl</code></br>
<em>
string
</em>
</td>
<td>
<p>APIURL is the URL of SDKMS API. Defaults to <code>sdkms.fortanix.com</code>.</p>
</td>
</tr>
<tr>
<td>
<code>apiKey</code></br>
<em>
<a href="#external-secrets.io/v1.FortanixProviderSecretRef">
FortanixProviderSecretRef
</a>
</em>
</td>
<td>
<p>APIKey is the API token to access SDKMS Applications.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.FortanixProviderSecretRef">FortanixProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.FortanixProvider">FortanixProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>SecretRef is a reference to a secret containing the SDKMS API Key.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GCPSMAuth">GCPSMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GCPSMProvider">GCPSMProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.GCPSMAuthSecretRef">
GCPSMAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>workloadIdentity</code></br>
<em>
<a href="#external-secrets.io/v1.GCPWorkloadIdentity">
GCPWorkloadIdentity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>workloadIdentityFederation</code></br>
<em>
<a href="#external-secrets.io/v1.GCPWorkloadIdentityFederation">
GCPWorkloadIdentityFederation
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GCPSMAuthSecretRef">GCPSMAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GCPSMAuth">GCPSMAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretAccessKeySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The SecretAccessKey is used for authentication</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GCPSMProvider">GCPSMProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>GCPSMProvider Configures a store to sync secrets using the GCP Secret Manager provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.GCPSMAuth">
GCPSMAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines the information necessary to authenticate against GCP</p>
</td>
</tr>
<tr>
<td>
<code>projectID</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectID project where secret is located</p>
</td>
</tr>
<tr>
<td>
<code>location</code></br>
<em>
string
</em>
</td>
<td>
<p>Location optionally defines a location for a secret</p>
</td>
</tr>
<tr>
<td>
<code>secretVersionSelectionPolicy</code></br>
<em>
<a href="#external-secrets.io/v1.SecretVersionSelectionPolicy">
SecretVersionSelectionPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretVersionSelectionPolicy specifies how the provider selects a secret version
when &ldquo;latest&rdquo; is disabled or destroyed.
Possible values are:
- LatestOrFail: the provider always uses &ldquo;latest&rdquo;, or fails if that version is disabled/destroyed.
- LatestOrFetch: the provider falls back to fetching the latest version if the version is DESTROYED or DISABLED</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GCPWorkloadIdentity">GCPWorkloadIdentity
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GCPSMAuth">GCPSMAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>clusterLocation</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterLocation is the location of the cluster
If not specified, it fetches information from the metadata server</p>
</td>
</tr>
<tr>
<td>
<code>clusterName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterName is the name of the cluster
If not specified, it fetches information from the metadata server</p>
</td>
</tr>
<tr>
<td>
<code>clusterProjectID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClusterProjectID is the project ID of the cluster
If not specified, it fetches information from the metadata server</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GCPWorkloadIdentityFederation">GCPWorkloadIdentityFederation
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GCPSMAuth">GCPSMAuth</a>)
</p>
<p>
<p>GCPWorkloadIdentityFederation holds the configurations required for generating federated access tokens.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>credConfig</code></br>
<em>
<a href="#external-secrets.io/v1.ConfigMapReference">
ConfigMapReference
</a>
</em>
</td>
<td>
<p>credConfig holds the configmap reference containing the GCP external account credential configuration in JSON format and the key name containing the json data.
For using Kubernetes cluster as the identity provider, use serviceAccountRef instead. Operators mounted serviceaccount token cannot be used as the token source, instead
serviceAccountRef must be used by providing operators service account details.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<p>serviceAccountRef is the reference to the kubernetes ServiceAccount to be used for obtaining the tokens,
when Kubernetes is configured as provider in workload identity pool.</p>
</td>
</tr>
<tr>
<td>
<code>awsSecurityCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.AwsCredentialsConfig">
AwsCredentialsConfig
</a>
</em>
</td>
<td>
<p>awsSecurityCredentials is for configuring AWS region and credentials to use for obtaining the access token,
when using the AWS metadata server is not an option.</p>
</td>
</tr>
<tr>
<td>
<code>audience</code></br>
<em>
string
</em>
</td>
<td>
<p>audience is the Secure Token Service (STS) audience which contains the resource name for the workload identity pool and the provider identifier in that pool.
If specified, Audience found in the external account credential config will be overridden with the configured value.
audience must be provided when serviceAccountRef or awsSecurityCredentials is configured.</p>
</td>
</tr>
<tr>
<td>
<code>externalTokenEndpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>externalTokenEndpoint is the endpoint explicitly set up to provide tokens, which will be matched against the
credential_source.url in the provided credConfig. This field is merely to double-check the external token source
URL is having the expected value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GcpIamAuthCredentials">GcpIamAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountKeyFilePath</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GcpIdTokenAuthCredentials">GcpIdTokenAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GeneratorRef">GeneratorRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.StoreGeneratorSourceRef">StoreGeneratorSourceRef</a>, 
<a href="#external-secrets.io/v1.StoreSourceRef">StoreSourceRef</a>)
</p>
<p>
<p>GeneratorRef points to a generator custom resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify the apiVersion of the generator resource</p>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify the Kind of the generator resource</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Specify the name of the generator resource</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GenericStore">GenericStore
</h3>
<p>
<p>GenericStore is a common interface for interacting with ClusterSecretStore
or a namespaced SecretStore.</p>
</p>
<h3 id="external-secrets.io/v1.GenericStoreValidator">GenericStoreValidator
</h3>
<p>
</p>
<h3 id="external-secrets.io/v1.GithubAppAuth">GithubAppAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GithubProvider">GithubProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>privateKey</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GithubProvider">GithubProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures a store to push secrets to Github Actions.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL configures the Github instance URL. Defaults to <a href="https://github.com/">https://github.com/</a>.</p>
</td>
</tr>
<tr>
<td>
<code>uploadURL</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Upload URL for enterprise instances. Default to URL.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.GithubAppAuth">
GithubAppAuth
</a>
</em>
</td>
<td>
<p>auth configures how secret-manager authenticates with a Github instance.</p>
</td>
</tr>
<tr>
<td>
<code>appID</code></br>
<em>
int64
</em>
</td>
<td>
<p>appID specifies the Github APP that will be used to authenticate the client</p>
</td>
</tr>
<tr>
<td>
<code>installationID</code></br>
<em>
int64
</em>
</td>
<td>
<p>installationID specifies the Github APP installation that will be used to authenticate the client</p>
</td>
</tr>
<tr>
<td>
<code>organization</code></br>
<em>
string
</em>
</td>
<td>
<p>organization will be used to fetch secrets from the Github organization</p>
</td>
</tr>
<tr>
<td>
<code>repository</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>repository will be used to fetch secrets from the Github repository within an organization</p>
</td>
</tr>
<tr>
<td>
<code>environment</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>environment will be used to fetch secrets from a particular environment within a github repository</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GitlabAuth">GitlabAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GitlabProvider">GitlabProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>SecretRef</code></br>
<em>
<a href="#external-secrets.io/v1.GitlabSecretRef">
GitlabSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GitlabProvider">GitlabProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures a store to sync secrets with a GitLab instance.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL configures the GitLab instance URL. Defaults to <a href="https://gitlab.com/">https://gitlab.com/</a>.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.GitlabAuth">
GitlabAuth
</a>
</em>
</td>
<td>
<p>Auth configures how secret-manager authenticates with a GitLab instance.</p>
</td>
</tr>
<tr>
<td>
<code>projectID</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectID specifies a project where secrets are located.</p>
</td>
</tr>
<tr>
<td>
<code>inheritFromGroups</code></br>
<em>
bool
</em>
</td>
<td>
<p>InheritFromGroups specifies whether parent groups should be discovered and checked for secrets.</p>
</td>
</tr>
<tr>
<td>
<code>groupIDs</code></br>
<em>
[]string
</em>
</td>
<td>
<p>GroupIDs specify, which gitlab groups to pull secrets from. Group secrets are read from left to right followed by the project variables.</p>
</td>
</tr>
<tr>
<td>
<code>environment</code></br>
<em>
string
</em>
</td>
<td>
<p>Environment environment_scope of gitlab CI/CD variables (Please see <a href="https://docs.gitlab.com/ee/ci/environments/#create-a-static-environment">https://docs.gitlab.com/ee/ci/environments/#create-a-static-environment</a> on how to create environments)</p>
</td>
</tr>
<tr>
<td>
<code>caBundle</code></br>
<em>
[]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>Base64 encoded certificate for the GitLab server sdk. The sdk MUST run with HTTPS to make sure no MITM attack
can be performed.</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.CAProvider">
CAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>see: <a href="https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider">https://external-secrets.io/latest/spec/#external-secrets.io/v1alpha1.CAProvider</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.GitlabSecretRef">GitlabSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GitlabAuth">GitlabAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessToken</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>AccessToken is used for authentication.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.IBMAuth">IBMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.IBMProvider">IBMProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.IBMAuthSecretRef">
IBMAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>containerAuth</code></br>
<em>
<a href="#external-secrets.io/v1.IBMAuthContainerAuth">
IBMAuthContainerAuth
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.IBMAuthContainerAuth">IBMAuthContainerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.IBMAuth">IBMAuth</a>)
</p>
<p>
<p>IBM Container-based auth with IAM Trusted Profile.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>profile</code></br>
<em>
string
</em>
</td>
<td>
<p>the IBM Trusted Profile</p>
</td>
</tr>
<tr>
<td>
<code>tokenLocation</code></br>
<em>
string
</em>
</td>
<td>
<p>Location the token is mounted on the pod</p>
</td>
</tr>
<tr>
<td>
<code>iamEndpoint</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.IBMAuthSecretRef">IBMAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.IBMAuth">IBMAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretApiKeySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The SecretAccessKey is used for authentication</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.IBMProvider">IBMProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures an store to sync secrets using a IBM Cloud Secrets Manager
backend.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.IBMAuth">
IBMAuth
</a>
</em>
</td>
<td>
<p>Auth configures how secret-manager authenticates with the IBM secrets manager.</p>
</td>
</tr>
<tr>
<td>
<code>serviceUrl</code></br>
<em>
string
</em>
</td>
<td>
<p>ServiceURL is the Endpoint URL that is specific to the Secrets Manager service instance</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.InfisicalAuth">InfisicalAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalProvider">InfisicalProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>universalAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.UniversalAuthCredentials">
UniversalAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>azureAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.AzureAuthCredentials">
AzureAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>gcpIdTokenAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.GcpIdTokenAuthCredentials">
GcpIdTokenAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>gcpIamAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.GcpIamAuthCredentials">
GcpIamAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>jwtAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.JwtAuthCredentials">
JwtAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>ldapAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.LdapAuthCredentials">
LdapAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>ociAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.OciAuthCredentials">
OciAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>kubernetesAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.KubernetesAuthCredentials">
KubernetesAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>awsAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.AwsAuthCredentials">
AwsAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tokenAuthCredentials</code></br>
<em>
<a href="#external-secrets.io/v1.TokenAuthCredentials">
TokenAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.InfisicalProvider">InfisicalProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>InfisicalProvider configures a store to sync secrets using the Infisical provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.InfisicalAuth">
InfisicalAuth
</a>
</em>
</td>
<td>
<p>Auth configures how the Operator authenticates with the Infisical API</p>
</td>
</tr>
<tr>
<td>
<code>secretsScope</code></br>
<em>
<a href="#external-secrets.io/v1.MachineIdentityScopeInWorkspace">
MachineIdentityScopeInWorkspace
</a>
</em>
</td>
<td>
<p>SecretsScope defines the scope of the secrets within the workspace</p>
</td>
</tr>
<tr>
<td>
<code>hostAPI</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HostAPI specifies the base URL of the Infisical API. If not provided, it defaults to &ldquo;<a href="https://app.infisical.com/api&quot;">https://app.infisical.com/api&rdquo;</a>.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.IntegrationInfo">IntegrationInfo
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OnePasswordSDKProvider">OnePasswordSDKProvider</a>)
</p>
<p>
<p>IntegrationInfo specifies the name and version of the integration built using the 1Password Go SDK.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name defaults to &ldquo;1Password SDK&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version defaults to &ldquo;v1.0.0&rdquo;.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.JwtAuthCredentials">JwtAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>jwt</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.KeeperSecurityProvider">KeeperSecurityProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>KeeperSecurityProvider Configures a store to sync secrets using Keeper Security.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>authRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>folderID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.KubernetesAuth">KubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.KubernetesProvider">KubernetesProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>cert</code></br>
<em>
<a href="#external-secrets.io/v1.CertAuth">
CertAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>has both clientCert and clientKey as secretKeySelector</p>
</td>
</tr>
<tr>
<td>
<code>token</code></br>
<em>
<a href="#external-secrets.io/v1.TokenAuth">
TokenAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>use static token to authenticate with</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>points to a service account that should be used for authentication</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.KubernetesAuthCredentials">KubernetesAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountTokenPath</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.KubernetesProvider">KubernetesProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures a store to sync secrets with a Kubernetes instance.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#external-secrets.io/v1.KubernetesServer">
KubernetesServer
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>configures the Kubernetes server Address.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.KubernetesAuth">
KubernetesAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth configures how secret-manager authenticates with a Kubernetes instance.</p>
</td>
</tr>
<tr>
<td>
<code>authRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to a secret that contains the auth information.</p>
</td>
</tr>
<tr>
<td>
<code>remoteNamespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Remote namespace to fetch the secrets from</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.KubernetesServer">KubernetesServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.KubernetesProvider">KubernetesProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>configures the Kubernetes server Address.</p>
</td>
</tr>
<tr>
<td>
<code>caBundle</code></br>
<em>
[]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>CABundle is a base64-encoded CA certificate</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.CAProvider">
CAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>see: <a href="https://external-secrets.io/v0.4.1/spec/#external-secrets.io/v1alpha1.CAProvider">https://external-secrets.io/v0.4.1/spec/#external-secrets.io/v1alpha1.CAProvider</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.LdapAuthCredentials">LdapAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ldapPassword</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ldapUsername</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.MachineIdentityScopeInWorkspace">MachineIdentityScopeInWorkspace
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalProvider">InfisicalProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretsPath</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretsPath specifies the path to the secrets within the workspace. Defaults to &ldquo;/&rdquo; if not provided.</p>
</td>
</tr>
<tr>
<td>
<code>recursive</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Recursive indicates whether the secrets should be fetched recursively. Defaults to false if not provided.</p>
</td>
</tr>
<tr>
<td>
<code>environmentSlug</code></br>
<em>
string
</em>
</td>
<td>
<p>EnvironmentSlug is the required slug identifier for the environment.</p>
</td>
</tr>
<tr>
<td>
<code>projectSlug</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectSlug is the required slug identifier for the project.</p>
</td>
</tr>
<tr>
<td>
<code>expandSecretReferences</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExpandSecretReferences indicates whether secret references should be expanded. Defaults to true if not provided.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.MaintenanceStatus">MaintenanceStatus
(<code>bool</code> alias)</p></h3>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>true</p></td>
<td></td>
</tr><tr><td><p>false</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.NTLMProtocol">NTLMProtocol
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AuthorizationProtocol">AuthorizationProtocol</a>)
</p>
<p>
<p>NTLMProtocol contains the NTLM-specific configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>usernameSecret</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>passwordSecret</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.NoSecretError">NoSecretError
</h3>
<p>
<p>NoSecretError shall be returned when a GetSecret can not find the
desired secret. This is used for deletionPolicy.</p>
</p>
<h3 id="external-secrets.io/v1.NotModifiedError">NotModifiedError
</h3>
<p>
<p>NotModifiedError to signal that the webhook received no changes,
and it should just return without doing anything.</p>
</p>
<h3 id="external-secrets.io/v1.OciAuthCredentials">OciAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>identityId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>privateKey</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>privateKeyPassphrase</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>fingerprint</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>userId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tenancyId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OnboardbaseAuthSecretRef">OnboardbaseAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OnboardbaseProvider">OnboardbaseProvider</a>)
</p>
<p>
<p>OnboardbaseAuthSecretRef holds secret references for onboardbase API Key credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiKeyRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>OnboardbaseAPIKey is the APIKey generated by an admin account.
It is used to recognize and authorize access to a project and environment within onboardbase</p>
</td>
</tr>
<tr>
<td>
<code>passcodeRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>OnboardbasePasscode is the passcode attached to the API Key</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OnboardbaseProvider">OnboardbaseProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>OnboardbaseProvider configures a store to sync secrets using the Onboardbase provider.
Project and Config are required if not using a Service Token.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.OnboardbaseAuthSecretRef">
OnboardbaseAuthSecretRef
</a>
</em>
</td>
<td>
<p>Auth configures how the Operator authenticates with the Onboardbase API</p>
</td>
</tr>
<tr>
<td>
<code>apiHost</code></br>
<em>
string
</em>
</td>
<td>
<p>APIHost use this to configure the host url for the API for selfhosted installation, default is <a href="https://public.onboardbase.com/api/v1/">https://public.onboardbase.com/api/v1/</a></p>
</td>
</tr>
<tr>
<td>
<code>project</code></br>
<em>
string
</em>
</td>
<td>
<p>Project is an onboardbase project that the secrets should be pulled from</p>
</td>
</tr>
<tr>
<td>
<code>environment</code></br>
<em>
string
</em>
</td>
<td>
<p>Environment is the name of an environmnent within a project to pull the secrets from</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OnePasswordAuth">OnePasswordAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OnePasswordProvider">OnePasswordProvider</a>)
</p>
<p>
<p>OnePasswordAuth contains a secretRef for credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.OnePasswordAuthSecretRef">
OnePasswordAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OnePasswordAuthSecretRef">OnePasswordAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OnePasswordAuth">OnePasswordAuth</a>)
</p>
<p>
<p>OnePasswordAuthSecretRef holds secret references for 1Password credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>connectTokenSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The ConnectToken is used for authentication to a 1Password Connect Server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OnePasswordProvider">OnePasswordProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>OnePasswordProvider configures a store to sync secrets using the 1Password Secret Manager provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.OnePasswordAuth">
OnePasswordAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against OnePassword Connect Server</p>
</td>
</tr>
<tr>
<td>
<code>connectHost</code></br>
<em>
string
</em>
</td>
<td>
<p>ConnectHost defines the OnePassword Connect Server to connect to</p>
</td>
</tr>
<tr>
<td>
<code>vaults</code></br>
<em>
map[string]int
</em>
</td>
<td>
<p>Vaults defines which OnePassword vaults to search in which order</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OnePasswordSDKAuth">OnePasswordSDKAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OnePasswordSDKProvider">OnePasswordSDKProvider</a>)
</p>
<p>
<p>OnePasswordSDKAuth contains a secretRef for the service account token.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serviceAccountSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>ServiceAccountSecretRef points to the secret containing the token to access 1Password vault.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OnePasswordSDKProvider">OnePasswordSDKProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>OnePasswordSDKProvider configures a store to sync secrets using the 1Password sdk.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>vault</code></br>
<em>
string
</em>
</td>
<td>
<p>Vault defines the vault&rsquo;s name or uuid to access. Do NOT add op:// prefix. This will be done automatically.</p>
</td>
</tr>
<tr>
<td>
<code>integrationInfo</code></br>
<em>
<a href="#external-secrets.io/v1.IntegrationInfo">
IntegrationInfo
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IntegrationInfo specifies the name and version of the integration built using the 1Password Go SDK.
If you don&rsquo;t know which name and version to use, use <code>DefaultIntegrationName</code> and <code>DefaultIntegrationVersion</code>, respectively.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.OnePasswordSDKAuth">
OnePasswordSDKAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against OnePassword API.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OracleAuth">OracleAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OracleProvider">OracleProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>tenancy</code></br>
<em>
string
</em>
</td>
<td>
<p>Tenancy is the tenancy OCID where user is located.</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
string
</em>
</td>
<td>
<p>User is an access OCID specific to the account.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.OracleSecretRef">
OracleSecretRef
</a>
</em>
</td>
<td>
<p>SecretRef to pass through sensitive information.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OraclePrincipalType">OraclePrincipalType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OracleProvider">OracleProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;InstancePrincipal&#34;</p></td>
<td><p>InstancePrincipal represents a instance principal.</p>
</td>
</tr><tr><td><p>&#34;UserPrincipal&#34;</p></td>
<td><p>UserPrincipal represents a user principal.</p>
</td>
</tr><tr><td><p>&#34;Workload&#34;</p></td>
<td><p>WorkloadPrincipal represents a workload principal.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.OracleProvider">OracleProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures an store to sync secrets using a Oracle Vault
backend.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region is the region where vault is located.</p>
</td>
</tr>
<tr>
<td>
<code>vault</code></br>
<em>
string
</em>
</td>
<td>
<p>Vault is the vault&rsquo;s OCID of the specific vault where secret is located.</p>
</td>
</tr>
<tr>
<td>
<code>compartment</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Compartment is the vault compartment OCID.
Required for PushSecret</p>
</td>
</tr>
<tr>
<td>
<code>encryptionKey</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>EncryptionKey is the OCID of the encryption key within the vault.
Required for PushSecret</p>
</td>
</tr>
<tr>
<td>
<code>principalType</code></br>
<em>
<a href="#external-secrets.io/v1.OraclePrincipalType">
OraclePrincipalType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The type of principal to use for authentication. If left blank, the Auth struct will
determine the principal type. This optional field must be specified if using
workload identity.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.OracleAuth">
OracleAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth configures how secret-manager authenticates with the Oracle Vault.
If empty, use the instance principal, otherwise the user credentials specified in Auth.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountRef specified the service account
that should be used when authenticating with WorkloadIdentity.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.OracleSecretRef">OracleSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.OracleAuth">OracleAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>privatekey</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>PrivateKey is the user&rsquo;s API Signing Key in PEM format, used for authentication.</p>
</td>
</tr>
<tr>
<td>
<code>fingerprint</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Fingerprint is the fingerprint of the API private key.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PassboltAuth">PassboltAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.PassboltProvider">PassboltProvider</a>)
</p>
<p>
<p>Passbolt contains a secretRef for the passbolt credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>passwordSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>privateKeySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PassboltProvider">PassboltProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.PassboltAuth">
PassboltAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against Passbolt Server</p>
</td>
</tr>
<tr>
<td>
<code>host</code></br>
<em>
string
</em>
</td>
<td>
<p>Host defines the Passbolt Server to connect to</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PasswordDepotAuth">PasswordDepotAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.PasswordDepotProvider">PasswordDepotProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.PasswordDepotSecretRef">
PasswordDepotSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PasswordDepotProvider">PasswordDepotProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures a store to sync secrets with a Password Depot instance.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>host</code></br>
<em>
string
</em>
</td>
<td>
<p>URL configures the Password Depot instance URL.</p>
</td>
</tr>
<tr>
<td>
<code>database</code></br>
<em>
string
</em>
</td>
<td>
<p>Database to use as source</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.PasswordDepotAuth">
PasswordDepotAuth
</a>
</em>
</td>
<td>
<p>Auth configures how secret-manager authenticates with a Password Depot instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PasswordDepotSecretRef">PasswordDepotSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.PasswordDepotAuth">PasswordDepotAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>credentials</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Username / Password is used for authentication.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PreviderAuth">PreviderAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.PreviderProvider">PreviderProvider</a>)
</p>
<p>
<p>PreviderAuth contains a secretRef for credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.PreviderAuthSecretRef">
PreviderAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PreviderAuthSecretRef">PreviderAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.PreviderAuth">PreviderAuth</a>)
</p>
<p>
<p>PreviderAuthSecretRef holds secret references for Previder Vault credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessToken</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>The AccessToken is used for authentication</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PreviderProvider">PreviderProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>PreviderProvider configures a store to sync secrets using the Previder Secret Manager provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.PreviderAuth">
PreviderAuth
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>baseUri</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.Provider">Provider
</h3>
<p>
<p>Provider is a common interface for interacting with secret backends.</p>
</p>
<h3 id="external-secrets.io/v1.PulumiProvider">PulumiProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiUrl</code></br>
<em>
string
</em>
</td>
<td>
<p>APIURL is the URL of the Pulumi API.</p>
</td>
</tr>
<tr>
<td>
<code>accessToken</code></br>
<em>
<a href="#external-secrets.io/v1.PulumiProviderSecretRef">
PulumiProviderSecretRef
</a>
</em>
</td>
<td>
<p>AccessToken is the access tokens to sign in to the Pulumi Cloud Console.</p>
</td>
</tr>
<tr>
<td>
<code>organization</code></br>
<em>
string
</em>
</td>
<td>
<p>Organization are a space to collaborate on shared projects and stacks.
To create a new organization, visit <a href="https://app.pulumi.com/">https://app.pulumi.com/</a> and click &ldquo;New Organization&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>project</code></br>
<em>
string
</em>
</td>
<td>
<p>Project is the name of the Pulumi ESC project the environment belongs to.</p>
</td>
</tr>
<tr>
<td>
<code>environment</code></br>
<em>
string
</em>
</td>
<td>
<p>Environment are YAML documents composed of static key-value pairs, programmatic expressions,
dynamically retrieved values from supported providers including all major clouds,
and other Pulumi ESC environments.
To create a new environment, visit <a href="https://www.pulumi.com/docs/esc/environments/">https://www.pulumi.com/docs/esc/environments/</a> for more information.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PulumiProviderSecretRef">PulumiProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.PulumiProvider">PulumiProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>SecretRef is a reference to a secret containing the Pulumi API token.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.PushSecretData">PushSecretData
</h3>
<p>
<p>PushSecretData is an interface to allow using v1alpha1.PushSecretData content in Provider registered in v1.</p>
</p>
<h3 id="external-secrets.io/v1.PushSecretRemoteRef">PushSecretRemoteRef
</h3>
<p>
<p>PushSecretRemoteRef is an interface to allow using v1alpha1.PushSecretRemoteRef in Provider registered in v1.</p>
</p>
<h3 id="external-secrets.io/v1.ScalewayProvider">ScalewayProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIURL is the url of the api to use. Defaults to <a href="https://api.scaleway.com">https://api.scaleway.com</a></p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region where your secrets are located: <a href="https://developers.scaleway.com/en/quickstart/#region-and-zone">https://developers.scaleway.com/en/quickstart/#region-and-zone</a></p>
</td>
</tr>
<tr>
<td>
<code>projectId</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectID is the id of your project, which you can find in the console: <a href="https://console.scaleway.com/project/settings">https://console.scaleway.com/project/settings</a></p>
</td>
</tr>
<tr>
<td>
<code>accessKey</code></br>
<em>
<a href="#external-secrets.io/v1.ScalewayProviderSecretRef">
ScalewayProviderSecretRef
</a>
</em>
</td>
<td>
<p>AccessKey is the non-secret part of the api key.</p>
</td>
</tr>
<tr>
<td>
<code>secretKey</code></br>
<em>
<a href="#external-secrets.io/v1.ScalewayProviderSecretRef">
ScalewayProviderSecretRef
</a>
</em>
</td>
<td>
<p>SecretKey is the non-secret part of the api key.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ScalewayProviderSecretRef">ScalewayProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ScalewayProvider">ScalewayProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value can be specified directly to set a value without using a secret.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef references a key in a secret that will be used as value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretReference">SecretReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AwsCredentialsConfig">AwsCredentialsConfig</a>)
</p>
<p>
<p>SecretReference holds the details of a secret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>name of the secret.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<p>namespace in which the secret exists. If empty, secret will looked up in local namespace.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretServerProvider">SecretServerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>See <a href="https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go">https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go</a>.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>username</code></br>
<em>
<a href="#external-secrets.io/v1.SecretServerProviderRef">
SecretServerProviderRef
</a>
</em>
</td>
<td>
<p>Username is the secret server account username.</p>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
<a href="#external-secrets.io/v1.SecretServerProviderRef">
SecretServerProviderRef
</a>
</em>
</td>
<td>
<p>Password is the secret server account password.</p>
</td>
</tr>
<tr>
<td>
<code>domain</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Domain is the secret server domain.</p>
</td>
</tr>
<tr>
<td>
<code>serverURL</code></br>
<em>
string
</em>
</td>
<td>
<p>ServerURL
URL to your secret server installation</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretServerProviderRef">SecretServerProviderRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretServerProvider">SecretServerProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Value can be specified directly to set a value without using a secret.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef references a key in a secret that will be used as value.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStore">SecretStore
</h3>
<p>
<p>SecretStore represents a secure external location for storing secrets, which can be referenced as part of <code>storeRef</code> fields.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreSpec">
SecretStoreSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>controller</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to select the correct ESO controller (think: ingress.ingressClassName)
The ESO controller is instantiated with a specific controller name and filters ES based on this property</p>
</td>
</tr>
<tr>
<td>
<code>provider</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreProvider">
SecretStoreProvider
</a>
</em>
</td>
<td>
<p>Used to configure the provider. Only one provider may be set</p>
</td>
</tr>
<tr>
<td>
<code>retrySettings</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreRetrySettings">
SecretStoreRetrySettings
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to configure http retries if failed</p>
</td>
</tr>
<tr>
<td>
<code>refreshInterval</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterSecretStoreCondition">
[]ClusterSecretStoreCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to constraint a ClusterSecretStore to specific namespaces. Relevant only to ClusterSecretStore</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreStatus">
SecretStoreStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreCapabilities">SecretStoreCapabilities
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreStatus">SecretStoreStatus</a>)
</p>
<p>
<p>SecretStoreCapabilities defines the possible operations a SecretStore can do.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ReadOnly&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;ReadWrite&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;WriteOnly&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreConditionType">SecretStoreConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreStatusCondition">SecretStoreStatusCondition</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreSpec">SecretStoreSpec</a>)
</p>
<p>
<p>SecretStoreProvider contains the provider-specific configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>aws</code></br>
<em>
<a href="#external-secrets.io/v1.AWSProvider">
AWSProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWS configures this store to sync secrets using AWS Secret Manager provider</p>
</td>
</tr>
<tr>
<td>
<code>azurekv</code></br>
<em>
<a href="#external-secrets.io/v1.AzureKVProvider">
AzureKVProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AzureKV configures this store to sync secrets using Azure Key Vault provider</p>
</td>
</tr>
<tr>
<td>
<code>akeyless</code></br>
<em>
<a href="#external-secrets.io/v1.AkeylessProvider">
AkeylessProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Akeyless configures this store to sync secrets using Akeyless Vault provider</p>
</td>
</tr>
<tr>
<td>
<code>bitwardensecretsmanager</code></br>
<em>
<a href="#external-secrets.io/v1.BitwardenSecretsManagerProvider">
BitwardenSecretsManagerProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BitwardenSecretsManager configures this store to sync secrets using BitwardenSecretsManager provider</p>
</td>
</tr>
<tr>
<td>
<code>vault</code></br>
<em>
<a href="#external-secrets.io/v1.VaultProvider">
VaultProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Vault configures this store to sync secrets using Hashi provider</p>
</td>
</tr>
<tr>
<td>
<code>gcpsm</code></br>
<em>
<a href="#external-secrets.io/v1.GCPSMProvider">
GCPSMProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>GCPSM configures this store to sync secrets using Google Cloud Platform Secret Manager provider</p>
</td>
</tr>
<tr>
<td>
<code>oracle</code></br>
<em>
<a href="#external-secrets.io/v1.OracleProvider">
OracleProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Oracle configures this store to sync secrets using Oracle Vault provider</p>
</td>
</tr>
<tr>
<td>
<code>ibm</code></br>
<em>
<a href="#external-secrets.io/v1.IBMProvider">
IBMProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IBM configures this store to sync secrets using IBM Cloud provider</p>
</td>
</tr>
<tr>
<td>
<code>yandexcertificatemanager</code></br>
<em>
<a href="#external-secrets.io/v1.YandexCertificateManagerProvider">
YandexCertificateManagerProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>YandexCertificateManager configures this store to sync secrets using Yandex Certificate Manager provider</p>
</td>
</tr>
<tr>
<td>
<code>yandexlockbox</code></br>
<em>
<a href="#external-secrets.io/v1.YandexLockboxProvider">
YandexLockboxProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>YandexLockbox configures this store to sync secrets using Yandex Lockbox provider</p>
</td>
</tr>
<tr>
<td>
<code>github</code></br>
<em>
<a href="#external-secrets.io/v1.GithubProvider">
GithubProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Github configures this store to push GitHub Action secrets using GitHub API provider.
Note: This provider only supports write operations (PushSecret) and cannot fetch secrets from GitHub</p>
</td>
</tr>
<tr>
<td>
<code>gitlab</code></br>
<em>
<a href="#external-secrets.io/v1.GitlabProvider">
GitlabProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>GitLab configures this store to sync secrets using GitLab Variables provider</p>
</td>
</tr>
<tr>
<td>
<code>alibaba</code></br>
<em>
<a href="#external-secrets.io/v1.AlibabaProvider">
AlibabaProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Alibaba configures this store to sync secrets using Alibaba Cloud provider</p>
</td>
</tr>
<tr>
<td>
<code>onepassword</code></br>
<em>
<a href="#external-secrets.io/v1.OnePasswordProvider">
OnePasswordProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>OnePassword configures this store to sync secrets using the 1Password Cloud provider</p>
</td>
</tr>
<tr>
<td>
<code>onepasswordSDK</code></br>
<em>
<a href="#external-secrets.io/v1.OnePasswordSDKProvider">
OnePasswordSDKProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>OnePasswordSDK configures this store to use 1Password&rsquo;s new Go SDK to sync secrets.</p>
</td>
</tr>
<tr>
<td>
<code>webhook</code></br>
<em>
<a href="#external-secrets.io/v1.WebhookProvider">
WebhookProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Webhook configures this store to sync secrets using a generic templated webhook</p>
</td>
</tr>
<tr>
<td>
<code>kubernetes</code></br>
<em>
<a href="#external-secrets.io/v1.KubernetesProvider">
KubernetesProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Kubernetes configures this store to sync secrets using a Kubernetes cluster provider</p>
</td>
</tr>
<tr>
<td>
<code>fake</code></br>
<em>
<a href="#external-secrets.io/v1.FakeProvider">
FakeProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Fake configures a store with static key/value pairs</p>
</td>
</tr>
<tr>
<td>
<code>senhasegura</code></br>
<em>
<a href="#external-secrets.io/v1.SenhaseguraProvider">
SenhaseguraProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Senhasegura configures this store to sync secrets using senhasegura provider</p>
</td>
</tr>
<tr>
<td>
<code>scaleway</code></br>
<em>
<a href="#external-secrets.io/v1.ScalewayProvider">
ScalewayProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Scaleway</p>
</td>
</tr>
<tr>
<td>
<code>doppler</code></br>
<em>
<a href="#external-secrets.io/v1.DopplerProvider">
DopplerProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Doppler configures this store to sync secrets using the Doppler provider</p>
</td>
</tr>
<tr>
<td>
<code>previder</code></br>
<em>
<a href="#external-secrets.io/v1.PreviderProvider">
PreviderProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Previder configures this store to sync secrets using the Previder provider</p>
</td>
</tr>
<tr>
<td>
<code>onboardbase</code></br>
<em>
<a href="#external-secrets.io/v1.OnboardbaseProvider">
OnboardbaseProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Onboardbase configures this store to sync secrets using the Onboardbase provider</p>
</td>
</tr>
<tr>
<td>
<code>keepersecurity</code></br>
<em>
<a href="#external-secrets.io/v1.KeeperSecurityProvider">
KeeperSecurityProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>KeeperSecurity configures this store to sync secrets using the KeeperSecurity provider</p>
</td>
</tr>
<tr>
<td>
<code>conjur</code></br>
<em>
<a href="#external-secrets.io/v1.ConjurProvider">
ConjurProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conjur configures this store to sync secrets using conjur provider</p>
</td>
</tr>
<tr>
<td>
<code>delinea</code></br>
<em>
<a href="#external-secrets.io/v1.DelineaProvider">
DelineaProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Delinea DevOps Secrets Vault
<a href="https://docs.delinea.com/online-help/products/devops-secrets-vault/current">https://docs.delinea.com/online-help/products/devops-secrets-vault/current</a></p>
</td>
</tr>
<tr>
<td>
<code>secretserver</code></br>
<em>
<a href="#external-secrets.io/v1.SecretServerProvider">
SecretServerProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretServer configures this store to sync secrets using SecretServer provider
<a href="https://docs.delinea.com/online-help/secret-server/start.htm">https://docs.delinea.com/online-help/secret-server/start.htm</a></p>
</td>
</tr>
<tr>
<td>
<code>chef</code></br>
<em>
<a href="#external-secrets.io/v1.ChefProvider">
ChefProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Chef configures this store to sync secrets with chef server</p>
</td>
</tr>
<tr>
<td>
<code>pulumi</code></br>
<em>
<a href="#external-secrets.io/v1.PulumiProvider">
PulumiProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Pulumi configures this store to sync secrets using the Pulumi provider</p>
</td>
</tr>
<tr>
<td>
<code>fortanix</code></br>
<em>
<a href="#external-secrets.io/v1.FortanixProvider">
FortanixProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Fortanix configures this store to sync secrets using the Fortanix provider</p>
</td>
</tr>
<tr>
<td>
<code>passworddepot</code></br>
<em>
<a href="#external-secrets.io/v1.PasswordDepotProvider">
PasswordDepotProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>passbolt</code></br>
<em>
<a href="#external-secrets.io/v1.PassboltProvider">
PassboltProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>device42</code></br>
<em>
<a href="#external-secrets.io/v1.Device42Provider">
Device42Provider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Device42 configures this store to sync secrets using the Device42 provider</p>
</td>
</tr>
<tr>
<td>
<code>infisical</code></br>
<em>
<a href="#external-secrets.io/v1.InfisicalProvider">
InfisicalProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Infisical configures this store to sync secrets using the Infisical provider</p>
</td>
</tr>
<tr>
<td>
<code>beyondtrust</code></br>
<em>
<a href="#external-secrets.io/v1.BeyondtrustProvider">
BeyondtrustProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Beyondtrust configures this store to sync secrets using Password Safe provider.</p>
</td>
</tr>
<tr>
<td>
<code>cloudrusm</code></br>
<em>
<a href="#external-secrets.io/v1.CloudruSMProvider">
CloudruSMProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CloudruSM configures this store to sync secrets using the Cloud.ru Secret Manager provider</p>
</td>
</tr>
<tr>
<td>
<code>volcengine</code></br>
<em>
<a href="#external-secrets.io/v1.VolcengineProvider">
VolcengineProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Volcengine configures this store to sync secrets using the Volcengine provider</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreRef">SecretStoreRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>, 
<a href="#external-secrets.io/v1.StoreGeneratorSourceRef">StoreGeneratorSourceRef</a>, 
<a href="#external-secrets.io/v1.StoreSourceRef">StoreSourceRef</a>)
</p>
<p>
<p>SecretStoreRef defines which SecretStore to fetch the ExternalSecret data.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the SecretStore resource</p>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Kind of the SecretStore resource (SecretStore or ClusterSecretStore)
Defaults to <code>SecretStore</code></p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreRetrySettings">SecretStoreRetrySettings
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreSpec">SecretStoreSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>maxRetries</code></br>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>retryInterval</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreSpec">SecretStoreSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterSecretStore">ClusterSecretStore</a>, 
<a href="#external-secrets.io/v1.SecretStore">SecretStore</a>)
</p>
<p>
<p>SecretStoreSpec defines the desired state of SecretStore.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>controller</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to select the correct ESO controller (think: ingress.ingressClassName)
The ESO controller is instantiated with a specific controller name and filters ES based on this property</p>
</td>
</tr>
<tr>
<td>
<code>provider</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreProvider">
SecretStoreProvider
</a>
</em>
</td>
<td>
<p>Used to configure the provider. Only one provider may be set</p>
</td>
</tr>
<tr>
<td>
<code>retrySettings</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreRetrySettings">
SecretStoreRetrySettings
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to configure http retries if failed</p>
</td>
</tr>
<tr>
<td>
<code>refreshInterval</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to configure store refresh interval in seconds. Empty or 0 will default to the controller config.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1.ClusterSecretStoreCondition">
[]ClusterSecretStoreCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to constraint a ClusterSecretStore to specific namespaces. Relevant only to ClusterSecretStore</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreStatus">SecretStoreStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ClusterSecretStore">ClusterSecretStore</a>, 
<a href="#external-secrets.io/v1.SecretStore">SecretStore</a>)
</p>
<p>
<p>SecretStoreStatus defines the observed state of the SecretStore.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreStatusCondition">
[]SecretStoreStatusCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>capabilities</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreCapabilities">
SecretStoreCapabilities
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreStatusCondition">SecretStoreStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreStatus">SecretStoreStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreConditionType">
SecretStoreConditionType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SecretVersionSelectionPolicy">SecretVersionSelectionPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GCPSMProvider">GCPSMProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;LatestOrFail&#34;</p></td>
<td><p>SecretVersionSelectionPolicyLatestOrFail means the provider always uses &ldquo;latest&rdquo;, or fails if that version is disabled/destroyed.</p>
</td>
</tr><tr><td><p>&#34;LatestOrFetch&#34;</p></td>
<td><p>SecretVersionSelectionPolicyLatestOrFetch behaves like SecretVersionSelectionPolicyLatestOrFail but falls back to fetching the latest version if the version is DESTROYED or DISABLED.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.SecretsClient">SecretsClient
</h3>
<p>
<p>SecretsClient provides access to secrets.</p>
</p>
<h3 id="external-secrets.io/v1.SecretsManager">SecretsManager
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.AWSProvider">AWSProvider</a>)
</p>
<p>
<p>SecretsManager defines how the provider behaves when interacting with AWS
SecretsManager. Some of these settings are only applicable to controlling how
secrets are deleted, and hence only apply to PushSecret (and only when
deletionPolicy is set to Delete).</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>forceDeleteWithoutRecovery</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies whether to delete the secret without any recovery window. You
can&rsquo;t use both this parameter and RecoveryWindowInDays in the same call.
If you don&rsquo;t use either, then by default Secrets Manager uses a 30 day
recovery window.
see: <a href="https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-ForceDeleteWithoutRecovery">https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-ForceDeleteWithoutRecovery</a></p>
</td>
</tr>
<tr>
<td>
<code>recoveryWindowInDays</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The number of days from 7 to 30 that Secrets Manager waits before
permanently deleting the secret. You can&rsquo;t use both this parameter and
ForceDeleteWithoutRecovery in the same call. If you don&rsquo;t use either,
then by default Secrets Manager uses a 30 day recovery window.
see: <a href="https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-RecoveryWindowInDays">https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_DeleteSecret.html#SecretsManager-DeleteSecret-request-RecoveryWindowInDays</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SenhaseguraAuth">SenhaseguraAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SenhaseguraProvider">SenhaseguraProvider</a>)
</p>
<p>
<p>SenhaseguraAuth tells the controller how to do auth in senhasegura.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientId</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>clientSecretSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.SenhaseguraModuleType">SenhaseguraModuleType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SenhaseguraProvider">SenhaseguraProvider</a>)
</p>
<p>
<p>SenhaseguraModuleType enum defines senhasegura target module to fetch secrets</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;DSM&#34;</p></td>
<td><pre><code>	SenhaseguraModuleDSM is the senhasegura DevOps Secrets Management module
see: https://senhasegura.com/devops
</code></pre>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.SenhaseguraProvider">SenhaseguraProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>SenhaseguraProvider setup a store to sync secrets with senhasegura.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL of senhasegura</p>
</td>
</tr>
<tr>
<td>
<code>module</code></br>
<em>
<a href="#external-secrets.io/v1.SenhaseguraModuleType">
SenhaseguraModuleType
</a>
</em>
</td>
<td>
<p>Module defines which senhasegura module should be used to get secrets</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.SenhaseguraAuth">
SenhaseguraAuth
</a>
</em>
</td>
<td>
<p>Auth defines parameters to authenticate in senhasegura</p>
</td>
</tr>
<tr>
<td>
<code>ignoreSslCertificate</code></br>
<em>
bool
</em>
</td>
<td>
<p>IgnoreSslCertificate defines if SSL certificate must be ignored</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.StoreGeneratorSourceRef">StoreGeneratorSourceRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
</p>
<p>
<p>StoreGeneratorSourceRef allows you to override the source
from which the secret will be pulled from.
You can define at maximum one property.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>storeRef</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreRef">
SecretStoreRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>generatorRef</code></br>
<em>
<a href="#external-secrets.io/v1.GeneratorRef">
GeneratorRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>GeneratorRef points to a generator custom resource.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.StoreSourceRef">StoreSourceRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretData">ExternalSecretData</a>)
</p>
<p>
<p>StoreSourceRef allows you to override the SecretStore source
from which the secret will be pulled from.
You can define at maximum one property.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>storeRef</code></br>
<em>
<a href="#external-secrets.io/v1.SecretStoreRef">
SecretStoreRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>generatorRef</code></br>
<em>
<a href="#external-secrets.io/v1.GeneratorRef">
GeneratorRef
</a>
</em>
</td>
<td>
<p>GeneratorRef points to a generator custom resource.</p>
<p>Deprecated: The generatorRef is not implemented in .data[].
this will be removed with v1.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.Tag">Tag
</h3>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateEngineVersion">TemplateEngineVersion
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;v2&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateFrom">TemplateFrom
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>configMap</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateRef">
TemplateRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>secret</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateRef">
TemplateRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateTarget">
TemplateTarget
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>literal</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateMergePolicy">TemplateMergePolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Merge&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Replace&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateRef">TemplateRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.TemplateFrom">TemplateFrom</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the ConfigMap/Secret resource</p>
</td>
</tr>
<tr>
<td>
<code>items</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateRefItem">
[]TemplateRefItem
</a>
</em>
</td>
<td>
<p>A list of keys in the ConfigMap/Secret to use as templates for Secret data</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateRefItem">TemplateRefItem
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.TemplateRef">TemplateRef</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>A key in the ConfigMap/Secret</p>
</td>
</tr>
<tr>
<td>
<code>templateAs</code></br>
<em>
<a href="#external-secrets.io/v1.TemplateScope">
TemplateScope
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateScope">TemplateScope
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.TemplateRefItem">TemplateRefItem</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;KeysAndValues&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Values&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateTarget">TemplateTarget
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.TemplateFrom">TemplateFrom</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Annotations&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Data&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Labels&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.TokenAuth">TokenAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.KubernetesAuth">KubernetesAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>bearerToken</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.TokenAuthCredentials">TokenAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessToken</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.UniversalAuthCredentials">UniversalAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientId</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>clientSecret</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ValidationResult">ValidationResult
(<code>byte</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.FakeProvider">FakeProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>2</p></td>
<td><p>Error indicates that there is a misconfiguration.</p>
</td>
</tr><tr><td><p>0</p></td>
<td><p>Ready indicates that the client is configured correctly
and can be used.</p>
</td>
</tr><tr><td><p>1</p></td>
<td><p>Unknown indicates that the client can be used
but information is missing and it can not be validated.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.VaultAppRole">VaultAppRole
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultAppRole authenticates with Vault using the App Role auth mechanism,
with the role and secret stored in a Kubernetes Secret resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path where the App Role authentication backend is mounted
in Vault, e.g: &ldquo;approle&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>roleId</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>RoleID configured in the App Role authentication backend when setting
up the authentication backend in Vault.</p>
</td>
</tr>
<tr>
<td>
<code>roleRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to a key in a Secret that contains the App Role ID used
to authenticate with Vault.
The <code>key</code> field must be specified and denotes which entry within the Secret
resource is used as the app role id.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Reference to a key in a Secret that contains the App Role secret used
to authenticate with Vault.
The <code>key</code> field must be specified and denotes which entry within the Secret
resource is used as the app role secret.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultAuth">VaultAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultProvider">VaultProvider</a>)
</p>
<p>
<p>VaultAuth is the configuration used to authenticate with a Vault server.
Only one of <code>tokenSecretRef</code>, <code>appRole</code>,  <code>kubernetes</code>, <code>ldap</code>, <code>userPass</code>, <code>jwt</code> or <code>cert</code>
can be specified. A namespace to authenticate against can optionally be specified.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Name of the vault namespace to authenticate to. This can be different than the namespace your secret is in.
Namespaces is a set of features within Vault Enterprise that allows
Vault environments to support Secure Multi-tenancy. e.g: &ldquo;ns1&rdquo;.
More about namespaces can be found here <a href="https://www.vaultproject.io/docs/enterprise/namespaces">https://www.vaultproject.io/docs/enterprise/namespaces</a>
This will default to Vault.Namespace field if set, or empty otherwise</p>
</td>
</tr>
<tr>
<td>
<code>tokenSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TokenSecretRef authenticates with Vault by presenting a token.</p>
</td>
</tr>
<tr>
<td>
<code>appRole</code></br>
<em>
<a href="#external-secrets.io/v1.VaultAppRole">
VaultAppRole
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AppRole authenticates with Vault using the App Role auth mechanism,
with the role and secret stored in a Kubernetes Secret resource.</p>
</td>
</tr>
<tr>
<td>
<code>kubernetes</code></br>
<em>
<a href="#external-secrets.io/v1.VaultKubernetesAuth">
VaultKubernetesAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Kubernetes authenticates with Vault by passing the ServiceAccount
token stored in the named Secret resource to the Vault server.</p>
</td>
</tr>
<tr>
<td>
<code>ldap</code></br>
<em>
<a href="#external-secrets.io/v1.VaultLdapAuth">
VaultLdapAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ldap authenticates with Vault by passing username/password pair using
the LDAP authentication method</p>
</td>
</tr>
<tr>
<td>
<code>jwt</code></br>
<em>
<a href="#external-secrets.io/v1.VaultJwtAuth">
VaultJwtAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Jwt authenticates with Vault by passing role and JWT token using the
JWT/OIDC authentication method</p>
</td>
</tr>
<tr>
<td>
<code>cert</code></br>
<em>
<a href="#external-secrets.io/v1.VaultCertAuth">
VaultCertAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Cert authenticates with TLS Certificates by passing client certificate, private key and ca certificate
Cert authentication method</p>
</td>
</tr>
<tr>
<td>
<code>iam</code></br>
<em>
<a href="#external-secrets.io/v1.VaultIamAuth">
VaultIamAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Iam authenticates with vault by passing a special AWS request signed with AWS IAM credentials
AWS IAM authentication method</p>
</td>
</tr>
<tr>
<td>
<code>userPass</code></br>
<em>
<a href="#external-secrets.io/v1.VaultUserPassAuth">
VaultUserPassAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>UserPass authenticates with Vault by passing username/password pair</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultAwsAuth">VaultAwsAuth
</h3>
<p>
<p>VaultAwsAuth tells the controller how to do authentication with aws.
Only one of secretRef or jwt can be specified.
if none is specified the controller will try to load credentials from its own service account assuming it is IRSA enabled.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.VaultAwsAuthSecretRef">
VaultAwsAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>jwt</code></br>
<em>
<a href="#external-secrets.io/v1.VaultAwsJWTAuth">
VaultAwsJWTAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultAwsAuthSecretRef">VaultAwsAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAwsAuth">VaultAwsAuth</a>, 
<a href="#external-secrets.io/v1.VaultIamAuth">VaultIamAuth</a>)
</p>
<p>
<p>VaultAWSAuthSecretRef holds secret references for AWS credentials
both AccessKeyID and SecretAccessKey must be defined in order to properly authenticate.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessKeyIDSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The AccessKeyID is used for authentication</p>
</td>
</tr>
<tr>
<td>
<code>secretAccessKeySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The SecretAccessKey is used for authentication</p>
</td>
</tr>
<tr>
<td>
<code>sessionTokenSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The SessionToken used for authentication
This must be defined if AccessKeyID and SecretAccessKey are temporary credentials
see: <a href="https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html">https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_use-resources.html</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultAwsJWTAuth">VaultAwsJWTAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAwsAuth">VaultAwsAuth</a>, 
<a href="#external-secrets.io/v1.VaultIamAuth">VaultIamAuth</a>)
</p>
<p>
<p>VaultAwsJWTAuth Authenticate against AWS using service account tokens.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultCertAuth">VaultCertAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultCertAuth authenticates with Vault using the JWT/OIDC authentication
method, with the role name and token stored in a Kubernetes Secret resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientCert</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClientCert is a certificate to authenticate using the Cert Vault
authentication method</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef to a key in a Secret resource containing client private key to
authenticate with Vault using the Cert authentication method</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultCheckAndSet">VaultCheckAndSet
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultProvider">VaultProvider</a>)
</p>
<p>
<p>VaultCheckAndSet defines the Check-And-Set (CAS) settings for Vault KV v2 PushSecret operations.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>required</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Required when true, all write operations must include a check-and-set parameter.
This helps prevent unintentional overwrites of secrets.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultClientTLS">VaultClientTLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultProvider">VaultProvider</a>)
</p>
<p>
<p>VaultClientTLS is the configuration used for client side related TLS communication,
when the Vault server requires mutual authentication.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CertSecretRef is a certificate added to the transport layer
when communicating with the Vault server.
If no key for the Secret is specified, external-secret will default to &lsquo;tls.crt&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>keySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>KeySecretRef to a key in a Secret resource containing client private key
added to the transport layer when communicating with the Vault server.
If no key for the Secret is specified, external-secret will default to &lsquo;tls.key&rsquo;.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultIamAuth">VaultIamAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultIamAuth authenticates with Vault using the Vault&rsquo;s AWS IAM authentication method. Refer: <a href="https://developer.hashicorp.com/vault/docs/auth/aws">https://developer.hashicorp.com/vault/docs/auth/aws</a></p>
<p>When JWTAuth and SecretRef are not specified, the provider will use the controller pod&rsquo;s
identity to authenticate with AWS. This supports both IRSA and EKS Pod Identity.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Path where the AWS auth method is enabled in Vault, e.g: &ldquo;aws&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWS region</p>
</td>
</tr>
<tr>
<td>
<code>role</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>This is the AWS role to be assumed before talking to vault</p>
</td>
</tr>
<tr>
<td>
<code>vaultRole</code></br>
<em>
string
</em>
</td>
<td>
<p>Vault Role. In vault, a role describes an identity with a set of permissions, groups, or policies you want to attach a user of the secrets engine</p>
</td>
</tr>
<tr>
<td>
<code>externalID</code></br>
<em>
string
</em>
</td>
<td>
<p>AWS External ID set on assumed IAM roles</p>
</td>
</tr>
<tr>
<td>
<code>vaultAwsIamServerID</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>X-Vault-AWS-IAM-Server-ID is an additional header used by Vault IAM auth method to mitigate against different types of replay attacks. More details here: <a href="https://developer.hashicorp.com/vault/docs/auth/aws">https://developer.hashicorp.com/vault/docs/auth/aws</a></p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.VaultAwsAuthSecretRef">
VaultAwsAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specify credentials in a Secret object</p>
</td>
</tr>
<tr>
<td>
<code>jwt</code></br>
<em>
<a href="#external-secrets.io/v1.VaultAwsJWTAuth">
VaultAwsJWTAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specify a service account with IRSA enabled</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultJwtAuth">VaultJwtAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultJwtAuth authenticates with Vault using the JWT/OIDC authentication
method, with the role name and a token stored in a Kubernetes Secret resource or
a Kubernetes service account token retrieved via <code>TokenRequest</code>.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path where the JWT authentication backend is mounted
in Vault, e.g: &ldquo;jwt&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>role</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Role is a JWT role to authenticate using the JWT/OIDC Vault
authentication method</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional SecretRef that refers to a key in a Secret resource containing JWT token to
authenticate with Vault using the JWT/OIDC authentication method.</p>
</td>
</tr>
<tr>
<td>
<code>kubernetesServiceAccountToken</code></br>
<em>
<a href="#external-secrets.io/v1.VaultKubernetesServiceAccountTokenAuth">
VaultKubernetesServiceAccountTokenAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional ServiceAccountToken specifies the Kubernetes service account for which to request
a token for with the <code>TokenRequest</code> API.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultKVStoreVersion">VaultKVStoreVersion
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultProvider">VaultProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;v1&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;v2&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.VaultKubernetesAuth">VaultKubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>Authenticate against Vault using a Kubernetes ServiceAccount token stored in
a Secret.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>mountPath</code></br>
<em>
string
</em>
</td>
<td>
<p>Path where the Kubernetes authentication backend is mounted in Vault, e.g:
&ldquo;kubernetes&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional service account field containing the name of a kubernetes ServiceAccount.
If the service account is specified, the service account secret token JWT will be used
for authenticating with Vault. If the service account selector is not supplied,
the secretRef will be used instead.</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional secret field containing a Kubernetes ServiceAccount JWT used
for authenticating with Vault. If a name is specified without a key,
<code>token</code> is the default. If one is not specified, the one bound to
the controller will be used.</p>
</td>
</tr>
<tr>
<td>
<code>role</code></br>
<em>
string
</em>
</td>
<td>
<p>A required field containing the Vault Role to assume. A Role binds a
Kubernetes ServiceAccount with a set of Vault policies.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultKubernetesServiceAccountTokenAuth">VaultKubernetesServiceAccountTokenAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultJwtAuth">VaultJwtAuth</a>)
</p>
<p>
<p>VaultKubernetesServiceAccountTokenAuth authenticates with Vault using a temporary
Kubernetes service account token retrieved by the <code>TokenRequest</code> API.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serviceAccountRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#ServiceAccountSelector">
External Secrets meta/v1.ServiceAccountSelector
</a>
</em>
</td>
<td>
<p>Service account field containing the name of a kubernetes ServiceAccount.</p>
</td>
</tr>
<tr>
<td>
<code>audiences</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional audiences field that will be used to request a temporary Kubernetes service
account token for the service account referenced by <code>serviceAccountRef</code>.
Defaults to a single audience <code>vault</code> it not specified.
Deprecated: use serviceAccountRef.Audiences instead</p>
</td>
</tr>
<tr>
<td>
<code>expirationSeconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional expiration time in seconds that will be used to request a temporary
Kubernetes service account token for the service account referenced by
<code>serviceAccountRef</code>.
Deprecated: this will be removed in the future.
Defaults to 10 minutes.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultLdapAuth">VaultLdapAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultLdapAuth authenticates with Vault using the LDAP authentication method,
with the username and password stored in a Kubernetes Secret resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path where the LDAP authentication backend is mounted
in Vault, e.g: &ldquo;ldap&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>username</code></br>
<em>
string
</em>
</td>
<td>
<p>Username is an LDAP username used to authenticate using the LDAP Vault
authentication method</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef to a key in a Secret resource containing password for the LDAP
user used to authenticate with Vault using the LDAP authentication
method</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultProvider">VaultProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>Configures an store to sync secrets using a HashiCorp Vault
KV backend.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.VaultAuth">
VaultAuth
</a>
</em>
</td>
<td>
<p>Auth configures how secret-manager authenticates with the Vault server.</p>
</td>
</tr>
<tr>
<td>
<code>server</code></br>
<em>
string
</em>
</td>
<td>
<p>Server is the connection address for the Vault server, e.g: &ldquo;<a href="https://vault.example.com:8200&quot;">https://vault.example.com:8200&rdquo;</a>.</p>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Path is the mount path of the Vault KV backend endpoint, e.g:
&ldquo;secret&rdquo;. The v2 KV secret engine version specific &ldquo;/data&rdquo; path suffix
for fetching secrets from Vault is optional and will be appended
if not present in specified path.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
<a href="#external-secrets.io/v1.VaultKVStoreVersion">
VaultKVStoreVersion
</a>
</em>
</td>
<td>
<p>Version is the Vault KV secret engine version. This can be either &ldquo;v1&rdquo; or
&ldquo;v2&rdquo;. Version defaults to &ldquo;v2&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Name of the vault namespace. Namespaces is a set of features within Vault Enterprise that allows
Vault environments to support Secure Multi-tenancy. e.g: &ldquo;ns1&rdquo;.
More about namespaces can be found here <a href="https://www.vaultproject.io/docs/enterprise/namespaces">https://www.vaultproject.io/docs/enterprise/namespaces</a></p>
</td>
</tr>
<tr>
<td>
<code>caBundle</code></br>
<em>
[]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>PEM encoded CA bundle used to validate Vault server certificate. Only used
if the Server URL is using HTTPS protocol. This parameter is ignored for
plain HTTP protocol connection. If not set the system root certificates
are used to validate the TLS connection.</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#external-secrets.io/v1.VaultClientTLS">
VaultClientTLS
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The configuration used for client side related TLS communication, when the Vault server
requires mutual authentication. Only used if the Server URL is using HTTPS protocol.
This parameter is ignored for plain HTTP protocol connection.
It&rsquo;s worth noting this configuration is different from the &ldquo;TLS certificates auth method&rdquo;,
which is available under the <code>auth.cert</code> section.</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.CAProvider">
CAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate Vault server certificate.</p>
</td>
</tr>
<tr>
<td>
<code>readYourWrites</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ReadYourWrites ensures isolated read-after-write semantics by
providing discovered cluster replication states in each request.
More information about eventual consistency in Vault can be found here
<a href="https://www.vaultproject.io/docs/enterprise/consistency">https://www.vaultproject.io/docs/enterprise/consistency</a></p>
</td>
</tr>
<tr>
<td>
<code>forwardInconsistent</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ForwardInconsistent tells Vault to forward read-after-write requests to the Vault
leader instead of simply retrying within a loop. This can increase performance if
the option is enabled serverside.
<a href="https://www.vaultproject.io/docs/configuration/replication#allow_forwarding_via_header">https://www.vaultproject.io/docs/configuration/replication#allow_forwarding_via_header</a></p>
</td>
</tr>
<tr>
<td>
<code>headers</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Headers to be added in Vault request</p>
</td>
</tr>
<tr>
<td>
<code>checkAndSet</code></br>
<em>
<a href="#external-secrets.io/v1.VaultCheckAndSet">
VaultCheckAndSet
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CheckAndSet defines the Check-And-Set (CAS) settings for PushSecret operations.
Only applies to Vault KV v2 stores. When enabled, write operations must include
the current version of the secret to prevent unintentional overwrites.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VaultUserPassAuth">VaultUserPassAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultUserPassAuth authenticates with Vault using UserPass authentication method,
with the username and password stored in a Kubernetes Secret resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path where the UserPassword authentication backend is mounted
in Vault, e.g: &ldquo;userpass&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>username</code></br>
<em>
string
</em>
</td>
<td>
<p>Username is a username used to authenticate using the UserPass Vault
authentication method</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef to a key in a Secret resource containing password for the
user used to authenticate with Vault using the UserPass authentication
method</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VolcengineAuth">VolcengineAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VolcengineProvider">VolcengineProvider</a>)
</p>
<p>
<p>VolcengineAuth defines the authentication method for the Volcengine provider.
Only one of the fields should be set.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#external-secrets.io/v1.VolcengineAuthSecretRef">
VolcengineAuthSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretRef defines the static credentials to use for authentication.
If not set, IRSA is used.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VolcengineAuthSecretRef">VolcengineAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VolcengineAuth">VolcengineAuth</a>)
</p>
<p>
<p>VolcengineAuthSecretRef defines the secret reference for static credentials.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessKeyID</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>AccessKeyID is the reference to the secret containing the Access Key ID.</p>
</td>
</tr>
<tr>
<td>
<code>secretAccessKey</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>SecretAccessKey is the reference to the secret containing the Secret Access Key.</p>
</td>
</tr>
<tr>
<td>
<code>token</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Token is the reference to the secret containing the STS(Security Token Service) Token.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.VolcengineProvider">VolcengineProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>VolcengineProvider defines the configuration for the Volcengine provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region specifies the Volcengine region to connect to.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.VolcengineAuth">
VolcengineAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines the authentication method to use.
If not specified, the provider will try to use IRSA (IAM Role for Service Account).</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.WebhookCAProvider">WebhookCAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.WebhookProvider">WebhookProvider</a>)
</p>
<p>
<p>Defines a location to fetch the cert for the webhook provider from.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#external-secrets.io/v1.WebhookCAProviderType">
WebhookCAProviderType
</a>
</em>
</td>
<td>
<p>The type of provider to use such as &ldquo;Secret&rdquo;, or &ldquo;ConfigMap&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the object located at the provider type.</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>The key where the CA certificate can be found in the Secret or ConfigMap.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The namespace the Provider type is in.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.WebhookCAProviderType">WebhookCAProviderType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.WebhookCAProvider">WebhookCAProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ConfigMap&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Secret&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.WebhookProvider">WebhookProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>WebHookProvider Configures an store to sync secrets from simple web apis.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>method</code></br>
<em>
string
</em>
</td>
<td>
<p>Webhook Method</p>
</td>
</tr>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>Webhook url to call</p>
</td>
</tr>
<tr>
<td>
<code>headers</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Headers</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.AuthorizationProtocol">
AuthorizationProtocol
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth specifies a authorization protocol. Only one protocol may be set.</p>
</td>
</tr>
<tr>
<td>
<code>body</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Body</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout</p>
</td>
</tr>
<tr>
<td>
<code>result</code></br>
<em>
<a href="#external-secrets.io/v1.WebhookResult">
WebhookResult
</a>
</em>
</td>
<td>
<p>Result formatting</p>
</td>
</tr>
<tr>
<td>
<code>secrets</code></br>
<em>
<a href="#external-secrets.io/v1.WebhookSecret">
[]WebhookSecret
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Secrets to fill in templates
These secrets will be passed to the templating function as key value pairs under the given name</p>
</td>
</tr>
<tr>
<td>
<code>caBundle</code></br>
<em>
[]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>PEM encoded CA bundle used to validate webhook server certificate. Only used
if the Server URL is using HTTPS protocol. This parameter is ignored for
plain HTTP protocol connection. If not set the system root certificates
are used to validate the TLS connection.</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.WebhookCAProvider">
WebhookCAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate webhook server certificate.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.WebhookResult">WebhookResult
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.WebhookProvider">WebhookProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>jsonPath</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Json path of return value</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.WebhookSecret">WebhookSecret
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.WebhookProvider">WebhookProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of this secret in templates</p>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Secret ref to fill in credentials</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.YandexAuth">YandexAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.YandexCertificateManagerProvider">YandexCertificateManagerProvider</a>, 
<a href="#external-secrets.io/v1.YandexLockboxProvider">YandexLockboxProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>authorizedKeySecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The authorized key used for authentication</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.YandexCAProvider">YandexCAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.YandexCertificateManagerProvider">YandexCertificateManagerProvider</a>, 
<a href="#external-secrets.io/v1.YandexLockboxProvider">YandexLockboxProvider</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.YandexCertificateManagerProvider">YandexCertificateManagerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>YandexCertificateManagerProvider Configures a store to sync secrets using the Yandex Certificate Manager provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiEndpoint</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Yandex.Cloud API endpoint (e.g. &lsquo;api.cloud.yandex.net:443&rsquo;)</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.YandexAuth">
YandexAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against Yandex.Cloud</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.YandexCAProvider">
YandexCAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate Yandex.Cloud server certificate.</p>
</td>
</tr>
<tr>
<td>
<code>fetching</code></br>
<em>
<a href="#external-secrets.io/v1.FetchingPolicy">
FetchingPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FetchingPolicy configures the provider to interpret the <code>data.secretKey.remoteRef.key</code> field in ExternalSecret as certificate ID or certificate name</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.YandexLockboxProvider">YandexLockboxProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>YandexLockboxProvider Configures a store to sync secrets using the Yandex Lockbox provider.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiEndpoint</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Yandex.Cloud API endpoint (e.g. &lsquo;api.cloud.yandex.net:443&rsquo;)</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.YandexAuth">
YandexAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against Yandex.Cloud</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1.YandexCAProvider">
YandexCAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate Yandex.Cloud server certificate.</p>
</td>
</tr>
<tr>
<td>
<code>fetching</code></br>
<em>
<a href="#external-secrets.io/v1.FetchingPolicy">
FetchingPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FetchingPolicy configures the provider to interpret the <code>data.secretKey.remoteRef.key</code> field in ExternalSecret as secret ID or secret name</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>.
</em></p>
