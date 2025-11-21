<p>Packages:</p>
<ul>
<li>
<a href="#external-secrets.io%2fv1">external-secrets.io/v1</a>
</li>
<li>
<a href="#external-secrets.io%2fv1alpha1">external-secrets.io/v1alpha1</a>
</li>
<li>
<a href="#external-secrets.io%2fv1beta1">external-secrets.io/v1beta1</a>
</li>
<li>
<a href="#generators.external-secrets.io%2fv1alpha1">generators.external-secrets.io/v1alpha1</a>
</li>
<li>
<a href="#scan.external-secrets.io%2fv1alpha1">scan.external-secrets.io/v1alpha1</a>
</li>
<li>
<a href="#target.external-secrets.io%2fv1alpha1">target.external-secrets.io/v1alpha1</a>
</li>
<li>
<a href="#workflows.external-secrets.io%2fv1alpha1">workflows.external-secrets.io/v1alpha1</a>
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
<p>AWSJWTAuth stores reference to Authenticate against AWS using service account tokens.</p>
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
<p>AkeylessAuth configures how the operator authenticates with Akeyless.</p>
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
<p>AkeylessAuthSecretRef references a Secret that contains the details
to authenticate with Akeyless.
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
<p>AkeylessKubernetesAuth configures Kubernetes authentication with Akeyless.
It authenticates with Kubernetes ServiceAccount token stored.</p>
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
<p>AlibabaRRSAAuth authenticates against Alibaba using RRSA.</p>
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
<p>AwsAuthCredentials represents the credentials for AWS authentication.</p>
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
<p>AzureAuthCredentials represents the credentials for Azure authentication.</p>
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
<p>AzureAuthType describes how to authenticate to the Azure Keyvault
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
<td><p>AzureManagedIdentity uses Managed Identity to authenticate. Used with aad-pod-identity installed in the cluster.</p>
</td>
</tr><tr><td><p>&#34;ServicePrincipal&#34;</p></td>
<td><p>AzureServicePrincipal uses service principal to authenticate, which needs a tenantId, a clientId and a clientSecret.</p>
</td>
</tr><tr><td><p>&#34;WorkloadIdentity&#34;</p></td>
<td><p>AzureWorkloadIdentity uses Workload Identity service accounts to authenticate.</p>
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
<a href="#external-secrets.io/v1.AzureKVProvider">AzureKVProvider</a>, 
<a href="#generators.external-secrets.io/v1alpha1.ACRAccessTokenSpec">ACRAccessTokenSpec</a>)
</p>
<p>
<p>AzureEnvironmentType specifies the Azure cloud environment endpoints to use for
connecting and authenticating with Azure. By default, it points to the public cloud AAD endpoint.
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
<p>AzureKVAuth is the configuration used to authenticate with Azure.</p>
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
<p>AzureKVProvider configures a store to sync secrets using Azure KV.</p>
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
<p>BeyondTrustProviderSecretRef references a value that can be specified directly or via a secret
for a BeyondTrustProvider.</p>
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
<p>BeyondtrustAuth provides different ways to authenticate to a BeyondtrustProvider server.</p>
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
<p>BeyondtrustProvider provides access to a BeyondTrust secrets provider.</p>
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
<p>BeyondtrustServer configures a store to sync secrets using BeyondTrust Password Safe.</p>
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
<a href="#external-secrets.io/v1.SecretServerProvider">SecretServerProvider</a>, 
<a href="#external-secrets.io/v1.VaultProvider">VaultProvider</a>, 
<a href="#target.external-secrets.io/v1alpha1.KubernetesServer">KubernetesServer</a>)
</p>
<p>
<p>CAProvider provides a custom certificate authority for accessing the provider&rsquo;s store.
The CAProvider points to a Secret or ConfigMap resource that contains a PEM-encoded certificate.</p>
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
<p>CAProviderType defines the type of provider for certificate authority.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ConfigMap&#34;</p></td>
<td><p>CAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap resource.</p>
</td>
</tr><tr><td><p>&#34;Secret&#34;</p></td>
<td><p>CAProviderTypeSecret indicates that the CA certificate is stored in a Secret resource.</p>
</td>
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
<p>CertAuth defines certificate-based authentication configuration for Kubernetes.</p>
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
<p>ClusterExternalSecretConditionType defines a value type for ClusterExternalSecret conditions.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>ClusterExternalSecretReady is a ClusterExternalSecretConditionType set when the ClusterExternalSecret is ready.</p>
</td>
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
<p>ClusterExternalSecretStatusCondition defines the observed state of a ClusterExternalSecret resource.</p>
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
<p>ConjurAPIKey contains references to a Secret resource that holds
the Conjur username and API key.</p>
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
<p>ConjurAuth is the way to provide authentication credentials to the ConjurProvider.</p>
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
<p>ConjurJWT defines the JWT authentication configuration for Conjur provider.</p>
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
<p>ConjurProvider provides access to a Conjur provider.</p>
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
<p>DelineaProvider provides access to Delinea secrets vault Server.
See: <a href="https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go">https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go</a>.</p>
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
<p>DelineaProviderSecretRef is a secret reference containing either a direct value or a reference to a secret key.</p>
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
<p>Device42Auth defines the authentication method for the Device42 provider.</p>
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
<p>Device42SecretRef contains the secret reference for accessing the Device42 instance.</p>
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
<p>DopplerAuth defines the authentication method for the Doppler provider.</p>
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
<p>DopplerAuthSecretRef contains the secret reference for accessing the Doppler API.</p>
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
<p>ExternalSecret is the Schema for the external-secrets API.
It defines how to fetch data from external APIs and make it available as Kubernetes Secrets.</p>
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
<p>ExternalSecretConditionType defines a value type for ExternalSecret conditions.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Deleted&#34;</p></td>
<td><p>ExternalSecretDeleted indicates that the external secret has been deleted.</p>
</td>
</tr><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>ExternalSecretReady indicates that the external secret is ready and synced.</p>
</td>
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
<p>ExternalSecretConversionStrategy defines strategies for converting secret values.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Default&#34;</p></td>
<td><p>ExternalSecretConversionDefault specifies the default conversion strategy.</p>
</td>
</tr><tr><td><p>&#34;Unicode&#34;</p></td>
<td><p>ExternalSecretConversionUnicode specifies that values should be treated as Unicode.</p>
</td>
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
<td><p>CreatePolicyMerge does not create the Secret, but merges the data fields to the Secret.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>CreatePolicyNone does not create a Secret (future use with injector).</p>
</td>
</tr><tr><td><p>&#34;Orphan&#34;</p></td>
<td><p>CreatePolicyOrphan creates the Secret and does not set the ownerReference.
I.e. it will be orphaned after the deletion of the ExternalSecret.</p>
</td>
</tr><tr><td><p>&#34;Owner&#34;</p></td>
<td><p>CreatePolicyOwner creates the Secret and sets .metadata.ownerReferences to the ExternalSecret resource.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretData">ExternalSecretData
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.PullStep">PullStep</a>)
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
<p>The key in the Kubernetes Secret to store the value.
Note: Needs to remove validation pattern for workflows to properly work.</p>
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
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.PullStep">PullStep</a>)
</p>
<p>
<p>ExternalSecretDataFromRemoteRef defines the connection between the Kubernetes Secret keys and the Provider data
when using DataFrom to fetch multiple values from a Provider.</p>
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
<p>ExternalSecretDecodingStrategy defines strategies for decoding secret values.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Auto&#34;</p></td>
<td><p>ExternalSecretDecodeAuto specifies automatic detection of the decoding method.</p>
</td>
</tr><tr><td><p>&#34;Base64&#34;</p></td>
<td><p>ExternalSecretDecodeBase64 specifies that values should be decoded using Base64.</p>
</td>
</tr><tr><td><p>&#34;Base64URL&#34;</p></td>
<td><p>ExternalSecretDecodeBase64URL specifies that values should be decoded using Base64URL.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>ExternalSecretDecodeNone specifies that no decoding should be performed.</p>
</td>
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
<td><p>DeletionPolicyDelete deletes the secret if all provider secrets are deleted.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.</p>
</td>
</tr><tr><td><p>&#34;Merge&#34;</p></td>
<td><p>DeletionPolicyMerge removes keys in the secret, but not the secret itself.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.</p>
</td>
</tr><tr><td><p>&#34;Retain&#34;</p></td>
<td><p>DeletionPolicyRetain will retain the secret if all provider secrets have been deleted.
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
<p>ExternalSecretFind defines configuration for finding secrets in the provider.</p>
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
<p>ExternalSecretMetadataPolicy defines policies for fetching metadata from provider secrets.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Fetch&#34;</p></td>
<td><p>ExternalSecretMetadataPolicyFetch specifies that metadata should be fetched from the provider.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>ExternalSecretMetadataPolicyNone specifies that no metadata should be fetched from the provider.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRefreshPolicy">ExternalSecretRefreshPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
<p>ExternalSecretRefreshPolicy defines how and when the ExternalSecret should be refreshed.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;CreatedOnce&#34;</p></td>
<td><p>RefreshPolicyCreatedOnce creates the Secret once and does not update it thereafter.</p>
</td>
</tr><tr><td><p>&#34;OnChange&#34;</p></td>
<td><p>RefreshPolicyOnChange only synchronizes when the ExternalSecret&rsquo;s metadata or spec changes.</p>
</td>
</tr><tr><td><p>&#34;Periodic&#34;</p></td>
<td><p>RefreshPolicyPeriodic synchronizes the Secret from the provider at regular intervals.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewrite">ExternalSecretRewrite
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.GeneratorStep">GeneratorStep</a>)
</p>
<p>
<p>ExternalSecretRewrite defines how to rewrite secret data values before they are written to the Secret.</p>
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
<p>ExternalSecretRewriteMerge defines configuration for merging secret values.</p>
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
<p>ExternalSecretRewriteMergeConflictPolicy defines the policy for resolving conflicts when merging secrets.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Error&#34;</p></td>
<td><p>ExternalSecretRewriteMergeConflictPolicyError returns an error when conflicts occur during merge.</p>
</td>
</tr><tr><td><p>&#34;Ignore&#34;</p></td>
<td><p>ExternalSecretRewriteMergeConflictPolicyIgnore ignores conflicts when merging secret values.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteMergePriorityPolicy">ExternalSecretRewriteMergePriorityPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewriteMerge">ExternalSecretRewriteMerge</a>)
</p>
<p>
<p>ExternalSecretRewriteMergePriorityPolicy defines the policy for handling missing keys in the priority
list during merge operations.</p>
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
<p>ExternalSecretRewriteMergeStrategy defines the strategy for merging secrets.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Extract&#34;</p></td>
<td><p>ExternalSecretRewriteMergeStrategyExtract merges secrets by extracting values.</p>
</td>
</tr><tr><td><p>&#34;JSON&#34;</p></td>
<td><p>ExternalSecretRewriteMergeStrategyJSON merges secrets using JSON merge strategy.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretRewriteRegexp">ExternalSecretRewriteRegexp
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretRewrite">ExternalSecretRewrite</a>)
</p>
<p>
<p>ExternalSecretRewriteRegexp defines configuration for rewriting secrets using regular expressions.</p>
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
<p>ExternalSecretRewriteTransform defines configuration for transforming secrets using templates.</p>
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
<p>ExternalSecretStatus defines the observed state of ExternalSecret.</p>
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
<p>ExternalSecretStatusCondition defines a status condition of an ExternalSecret resource.</p>
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
<p>ExternalSecretTarget defines the Kubernetes Secret to be created,
there can be only one target per ExternalSecret.</p>
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
<code>manifest</code></br>
<em>
<a href="#external-secrets.io/v1.ManifestReference">
ManifestReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Manifest defines a custom Kubernetes resource to create instead of a Secret.
When specified, ExternalSecret will create the resource type defined here
(e.g., ConfigMap, Custom Resource) instead of a Secret.
Warning: Using Generic target. Make sure access policies and encryption are properly configured.</p>
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
<a href="#external-secrets.io/v1.ExternalSecretTarget">ExternalSecretTarget</a>, 
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">PushSecretSpec</a>)
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
<p>ExternalSecretValidator implements a validating webhook for ExternalSecrets.</p>
</p>
<h3 id="external-secrets.io/v1.ExternalSecretsAuth">ExternalSecretsAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretsProvider">ExternalSecretsProvider</a>)
</p>
<p>
<p>ExternalSecretsAuth defines authentication methods for External Secrets Enterprise.</p>
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
<code>kubernetes</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretsKubernetesAuth">
ExternalSecretsKubernetesAuth
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretsCARef">ExternalSecretsCARef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretsKubernetesAuth">ExternalSecretsKubernetesAuth</a>, 
<a href="#external-secrets.io/v1.ExternalSecretsServer">ExternalSecretsServer</a>)
</p>
<p>
<p>ExternalSecretsCARef defines a reference to a CA certificate.</p>
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
<code>bundle</code></br>
<em>
[]byte
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>configMapRef</code></br>
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
<h3 id="external-secrets.io/v1.ExternalSecretsKubernetesAuth">ExternalSecretsKubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretsAuth">ExternalSecretsAuth</a>)
</p>
<p>
<p>ExternalSecretsKubernetesAuth defines Kubernetes-based authentication for External Secrets Enterprise.</p>
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
<code>caCertRef</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretsCARef">
ExternalSecretsCARef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretsProvider">ExternalSecretsProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>ExternalSecretsProvider configures the External Secrets Enterprise provider.</p>
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
<a href="#external-secrets.io/v1.ExternalSecretsServer">
ExternalSecretsServer
</a>
</em>
</td>
<td>
<p>URL For the External Secrets Enterprise Server.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretsAuth">
ExternalSecretsAuth
</a>
</em>
</td>
<td>
<p>Authentication parameters for External Secrets Enterprise</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretsTarget">
ExternalSecretsTarget
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretsServer">ExternalSecretsServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretsProvider">ExternalSecretsProvider</a>)
</p>
<p>
<p>ExternalSecretsServer defines the server configuration for External Secrets Enterprise.</p>
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
<code>caRef</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretsCARef">
ExternalSecretsCARef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
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
<p>URL For the External Secrets Enterprise Server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.ExternalSecretsTarget">ExternalSecretsTarget
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretsProvider">ExternalSecretsProvider</a>)
</p>
<p>
<p>ExternalSecretsTarget specifies the target for External Secrets Enterprise operations.</p>
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
<code>clusterSecretStoreName</code></br>
<em>
string
</em>
</td>
<td>
<p>Remote clusterSecretStore to connect. Eventually, support more fields</p>
</td>
</tr>
</tbody>
</table>
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
<p>FakeProviderData defines a key-value pair with optional version for the fake provider.</p>
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
<p>FetchingPolicy configures how the provider interprets the <code>data.secretKey.remoteRef.key</code> field in ExternalSecret.</p>
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
<p>FindName defines criteria for finding secrets by name patterns.</p>
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
<p>FortanixProvider provides access to Fortanix SDKMS API using the provided credentials.</p>
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
<p>FortanixProviderSecretRef is a secret reference containing the SDKMS API Key.</p>
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
<p>GCPSMAuth defines the authentication methods for Google Cloud Platform Secret Manager.</p>
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
<a href="#external-secrets.io/v1.GCPSMAuth">GCPSMAuth</a>, 
<a href="#external-secrets.io/v1.VaultGCPAuth">VaultGCPAuth</a>)
</p>
<p>
<p>GCPSMAuthSecretRef contains the secret references for GCP Secret Manager authentication.</p>
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
<a href="#external-secrets.io/v1.GCPSMAuth">GCPSMAuth</a>, 
<a href="#external-secrets.io/v1.VaultGCPAuth">VaultGCPAuth</a>)
</p>
<p>
<p>GCPWorkloadIdentity defines configuration for workload identity authentication to GCP.</p>
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
<a href="#external-secrets.io/v1.GCPSMAuth">GCPSMAuth</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GCPSMAuth">GCPSMAuth</a>)
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
<h3 id="external-secrets.io/v1.GcpIDTokenAuthCredentials">GcpIDTokenAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
<p>GcpIDTokenAuthCredentials represents the credentials for GCP ID token authentication.</p>
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
<h3 id="external-secrets.io/v1.GcpIamAuthCredentials">GcpIamAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
<p>GcpIamAuthCredentials represents the credentials for GCP IAM authentication.</p>
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
<h3 id="external-secrets.io/v1.GeneratorRef">GeneratorRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.StoreGeneratorSourceRef">StoreGeneratorSourceRef</a>, 
<a href="#external-secrets.io/v1.StoreSourceRef">StoreSourceRef</a>, 
<a href="#external-secrets.io/v1alpha1.PushSecretSelector">PushSecretSelector</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.GeneratorStep">GeneratorStep</a>)
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
<p>Specify the name of the generator resource
Note: Needs to remove validation pattern for workflows to properly work.</p>
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
<p>GenericStoreValidator implements webhook validation for SecretStore and ClusterSecretStore resources.</p>
</p>
<h3 id="external-secrets.io/v1.GithubAppAuth">GithubAppAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.GithubProvider">GithubProvider</a>)
</p>
<p>
<p>GithubAppAuth defines authentication configuration using a GitHub App for accessing GitHub API.</p>
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
<p>GithubProvider provides access and authentication to a GitHub instance .</p>
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
<p>GitlabAuth defines the authentication method for accessing GitLab API.</p>
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
<p>GitlabProvider configures a store to sync secrets with a GitLab instance.</p>
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
<p>GitlabSecretRef contains the secret reference for GitLab authentication credentials.</p>
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
<p>IBMAuth defines authentication options for connecting to IBM Cloud Secrets Manager.</p>
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
<p>IBMAuthContainerAuth defines container-based authentication with IAM Trusted Profile.</p>
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
<p>IBMAuthSecretRef contains the secret reference for IBM Cloud API key authentication.</p>
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
<tr>
<td>
<code>iamEndpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>The IAM endpoint used to obain a token</p>
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
<p>IBMProvider configures a store to sync secrets using a IBM Cloud Secrets Manager
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
<p>InfisicalAuth specifies the authentication configuration for Infisical.</p>
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
<a href="#external-secrets.io/v1.GcpIDTokenAuthCredentials">
GcpIDTokenAuthCredentials
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
<p>JwtAuthCredentials represents the credentials for JWT authentication.</p>
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
<p>KubernetesAuth defines authentication options for connecting to a Kubernetes cluster.</p>
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
<p>KubernetesAuthCredentials represents the credentials for Kubernetes authentication.</p>
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
<p>KubernetesProvider configures a store to sync secrets with a Kubernetes instance.</p>
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
<p>KubernetesServer defines configuration for connecting to a Kubernetes API server.</p>
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
<p>LdapAuthCredentials represents the credentials for LDAP authentication.</p>
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
<p>MachineIdentityScopeInWorkspace defines the scope for machine identity within a workspace.</p>
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
<p>MaintenanceStatus defines a type for different maintenance states of a provider schema.</p>
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
<h3 id="external-secrets.io/v1.ManifestReference">ManifestReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTarget">ExternalSecretTarget</a>)
</p>
<p>
<p>ManifestReference defines a custom Kubernetes resource type to be created
instead of a Secret. This allows ExternalSecret to create ConfigMaps,
Custom Resources, or any other Kubernetes resource type.</p>
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
<p>APIVersion of the target resource (e.g., &ldquo;v1&rdquo; for ConfigMap, &ldquo;argoproj.io/v1alpha1&rdquo; for ArgoCD Application)</p>
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
<p>Kind of the target resource (e.g., &ldquo;ConfigMap&rdquo;, &ldquo;Application&rdquo;)</p>
</td>
</tr>
</tbody>
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
<h3 id="external-secrets.io/v1.NgrokAuth">NgrokAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.NgrokProvider">NgrokProvider</a>)
</p>
<p>
<p>NgrokAuth configures the authentication method for the ngrok provider.</p>
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
<a href="#external-secrets.io/v1.NgrokProviderSecretRef">
NgrokProviderSecretRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>APIKey is the API Key used to authenticate with ngrok. See <a href="https://ngrok.com/docs/api/#authentication">https://ngrok.com/docs/api/#authentication</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.NgrokProvider">NgrokProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>NgrokProvider configures a store to sync secrets with a ngrok vault to use in traffic policies.
See: <a href="https://ngrok.com/blog-post/secrets-for-traffic-policy">https://ngrok.com/blog-post/secrets-for-traffic-policy</a></p>
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
<p>APIURL is the URL of the ngrok API.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1.NgrokAuth">
NgrokAuth
</a>
</em>
</td>
<td>
<p>Auth configures how the ngrok provider authenticates with the ngrok API.</p>
</td>
</tr>
<tr>
<td>
<code>vault</code></br>
<em>
<a href="#external-secrets.io/v1.NgrokVault">
NgrokVault
</a>
</em>
</td>
<td>
<p>Vault configures the ngrok vault to sync secrets with.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.NgrokProviderSecretRef">NgrokProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.NgrokAuth">NgrokAuth</a>)
</p>
<p>
<p>NgrokProviderSecretRef contains the secret reference for the ngrok provider.</p>
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
<em>(Optional)</em>
<p>SecretRef is a reference to a secret containing the ngrok API key.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1.NgrokVault">NgrokVault
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.NgrokProvider">NgrokProvider</a>)
</p>
<p>
<p>NgrokVault configures the ngrok vault to sync secrets with.</p>
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
<p>Name is the name of the ngrok vault to sync secrets with.</p>
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
<p>OciAuthCredentials represents the credentials for OCI authentication.</p>
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
<p>OracleAuth defines the authentication method for the Oracle Vault provider.</p>
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
<p>OraclePrincipalType defines the type of principal used for authentication with Oracle Vault.</p>
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
<p>OracleProvider configures a store to sync secrets using an Oracle Vault
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
<p>OracleSecretRef contains the secret reference for Oracle Vault authentication credentials.</p>
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
<p>PassboltAuth contains a secretRef for the passbolt credentials.</p>
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
<p>PassboltProvider provides access to Passbolt secrets manager.
See: <a href="https://www.passbolt.com">https://www.passbolt.com</a>.</p>
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
<p>PasswordDepotAuth defines the authentication method for the Password Depot provider.</p>
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
<p>PasswordDepotProvider configures a store to sync secrets with a Password Depot instance.</p>
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
<p>PasswordDepotSecretRef contains the secret reference for Password Depot authentication.</p>
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
<p>PulumiProvider defines configuration for accessing secrets from Pulumi ESC.</p>
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
<p>PulumiProviderSecretRef contains the secret reference for Pulumi authentication.</p>
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
<p>ScalewayProvider defines the configuration for the Scaleway Secret Manager provider.</p>
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
<p>ScalewayProviderSecretRef defines the configuration for Scaleway secret references.</p>
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
<p>SecretServerProvider provides access to authenticate to a secrets provider server.
See: <a href="https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go">https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go</a>.</p>
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
<tr>
<td>
<code>caBundle</code></br>
<em>
[]byte
</em>
</td>
<td>
<em>(Optional)</em>
<p>PEM/base64 encoded CA bundle used to validate Secret ServerURL. Only used
if the ServerURL URL is using HTTPS protocol. If not set the system root certificates
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
<p>The provider for the CA bundle to use to validate Secret ServerURL certificate.</p>
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
<p>SecretServerProviderRef references a value that can be specified directly or via a secret
for a SecretServerProvider.</p>
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
<td><p>SecretStoreReadOnly indicates that the store can only read secrets.</p>
</td>
</tr><tr><td><p>&#34;ReadWrite&#34;</p></td>
<td><p>SecretStoreReadWrite indicates that the store can both read and write secrets.</p>
</td>
</tr><tr><td><p>&#34;WriteOnly&#34;</p></td>
<td><p>SecretStoreWriteOnly indicates that the store can only write secrets.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.SecretStoreConditionType">SecretStoreConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreStatusCondition">SecretStoreStatusCondition</a>)
</p>
<p>
<p>SecretStoreConditionType represents the condition of the SecretStore.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>SecretStoreReady indicates that the store is ready and able to serve requests.</p>
</td>
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
<code>externalsecrets</code></br>
<em>
<a href="#external-secrets.io/v1.ExternalSecretsProvider">
ExternalSecretsProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalSecrets configures this store to sync secrets using the ExternalSecrets provider</p>
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
<tr>
<td>
<code>ngrok</code></br>
<em>
<a href="#external-secrets.io/v1.NgrokProvider">
NgrokProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Ngrok configures this store to sync secrets using the ngrok provider.</p>
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
<a href="#external-secrets.io/v1.StoreSourceRef">StoreSourceRef</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.DestinationRef">DestinationRef</a>)
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
<p>Name of the SecretStore resource
Note: needs to remove validation pattern for workflows to properly work</p>
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
<p>Kind of the SecretStore resource (SecretStore, ClusterSecretStore or any Target kind)
Defaults to <code>SecretStore</code></p>
</td>
</tr>
<tr>
<td>
<code>group</code></br>
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
<h3 id="external-secrets.io/v1.SecretStoreRetrySettings">SecretStoreRetrySettings
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreSpec">SecretStoreSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecretSpec">VaultDynamicSecretSpec</a>)
</p>
<p>
<p>SecretStoreRetrySettings defines the retry settings for accessing external secrets manager stores.</p>
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
<p>SecretStoreStatusCondition contains condition information for a SecretStore.</p>
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
<p>SecretVersionSelectionPolicy defines the policy for selecting secret versions in GCP Secret Manager.</p>
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
then by default Secrets Manager uses a 30-day recovery window.
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
<a href="#external-secrets.io/v1.ExternalSecretData">ExternalSecretData</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.PullStep">PullStep</a>)
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
<p>Tag is a key-value pair that can be attached to an AWS resource.
see: <a href="https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html">https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html</a></p>
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
<p>TemplateEngineVersion specifies the template engine version that should be used to
compile/execute the template.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;v2&#34;</p></td>
<td><p>TemplateEngineV2 is the currently supported template engine version.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.TemplateFrom">TemplateFrom
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
<p>TemplateFrom specifies a source for templates.
Each item in the list can either reference a ConfigMap or a Secret resource.</p>
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
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Target specifies where to place the template result.
For Secret resources, common values are: &ldquo;Data&rdquo;, &ldquo;Annotations&rdquo;, &ldquo;Labels&rdquo;.
For custom resources (when spec.target.manifest is set), this supports
nested paths like &ldquo;spec.database.config&rdquo; or &ldquo;data&rdquo;.</p>
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
<p>TemplateMergePolicy defines how the rendered template should be merged with the existing Secret data.</p>
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
<p>TemplateRef specifies a reference to either a ConfigMap or a Secret resource.</p>
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
<p>TemplateRefItem specifies a key in the ConfigMap/Secret to use as a template for Secret data.</p>
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
<p>TemplateScope specifies how the template keys should be interpreted.</p>
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
<h3 id="external-secrets.io/v1.TokenAuth">TokenAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.KubernetesAuth">KubernetesAuth</a>)
</p>
<p>
<p>TokenAuth defines token-based authentication configuration for Kubernetes.</p>
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
<p>TokenAuthCredentials represents the credentials for access token-based authentication.</p>
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
<p>UniversalAuthCredentials represents the client credentials for universal authentication.</p>
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
<p>ValidationResult is defined type for the number of validation results.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>2</p></td>
<td><p>ValidationResultError indicates that there is a misconfiguration.</p>
</td>
</tr><tr><td><p>0</p></td>
<td><p>ValidationResultReady indicates that the client is configured correctly
and can be used.</p>
</td>
</tr><tr><td><p>1</p></td>
<td><p>ValidationResultUnknown indicates that the client can be used
but information is missing, and it can not be validated.</p>
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
Only one of <code>tokenSecretRef</code>, <code>appRole</code>,  <code>kubernetes</code>, <code>ldap</code>, <code>userPass</code>, <code>jwt</code>, <code>cert</code>, <code>iam</code> or <code>gcp</code>
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
<tr>
<td>
<code>gcp</code></br>
<em>
<a href="#external-secrets.io/v1.VaultGCPAuth">
VaultGCPAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Gcp authenticates with Vault using Google Cloud Platform authentication method
GCP authentication method</p>
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
<p>VaultAwsAuthSecretRef holds secret references for AWS credentials
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
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Path where the Certificate authentication backend is mounted
in Vault, e.g: &ldquo;cert&rdquo;</p>
</td>
</tr>
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
<h3 id="external-secrets.io/v1.VaultGCPAuth">VaultGCPAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultGCPAuth authenticates with Vault using Google Cloud Platform authentication method.
Refer: <a href="https://developer.hashicorp.com/vault/docs/auth/gcp">https://developer.hashicorp.com/vault/docs/auth/gcp</a></p>
<p>When ServiceAccountRef, SecretRef and WorkloadIdentity are not specified, the provider will use the controller pod&rsquo;s
identity to authenticate with GCP. This supports both GKE Workload Identity and service account keys.</p>
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
<p>Path where the GCP auth method is enabled in Vault, e.g: &ldquo;gcp&rdquo;</p>
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
<p>Vault Role. In Vault, a role describes an identity with a set of permissions, groups, or policies you want to attach to a user of the secrets engine.</p>
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
<em>(Optional)</em>
<p>Project ID of the Google Cloud Platform project</p>
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
<em>(Optional)</em>
<p>Location optionally defines a location/region for the secret</p>
</td>
</tr>
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
<p>Specify credentials in a Secret object</p>
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
<p>Specify a service account with Workload Identity</p>
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
<p>ServiceAccountRef to a service account for impersonation</p>
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
<p>VaultKVStoreVersion represents the version of the Vault KV secret engine.</p>
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
<p>VaultKubernetesAuth authenticates against Vault using a Kubernetes ServiceAccount token stored in
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
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>, 
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecretSpec">VaultDynamicSecretSpec</a>)
</p>
<p>
<p>VaultProvider configures a store to sync secrets using a Hashicorp Vault KV backend.</p>
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
<p>WebhookCAProvider defines a location to fetch the cert for the webhook provider from.</p>
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
<p>WebhookCAProviderType defines the type of provider for certificate authority in webhook connections.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ConfigMap&#34;</p></td>
<td><p>WebhookCAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap resource.</p>
</td>
</tr><tr><td><p>&#34;Secret&#34;</p></td>
<td><p>WebhookCAProviderTypeSecret indicates that the CA certificate is stored in a Secret resource.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1.WebhookProvider">WebhookProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>WebhookProvider configures a store to sync secrets from simple web APIs.</p>
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
<em>(Optional)</em>
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
<p>WebhookResult defines how to process and extract secrets from the webhook response.</p>
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
<p>WebhookSecret defines a secret that will be passed to the webhook request.</p>
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
<p>YandexAuth defines the authentication method for the Yandex provider.</p>
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
<p>YandexCAProvider defines the configuration for Yandex custom certificate authority.</p>
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
<h2 id="external-secrets.io/v1alpha1">external-secrets.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains resources for external-secrets</p>
</p>
Resource Types:
<ul></ul>
<h3 id="external-secrets.io/v1alpha1.ClusterPushSecret">ClusterPushSecret
</h3>
<p>
<p>ClusterPushSecret is the Schema for the ClusterPushSecrets API that enables cluster-wide management of pushing Kubernetes secrets to external providers.</p>
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
<a href="#external-secrets.io/v1alpha1.ClusterPushSecretSpec">
ClusterPushSecretSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>pushSecretSpec</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">
PushSecretSpec
</a>
</em>
</td>
<td>
<p>PushSecretSpec defines what to do with the secrets.</p>
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
<tr>
<td>
<code>pushSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the push secrets to be created.
Defaults to the name of the ClusterPushSecret</p>
</td>
</tr>
<tr>
<td>
<code>pushSecretMetadata</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretMetadata">
PushSecretMetadata
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
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecretStatus">
ClusterPushSecretStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.ClusterPushSecretCondition">ClusterPushSecretCondition
</h3>
<p>
<p>ClusterPushSecretCondition used to refine PushSecrets to specific namespaces and names.</p>
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
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.ClusterPushSecretNamespaceFailure">ClusterPushSecretNamespaceFailure
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecretStatus">ClusterPushSecretStatus</a>)
</p>
<p>
<p>ClusterPushSecretNamespaceFailure represents a failed namespace deployment and it&rsquo;s reason.</p>
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
<p>Namespace is the namespace that failed when trying to apply an PushSecret</p>
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
<p>Reason is why the PushSecret failed to apply to the namespace</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.ClusterPushSecretSpec">ClusterPushSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecret">ClusterPushSecret</a>)
</p>
<p>
<p>ClusterPushSecretSpec defines the configuration for a ClusterPushSecret resource.</p>
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
<code>pushSecretSpec</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">
PushSecretSpec
</a>
</em>
</td>
<td>
<p>PushSecretSpec defines what to do with the secrets.</p>
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
<tr>
<td>
<code>pushSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The name of the push secrets to be created.
Defaults to the name of the ClusterPushSecret</p>
</td>
</tr>
<tr>
<td>
<code>pushSecretMetadata</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretMetadata">
PushSecretMetadata
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
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.ClusterPushSecretStatus">ClusterPushSecretStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecret">ClusterPushSecret</a>)
</p>
<p>
<p>ClusterPushSecretStatus contains the status information for the ClusterPushSecret resource.</p>
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
<code>failedNamespaces</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecretNamespaceFailure">
[]ClusterPushSecretNamespaceFailure
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Failed namespaces are the namespaces that failed to apply an PushSecret</p>
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
<p>ProvisionedNamespaces are the namespaces where the ClusterPushSecret has secrets</p>
</td>
</tr>
<tr>
<td>
<code>pushSecretName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretStatusCondition">
[]PushSecretStatusCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecret">PushSecret
</h3>
<p>
<p>PushSecret is the Schema for the PushSecrets API that enables pushing Kubernetes secrets to external secret providers.</p>
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
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">
PushSecretSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
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
<p>The Interval to which External Secrets will try to push a secret definition</p>
</td>
</tr>
<tr>
<td>
<code>secretStoreRefs</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretStoreRef">
[]PushSecretStoreRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>updatePolicy</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretUpdatePolicy">
PushSecretUpdatePolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>UpdatePolicy to handle Secrets in the provider.</p>
</td>
</tr>
<tr>
<td>
<code>deletionPolicy</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretDeletionPolicy">
PushSecretDeletionPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deletion Policy to handle Secrets in the provider.</p>
</td>
</tr>
<tr>
<td>
<code>selector</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretSelector">
PushSecretSelector
</a>
</em>
</td>
<td>
<p>The Secret Selector (k8s source) for the Push Secret</p>
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretData">
[]PushSecretData
</a>
</em>
</td>
<td>
<p>Secret Data that should be pushed to providers</p>
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
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretStatus">
PushSecretStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretConditionType">PushSecretConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretStatusCondition">PushSecretStatusCondition</a>)
</p>
<p>
<p>PushSecretConditionType indicates the condition of the PushSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>PushSecretReady indicates the PushSecret resource is ready.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretConversionStrategy">PushSecretConversionStrategy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretData">PushSecretData</a>)
</p>
<p>
<p>PushSecretConversionStrategy defines how secret values are converted when pushed to providers.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;None&#34;</p></td>
<td><p>PushSecretConversionNone indicates no conversion will be performed on the secret value.</p>
</td>
</tr><tr><td><p>&#34;ReverseUnicode&#34;</p></td>
<td><p>PushSecretConversionReverseUnicode indicates that unicode escape sequences will be reversed.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretData">PushSecretData
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">PushSecretSpec</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.PushStep">PushStep</a>)
</p>
<p>
<p>PushSecretData defines data to be pushed to the provider and associated metadata.</p>
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
<code>match</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretMatch">
PushSecretMatch
</a>
</em>
</td>
<td>
<p>Match a given Secret Key to be pushed to the provider.</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Metadata is metadata attached to the secret.
The structure of metadata is provider specific, please look it up in the provider documentation.</p>
</td>
</tr>
<tr>
<td>
<code>conversionStrategy</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretConversionStrategy">
PushSecretConversionStrategy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Used to define a conversion Strategy for the secret keys</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretDeletionPolicy">PushSecretDeletionPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">PushSecretSpec</a>)
</p>
<p>
<p>PushSecretDeletionPolicy defines how push secrets are deleted in the provider.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Delete&#34;</p></td>
<td><p>PushSecretDeletionPolicyDelete deletes secrets from the provider when the PushSecret is deleted.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>PushSecretDeletionPolicyNone keeps secrets in the provider when the PushSecret is deleted.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretMatch">PushSecretMatch
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretData">PushSecretData</a>)
</p>
<p>
<p>PushSecretMatch defines how a source Secret key maps to a destination in the provider.</p>
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
<em>(Optional)</em>
<p>Secret Key to be pushed</p>
</td>
</tr>
<tr>
<td>
<code>remoteRef</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretRemoteRef">
PushSecretRemoteRef
</a>
</em>
</td>
<td>
<p>Remote Refs to push to providers.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretMetadata">PushSecretMetadata
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecretSpec">ClusterPushSecretSpec</a>)
</p>
<p>
<p>PushSecretMetadata defines metadata fields for the PushSecret generated by the ClusterPushSecret.</p>
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
<h3 id="external-secrets.io/v1alpha1.PushSecretRemoteRef">PushSecretRemoteRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretMatch">PushSecretMatch</a>)
</p>
<p>
<p>PushSecretRemoteRef defines the location of the secret in the provider.</p>
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
<code>remoteKey</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the resulting provider secret.</p>
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
<p>Name of the property in the resulting secret</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretSecret">PushSecretSecret
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretSelector">PushSecretSelector</a>)
</p>
<p>
<p>PushSecretSecret defines a Secret that will be used as a source for pushing to providers.</p>
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
<p>Name of the Secret.
The Secret must exist in the same namespace as the PushSecret manifest.</p>
</td>
</tr>
<tr>
<td>
<code>selector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Selector chooses secrets using a labelSelector.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretSelector">PushSecretSelector
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">PushSecretSpec</a>)
</p>
<p>
<p>PushSecretSelector defines criteria for selecting the source Secret for pushing to providers.</p>
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
<code>secret</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretSecret">
PushSecretSecret
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Select a Secret to Push.</p>
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
<p>Point to a generator to create a Secret.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretSpec">PushSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecretSpec">ClusterPushSecretSpec</a>, 
<a href="#external-secrets.io/v1alpha1.PushSecret">PushSecret</a>)
</p>
<p>
<p>PushSecretSpec configures the behavior of the PushSecret.</p>
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
<code>refreshInterval</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>The Interval to which External Secrets will try to push a secret definition</p>
</td>
</tr>
<tr>
<td>
<code>secretStoreRefs</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretStoreRef">
[]PushSecretStoreRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>updatePolicy</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretUpdatePolicy">
PushSecretUpdatePolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>UpdatePolicy to handle Secrets in the provider.</p>
</td>
</tr>
<tr>
<td>
<code>deletionPolicy</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretDeletionPolicy">
PushSecretDeletionPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Deletion Policy to handle Secrets in the provider.</p>
</td>
</tr>
<tr>
<td>
<code>selector</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretSelector">
PushSecretSelector
</a>
</em>
</td>
<td>
<p>The Secret Selector (k8s source) for the Push Secret</p>
</td>
</tr>
<tr>
<td>
<code>data</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretData">
[]PushSecretData
</a>
</em>
</td>
<td>
<p>Secret Data that should be pushed to providers</p>
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
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretStatus">PushSecretStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecret">PushSecret</a>)
</p>
<p>
<p>PushSecretStatus indicates the history of the status of PushSecret.</p>
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
<p>SyncedResourceVersion keeps track of the last synced version.</p>
</td>
</tr>
<tr>
<td>
<code>syncedPushSecrets</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.SyncedPushSecretsMap">
SyncedPushSecretsMap
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Synced PushSecrets, including secrets that already exist in provider.
Matches secret stores to PushSecretData that was stored to that secret store.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#external-secrets.io/v1alpha1.PushSecretStatusCondition">
[]PushSecretStatusCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretStatusCondition">PushSecretStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.ClusterPushSecretStatus">ClusterPushSecretStatus</a>, 
<a href="#external-secrets.io/v1alpha1.PushSecretStatus">PushSecretStatus</a>)
</p>
<p>
<p>PushSecretStatusCondition indicates the status of the PushSecret.</p>
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
<a href="#external-secrets.io/v1alpha1.PushSecretConditionType">
PushSecretConditionType
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
<h3 id="external-secrets.io/v1alpha1.PushSecretStoreRef">PushSecretStoreRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">PushSecretSpec</a>)
</p>
<p>
<p>PushSecretStoreRef contains a reference on how to sync to a SecretStore.</p>
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
<p>Optionally, sync to the SecretStore of the given name</p>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optionally, sync to secret stores with label selector</p>
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
<p>Kind of the SecretStore resource (SecretStore or ClusterSecretStore)</p>
</td>
</tr>
<tr>
<td>
<code>group</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Group of the SecretStore resource (SecretStore or ClusterSecretStore, or any Target)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.PushSecretUpdatePolicy">PushSecretUpdatePolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretSpec">PushSecretSpec</a>)
</p>
<p>
<p>PushSecretUpdatePolicy defines how push secrets are updated in the provider.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;IfNotExists&#34;</p></td>
<td><p>PushSecretUpdatePolicyIfNotExists only creates secrets that don&rsquo;t exist in the provider.</p>
</td>
</tr><tr><td><p>&#34;Replace&#34;</p></td>
<td><p>PushSecretUpdatePolicyReplace replaces existing secrets in the provider.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1alpha1.SyncedPushSecretsMap">SyncedPushSecretsMap
(<code>map[string]map[string]github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1.PushSecretData</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1alpha1.PushSecretStatus">PushSecretStatus</a>)
</p>
<p>
<p>SyncedPushSecretsMap is a map that tracks which PushSecretData was stored to which secret store.
The outer map&rsquo;s key is the secret store name, and the inner map&rsquo;s key is the remote key name.</p>
</p>
<hr/>
<h2 id="external-secrets.io/v1beta1">external-secrets.io/v1beta1</h2>
<p>
<p>Package v1beta1 contains resources for external-secrets</p>
</p>
Resource Types:
<ul></ul>
<h3 id="external-secrets.io/v1beta1.AWSAuth">AWSAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AWSProvider">AWSProvider</a>)
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
<a href="#external-secrets.io/v1beta1.AWSAuthSecretRef">
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
<a href="#external-secrets.io/v1beta1.AWSJWTAuth">
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
<h3 id="external-secrets.io/v1beta1.AWSAuthSecretRef">AWSAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AWSAuth">AWSAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.AWSJWTAuth">AWSJWTAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AWSAuth">AWSAuth</a>)
</p>
<p>
<p>AWSJWTAuth authenticates against AWS using service account tokens from the Kubernetes cluster.</p>
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
<h3 id="external-secrets.io/v1beta1.AWSProvider">AWSProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.AWSServiceType">
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
<a href="#external-secrets.io/v1beta1.AWSAuth">
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
<a href="#external-secrets.io/v1beta1.*github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1.Tag">
[]*github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1.Tag
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
<a href="#external-secrets.io/v1beta1.SecretsManager">
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
[]*string
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
<h3 id="external-secrets.io/v1beta1.AWSServiceType">AWSServiceType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AWSProvider">AWSProvider</a>)
</p>
<p>
<p>AWSServiceType is an enum that defines the service/API that is used to fetch the secrets.</p>
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
<h3 id="external-secrets.io/v1beta1.AkeylessAuth">AkeylessAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AkeylessProvider">AkeylessProvider</a>)
</p>
<p>
<p>AkeylessAuth defines methods of authentication with Akeyless Vault.</p>
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
<a href="#external-secrets.io/v1beta1.AkeylessAuthSecretRef">
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
<a href="#external-secrets.io/v1beta1.AkeylessKubernetesAuth">
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
<h3 id="external-secrets.io/v1beta1.AkeylessAuthSecretRef">AkeylessAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AkeylessAuth">AkeylessAuth</a>)
</p>
<p>
<p>AkeylessAuthSecretRef defines how to authenticate using a secret reference.
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
<h3 id="external-secrets.io/v1beta1.AkeylessKubernetesAuth">AkeylessKubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AkeylessAuth">AkeylessAuth</a>)
</p>
<p>
<p>AkeylessKubernetesAuth authenticates with Akeyless using a Kubernetes ServiceAccount token.</p>
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
<h3 id="external-secrets.io/v1beta1.AkeylessProvider">AkeylessProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.AkeylessAuth">
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
<a href="#external-secrets.io/v1beta1.CAProvider">
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
<h3 id="external-secrets.io/v1beta1.AlibabaAuth">AlibabaAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AlibabaProvider">AlibabaProvider</a>)
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
<a href="#external-secrets.io/v1beta1.AlibabaAuthSecretRef">
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
<a href="#external-secrets.io/v1beta1.AlibabaRRSAAuth">
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
<h3 id="external-secrets.io/v1beta1.AlibabaAuthSecretRef">AlibabaAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AlibabaAuth">AlibabaAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.AlibabaProvider">AlibabaProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.AlibabaAuth">
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
<h3 id="external-secrets.io/v1beta1.AlibabaRRSAAuth">AlibabaRRSAAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AlibabaAuth">AlibabaAuth</a>)
</p>
<p>
<p>AlibabaRRSAAuth authenticates against Alibaba using RRSA (Resource-oriented RAM-based Service Authentication).</p>
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
<h3 id="external-secrets.io/v1beta1.AuthorizationProtocol">AuthorizationProtocol
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.WebhookProvider">WebhookProvider</a>)
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
<a href="#external-secrets.io/v1beta1.NTLMProtocol">
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
<h3 id="external-secrets.io/v1beta1.AzureAuthType">AzureAuthType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AzureKVProvider">AzureKVProvider</a>)
</p>
<p>
<p>AzureAuthType describes how to authenticate to the Azure Keyvault.
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
<td><p>AzureManagedIdentity uses Managed Identity to authenticate. Used with aad-pod-identity installed in the cluster.</p>
</td>
</tr><tr><td><p>&#34;ServicePrincipal&#34;</p></td>
<td><p>AzureServicePrincipal uses service principal to authenticate, which needs a tenantId, a clientId and a clientSecret.</p>
</td>
</tr><tr><td><p>&#34;WorkloadIdentity&#34;</p></td>
<td><p>AzureWorkloadIdentity uses Workload Identity service accounts to authenticate.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.AzureEnvironmentType">AzureEnvironmentType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AzureKVProvider">AzureKVProvider</a>)
</p>
<p>
<p>AzureEnvironmentType specifies the Azure cloud environment endpoints to use for
connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: <a href="https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152">https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152</a>
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ChinaCloud&#34;</p></td>
<td><p>AzureEnvironmentChinaCloud represents the Azure China cloud environment.</p>
</td>
</tr><tr><td><p>&#34;GermanCloud&#34;</p></td>
<td><p>AzureEnvironmentGermanCloud represents the Azure German cloud environment.</p>
</td>
</tr><tr><td><p>&#34;PublicCloud&#34;</p></td>
<td><p>AzureEnvironmentPublicCloud represents the Azure public cloud environment.</p>
</td>
</tr><tr><td><p>&#34;USGovernmentCloud&#34;</p></td>
<td><p>AzureEnvironmentUSGovernmentCloud represents the Azure US government cloud environment.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.AzureKVAuth">AzureKVAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AzureKVProvider">AzureKVProvider</a>)
</p>
<p>
<p>AzureKVAuth defines configuration for authentication with Azure Key Vault.</p>
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
<h3 id="external-secrets.io/v1beta1.AzureKVProvider">AzureKVProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>AzureKVProvider configures a store to sync secrets using Azure Key Vault.</p>
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
<a href="#external-secrets.io/v1beta1.AzureAuthType">
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
<a href="#external-secrets.io/v1beta1.AzureEnvironmentType">
AzureEnvironmentType
</a>
</em>
</td>
<td>
<p>EnvironmentType specifies the Azure cloud environment endpoints to use for
connecting and authenticating with Azure. By default it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: <a href="https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152">https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152</a>
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud</p>
</td>
</tr>
<tr>
<td>
<code>authSecretRef</code></br>
<em>
<a href="#external-secrets.io/v1beta1.AzureKVAuth">
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
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.BeyondTrustProviderSecretRef">BeyondTrustProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.BeyondtrustAuth">BeyondtrustAuth</a>)
</p>
<p>
<p>BeyondTrustProviderSecretRef defines a reference to a secret containing credentials for the BeyondTrust provider.</p>
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
<h3 id="external-secrets.io/v1beta1.BeyondtrustAuth">BeyondtrustAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.BeyondtrustProvider">BeyondtrustProvider</a>)
</p>
<p>
<p>BeyondtrustAuth configures authentication for BeyondTrust Password Safe.</p>
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
<a href="#external-secrets.io/v1beta1.BeyondTrustProviderSecretRef">
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
<a href="#external-secrets.io/v1beta1.BeyondTrustProviderSecretRef">
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
<a href="#external-secrets.io/v1beta1.BeyondTrustProviderSecretRef">
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
<a href="#external-secrets.io/v1beta1.BeyondTrustProviderSecretRef">
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
<a href="#external-secrets.io/v1beta1.BeyondTrustProviderSecretRef">
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
<h3 id="external-secrets.io/v1beta1.BeyondtrustProvider">BeyondtrustProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>BeyondtrustProvider defines configuration for the BeyondTrust Password Safe provider.</p>
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
<a href="#external-secrets.io/v1beta1.BeyondtrustAuth">
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
<a href="#external-secrets.io/v1beta1.BeyondtrustServer">
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
<h3 id="external-secrets.io/v1beta1.BeyondtrustServer">BeyondtrustServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.BeyondtrustProvider">BeyondtrustProvider</a>)
</p>
<p>
<p>BeyondtrustServer defines configuration for connecting to BeyondTrust Password Safe server.</p>
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
<h3 id="external-secrets.io/v1beta1.BitwardenSecretsManagerAuth">BitwardenSecretsManagerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.BitwardenSecretsManagerProvider">BitwardenSecretsManagerProvider</a>)
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
<a href="#external-secrets.io/v1beta1.BitwardenSecretsManagerSecretRef">
BitwardenSecretsManagerSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.BitwardenSecretsManagerProvider">BitwardenSecretsManagerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.CAProvider">
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
<a href="#external-secrets.io/v1beta1.BitwardenSecretsManagerAuth">
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
<h3 id="external-secrets.io/v1beta1.BitwardenSecretsManagerSecretRef">BitwardenSecretsManagerSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.BitwardenSecretsManagerAuth">BitwardenSecretsManagerAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.CAProvider">CAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AkeylessProvider">AkeylessProvider</a>, 
<a href="#external-secrets.io/v1beta1.BitwardenSecretsManagerProvider">BitwardenSecretsManagerProvider</a>, 
<a href="#external-secrets.io/v1beta1.ConjurProvider">ConjurProvider</a>, 
<a href="#external-secrets.io/v1beta1.GitlabProvider">GitlabProvider</a>, 
<a href="#external-secrets.io/v1beta1.KubernetesServer">KubernetesServer</a>, 
<a href="#external-secrets.io/v1beta1.VaultProvider">VaultProvider</a>)
</p>
<p>
<p>CAProvider provides custom certificate authority (CA) certificates
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
<a href="#external-secrets.io/v1beta1.CAProviderType">
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
<h3 id="external-secrets.io/v1beta1.CAProviderType">CAProviderType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.CAProvider">CAProvider</a>)
</p>
<p>
<p>CAProviderType defines the type of provider to use for CA certificates.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ConfigMap&#34;</p></td>
<td><p>CAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap.</p>
</td>
</tr><tr><td><p>&#34;Secret&#34;</p></td>
<td><p>CAProviderTypeSecret indicates that the CA certificate is stored in a Secret.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.CSMAuth">CSMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.CloudruSMProvider">CloudruSMProvider</a>)
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
<a href="#external-secrets.io/v1beta1.CSMAuthSecretRef">
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
<h3 id="external-secrets.io/v1beta1.CSMAuthSecretRef">CSMAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.CSMAuth">CSMAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.CertAuth">CertAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.KubernetesAuth">KubernetesAuth</a>)
</p>
<p>
<p>CertAuth defines certificate-based authentication for the Kubernetes provider.</p>
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
<h3 id="external-secrets.io/v1beta1.ChefAuth">ChefAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ChefProvider">ChefProvider</a>)
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
<a href="#external-secrets.io/v1beta1.ChefAuthSecretRef">
ChefAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ChefAuthSecretRef">ChefAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ChefAuth">ChefAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.ChefProvider">ChefProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.ChefAuth">
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
<h3 id="external-secrets.io/v1beta1.CloudruSMProvider">CloudruSMProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.CSMAuth">
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
<h3 id="external-secrets.io/v1beta1.ClusterExternalSecret">ClusterExternalSecret
</h3>
<p>
<p>ClusterExternalSecret is the schema for the clusterexternalsecrets API.</p>
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
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretSpec">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretMetadata">
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
<p>The labels to select by to find the Namespaces to create the ExternalSecrets in</p>
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
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretStatus">
ClusterExternalSecretStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ClusterExternalSecretConditionType">ClusterExternalSecretConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretStatusCondition">ClusterExternalSecretStatusCondition</a>)
</p>
<p>
<p>ClusterExternalSecretConditionType indicates the condition of the ClusterExternalSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>ClusterExternalSecretReady indicates the ClusterExternalSecret resource is ready.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ClusterExternalSecretNamespaceFailure">ClusterExternalSecretNamespaceFailure
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretStatus">ClusterExternalSecretStatus</a>)
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
<h3 id="external-secrets.io/v1beta1.ClusterExternalSecretSpec">ClusterExternalSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterExternalSecret">ClusterExternalSecret</a>)
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
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretMetadata">
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
<p>The labels to select by to find the Namespaces to create the ExternalSecrets in</p>
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
<h3 id="external-secrets.io/v1beta1.ClusterExternalSecretStatus">ClusterExternalSecretStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterExternalSecret">ClusterExternalSecret</a>)
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
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretNamespaceFailure">
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
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretStatusCondition">
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
<h3 id="external-secrets.io/v1beta1.ClusterExternalSecretStatusCondition">ClusterExternalSecretStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretStatus">ClusterExternalSecretStatus</a>)
</p>
<p>
<p>ClusterExternalSecretStatusCondition indicates the status of the ClusterExternalSecret.</p>
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
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretConditionType">
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
<h3 id="external-secrets.io/v1beta1.ClusterSecretStore">ClusterSecretStore
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
<a href="#external-secrets.io/v1beta1.SecretStoreSpec">
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
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">
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
<a href="#external-secrets.io/v1beta1.SecretStoreRetrySettings">
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
<a href="#external-secrets.io/v1beta1.ClusterSecretStoreCondition">
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
<a href="#external-secrets.io/v1beta1.SecretStoreStatus">
SecretStoreStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ClusterSecretStoreCondition">ClusterSecretStoreCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreSpec">SecretStoreSpec</a>)
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
<h3 id="external-secrets.io/v1beta1.ConjurAPIKey">ConjurAPIKey
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ConjurAuth">ConjurAuth</a>)
</p>
<p>
<p>ConjurAPIKey defines authentication using a Conjur API key.</p>
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
<h3 id="external-secrets.io/v1beta1.ConjurAuth">ConjurAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ConjurProvider">ConjurProvider</a>)
</p>
<p>
<p>ConjurAuth defines the methods of authentication with Conjur.</p>
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
<a href="#external-secrets.io/v1beta1.ConjurAPIKey">
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
<a href="#external-secrets.io/v1beta1.ConjurJWT">
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
<h3 id="external-secrets.io/v1beta1.ConjurJWT">ConjurJWT
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ConjurAuth">ConjurAuth</a>)
</p>
<p>
<p>ConjurJWT defines authentication using a JWT service account token.</p>
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
<h3 id="external-secrets.io/v1beta1.ConjurProvider">ConjurProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>ConjurProvider defines configuration for the CyberArk Conjur provider.</p>
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
<a href="#external-secrets.io/v1beta1.CAProvider">
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
<a href="#external-secrets.io/v1beta1.ConjurAuth">
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
<h3 id="external-secrets.io/v1beta1.DelineaProvider">DelineaProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>DelineaProvider defines configuration for the Delinea DevOps Secrets Vault provider.
See <a href="https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go">https://github.com/DelineaXPM/dsv-sdk-go/blob/main/vault/vault.go</a>.</p>
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
<a href="#external-secrets.io/v1beta1.DelineaProviderSecretRef">
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
<a href="#external-secrets.io/v1beta1.DelineaProviderSecretRef">
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
<h3 id="external-secrets.io/v1beta1.DelineaProviderSecretRef">DelineaProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.DelineaProvider">DelineaProvider</a>)
</p>
<p>
<p>DelineaProviderSecretRef defines a reference to a secret containing credentials for the Delinea provider.</p>
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
<h3 id="external-secrets.io/v1beta1.Device42Auth">Device42Auth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.Device42Provider">Device42Provider</a>)
</p>
<p>
<p>Device42Auth defines the authentication method for the Device42 provider.</p>
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
<a href="#external-secrets.io/v1beta1.Device42SecretRef">
Device42SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.Device42Provider">Device42Provider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.Device42Auth">
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
<h3 id="external-secrets.io/v1beta1.Device42SecretRef">Device42SecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.Device42Auth">Device42Auth</a>)
</p>
<p>
<p>Device42SecretRef defines a reference to a secret containing credentials for the Device42 provider.</p>
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
<h3 id="external-secrets.io/v1beta1.DopplerAuth">DopplerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.DopplerProvider">DopplerProvider</a>)
</p>
<p>
<p>DopplerAuth defines the authentication method for the Doppler provider.</p>
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
<a href="#external-secrets.io/v1beta1.DopplerAuthSecretRef">
DopplerAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.DopplerAuthSecretRef">DopplerAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.DopplerAuth">DopplerAuth</a>)
</p>
<p>
<p>DopplerAuthSecretRef defines a reference to a secret containing credentials for the Doppler provider.</p>
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
<h3 id="external-secrets.io/v1beta1.DopplerProvider">DopplerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.DopplerAuth">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecret">ExternalSecret
</h3>
<p>
<p>ExternalSecret is the schema for the external-secrets API.</p>
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
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">
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
<a href="#external-secrets.io/v1beta1.SecretStoreRef">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretTarget">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretRefreshPolicy">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretData">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretDataFromRemoteRef">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretStatus">
ExternalSecretStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretConditionType">ExternalSecretConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretStatusCondition">ExternalSecretStatusCondition</a>)
</p>
<p>
<p>ExternalSecretConditionType defines the condition type for an ExternalSecret.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Deleted&#34;</p></td>
<td><p>ExternalSecretDeleted indicates the ExternalSecret has been deleted.</p>
</td>
</tr><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>ExternalSecretReady indicates the ExternalSecret has been successfully reconciled.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretConversionStrategy">ExternalSecretConversionStrategy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef</a>, 
<a href="#external-secrets.io/v1beta1.ExternalSecretFind">ExternalSecretFind</a>)
</p>
<p>
<p>ExternalSecretConversionStrategy defines how secret values are converted.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Default&#34;</p></td>
<td><p>ExternalSecretConversionDefault indicates the default conversion strategy.</p>
</td>
</tr><tr><td><p>&#34;Unicode&#34;</p></td>
<td><p>ExternalSecretConversionUnicode indicates that unicode conversion will be performed.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretCreationPolicy">ExternalSecretCreationPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretTarget">ExternalSecretTarget</a>)
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
<td><p>CreatePolicyMerge does not create the Secret, but merges the data fields to the Secret.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>CreatePolicyNone does not create a Secret (future use with injector).</p>
</td>
</tr><tr><td><p>&#34;Orphan&#34;</p></td>
<td><p>CreatePolicyOrphan creates the Secret and does not set the ownerReference.
I.e. it will be orphaned after the deletion of the ExternalSecret.</p>
</td>
</tr><tr><td><p>&#34;Owner&#34;</p></td>
<td><p>CreatePolicyOwner creates the Secret and sets .metadata.ownerReferences to the ExternalSecret resource.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretData">ExternalSecretData
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">ExternalSecretSpec</a>)
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
<a href="#external-secrets.io/v1beta1.ExternalSecretDataRemoteRef">
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
<a href="#external-secrets.io/v1beta1.StoreSourceRef">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
<p>ExternalSecretDataFromRemoteRef defines a reference to multiple secrets in the provider to be fetched using options.</p>
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
<a href="#external-secrets.io/v1beta1.ExternalSecretDataRemoteRef">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretFind">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretRewrite">
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
<a href="#external-secrets.io/v1beta1.StoreGeneratorSourceRef">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretData">ExternalSecretData</a>, 
<a href="#external-secrets.io/v1beta1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
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
<a href="#external-secrets.io/v1beta1.ExternalSecretMetadataPolicy">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretConversionStrategy">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretDecodingStrategy">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretDecodingStrategy">ExternalSecretDecodingStrategy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef</a>, 
<a href="#external-secrets.io/v1beta1.ExternalSecretFind">ExternalSecretFind</a>)
</p>
<p>
<p>ExternalSecretDecodingStrategy defines how secret values are decoded.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Auto&#34;</p></td>
<td><p>ExternalSecretDecodeAuto indicates that the decoding strategy will be automatically determined.</p>
</td>
</tr><tr><td><p>&#34;Base64&#34;</p></td>
<td><p>ExternalSecretDecodeBase64 indicates that base64 decoding will be used.</p>
</td>
</tr><tr><td><p>&#34;Base64URL&#34;</p></td>
<td><p>ExternalSecretDecodeBase64URL indicates that base64url decoding will be used.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>ExternalSecretDecodeNone indicates that no decoding will be performed.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretDeletionPolicy">ExternalSecretDeletionPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretTarget">ExternalSecretTarget</a>)
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
<td><p>DeletionPolicyDelete deletes the secret if all provider secrets are deleted.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.</p>
</td>
</tr><tr><td><p>&#34;Merge&#34;</p></td>
<td><p>DeletionPolicyMerge removes keys in the secret, but not the secret itself.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.</p>
</td>
</tr><tr><td><p>&#34;Retain&#34;</p></td>
<td><p>DeletionPolicyRetain will retain the secret if all provider secrets have been deleted.
If a provider secret does not exist the ExternalSecret gets into the
SecretSyncedError status.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretFind">ExternalSecretFind
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
</p>
<p>
<p>ExternalSecretFind defines criteria for finding secrets in the provider.</p>
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
<a href="#external-secrets.io/v1beta1.FindName">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretConversionStrategy">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretDecodingStrategy">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretMetadata">ExternalSecretMetadata
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretSpec">ClusterExternalSecretSpec</a>)
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretMetadataPolicy">ExternalSecretMetadataPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretDataRemoteRef">ExternalSecretDataRemoteRef</a>)
</p>
<p>
<p>ExternalSecretMetadataPolicy defines the policy for fetching tags/labels from provider secrets.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Fetch&#34;</p></td>
<td><p>ExternalSecretMetadataPolicyFetch indicates that metadata will be fetched from the provider.</p>
</td>
</tr><tr><td><p>&#34;None&#34;</p></td>
<td><p>ExternalSecretMetadataPolicyNone indicates that no metadata will be fetched.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretRefreshPolicy">ExternalSecretRefreshPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">ExternalSecretSpec</a>)
</p>
<p>
<p>ExternalSecretRefreshPolicy defines how and when the ExternalSecret should be refreshed.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;CreatedOnce&#34;</p></td>
<td><p>RefreshPolicyCreatedOnce creates the Secret only if it does not exist and does not update it thereafter.</p>
</td>
</tr><tr><td><p>&#34;OnChange&#34;</p></td>
<td><p>RefreshPolicyOnChange only synchronizes the Secret when the ExternalSecret&rsquo;s metadata or specification changes.</p>
</td>
</tr><tr><td><p>&#34;Periodic&#34;</p></td>
<td><p>RefreshPolicyPeriodic synchronizes the Secret from the external source at regular intervals.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretRewrite">ExternalSecretRewrite
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
</p>
<p>
<p>ExternalSecretRewrite defines rules on how to rewrite secret keys.</p>
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
<a href="#external-secrets.io/v1beta1.ExternalSecretRewriteRegexp">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretRewriteTransform">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretRewriteRegexp">ExternalSecretRewriteRegexp
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretRewrite">ExternalSecretRewrite</a>)
</p>
<p>
<p>ExternalSecretRewriteRegexp defines how to use regular expressions for rewriting secret keys.</p>
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretRewriteTransform">ExternalSecretRewriteTransform
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretRewrite">ExternalSecretRewrite</a>)
</p>
<p>
<p>ExternalSecretRewriteTransform defines how to use string templates for transforming secret keys.</p>
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretSpec">ExternalSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterExternalSecretSpec">ClusterExternalSecretSpec</a>, 
<a href="#external-secrets.io/v1beta1.ExternalSecret">ExternalSecret</a>)
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
<a href="#external-secrets.io/v1beta1.SecretStoreRef">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretTarget">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretRefreshPolicy">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretData">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretDataFromRemoteRef">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretStatus">ExternalSecretStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecret">ExternalSecret</a>)
</p>
<p>
<p>ExternalSecretStatus defines the observed state of ExternalSecret.</p>
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
<a href="#external-secrets.io/v1beta1.ExternalSecretStatusCondition">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretStatusCondition">ExternalSecretStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretStatus">ExternalSecretStatus</a>)
</p>
<p>
<p>ExternalSecretStatusCondition contains condition information for an ExternalSecret.</p>
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
<a href="#external-secrets.io/v1beta1.ExternalSecretConditionType">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretTarget">ExternalSecretTarget
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">ExternalSecretSpec</a>)
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
<a href="#external-secrets.io/v1beta1.ExternalSecretCreationPolicy">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretDeletionPolicy">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretTemplate">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretTemplate">ExternalSecretTemplate
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretTarget">ExternalSecretTarget</a>)
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
<a href="#external-secrets.io/v1beta1.TemplateEngineVersion">
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
<a href="#external-secrets.io/v1beta1.ExternalSecretTemplateMetadata">
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
<a href="#external-secrets.io/v1beta1.TemplateMergePolicy">
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
<a href="#external-secrets.io/v1beta1.TemplateFrom">
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretTemplateMetadata">ExternalSecretTemplateMetadata
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
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
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretValidator">ExternalSecretValidator
</h3>
<p>
<p>ExternalSecretValidator implements webhook validation for ExternalSecret resources.</p>
</p>
<h3 id="external-secrets.io/v1beta1.ExternalSecretsAuth">ExternalSecretsAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsProvider">ExternalSecretsProvider</a>)
</p>
<p>
<p>ExternalSecretsAuth defines authentication methods for External Secrets Enterprise.</p>
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
<code>kubernetes</code></br>
<em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsKubernetesAuth">
ExternalSecretsKubernetesAuth
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretsCARef">ExternalSecretsCARef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsKubernetesAuth">ExternalSecretsKubernetesAuth</a>, 
<a href="#external-secrets.io/v1beta1.ExternalSecretsServer">ExternalSecretsServer</a>)
</p>
<p>
<p>ExternalSecretsCARef defines a reference to a CA certificate.</p>
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
<code>bundle</code></br>
<em>
[]byte
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>configMapRef</code></br>
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
<h3 id="external-secrets.io/v1beta1.ExternalSecretsKubernetesAuth">ExternalSecretsKubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsAuth">ExternalSecretsAuth</a>)
</p>
<p>
<p>ExternalSecretsKubernetesAuth defines Kubernetes-based authentication for External Secrets Enterprise.</p>
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
<code>caCertRef</code></br>
<em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsCARef">
ExternalSecretsCARef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretsProvider">ExternalSecretsProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>ExternalSecretsProvider configures the External Secrets Enterprise provider.</p>
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
<a href="#external-secrets.io/v1beta1.ExternalSecretsServer">
ExternalSecretsServer
</a>
</em>
</td>
<td>
<p>URL For the External Secrets Enterprise Server.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsAuth">
ExternalSecretsAuth
</a>
</em>
</td>
<td>
<p>Authentication parameters for External Secrets Enterprise</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsTarget">
ExternalSecretsTarget
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretsServer">ExternalSecretsServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsProvider">ExternalSecretsProvider</a>)
</p>
<p>
<p>ExternalSecretsServer defines the server configuration for External Secrets Enterprise.</p>
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
<code>caRef</code></br>
<em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsCARef">
ExternalSecretsCARef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
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
<p>URL For the External Secrets Enterprise Server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.ExternalSecretsTarget">ExternalSecretsTarget
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsProvider">ExternalSecretsProvider</a>)
</p>
<p>
<p>ExternalSecretsTarget specifies the target for External Secrets Enterprise operations.</p>
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
<code>clusterSecretStoreName</code></br>
<em>
string
</em>
</td>
<td>
<p>Remote clusterSecretStore to connect. Eventually, support more fields</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.FakeProvider">FakeProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.FakeProviderData">
[]FakeProviderData
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.FakeProviderData">FakeProviderData
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.FakeProvider">FakeProvider</a>)
</p>
<p>
<p>FakeProviderData defines a key-value pair for the fake provider used in testing.</p>
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
<h3 id="external-secrets.io/v1beta1.FindName">FindName
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretFind">ExternalSecretFind</a>)
</p>
<p>
<p>FindName defines name matching criteria for finding secrets.</p>
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
<h3 id="external-secrets.io/v1beta1.FortanixProvider">FortanixProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>FortanixProvider configures a store to sync secrets using the Fortanix SDKMS provider.</p>
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
<a href="#external-secrets.io/v1beta1.FortanixProviderSecretRef">
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
<h3 id="external-secrets.io/v1beta1.FortanixProviderSecretRef">FortanixProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.FortanixProvider">FortanixProvider</a>)
</p>
<p>
<p>FortanixProviderSecretRef defines a reference to a secret containing credentials for the Fortanix provider.</p>
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
<h3 id="external-secrets.io/v1beta1.GCPSMAuth">GCPSMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.GCPSMProvider">GCPSMProvider</a>)
</p>
<p>
<p>GCPSMAuth defines the authentication methods for the GCP Secret Manager provider.</p>
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
<a href="#external-secrets.io/v1beta1.GCPSMAuthSecretRef">
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
<a href="#external-secrets.io/v1beta1.GCPWorkloadIdentity">
GCPWorkloadIdentity
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.GCPSMAuthSecretRef">GCPSMAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.GCPSMAuth">GCPSMAuth</a>)
</p>
<p>
<p>GCPSMAuthSecretRef defines a reference to a secret containing credentials for the GCP Secret Manager provider.</p>
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
<h3 id="external-secrets.io/v1beta1.GCPSMProvider">GCPSMProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.GCPSMAuth">
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
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.GCPWorkloadIdentity">GCPWorkloadIdentity
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.GCPSMAuth">GCPSMAuth</a>)
</p>
<p>
<p>GCPWorkloadIdentity defines configuration for using GCP Workload Identity authentication.</p>
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
<h3 id="external-secrets.io/v1beta1.GeneratorRef">GeneratorRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.StoreGeneratorSourceRef">StoreGeneratorSourceRef</a>, 
<a href="#external-secrets.io/v1beta1.StoreSourceRef">StoreSourceRef</a>)
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
<h3 id="external-secrets.io/v1beta1.GenericStore">GenericStore
</h3>
<p>
<p>GenericStore is a common interface for interacting with ClusterSecretStore
or a namespaced SecretStore.</p>
</p>
<h3 id="external-secrets.io/v1beta1.GenericStoreValidator">GenericStoreValidator
</h3>
<p>
<p>GenericStoreValidator provides validation for SecretStore and ClusterSecretStore resources.</p>
</p>
<h3 id="external-secrets.io/v1beta1.GithubAppAuth">GithubAppAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.GithubProvider">GithubProvider</a>)
</p>
<p>
<p>GithubAppAuth defines the GitHub App authentication mechanism for the GitHub provider.</p>
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
<h3 id="external-secrets.io/v1beta1.GithubProvider">GithubProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>GithubProvider configures a store to push secrets to Github Actions.</p>
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
<a href="#external-secrets.io/v1beta1.GithubAppAuth">
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
<h3 id="external-secrets.io/v1beta1.GitlabAuth">GitlabAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.GitlabProvider">GitlabProvider</a>)
</p>
<p>
<p>GitlabAuth defines the authentication method for the GitLab provider.</p>
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
<a href="#external-secrets.io/v1beta1.GitlabSecretRef">
GitlabSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.GitlabProvider">GitlabProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>GitlabProvider configures a store to sync secrets with a GitLab instance.</p>
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
<a href="#external-secrets.io/v1beta1.GitlabAuth">
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
<a href="#external-secrets.io/v1beta1.CAProvider">
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
<h3 id="external-secrets.io/v1beta1.GitlabSecretRef">GitlabSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.GitlabAuth">GitlabAuth</a>)
</p>
<p>
<p>GitlabSecretRef defines a reference to a secret containing credentials for the GitLab provider.</p>
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
<h3 id="external-secrets.io/v1beta1.IBMAuth">IBMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.IBMProvider">IBMProvider</a>)
</p>
<p>
<p>IBMAuth defines the authentication methods for the IBM Cloud Secrets Manager provider.</p>
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
<a href="#external-secrets.io/v1beta1.IBMAuthSecretRef">
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
<a href="#external-secrets.io/v1beta1.IBMAuthContainerAuth">
IBMAuthContainerAuth
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.IBMAuthContainerAuth">IBMAuthContainerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.IBMAuth">IBMAuth</a>)
</p>
<p>
<p>IBMAuthContainerAuth defines authentication using IBM Container-based auth with IAM Trusted Profile.</p>
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
<h3 id="external-secrets.io/v1beta1.IBMAuthSecretRef">IBMAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.IBMAuth">IBMAuth</a>)
</p>
<p>
<p>IBMAuthSecretRef defines a reference to a secret containing credentials for the IBM provider.</p>
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
<h3 id="external-secrets.io/v1beta1.IBMProvider">IBMProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>IBMProvider configures a store to sync secrets using a IBM Cloud Secrets Manager backend.</p>
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
<a href="#external-secrets.io/v1beta1.IBMAuth">
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
<h3 id="external-secrets.io/v1beta1.InfisicalAuth">InfisicalAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.InfisicalProvider">InfisicalProvider</a>)
</p>
<p>
<p>InfisicalAuth defines the authentication methods for the Infisical provider.</p>
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
<a href="#external-secrets.io/v1beta1.UniversalAuthCredentials">
UniversalAuthCredentials
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.InfisicalProvider">InfisicalProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.InfisicalAuth">
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
<a href="#external-secrets.io/v1beta1.MachineIdentityScopeInWorkspace">
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
<h3 id="external-secrets.io/v1beta1.KeeperSecurityProvider">KeeperSecurityProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<h3 id="external-secrets.io/v1beta1.KubernetesAuth">KubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.KubernetesProvider">KubernetesProvider</a>)
</p>
<p>
<p>KubernetesAuth defines authentication methods for the Kubernetes provider.</p>
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
<a href="#external-secrets.io/v1beta1.CertAuth">
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
<a href="#external-secrets.io/v1beta1.TokenAuth">
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
<h3 id="external-secrets.io/v1beta1.KubernetesProvider">KubernetesProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>KubernetesProvider configures a store to sync secrets with a Kubernetes instance.</p>
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
<a href="#external-secrets.io/v1beta1.KubernetesServer">
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
<a href="#external-secrets.io/v1beta1.KubernetesAuth">
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
<h3 id="external-secrets.io/v1beta1.KubernetesServer">KubernetesServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.KubernetesProvider">KubernetesProvider</a>)
</p>
<p>
<p>KubernetesServer defines the Kubernetes server connection configuration.</p>
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
<a href="#external-secrets.io/v1beta1.CAProvider">
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
<h3 id="external-secrets.io/v1beta1.MachineIdentityScopeInWorkspace">MachineIdentityScopeInWorkspace
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.InfisicalProvider">InfisicalProvider</a>)
</p>
<p>
<p>MachineIdentityScopeInWorkspace defines the scope of a machine identity in an Infisical workspace.</p>
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
<h3 id="external-secrets.io/v1beta1.NTLMProtocol">NTLMProtocol
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AuthorizationProtocol">AuthorizationProtocol</a>)
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
<h3 id="external-secrets.io/v1beta1.NoSecretError">NoSecretError
</h3>
<p>
<p>NoSecretError shall be returned when a GetSecret can not find the
desired secret. This is used for deletionPolicy.</p>
</p>
<h3 id="external-secrets.io/v1beta1.NotModifiedError">NotModifiedError
</h3>
<p>
<p>NotModifiedError to signal that the webhook received no changes,
and it should just return without doing anything.</p>
</p>
<h3 id="external-secrets.io/v1beta1.OnboardbaseAuthSecretRef">OnboardbaseAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.OnboardbaseProvider">OnboardbaseProvider</a>)
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
<h3 id="external-secrets.io/v1beta1.OnboardbaseProvider">OnboardbaseProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.OnboardbaseAuthSecretRef">
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
<h3 id="external-secrets.io/v1beta1.OnePasswordAuth">OnePasswordAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.OnePasswordProvider">OnePasswordProvider</a>)
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
<a href="#external-secrets.io/v1beta1.OnePasswordAuthSecretRef">
OnePasswordAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.OnePasswordAuthSecretRef">OnePasswordAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.OnePasswordAuth">OnePasswordAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.OnePasswordProvider">OnePasswordProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.OnePasswordAuth">
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
<h3 id="external-secrets.io/v1beta1.OracleAuth">OracleAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.OracleProvider">OracleProvider</a>)
</p>
<p>
<p>OracleAuth defines authentication configuration for the Oracle Vault provider.</p>
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
<a href="#external-secrets.io/v1beta1.OracleSecretRef">
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
<h3 id="external-secrets.io/v1beta1.OraclePrincipalType">OraclePrincipalType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.OracleProvider">OracleProvider</a>)
</p>
<p>
<p>OraclePrincipalType defines the type of principal used for authentication to Oracle Vault.</p>
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
<h3 id="external-secrets.io/v1beta1.OracleProvider">OracleProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>OracleProvider configures a store to sync secrets using an Oracle Vault backend.</p>
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
<a href="#external-secrets.io/v1beta1.OraclePrincipalType">
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
<a href="#external-secrets.io/v1beta1.OracleAuth">
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
<h3 id="external-secrets.io/v1beta1.OracleSecretRef">OracleSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.OracleAuth">OracleAuth</a>)
</p>
<p>
<p>OracleSecretRef defines references to secrets containing Oracle credentials.</p>
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
<h3 id="external-secrets.io/v1beta1.PassboltAuth">PassboltAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.PassboltProvider">PassboltProvider</a>)
</p>
<p>
<p>PassboltAuth contains credentials and configuration for authenticating with the Passbolt server.</p>
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
<p>PasswordSecretRef is a reference to the secret containing the Passbolt password</p>
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
<p>PrivateKeySecretRef is a reference to the secret containing the Passbolt private key</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.PassboltProvider">PassboltProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>PassboltProvider defines configuration for the Passbolt provider.</p>
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
<a href="#external-secrets.io/v1beta1.PassboltAuth">
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
<h3 id="external-secrets.io/v1beta1.PasswordDepotAuth">PasswordDepotAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.PasswordDepotProvider">PasswordDepotProvider</a>)
</p>
<p>
<p>PasswordDepotAuth defines the authentication method for the Password Depot provider.</p>
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
<a href="#external-secrets.io/v1beta1.PasswordDepotSecretRef">
PasswordDepotSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.PasswordDepotProvider">PasswordDepotProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>PasswordDepotProvider configures a store to sync secrets with a Password Depot instance.</p>
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
<a href="#external-secrets.io/v1beta1.PasswordDepotAuth">
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
<h3 id="external-secrets.io/v1beta1.PasswordDepotSecretRef">PasswordDepotSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.PasswordDepotAuth">PasswordDepotAuth</a>)
</p>
<p>
<p>PasswordDepotSecretRef defines a reference to a secret containing credentials for the Password Depot provider.</p>
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
<h3 id="external-secrets.io/v1beta1.PreviderAuth">PreviderAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.PreviderProvider">PreviderProvider</a>)
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
<a href="#external-secrets.io/v1beta1.PreviderAuthSecretRef">
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
<h3 id="external-secrets.io/v1beta1.PreviderAuthSecretRef">PreviderAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.PreviderAuth">PreviderAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.PreviderProvider">PreviderProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.PreviderAuth">
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
<h3 id="external-secrets.io/v1beta1.Provider">Provider
</h3>
<p>
<p>Provider is a common interface for interacting with secret backends.</p>
</p>
<h3 id="external-secrets.io/v1beta1.PulumiProvider">PulumiProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>PulumiProvider defines configuration for the Pulumi provider.</p>
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
<a href="#external-secrets.io/v1beta1.PulumiProviderSecretRef">
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
<h3 id="external-secrets.io/v1beta1.PulumiProviderSecretRef">PulumiProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.PulumiProvider">PulumiProvider</a>)
</p>
<p>
<p>PulumiProviderSecretRef defines a reference to a secret containing credentials for the Pulumi provider.</p>
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
<h3 id="external-secrets.io/v1beta1.PushSecretData">PushSecretData
</h3>
<p>
<p>PushSecretData is an interface to allow using v1alpha1.PushSecretData content in Provider registered in v1beta1.</p>
</p>
<h3 id="external-secrets.io/v1beta1.PushSecretRemoteRef">PushSecretRemoteRef
</h3>
<p>
<p>PushSecretRemoteRef is an interface to allow using v1alpha1.PushSecretRemoteRef in Provider registered in v1beta1.</p>
</p>
<h3 id="external-secrets.io/v1beta1.ScalewayProvider">ScalewayProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>ScalewayProvider defines configuration for the Scaleway provider.</p>
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
<a href="#external-secrets.io/v1beta1.ScalewayProviderSecretRef">
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
<a href="#external-secrets.io/v1beta1.ScalewayProviderSecretRef">
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
<h3 id="external-secrets.io/v1beta1.ScalewayProviderSecretRef">ScalewayProviderSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ScalewayProvider">ScalewayProvider</a>)
</p>
<p>
<p>ScalewayProviderSecretRef defines a reference to a secret containing credentials for the Scaleway provider.</p>
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
<h3 id="external-secrets.io/v1beta1.SecretServerProvider">SecretServerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>SecretServerProvider defines configuration for the Delinea Secret Server provider.
See <a href="https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go">https://github.com/DelineaXPM/tss-sdk-go/blob/main/server/server.go</a>.</p>
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
<a href="#external-secrets.io/v1beta1.SecretServerProviderRef">
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
<a href="#external-secrets.io/v1beta1.SecretServerProviderRef">
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
<h3 id="external-secrets.io/v1beta1.SecretServerProviderRef">SecretServerProviderRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretServerProvider">SecretServerProvider</a>)
</p>
<p>
<p>SecretServerProviderRef defines a reference to a secret containing credentials for the Secret Server provider.</p>
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
<h3 id="external-secrets.io/v1beta1.SecretStore">SecretStore
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
<a href="#external-secrets.io/v1beta1.SecretStoreSpec">
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
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">
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
<a href="#external-secrets.io/v1beta1.SecretStoreRetrySettings">
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
<a href="#external-secrets.io/v1beta1.ClusterSecretStoreCondition">
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
<a href="#external-secrets.io/v1beta1.SecretStoreStatus">
SecretStoreStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.SecretStoreCapabilities">SecretStoreCapabilities
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreStatus">SecretStoreStatus</a>)
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
<td><p>SecretStoreReadOnly indicates that the SecretStore only supports reading secrets.</p>
</td>
</tr><tr><td><p>&#34;ReadWrite&#34;</p></td>
<td><p>SecretStoreReadWrite indicates that the SecretStore supports both reading and writing secrets.</p>
</td>
</tr><tr><td><p>&#34;WriteOnly&#34;</p></td>
<td><p>SecretStoreWriteOnly indicates that the SecretStore only supports writing secrets.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.SecretStoreConditionType">SecretStoreConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreStatusCondition">SecretStoreStatusCondition</a>)
</p>
<p>
<p>SecretStoreConditionType represents the condition type of the SecretStore.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>SecretStoreReady indicates that the SecretStore has been successfully configured.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreSpec">SecretStoreSpec</a>)
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
<a href="#external-secrets.io/v1beta1.AWSProvider">
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
<a href="#external-secrets.io/v1beta1.AzureKVProvider">
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
<a href="#external-secrets.io/v1beta1.AkeylessProvider">
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
<a href="#external-secrets.io/v1beta1.BitwardenSecretsManagerProvider">
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
<code>externalsecrets</code></br>
<em>
<a href="#external-secrets.io/v1beta1.ExternalSecretsProvider">
ExternalSecretsProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExternalSecrets configures the store to sync secrets using ExternalSecrets Enterprise Server</p>
</td>
</tr>
<tr>
<td>
<code>vault</code></br>
<em>
<a href="#external-secrets.io/v1beta1.VaultProvider">
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
<a href="#external-secrets.io/v1beta1.GCPSMProvider">
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
<a href="#external-secrets.io/v1beta1.OracleProvider">
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
<a href="#external-secrets.io/v1beta1.IBMProvider">
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
<a href="#external-secrets.io/v1beta1.YandexCertificateManagerProvider">
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
<a href="#external-secrets.io/v1beta1.YandexLockboxProvider">
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
<a href="#external-secrets.io/v1beta1.GithubProvider">
GithubProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Github configures this store to push Github Action secrets using Github API provider</p>
</td>
</tr>
<tr>
<td>
<code>gitlab</code></br>
<em>
<a href="#external-secrets.io/v1beta1.GitlabProvider">
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
<a href="#external-secrets.io/v1beta1.AlibabaProvider">
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
<a href="#external-secrets.io/v1beta1.OnePasswordProvider">
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
<code>webhook</code></br>
<em>
<a href="#external-secrets.io/v1beta1.WebhookProvider">
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
<a href="#external-secrets.io/v1beta1.KubernetesProvider">
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
<a href="#external-secrets.io/v1beta1.FakeProvider">
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
<a href="#external-secrets.io/v1beta1.SenhaseguraProvider">
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
<a href="#external-secrets.io/v1beta1.ScalewayProvider">
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
<a href="#external-secrets.io/v1beta1.DopplerProvider">
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
<a href="#external-secrets.io/v1beta1.PreviderProvider">
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
<a href="#external-secrets.io/v1beta1.OnboardbaseProvider">
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
<a href="#external-secrets.io/v1beta1.KeeperSecurityProvider">
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
<a href="#external-secrets.io/v1beta1.ConjurProvider">
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
<a href="#external-secrets.io/v1beta1.DelineaProvider">
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
<a href="#external-secrets.io/v1beta1.SecretServerProvider">
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
<a href="#external-secrets.io/v1beta1.ChefProvider">
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
<a href="#external-secrets.io/v1beta1.PulumiProvider">
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
<a href="#external-secrets.io/v1beta1.FortanixProvider">
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
<a href="#external-secrets.io/v1beta1.PasswordDepotProvider">
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
<a href="#external-secrets.io/v1beta1.PassboltProvider">
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
<a href="#external-secrets.io/v1beta1.Device42Provider">
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
<a href="#external-secrets.io/v1beta1.InfisicalProvider">
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
<a href="#external-secrets.io/v1beta1.BeyondtrustProvider">
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
<a href="#external-secrets.io/v1beta1.CloudruSMProvider">
CloudruSMProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CloudruSM configures this store to sync secrets using the Cloud.ru Secret Manager provider</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.SecretStoreRef">SecretStoreRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretSpec">ExternalSecretSpec</a>, 
<a href="#external-secrets.io/v1beta1.StoreGeneratorSourceRef">StoreGeneratorSourceRef</a>, 
<a href="#external-secrets.io/v1beta1.StoreSourceRef">StoreSourceRef</a>)
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
<h3 id="external-secrets.io/v1beta1.SecretStoreRetrySettings">SecretStoreRetrySettings
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreSpec">SecretStoreSpec</a>)
</p>
<p>
<p>SecretStoreRetrySettings defines configuration for retrying failed requests to the provider.</p>
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
<p>MaxRetries is the maximum number of retry attempts.</p>
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
<p>RetryInterval is the interval between retry attempts.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.SecretStoreSpec">SecretStoreSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterSecretStore">ClusterSecretStore</a>, 
<a href="#external-secrets.io/v1beta1.SecretStore">SecretStore</a>)
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
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">
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
<a href="#external-secrets.io/v1beta1.SecretStoreRetrySettings">
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
<a href="#external-secrets.io/v1beta1.ClusterSecretStoreCondition">
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
<h3 id="external-secrets.io/v1beta1.SecretStoreStatus">SecretStoreStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ClusterSecretStore">ClusterSecretStore</a>, 
<a href="#external-secrets.io/v1beta1.SecretStore">SecretStore</a>)
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
<a href="#external-secrets.io/v1beta1.SecretStoreStatusCondition">
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
<a href="#external-secrets.io/v1beta1.SecretStoreCapabilities">
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
<h3 id="external-secrets.io/v1beta1.SecretStoreStatusCondition">SecretStoreStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreStatus">SecretStoreStatus</a>)
</p>
<p>
<p>SecretStoreStatusCondition defines the observed condition of the SecretStore.</p>
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
<a href="#external-secrets.io/v1beta1.SecretStoreConditionType">
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
<h3 id="external-secrets.io/v1beta1.SecretsClient">SecretsClient
</h3>
<p>
<p>SecretsClient provides access to secrets.</p>
</p>
<h3 id="external-secrets.io/v1beta1.SecretsManager">SecretsManager
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.AWSProvider">AWSProvider</a>)
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
<h3 id="external-secrets.io/v1beta1.SenhaseguraAuth">SenhaseguraAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SenhaseguraProvider">SenhaseguraProvider</a>)
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
<h3 id="external-secrets.io/v1beta1.SenhaseguraModuleType">SenhaseguraModuleType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SenhaseguraProvider">SenhaseguraProvider</a>)
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
<h3 id="external-secrets.io/v1beta1.SenhaseguraProvider">SenhaseguraProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
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
<a href="#external-secrets.io/v1beta1.SenhaseguraModuleType">
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
<a href="#external-secrets.io/v1beta1.SenhaseguraAuth">
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
<h3 id="external-secrets.io/v1beta1.StoreGeneratorSourceRef">StoreGeneratorSourceRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretDataFromRemoteRef">ExternalSecretDataFromRemoteRef</a>)
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
<a href="#external-secrets.io/v1beta1.SecretStoreRef">
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
<a href="#external-secrets.io/v1beta1.GeneratorRef">
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
<h3 id="external-secrets.io/v1beta1.StoreSourceRef">StoreSourceRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretData">ExternalSecretData</a>)
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
<a href="#external-secrets.io/v1beta1.SecretStoreRef">
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
<a href="#external-secrets.io/v1beta1.GeneratorRef">
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
<h3 id="external-secrets.io/v1beta1.Tag">Tag
</h3>
<p>
<p>Tag defines a tag key and value for AWS resources.</p>
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
<h3 id="external-secrets.io/v1beta1.TemplateEngineVersion">TemplateEngineVersion
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
<p>TemplateEngineVersion defines the version of the template engine to use.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;v2&#34;</p></td>
<td><p>TemplateEngineV2 specifies the v2 template engine version.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.TemplateFrom">TemplateFrom
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
<p>TemplateFrom defines a source for template data.</p>
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
<a href="#external-secrets.io/v1beta1.TemplateRef">
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
<a href="#external-secrets.io/v1beta1.TemplateRef">
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
<a href="#external-secrets.io/v1beta1.TemplateTarget">
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
<h3 id="external-secrets.io/v1beta1.TemplateMergePolicy">TemplateMergePolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.ExternalSecretTemplate">ExternalSecretTemplate</a>)
</p>
<p>
<p>TemplateMergePolicy defines how template values should be merged when generating a secret.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Merge&#34;</p></td>
<td><p>MergePolicyMerge merges the template content with existing values.</p>
</td>
</tr><tr><td><p>&#34;Replace&#34;</p></td>
<td><p>MergePolicyReplace replaces the entire template content during merge operations.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.TemplateRef">TemplateRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.TemplateFrom">TemplateFrom</a>)
</p>
<p>
<p>TemplateRef defines a reference to a template source in a ConfigMap or Secret.</p>
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
<a href="#external-secrets.io/v1beta1.TemplateRefItem">
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
<h3 id="external-secrets.io/v1beta1.TemplateRefItem">TemplateRefItem
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.TemplateRef">TemplateRef</a>)
</p>
<p>
<p>TemplateRefItem defines which key in the referenced ConfigMap or Secret to use as a template.</p>
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
<a href="#external-secrets.io/v1beta1.TemplateScope">
TemplateScope
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.TemplateScope">TemplateScope
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.TemplateRefItem">TemplateRefItem</a>)
</p>
<p>
<p>TemplateScope defines the scope of the template when processing template data.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;KeysAndValues&#34;</p></td>
<td><p>TemplateScopeKeysAndValues processes both keys and values of the data.</p>
</td>
</tr><tr><td><p>&#34;Values&#34;</p></td>
<td><p>TemplateScopeValues processes only the values of the data.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.TemplateTarget">TemplateTarget
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.TemplateFrom">TemplateFrom</a>)
</p>
<p>
<p>TemplateTarget defines the target field where the template result will be stored.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Annotations&#34;</p></td>
<td><p>TemplateTargetAnnotations stores template results in the annotations field of the secret.</p>
</td>
</tr><tr><td><p>&#34;Data&#34;</p></td>
<td><p>TemplateTargetData stores template results in the data field of the secret.</p>
</td>
</tr><tr><td><p>&#34;Labels&#34;</p></td>
<td><p>TemplateTargetLabels stores template results in the labels field of the secret.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.TokenAuth">TokenAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.KubernetesAuth">KubernetesAuth</a>)
</p>
<p>
<p>TokenAuth defines token-based authentication for the Kubernetes provider.</p>
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
<h3 id="external-secrets.io/v1beta1.UniversalAuthCredentials">UniversalAuthCredentials
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.InfisicalAuth">InfisicalAuth</a>)
</p>
<p>
<p>UniversalAuthCredentials defines the credentials for Infisical Universal Auth.</p>
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
<h3 id="external-secrets.io/v1beta1.ValidationResult">ValidationResult
(<code>byte</code> alias)</p></h3>
<p>
<p>ValidationResult represents the result of validating a provider client configuration.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>2</p></td>
<td><p>ValidationResultError indicates that there is a misconfiguration.</p>
</td>
</tr><tr><td><p>0</p></td>
<td><p>ValidationResultReady indicates that the client is configured correctly and can be used.</p>
</td>
</tr><tr><td><p>1</p></td>
<td><p>ValidationResultUnknown indicates that the client can be used but information is missing and it can not be validated.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.VaultAppRole">VaultAppRole
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAuth">VaultAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.VaultAuth">VaultAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultProvider">VaultProvider</a>)
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
<a href="#external-secrets.io/v1beta1.VaultAppRole">
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
<a href="#external-secrets.io/v1beta1.VaultKubernetesAuth">
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
<a href="#external-secrets.io/v1beta1.VaultLdapAuth">
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
<a href="#external-secrets.io/v1beta1.VaultJwtAuth">
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
<a href="#external-secrets.io/v1beta1.VaultCertAuth">
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
<a href="#external-secrets.io/v1beta1.VaultIamAuth">
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
<a href="#external-secrets.io/v1beta1.VaultUserPassAuth">
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
<h3 id="external-secrets.io/v1beta1.VaultAwsAuth">VaultAwsAuth
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
<a href="#external-secrets.io/v1beta1.VaultAwsAuthSecretRef">
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
<a href="#external-secrets.io/v1beta1.VaultAwsJWTAuth">
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
<h3 id="external-secrets.io/v1beta1.VaultAwsAuthSecretRef">VaultAwsAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAwsAuth">VaultAwsAuth</a>, 
<a href="#external-secrets.io/v1beta1.VaultIamAuth">VaultIamAuth</a>)
</p>
<p>
<p>VaultAwsAuthSecretRef holds secret references for AWS credentials
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
<h3 id="external-secrets.io/v1beta1.VaultAwsJWTAuth">VaultAwsJWTAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAwsAuth">VaultAwsAuth</a>, 
<a href="#external-secrets.io/v1beta1.VaultIamAuth">VaultIamAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.VaultCertAuth">VaultCertAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAuth">VaultAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.VaultClientTLS">VaultClientTLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultProvider">VaultProvider</a>)
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
<h3 id="external-secrets.io/v1beta1.VaultIamAuth">VaultIamAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultIamAuth authenticates with Vault using the Vault&rsquo;s AWS IAM authentication method. Refer: <a href="https://developer.hashicorp.com/vault/docs/auth/aws">https://developer.hashicorp.com/vault/docs/auth/aws</a></p>
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
<a href="#external-secrets.io/v1beta1.VaultAwsAuthSecretRef">
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
<a href="#external-secrets.io/v1beta1.VaultAwsJWTAuth">
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
<h3 id="external-secrets.io/v1beta1.VaultJwtAuth">VaultJwtAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAuth">VaultAuth</a>)
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
<a href="#external-secrets.io/v1beta1.VaultKubernetesServiceAccountTokenAuth">
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
<h3 id="external-secrets.io/v1beta1.VaultKVStoreVersion">VaultKVStoreVersion
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultProvider">VaultProvider</a>)
</p>
<p>
<p>VaultKVStoreVersion defines the version of the KV store in Vault.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;v1&#34;</p></td>
<td><p>VaultKVStoreV1 represents version 1 of the Vault KV store.</p>
</td>
</tr><tr><td><p>&#34;v2&#34;</p></td>
<td><p>VaultKVStoreV2 represents version 2 of the Vault KV store.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.VaultKubernetesAuth">VaultKubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAuth">VaultAuth</a>)
</p>
<p>
<p>VaultKubernetesAuth authenticates against Vault using a Kubernetes ServiceAccount token stored in a Secret.</p>
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
<h3 id="external-secrets.io/v1beta1.VaultKubernetesServiceAccountTokenAuth">VaultKubernetesServiceAccountTokenAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultJwtAuth">VaultJwtAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.VaultLdapAuth">VaultLdapAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAuth">VaultAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.VaultProvider">VaultProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>VaultProvider configures a store to sync secrets using a HashiCorp Vault KV backend.</p>
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
<a href="#external-secrets.io/v1beta1.VaultAuth">
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
<a href="#external-secrets.io/v1beta1.VaultKVStoreVersion">
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
<a href="#external-secrets.io/v1beta1.VaultClientTLS">
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
<a href="#external-secrets.io/v1beta1.CAProvider">
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
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.VaultUserPassAuth">VaultUserPassAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.VaultAuth">VaultAuth</a>)
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
<h3 id="external-secrets.io/v1beta1.WebhookCAProvider">WebhookCAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.WebhookProvider">WebhookProvider</a>)
</p>
<p>
<p>WebhookCAProvider defines a location to fetch the certificate for the webhook provider.</p>
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
<a href="#external-secrets.io/v1beta1.WebhookCAProviderType">
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
<h3 id="external-secrets.io/v1beta1.WebhookCAProviderType">WebhookCAProviderType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.WebhookCAProvider">WebhookCAProvider</a>)
</p>
<p>
<p>WebhookCAProviderType defines the type of provider to use for CA certificates with Webhook providers.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ConfigMap&#34;</p></td>
<td><p>WebhookCAProviderTypeConfigMap indicates that the CA certificate is stored in a ConfigMap.</p>
</td>
</tr><tr><td><p>&#34;Secret&#34;</p></td>
<td><p>WebhookCAProviderTypeSecret indicates that the CA certificate is stored in a Secret.</p>
</td>
</tr></tbody>
</table>
<h3 id="external-secrets.io/v1beta1.WebhookProvider">WebhookProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>WebhookProvider configures a store to sync secrets from simple web APIs.</p>
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
<a href="#external-secrets.io/v1beta1.AuthorizationProtocol">
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
<a href="#external-secrets.io/v1beta1.WebhookResult">
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
<a href="#external-secrets.io/v1beta1.WebhookSecret">
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
<a href="#external-secrets.io/v1beta1.WebhookCAProvider">
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
<h3 id="external-secrets.io/v1beta1.WebhookResult">WebhookResult
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.WebhookProvider">WebhookProvider</a>)
</p>
<p>
<p>WebhookResult defines how to extract and format the result from the webhook response.</p>
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
<h3 id="external-secrets.io/v1beta1.WebhookSecret">WebhookSecret
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.WebhookProvider">WebhookProvider</a>)
</p>
<p>
<p>WebhookSecret defines a secret to be used in webhook templates.</p>
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
<h3 id="external-secrets.io/v1beta1.YandexCertificateManagerAuth">YandexCertificateManagerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.YandexCertificateManagerProvider">YandexCertificateManagerProvider</a>)
</p>
<p>
<p>YandexCertificateManagerAuth defines authentication configuration for the Yandex Certificate Manager provider.</p>
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
<h3 id="external-secrets.io/v1beta1.YandexCertificateManagerCAProvider">YandexCertificateManagerCAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.YandexCertificateManagerProvider">YandexCertificateManagerProvider</a>)
</p>
<p>
<p>YandexCertificateManagerCAProvider defines CA certificate configuration for Yandex Certificate Manager.</p>
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
<h3 id="external-secrets.io/v1beta1.YandexCertificateManagerProvider">YandexCertificateManagerProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>YandexCertificateManagerProvider configures a store to sync secrets using the Yandex Certificate Manager provider.</p>
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
<a href="#external-secrets.io/v1beta1.YandexCertificateManagerAuth">
YandexCertificateManagerAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against Yandex Certificate Manager</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1beta1.YandexCertificateManagerCAProvider">
YandexCertificateManagerCAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate Yandex.Cloud server certificate.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="external-secrets.io/v1beta1.YandexLockboxAuth">YandexLockboxAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.YandexLockboxProvider">YandexLockboxProvider</a>)
</p>
<p>
<p>YandexLockboxAuth defines authentication configuration for the Yandex Lockbox provider.</p>
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
<h3 id="external-secrets.io/v1beta1.YandexLockboxCAProvider">YandexLockboxCAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.YandexLockboxProvider">YandexLockboxProvider</a>)
</p>
<p>
<p>YandexLockboxCAProvider defines CA certificate configuration for Yandex Lockbox.</p>
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
<h3 id="external-secrets.io/v1beta1.YandexLockboxProvider">YandexLockboxProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#external-secrets.io/v1beta1.SecretStoreProvider">SecretStoreProvider</a>)
</p>
<p>
<p>YandexLockboxProvider configures a store to sync secrets using the Yandex Lockbox provider.</p>
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
<a href="#external-secrets.io/v1beta1.YandexLockboxAuth">
YandexLockboxAuth
</a>
</em>
</td>
<td>
<p>Auth defines the information necessary to authenticate against Yandex Lockbox</p>
</td>
</tr>
<tr>
<td>
<code>caProvider</code></br>
<em>
<a href="#external-secrets.io/v1beta1.YandexLockboxCAProvider">
YandexLockboxCAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate Yandex.Cloud server certificate.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<h2 id="generators.external-secrets.io/v1alpha1">generators.external-secrets.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains resources for generators</p>
</p>
Resource Types:
<ul></ul>
<h3 id="generators.external-secrets.io/v1alpha1.ACRAccessToken">ACRAccessToken
</h3>
<p>
<p>ACRAccessToken returns an Azure Container Registry token
that can be used for pushing/pulling images.
Note: by default it will return an ACR Refresh Token with full access
(depending on the identity).
This can be scoped down to the repository level using .spec.scope.
In case scope is defined it will return an ACR Access Token.</p>
<p>See docs: <a href="https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md">https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md</a></p>
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
<a href="#generators.external-secrets.io/v1alpha1.ACRAccessTokenSpec">
ACRAccessTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAuth">
ACRAuth
</a>
</em>
</td>
<td>
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
<p>TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type.</p>
</td>
</tr>
<tr>
<td>
<code>registry</code></br>
<em>
string
</em>
</td>
<td>
<p>the domain name of the ACR registry
e.g. foobarexample.azurecr.io</p>
</td>
</tr>
<tr>
<td>
<code>scope</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Define the scope for the access token, e.g. pull/push access for a repository.
if not provided it will return a refresh token that has full scope.
Note: you need to pin it down to the repository level, there is no wildcard available.</p>
<p>examples:
repository:my-repository:pull,push
repository:my-repository:pull</p>
<p>see docs for details: <a href="https://docs.docker.com/registry/spec/auth/scope/">https://docs.docker.com/registry/spec/auth/scope/</a></p>
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
connecting and authenticating with Azure. By default, it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: <a href="https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152">https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152</a>
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.ACRAccessTokenSpec">ACRAccessTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAccessToken">ACRAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>)
</p>
<p>
<p>ACRAccessTokenSpec defines how to generate the access token
e.g. how to authenticate and which registry to use.
see: <a href="https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#overview">https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#overview</a></p>
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
<a href="#generators.external-secrets.io/v1alpha1.ACRAuth">
ACRAuth
</a>
</em>
</td>
<td>
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
<p>TenantID configures the Azure Tenant to send requests to. Required for ServicePrincipal auth type.</p>
</td>
</tr>
<tr>
<td>
<code>registry</code></br>
<em>
string
</em>
</td>
<td>
<p>the domain name of the ACR registry
e.g. foobarexample.azurecr.io</p>
</td>
</tr>
<tr>
<td>
<code>scope</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Define the scope for the access token, e.g. pull/push access for a repository.
if not provided it will return a refresh token that has full scope.
Note: you need to pin it down to the repository level, there is no wildcard available.</p>
<p>examples:
repository:my-repository:pull,push
repository:my-repository:pull</p>
<p>see docs for details: <a href="https://docs.docker.com/registry/spec/auth/scope/">https://docs.docker.com/registry/spec/auth/scope/</a></p>
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
connecting and authenticating with Azure. By default, it points to the public cloud AAD endpoint.
The following endpoints are available, also see here: <a href="https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152">https://github.com/Azure/go-autorest/blob/main/autorest/azure/environments.go#L152</a>
PublicCloud, USGovernmentCloud, ChinaCloud, GermanCloud</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.ACRAuth">ACRAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAccessTokenSpec">ACRAccessTokenSpec</a>)
</p>
<p>
<p>ACRAuth defines the authentication methods for Azure Container Registry.</p>
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
<code>servicePrincipal</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AzureACRServicePrincipalAuth">
AzureACRServicePrincipalAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServicePrincipal uses Azure Service Principal credentials to authenticate with Azure.</p>
</td>
</tr>
<tr>
<td>
<code>managedIdentity</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AzureACRManagedIdentityAuth">
AzureACRManagedIdentityAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ManagedIdentity uses Azure Managed Identity to authenticate with Azure.</p>
</td>
</tr>
<tr>
<td>
<code>workloadIdentity</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AzureACRWorkloadIdentityAuth">
AzureACRWorkloadIdentityAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>WorkloadIdentity uses Azure Workload Identity to authenticate with Azure.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.AWSAuth">AWSAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ECRAuthorizationTokenSpec">ECRAuthorizationTokenSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.IAMKeysSpec">IAMKeysSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.STSSessionTokenSpec">STSSessionTokenSpec</a>)
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
<a href="#generators.external-secrets.io/v1alpha1.AWSAuthSecretRef">
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
<a href="#generators.external-secrets.io/v1alpha1.AWSJWTAuth">
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
<h3 id="generators.external-secrets.io/v1alpha1.AWSAuthSecretRef">AWSAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">AWSAuth</a>)
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
<h3 id="generators.external-secrets.io/v1alpha1.AWSIAMKey">AWSIAMKey
</h3>
<p>
<p>AWSIAMKey uses the CreateAccessKey API to retrieve an
access key. It also rotates the key by making sure only X keys exist on a given user.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.IAMKeysSpec">
IAMKeysSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region specifies the region to operate in.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">
AWSAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines how to authenticate with AWS</p>
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
<p>You can assume a role before making calls to the
desired AWS service.</p>
</td>
</tr>
<tr>
<td>
<code>iamRef</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.IAMRef">
IAMRef
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.AWSJWTAuth">AWSJWTAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">AWSAuth</a>)
</p>
<p>
<p>AWSJWTAuth provides configuration to authenticate against AWS using service account tokens.</p>
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
<h3 id="generators.external-secrets.io/v1alpha1.AuthorizationProtocol">AuthorizationProtocol
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.WebhookSpec">WebhookSpec</a>)
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
<a href="#generators.external-secrets.io/v1alpha1.NTLMProtocol">
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
<h3 id="generators.external-secrets.io/v1alpha1.AzureACRManagedIdentityAuth">AzureACRManagedIdentityAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAuth">ACRAuth</a>)
</p>
<p>
<p>AzureACRManagedIdentityAuth defines the configuration for using Azure Managed Identity authentication.</p>
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
string
</em>
</td>
<td>
<p>If multiple Managed Identity is assigned to the pod, you can select the one to be used</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.AzureACRServicePrincipalAuth">AzureACRServicePrincipalAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAuth">ACRAuth</a>)
</p>
<p>
<p>AzureACRServicePrincipalAuth defines the configuration for using Azure Service Principal authentication.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.AzureACRServicePrincipalAuthSecretRef">
AzureACRServicePrincipalAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.AzureACRServicePrincipalAuthSecretRef">AzureACRServicePrincipalAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.AzureACRServicePrincipalAuth">AzureACRServicePrincipalAuth</a>)
</p>
<p>
<p>AzureACRServicePrincipalAuthSecretRef defines the secret references for Azure Service Principal authentication.
It uses static credentials stored in a Kind=Secret.</p>
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
<p>The Azure clientId of the service principle used for authentication.</p>
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
<p>The Azure ClientSecret of the service principle used for authentication.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.AzureACRWorkloadIdentityAuth">AzureACRWorkloadIdentityAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAuth">ACRAuth</a>)
</p>
<p>
<p>AzureACRWorkloadIdentityAuth defines the configuration for using Azure Workload Identity authentication.</p>
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
<p>ServiceAccountRef specified the service account
that should be used when authenticating with WorkloadIdentity.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.BasicAuth">BasicAuth
</h3>
<p>
<p>BasicAuth generates a random basic auth based on the
configuration parameters in spec.
You can specify the length, characterset and other attributes.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.BasicAuthSpec">
BasicAuthSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>username</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.UsernameSpec">
UsernameSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PasswordSpec">
PasswordSpec
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.BasicAuthSpec">BasicAuthSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.BasicAuth">BasicAuth</a>)
</p>
<p>
<p>BasicAuthSpec controls the behavior of the basic auth generator.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.UsernameSpec">
UsernameSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PasswordSpec">
PasswordSpec
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.CleanupPolicy">CleanupPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.OpenAISpec">OpenAISpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLCleanupPolicy">PostgreSQLCleanupPolicy</a>)
</p>
<p>
<p>CleanupPolicy defines the cleanup policy for generated secrets.</p>
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
string
</em>
</td>
<td>
<p>Type of the cleanup policy. Supported values: &ldquo;idle&rdquo;, &ldquo;retainLatest&rdquo;.
idle: delete the secret if it has not been used for a while
retainLatest: delete older secrets when a new one is created</p>
</td>
</tr>
<tr>
<td>
<code>idleTimeout</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IdleTimeout Indicates how long without activity a secret is considered inactive and can be removed.
Used only when type is &ldquo;idle&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>gracePeriod</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>GracePeriod is the amount of time to wait before deleting a secret.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.CloudsmithAccessToken">CloudsmithAccessToken
</h3>
<p>
<p>CloudsmithAccessToken generates Cloudsmith access token using OIDC authentication</p>
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
<a href="#generators.external-secrets.io/v1alpha1.CloudsmithAccessTokenSpec">
CloudsmithAccessTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>apiUrl</code></br>
<em>
string
</em>
</td>
<td>
<p>APIURL configures the Cloudsmith API URL. Defaults to <a href="https://api.cloudsmith.io">https://api.cloudsmith.io</a>.</p>
</td>
</tr>
<tr>
<td>
<code>orgSlug</code></br>
<em>
string
</em>
</td>
<td>
<p>OrgSlug is the organization slug in Cloudsmith</p>
</td>
</tr>
<tr>
<td>
<code>serviceSlug</code></br>
<em>
string
</em>
</td>
<td>
<p>ServiceSlug is the service slug in Cloudsmith for OIDC authentication</p>
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
<p>Name of the service account you are federating with</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.CloudsmithAccessTokenSpec">CloudsmithAccessTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.CloudsmithAccessToken">CloudsmithAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>)
</p>
<p>
<p>CloudsmithAccessTokenSpec defines the configuration for generating a Cloudsmith access token using OIDC authentication.</p>
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
<p>APIURL configures the Cloudsmith API URL. Defaults to <a href="https://api.cloudsmith.io">https://api.cloudsmith.io</a>.</p>
</td>
</tr>
<tr>
<td>
<code>orgSlug</code></br>
<em>
string
</em>
</td>
<td>
<p>OrgSlug is the organization slug in Cloudsmith</p>
</td>
</tr>
<tr>
<td>
<code>serviceSlug</code></br>
<em>
string
</em>
</td>
<td>
<p>ServiceSlug is the service slug in Cloudsmith for OIDC authentication</p>
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
<p>Name of the service account you are federating with</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.ClusterGenerator">ClusterGenerator
</h3>
<p>
<p>ClusterGenerator represents a cluster-wide generator which can be referenced as part of <code>generatorRef</code> fields.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.ClusterGeneratorSpec">
ClusterGeneratorSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>kind</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorKind">
GeneratorKind
</a>
</em>
</td>
<td>
<p>Kind the kind of this generator.</p>
</td>
</tr>
<tr>
<td>
<code>generator</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">
GeneratorSpec
</a>
</em>
</td>
<td>
<p>Generator the spec for this generator, must match the kind.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.ClusterGeneratorSpec">ClusterGeneratorSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ClusterGenerator">ClusterGenerator</a>)
</p>
<p>
<p>ClusterGeneratorSpec defines the desired state of a ClusterGenerator.</p>
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
<code>kind</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorKind">
GeneratorKind
</a>
</em>
</td>
<td>
<p>Kind the kind of this generator.</p>
</td>
</tr>
<tr>
<td>
<code>generator</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">
GeneratorSpec
</a>
</em>
</td>
<td>
<p>Generator the spec for this generator, must match the kind.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.ControllerClassResource">ControllerClassResource
</h3>
<p>
<p>ControllerClassResource defines a resource that can be assigned to a specific controller class.</p>
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
<code>spec</code></br>
<em>
struct{ControllerClass string &#34;json:\&#34;controller\&#34;&#34;}
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
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.ECRAuthorizationToken">ECRAuthorizationToken
</h3>
<p>
<p>ECRAuthorizationToken uses the GetAuthorizationToken API to retrieve an authorization token.
The authorization token is valid for 12 hours.
The authorizationToken returned is a base64 encoded string that can be decoded
and used in a docker login command to authenticate to a registry.
For more information, see Registry authentication (<a href="https://docs.aws.amazon.com/AmazonECR/latest/userguide/Registries.html#registry_auth">https://docs.aws.amazon.com/AmazonECR/latest/userguide/Registries.html#registry_auth</a>) in the Amazon Elastic Container Registry User Guide.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.ECRAuthorizationTokenSpec">
ECRAuthorizationTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region specifies the region to operate in.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">
AWSAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines how to authenticate with AWS</p>
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
<p>You can assume a role before making calls to the
desired AWS service.</p>
</td>
</tr>
<tr>
<td>
<code>scope</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Scope specifies the ECR service scope.
Valid options are private and public.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.ECRAuthorizationTokenSpec">ECRAuthorizationTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ECRAuthorizationToken">ECRAuthorizationToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>)
</p>
<p>
<p>ECRAuthorizationTokenSpec defines the desired state to generate an AWS ECR authorization token.</p>
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
<p>Region specifies the region to operate in.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">
AWSAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines how to authenticate with AWS</p>
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
<p>You can assume a role before making calls to the
desired AWS service.</p>
</td>
</tr>
<tr>
<td>
<code>scope</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Scope specifies the ECR service scope.
Valid options are private and public.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Fake">Fake
</h3>
<p>
<p>Fake generator is used for testing. It lets you define
a static set of credentials that is always returned.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.FakeSpec">
FakeSpec
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
The ESO controller is instantiated with a specific controller name and filters VDS based on this property</p>
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
<p>Data defines the static data returned
by this generator.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.FakeSpec">FakeSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Fake">Fake</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>)
</p>
<p>
<p>FakeSpec contains the static data.</p>
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
The ESO controller is instantiated with a specific controller name and filters VDS based on this property</p>
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
<p>Data defines the static data returned
by this generator.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Federation">Federation
</h3>
<p>
<p>Federation represents a federation generator configuration.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.FederationSpec">
FederationSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.FederationServer">
FederationServer
</a>
</em>
</td>
<td>
<p>Server specifies the federation server configuration</p>
</td>
</tr>
<tr>
<td>
<code>generator</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.FederationGeneratorRef">
FederationGeneratorRef
</a>
</em>
</td>
<td>
<p>Generator specifies the target generator to use</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.FederationAuthKubernetes">
FederationAuthKubernetes
</a>
</em>
</td>
<td>
<p>Auth specifies the authentication configuration</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.FederationAuthKubernetes">FederationAuthKubernetes
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.FederationSpec">FederationSpec</a>)
</p>
<p>
<p>FederationAuthKubernetes defines the authentication configuration.</p>
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
<code>tokenSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TokenSecretRef references a secret containing the auth token</p>
</td>
</tr>
<tr>
<td>
<code>caCertSecretRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CACertSecretRef references a secret containing the CA certificate</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.FederationGeneratorRef">FederationGeneratorRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.FederationSpec">FederationSpec</a>)
</p>
<p>
<p>FederationGeneratorRef defines the target generator.</p>
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
<p>Namespace is the namespace of the generator</p>
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
<p>Kind is the kind of the generator</p>
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
<p>Name is the name of the generator</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.FederationServer">FederationServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.FederationSpec">FederationSpec</a>)
</p>
<p>
<p>FederationServer defines the federation server configuration.</p>
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
<p>URL is the URL of the federation server</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.FederationSpec">FederationSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Federation">Federation</a>)
</p>
<p>
<p>FederationSpec defines the configuration for the federation generator.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.FederationServer">
FederationServer
</a>
</em>
</td>
<td>
<p>Server specifies the federation server configuration</p>
</td>
</tr>
<tr>
<td>
<code>generator</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.FederationGeneratorRef">
FederationGeneratorRef
</a>
</em>
</td>
<td>
<p>Generator specifies the target generator to use</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.FederationAuthKubernetes">
FederationAuthKubernetes
</a>
</em>
</td>
<td>
<p>Auth specifies the authentication configuration</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GCPSMAuth">GCPSMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GCRAccessTokenSpec">GCRAccessTokenSpec</a>)
</p>
<p>
<p>GCPSMAuth defines the authentication methods for Google Cloud Platform.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GCPSMAuthSecretRef">
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
<a href="#generators.external-secrets.io/v1alpha1.GCPWorkloadIdentity">
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
<h3 id="generators.external-secrets.io/v1alpha1.GCPSMAuthSecretRef">GCPSMAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GCPSMAuth">GCPSMAuth</a>)
</p>
<p>
<p>GCPSMAuthSecretRef defines the reference to a secret containing Google Cloud Platform credentials.</p>
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
<h3 id="generators.external-secrets.io/v1alpha1.GCPWorkloadIdentity">GCPWorkloadIdentity
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GCPSMAuth">GCPSMAuth</a>)
</p>
<p>
<p>GCPWorkloadIdentity defines the configuration for using GCP Workload Identity authentication.</p>
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
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GCRAccessToken">GCRAccessToken
</h3>
<p>
<p>GCRAccessToken generates an GCP access token
that can be used to authenticate with GCR.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GCRAccessTokenSpec">
GCRAccessTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GCPSMAuth">
GCPSMAuth
</a>
</em>
</td>
<td>
<p>Auth defines the means for authenticating with GCP</p>
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
<p>ProjectID defines which project to use to authenticate with</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GCRAccessTokenSpec">GCRAccessTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GCRAccessToken">GCRAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>)
</p>
<p>
<p>GCRAccessTokenSpec defines the desired state to generate a Google Container Registry access token.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GCPSMAuth">
GCPSMAuth
</a>
</em>
</td>
<td>
<p>Auth defines the means for authenticating with GCP</p>
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
<p>ProjectID defines which project to use to authenticate with</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Generator">Generator
</h3>
<p>
<p>Generator is the common interface for all generators that is actually used to generate whatever is needed.</p>
</p>
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorKind">GeneratorKind
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ClusterGeneratorSpec">ClusterGeneratorSpec</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.GeneratorStep">GeneratorStep</a>)
</p>
<p>
<p>GeneratorKind represents a kind of generator.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ACRAccessToken&#34;</p></td>
<td><p>GeneratorKindACRAccessToken represents an Azure Container Registry access token generator.</p>
</td>
</tr><tr><td><p>&#34;CloudsmithAccessToken&#34;</p></td>
<td><p>GeneratorKindCloudsmithAccessToken represents a Cloudsmith access token generator.</p>
</td>
</tr><tr><td><p>&#34;ECRAuthorizationToken&#34;</p></td>
<td><p>GeneratorKindECRAuthorizationToken represents an AWS ECR authorization token generator.</p>
</td>
</tr><tr><td><p>&#34;Fake&#34;</p></td>
<td><p>GeneratorKindFake represents a fake generator for testing purposes.</p>
</td>
</tr><tr><td><p>&#34;GCRAccessToken&#34;</p></td>
<td><p>GeneratorKindGCRAccessToken represents a Google Container Registry access token generator.</p>
</td>
</tr><tr><td><p>&#34;GithubAccessToken&#34;</p></td>
<td><p>GeneratorKindGithubAccessToken represents a GitHub access token generator.</p>
</td>
</tr><tr><td><p>&#34;Grafana&#34;</p></td>
<td><p>GeneratorKindGrafana represents a Grafana token generator.</p>
</td>
</tr><tr><td><p>&#34;MFA&#34;</p></td>
<td><p>GeneratorKindMFA represents a Multi-Factor Authentication generator.</p>
</td>
</tr><tr><td><p>&#34;Password&#34;</p></td>
<td><p>GeneratorKindPassword represents a password generator.</p>
</td>
</tr><tr><td><p>&#34;QuayAccessToken&#34;</p></td>
<td><p>GeneratorKindQuayAccessToken represents a Quay access token generator.</p>
</td>
</tr><tr><td><p>&#34;SSHKey&#34;</p></td>
<td><p>GeneratorKindSSHKey represents an SSH key generator.</p>
</td>
</tr><tr><td><p>&#34;STSSessionToken&#34;</p></td>
<td><p>GeneratorKindSTSSessionToken represents an AWS STS session token generator.</p>
</td>
</tr><tr><td><p>&#34;UUID&#34;</p></td>
<td><p>GeneratorKindUUID represents a UUID generator.</p>
</td>
</tr><tr><td><p>&#34;VaultDynamicSecret&#34;</p></td>
<td><p>GeneratorKindVaultDynamicSecret represents a HashiCorp Vault dynamic secret generator.</p>
</td>
</tr><tr><td><p>&#34;Webhook&#34;</p></td>
<td><p>GeneratorKindWebhook represents a webhook-based generator.</p>
</td>
</tr></tbody>
</table>
<h3 id="&lt;UNKNOWN_API_GROUP&gt;.GeneratorProviderState">GeneratorProviderState
</h3>
<p>
<p>GeneratorProviderState represents the state of a generator provider that can be stored and retrieved.</p>
</p>
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ClusterGeneratorSpec">ClusterGeneratorSpec</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.GeneratorStep">GeneratorStep</a>)
</p>
<p>
<p>GeneratorSpec defines the configuration for various supported generator types.</p>
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
<code>acrAccessTokenSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAccessTokenSpec">
ACRAccessTokenSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cloudsmithAccessTokenSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.CloudsmithAccessTokenSpec">
CloudsmithAccessTokenSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ecrAuthorizationTokenSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.ECRAuthorizationTokenSpec">
ECRAuthorizationTokenSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>fakeSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.FakeSpec">
FakeSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>gcrAccessTokenSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GCRAccessTokenSpec">
GCRAccessTokenSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>githubAccessTokenSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GithubAccessTokenSpec">
GithubAccessTokenSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>quayAccessTokenSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.QuayAccessTokenSpec">
QuayAccessTokenSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>passwordSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PasswordSpec">
PasswordSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>sshKeySpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.SSHKeySpec">
SSHKeySpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>stsSessionTokenSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.STSSessionTokenSpec">
STSSessionTokenSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>uuidSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.UUIDSpec">
UUIDSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>vaultDynamicSecretSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecretSpec">
VaultDynamicSecretSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>webhookSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.WebhookSpec">
WebhookSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>grafanaSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaSpec">
GrafanaSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>mfaSpec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MFASpec">
MFASpec
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorState">GeneratorState
</h3>
<p>
<p>GeneratorState represents the state created and managed by a generator resource.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateSpec">
GeneratorStateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>garbageCollectionDeadline</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>GarbageCollectionDeadline is the time after which the generator state
will be deleted.
It is set by the controller which creates the generator state and
can be set configured by the user.
If the garbage collection deadline is not set the generator state will not be deleted.</p>
</td>
</tr>
<tr>
<td>
<code>resource</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<p>Resource is the generator manifest that produced the state.
It is a snapshot of the generator manifest at the time the state was produced.
This manifest will be used to delete the resource. Any configuration that is referenced
in the manifest should be available at the time of garbage collection. If that is not the case deletion will
be blocked by a finalizer.</p>
</td>
</tr>
<tr>
<td>
<code>state</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<p>State is the state that was produced by the generator implementation.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateStatus">
GeneratorStateStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorStateConditionType">GeneratorStateConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateStatus">GeneratorStateStatus</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateStatusCondition">GeneratorStateStatusCondition</a>)
</p>
<p>
<p>GeneratorStateConditionType represents the type of condition for a generator state.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Deletion Scheduled&#34;</p></td>
<td><p>GeneratorStateDeletionScheduled indicates the generator state is scheduled for deletion.</p>
</td>
</tr><tr><td><p>&#34;Pending Deletion&#34;</p></td>
<td><p>GeneratorStatePendingDeletion indicates the generator state is pending deletion.</p>
</td>
</tr><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>GeneratorStateReady indicates the generator state is ready and available.</p>
</td>
</tr><tr><td><p>&#34;Terminating&#34;</p></td>
<td><p>GeneratorStateTerminating indicates the generator state is terminating.</p>
</td>
</tr></tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorStateSpec">GeneratorStateSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorState">GeneratorState</a>)
</p>
<p>
<p>GeneratorStateSpec defines the desired state of a generator state resource.</p>
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
<code>garbageCollectionDeadline</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>GarbageCollectionDeadline is the time after which the generator state
will be deleted.
It is set by the controller which creates the generator state and
can be set configured by the user.
If the garbage collection deadline is not set the generator state will not be deleted.</p>
</td>
</tr>
<tr>
<td>
<code>resource</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<p>Resource is the generator manifest that produced the state.
It is a snapshot of the generator manifest at the time the state was produced.
This manifest will be used to delete the resource. Any configuration that is referenced
in the manifest should be available at the time of garbage collection. If that is not the case deletion will
be blocked by a finalizer.</p>
</td>
</tr>
<tr>
<td>
<code>state</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<p>State is the state that was produced by the generator implementation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorStateStatus">GeneratorStateStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorState">GeneratorState</a>)
</p>
<p>
<p>GeneratorStateStatus defines the observed state of a generator state resource.</p>
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
<code>lastType</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateConditionType">
GeneratorStateConditionType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastReason</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastMessage</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateStatusCondition">
[]GeneratorStateStatusCondition
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorStateStatusCondition">GeneratorStateStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateStatus">GeneratorStateStatus</a>)
</p>
<p>
<p>GeneratorStateStatusCondition represents the observed condition of a generator state.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStateConditionType">
GeneratorStateConditionType
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
<h3 id="generators.external-secrets.io/v1alpha1.GeneratorStatus">GeneratorStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.ACRAccessToken">ACRAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.AWSIAMKey">AWSIAMKey</a>, 
<a href="#generators.external-secrets.io/v1alpha1.BasicAuth">BasicAuth</a>, 
<a href="#generators.external-secrets.io/v1alpha1.CloudsmithAccessToken">CloudsmithAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.ClusterGenerator">ClusterGenerator</a>, 
<a href="#generators.external-secrets.io/v1alpha1.ECRAuthorizationToken">ECRAuthorizationToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Fake">Fake</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Federation">Federation</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GCRAccessToken">GCRAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GithubAccessToken">GithubAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Grafana">Grafana</a>, 
<a href="#generators.external-secrets.io/v1alpha1.MFA">MFA</a>, 
<a href="#generators.external-secrets.io/v1alpha1.MongoDB">MongoDB</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Neo4j">Neo4j</a>, 
<a href="#generators.external-secrets.io/v1alpha1.OpenAI">OpenAI</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Password">Password</a>, 
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQL">PostgreSQL</a>, 
<a href="#generators.external-secrets.io/v1alpha1.QuayAccessToken">QuayAccessToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQ">RabbitMQ</a>, 
<a href="#generators.external-secrets.io/v1alpha1.SSHKey">SSHKey</a>, 
<a href="#generators.external-secrets.io/v1alpha1.STSSessionToken">STSSessionToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.SendgridAuthorizationToken">SendgridAuthorizationToken</a>, 
<a href="#generators.external-secrets.io/v1alpha1.UUID">UUID</a>, 
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecret">VaultDynamicSecret</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Webhook">Webhook</a>)
</p>
<p>
<p>GeneratorStatus represents the status of a generator.</p>
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
<code>output</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GenericGenerator">GenericGenerator
</h3>
<p>
<p>GenericGenerator is a common interface for interacting with Generators.</p>
</p>
<h3 id="generators.external-secrets.io/v1alpha1.GithubAccessToken">GithubAccessToken
</h3>
<p>
<p>GithubAccessToken generates ghs_ accessToken</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GithubAccessTokenSpec">
GithubAccessTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL configures the GitHub instance URL. Defaults to <a href="https://github.com/">https://github.com/</a>.</p>
</td>
</tr>
<tr>
<td>
<code>appID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>installID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>repositories</code></br>
<em>
[]string
</em>
</td>
<td>
<p>List of repositories the token will have access to. If omitted, defaults to all repositories the GitHub App
is installed to.</p>
</td>
</tr>
<tr>
<td>
<code>permissions</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Map of permissions the token will have. If omitted, defaults to all permissions the GitHub App has.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GithubAuth">
GithubAuth
</a>
</em>
</td>
<td>
<p>Auth configures how ESO authenticates with a Github instance.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GithubAccessTokenSpec">GithubAccessTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GithubAccessToken">GithubAccessToken</a>)
</p>
<p>
<p>GithubAccessTokenSpec defines the desired state to generate a GitHub access token.</p>
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
<p>URL configures the GitHub instance URL. Defaults to <a href="https://github.com/">https://github.com/</a>.</p>
</td>
</tr>
<tr>
<td>
<code>appID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>installID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>repositories</code></br>
<em>
[]string
</em>
</td>
<td>
<p>List of repositories the token will have access to. If omitted, defaults to all repositories the GitHub App
is installed to.</p>
</td>
</tr>
<tr>
<td>
<code>permissions</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Map of permissions the token will have. If omitted, defaults to all permissions the GitHub App has.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GithubAuth">
GithubAuth
</a>
</em>
</td>
<td>
<p>Auth configures how ESO authenticates with a Github instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GithubAuth">GithubAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GithubAccessTokenSpec">GithubAccessTokenSpec</a>)
</p>
<p>
<p>GithubAuth defines the authentication configuration for GitHub access.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GithubSecretRef">
GithubSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GithubSecretRef">GithubSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GithubAuth">GithubAuth</a>)
</p>
<p>
<p>GithubSecretRef references a secret containing GitHub credentials.</p>
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
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Grafana">Grafana
</h3>
<p>
<p>Grafana represents a generator for Grafana service account tokens.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.GrafanaSpec">
GrafanaSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL is the URL of the Grafana instance.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaAuth">
GrafanaAuth
</a>
</em>
</td>
<td>
<p>Auth is the authentication configuration to authenticate
against the Grafana instance.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaServiceAccount">
GrafanaServiceAccount
</a>
</em>
</td>
<td>
<p>ServiceAccount is the configuration for the service account that
is supposed to be generated by the generator.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GrafanaAuth">GrafanaAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaSpec">GrafanaSpec</a>)
</p>
<p>
<p>GrafanaAuth defines the authentication methods for connecting to a Grafana instance.</p>
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
<code>token</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.SecretKeySelector">
SecretKeySelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A service account token used to authenticate against the Grafana instance.
Note: you need a token which has elevated permissions to create service accounts.
See here for the documentation on basic roles offered by Grafana:
<a href="https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/">https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/</a></p>
</td>
</tr>
<tr>
<td>
<code>basic</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaBasicAuth">
GrafanaBasicAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Basic auth credentials used to authenticate against the Grafana instance.
Note: you need a token which has elevated permissions to create service accounts.
See here for the documentation on basic roles offered by Grafana:
<a href="https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/">https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GrafanaBasicAuth">GrafanaBasicAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaAuth">GrafanaAuth</a>)
</p>
<p>
<p>GrafanaBasicAuth defines the credentials for basic authentication with Grafana.</p>
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
string
</em>
</td>
<td>
<p>A basic auth username used to authenticate against the Grafana instance.</p>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.SecretKeySelector">
SecretKeySelector
</a>
</em>
</td>
<td>
<p>A basic auth password used to authenticate against the Grafana instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GrafanaServiceAccount">GrafanaServiceAccount
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaSpec">GrafanaSpec</a>)
</p>
<p>
<p>GrafanaServiceAccount defines the configuration for a Grafana service account to be created.</p>
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
<p>Name is the name of the service account that will be created by ESO.</p>
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
<p>Role is the role of the service account.
See here for the documentation on basic roles offered by Grafana:
<a href="https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/">https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/access-control/rbac-fixed-basic-role-definitions/</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GrafanaServiceAccountTokenState">GrafanaServiceAccountTokenState
</h3>
<p>
<p>GrafanaServiceAccountTokenState is the state type produced by the Grafana generator.
It contains the service account ID, login and token ID which is enough to
identify the service account.</p>
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
<code>serviceAccount</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaStateServiceAccount">
GrafanaStateServiceAccount
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GrafanaSpec">GrafanaSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Grafana">Grafana</a>)
</p>
<p>
<p>GrafanaSpec controls the behavior of the grafana generator.</p>
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
<p>URL is the URL of the Grafana instance.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaAuth">
GrafanaAuth
</a>
</em>
</td>
<td>
<p>Auth is the authentication configuration to authenticate
against the Grafana instance.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccount</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaServiceAccount">
GrafanaServiceAccount
</a>
</em>
</td>
<td>
<p>ServiceAccount is the configuration for the service account that
is supposed to be generated by the generator.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.GrafanaStateServiceAccount">GrafanaStateServiceAccount
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaServiceAccountTokenState">GrafanaServiceAccountTokenState</a>)
</p>
<p>
<p>GrafanaStateServiceAccount contains the service account ID, login and token ID.</p>
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
<code>id</code></br>
<em>
int64
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>login</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tokenID</code></br>
<em>
int64
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.IAMKeysSpec">IAMKeysSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.AWSIAMKey">AWSIAMKey</a>)
</p>
<p>
<p>IAMKeysSpec defines the specification for IAM key generation.</p>
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
<p>Region specifies the region to operate in.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">
AWSAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines how to authenticate with AWS</p>
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
<p>You can assume a role before making calls to the
desired AWS service.</p>
</td>
</tr>
<tr>
<td>
<code>iamRef</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.IAMRef">
IAMRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.IAMRef">IAMRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.IAMKeysSpec">IAMKeysSpec</a>)
</p>
<p>
<p>IAMRef defines a reference to an IAM user.</p>
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
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>maxKeys</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MFA">MFA
</h3>
<p>
<p>MFA generates a new TOTP token that is compliant with RFC 6238.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.MFASpec">
MFASpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>secret</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Secret is a secret selector to a secret containing the seed secret to generate the TOTP value from.</p>
</td>
</tr>
<tr>
<td>
<code>length</code></br>
<em>
int
</em>
</td>
<td>
<p>Length defines the token length. Defaults to 6 characters.</p>
</td>
</tr>
<tr>
<td>
<code>timePeriod</code></br>
<em>
int
</em>
</td>
<td>
<p>TimePeriod defines how long the token can be active. Defaults to 30 seconds.</p>
</td>
</tr>
<tr>
<td>
<code>algorithm</code></br>
<em>
string
</em>
</td>
<td>
<p>Algorithm to use for encoding. Defaults to SHA1 as per the RFC.</p>
</td>
</tr>
<tr>
<td>
<code>when</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>When defines a time parameter that can be used to pin the origin time of the generated token.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MFASpec">MFASpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.MFA">MFA</a>)
</p>
<p>
<p>MFASpec controls the behavior of the mfa generator.</p>
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
<code>secret</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Secret is a secret selector to a secret containing the seed secret to generate the TOTP value from.</p>
</td>
</tr>
<tr>
<td>
<code>length</code></br>
<em>
int
</em>
</td>
<td>
<p>Length defines the token length. Defaults to 6 characters.</p>
</td>
</tr>
<tr>
<td>
<code>timePeriod</code></br>
<em>
int
</em>
</td>
<td>
<p>TimePeriod defines how long the token can be active. Defaults to 30 seconds.</p>
</td>
</tr>
<tr>
<td>
<code>algorithm</code></br>
<em>
string
</em>
</td>
<td>
<p>Algorithm to use for encoding. Defaults to SHA1 as per the RFC.</p>
</td>
</tr>
<tr>
<td>
<code>when</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>When defines a time parameter that can be used to pin the origin time of the generated token.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDB">MongoDB
</h3>
<p>
<p>MongoDB represents a MongoDB generator configuration.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.MongoDBSpec">
MongoDBSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>database</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBDatabase">
MongoDBDatabase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBAuth">
MongoDBAuth
</a>
</em>
</td>
<td>
<p>Auth is the authentication configuration to authenticate
against the MongoDB instance.</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBUser">
MongoDBUser
</a>
</em>
</td>
<td>
<p>User is the configuration for the service account that
is supposed to be generated by the generator.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBAuth">MongoDBAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBSpec">MongoDBSpec</a>)
</p>
<p>
<p>MongoDBAuth defines MongoDB authentication configuration.</p>
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
<code>scram</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBSCRAMAuth">
MongoDBSCRAMAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Basic auth credentials used to authenticate against the MongoDB instance.
Note: you need a token which has elevated permissions to create service accounts.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBAuthSecretRef">MongoDBAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBSCRAMAuth">MongoDBSCRAMAuth</a>)
</p>
<p>
<p>MongoDBAuthSecretRef defines secret references for MongoDB authentication.</p>
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
<code>usernameSecretRef</code></br>
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
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBDatabase">MongoDBDatabase
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBSpec">MongoDBSpec</a>)
</p>
<p>
<p>MongoDBDatabase defines the MongoDB database configuration.</p>
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
<code>adminDB</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AdminDB is the name of the admin database</p>
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
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Port is the port of the MongoDB instance. Defaults to 27017</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBRole">MongoDBRole
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBUser">MongoDBUser</a>)
</p>
<p>
<p>MongoDBRole defines a MongoDB role configuration.</p>
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
<p>Name is the name of the role linked to the service account.</p>
</td>
</tr>
<tr>
<td>
<code>db</code></br>
<em>
string
</em>
</td>
<td>
<p>DB is where the role will be applied.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBSCRAMAuth">MongoDBSCRAMAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBAuth">MongoDBAuth</a>)
</p>
<p>
<p>MongoDBSCRAMAuth defines SCRAM authentication for MongoDB.</p>
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
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>secretRef</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBAuthSecretRef">
MongoDBAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBSpec">MongoDBSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDB">MongoDB</a>)
</p>
<p>
<p>MongoDBSpec controls the behavior of the MongoDB generator.</p>
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
<code>database</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBDatabase">
MongoDBDatabase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBAuth">
MongoDBAuth
</a>
</em>
</td>
<td>
<p>Auth is the authentication configuration to authenticate
against the MongoDB instance.</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBUser">
MongoDBUser
</a>
</em>
</td>
<td>
<p>User is the configuration for the service account that
is supposed to be generated by the generator.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBUser">MongoDBUser
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBSpec">MongoDBSpec</a>)
</p>
<p>
<p>MongoDBUser defines a MongoDB user configuration.</p>
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
<p>Name is the name of the service account.
If not provided, the generator will create one.</p>
</td>
</tr>
<tr>
<td>
<code>roles</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.MongoDBRole">
[]MongoDBRole
</a>
</em>
</td>
<td>
<p>Roles is the configuration for the service account</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.MongoDBUserState">MongoDBUserState
</h3>
<p>
<p>MongoDBUserState represents the state of a MongoDB user.</p>
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
<code>user</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.NTLMProtocol">NTLMProtocol
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.AuthorizationProtocol">AuthorizationProtocol</a>)
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
<h3 id="generators.external-secrets.io/v1alpha1.Neo4j">Neo4j
</h3>
<p>
<p>Neo4j generates a Neo4j user based on the configuration parameters in spec.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.Neo4jSpec">
Neo4jSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>database</code></br>
<em>
string
</em>
</td>
<td>
<p>Database is the name of the database to connect to.
If not specified, the &ldquo;neo4j&rdquo; database will be used.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jAuth">
Neo4jAuth
</a>
</em>
</td>
<td>
<p>Auth contains the credentials or auth configuration</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jUser">
Neo4jUser
</a>
</em>
</td>
<td>
<p>User is the data of the user to be created.</p>
</td>
</tr>
<tr>
<td>
<code>enterprise</code></br>
<em>
bool
</em>
</td>
<td>
<p>If the neo4j instance is running in enterprise mode.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Neo4jAuth">Neo4jAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jSpec">Neo4jSpec</a>)
</p>
<p>
<p>Neo4jAuth defines Neo4j authentication configuration.</p>
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
<code>uri</code></br>
<em>
string
</em>
</td>
<td>
<p>URI is the connection URI for the Neo4j database.
Example: bolt://neo4j.default.svc.cluster.local:7687</p>
</td>
</tr>
<tr>
<td>
<code>basic</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jBasicAuth">
Neo4jBasicAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Basic auth credentials used to authenticate against the Neo4j instance.</p>
</td>
</tr>
<tr>
<td>
<code>bearer</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jBearerAuth">
Neo4jBearerAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Bearer auth token used to authenticate against the Neo4j instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Neo4jAuthProvider">Neo4jAuthProvider
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jUser">Neo4jUser</a>)
</p>
<p>
<p>Neo4jAuthProvider represents the authentication provider type.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;native&#34;</p></td>
<td><p>Neo4jAuthProviderNative represents native Neo4j authentication.</p>
</td>
</tr></tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Neo4jBasicAuth">Neo4jBasicAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jAuth">Neo4jAuth</a>)
</p>
<p>
<p>Neo4jBasicAuth defines basic authentication for Neo4j.</p>
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
string
</em>
</td>
<td>
<p>A basic auth username used to authenticate against the Neo4j instance.</p>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>A basic auth password used to authenticate against the Neo4j instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Neo4jBearerAuth">Neo4jBearerAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jAuth">Neo4jAuth</a>)
</p>
<p>
<p>Neo4jBearerAuth defines bearer token authentication for Neo4j.</p>
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
<code>token</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>A bearer auth token used to authenticate against the Neo4j instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Neo4jSpec">Neo4jSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4j">Neo4j</a>)
</p>
<p>
<p>Neo4jSpec controls the behavior of the neo4j generator.</p>
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
<code>database</code></br>
<em>
string
</em>
</td>
<td>
<p>Database is the name of the database to connect to.
If not specified, the &ldquo;neo4j&rdquo; database will be used.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jAuth">
Neo4jAuth
</a>
</em>
</td>
<td>
<p>Auth contains the credentials or auth configuration</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jUser">
Neo4jUser
</a>
</em>
</td>
<td>
<p>User is the data of the user to be created.</p>
</td>
</tr>
<tr>
<td>
<code>enterprise</code></br>
<em>
bool
</em>
</td>
<td>
<p>If the neo4j instance is running in enterprise mode.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Neo4jUser">Neo4jUser
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jSpec">Neo4jSpec</a>)
</p>
<p>
<p>Neo4jUser defines a Neo4j user configuration.</p>
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
<code>user</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the user to be created.
Must not contain a dash (&ldquo;-&rdquo;) character.</p>
</td>
</tr>
<tr>
<td>
<code>suffixSize</code></br>
<em>
int
</em>
</td>
<td>
<p>SuffixSize define the size of the random suffix added after the defined username.
If not specified, a random suffix of size 8 will be used.
If set to 0, no suffix will be added.</p>
</td>
</tr>
<tr>
<td>
<code>roles</code></br>
<em>
[]string
</em>
</td>
<td>
<p>The roles to be assigned to the user (Neo4j Enterprise only).
See <a href="https://neo4j.com/docs/operations-manual/current/authentication-authorization/built-in-roles/">https://neo4j.com/docs/operations-manual/current/authentication-authorization/built-in-roles/</a>
for a list of built-in roles.
If contains non-existing roles, they will be created as copy of &ldquo;PUBLIC&rdquo; role.
If empty, the user will be created with no role.</p>
</td>
</tr>
<tr>
<td>
<code>home</code></br>
<em>
string
</em>
</td>
<td>
<p>The home database of the user (Neo4j Enterprise only).</p>
</td>
</tr>
<tr>
<td>
<code>provider</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.Neo4jAuthProvider">
Neo4jAuthProvider
</a>
</em>
</td>
<td>
<p>The auth provider to be used for the user (Neo4j Enterprise only).
Currently only &ldquo;native&rdquo; is supported.
&ldquo;native&rdquo; provider is used for Neo4j Community</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Neo4jUserState">Neo4jUserState
</h3>
<p>
<p>Neo4jUserState represents the state of a Neo4j user.</p>
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
<code>user</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.OpenAI">OpenAI
</h3>
<p>
<p>OpenAI generates an OpenAI service account based on the configuration parameters in spec.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.OpenAISpec">
OpenAISpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>projectId</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectID is the id of the project the account will related to.</p>
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
<p>Host is the server where the openAI api is hosted.
Default: &ldquo;<a href="https://api.openai.com/v1&quot;">https://api.openai.com/v1&rdquo;</a></p>
</td>
</tr>
<tr>
<td>
<code>openAiAdminKey</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>OpenAiAdminKey contains the Admin API Key used to authenticate against the OpenAI server.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountNamePrefix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountNamePrefix define a prefix to add before the generated name for the service account</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountNameSize</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountNameSize define the size of the generated name for the service account
Default: 12</p>
</td>
</tr>
<tr>
<td>
<code>cleanupPolicy</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.CleanupPolicy">
CleanupPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CleanupPolicy controls the behavior of the cleanup process</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.OpenAISpec">OpenAISpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.OpenAI">OpenAI</a>)
</p>
<p>
<p>OpenAISpec controls the behavior of the openAI generator.</p>
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
<code>projectId</code></br>
<em>
string
</em>
</td>
<td>
<p>ProjectID is the id of the project the account will related to.</p>
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
<p>Host is the server where the openAI api is hosted.
Default: &ldquo;<a href="https://api.openai.com/v1&quot;">https://api.openai.com/v1&rdquo;</a></p>
</td>
</tr>
<tr>
<td>
<code>openAiAdminKey</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>OpenAiAdminKey contains the Admin API Key used to authenticate against the OpenAI server.</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountNamePrefix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountNamePrefix define a prefix to add before the generated name for the service account</p>
</td>
</tr>
<tr>
<td>
<code>serviceAccountNameSize</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountNameSize define the size of the generated name for the service account
Default: 12</p>
</td>
</tr>
<tr>
<td>
<code>cleanupPolicy</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.CleanupPolicy">
CleanupPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CleanupPolicy controls the behavior of the cleanup process</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.OpenAiAPIKey">OpenAiAPIKey
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.OpenAiServiceAccount">OpenAiServiceAccount</a>)
</p>
<p>
<p>OpenAiAPIKey represents an OpenAI API key.</p>
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
<code>object</code></br>
<em>
string
</em>
</td>
<td>
<p>Object defines the type of this OpenAI API key resource.
Example: &ldquo;organization.project.service_account.api_key&rdquo;</p>
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
<p>Value is the actual secret API key (e.g., &ldquo;sk-&hellip;&rdquo;).</p>
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
<p>Name is the display name of the API key.</p>
</td>
</tr>
<tr>
<td>
<code>created_at</code></br>
<em>
int64
</em>
</td>
<td>
<p>CreatedAt is the Unix timestamp representing creation time.</p>
</td>
</tr>
<tr>
<td>
<code>last_used_at</code></br>
<em>
int64
</em>
</td>
<td>
<p>LastUsedAt is the Unix timestamp representing the last time the API key was used.</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the unique identifier of the API key.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.OpenAiServiceAccount">OpenAiServiceAccount
</h3>
<p>
<p>OpenAiServiceAccount represents an OpenAI service account.</p>
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
<code>object</code></br>
<em>
string
</em>
</td>
<td>
<p>Object defines the type of this OpenAI resource.
Example: &ldquo;organization.project.service_account&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the unique identifier of the service account.</p>
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
<p>Name is the display name of the service account.</p>
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
<p>Role defines the role assigned to this service account (e.g., &ldquo;member&rdquo;).</p>
</td>
</tr>
<tr>
<td>
<code>created_at</code></br>
<em>
int64
</em>
</td>
<td>
<p>CreatedAt is the Unix timestamp representing creation time.</p>
</td>
</tr>
<tr>
<td>
<code>api_key</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.OpenAiAPIKey">
OpenAiAPIKey
</a>
</em>
</td>
<td>
<p>APIKey contains the API key associated with this service account.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.OpenAiServiceAccountState">OpenAiServiceAccountState
</h3>
<p>
<p>OpenAiServiceAccountState represents the state of an OpenAI service account.</p>
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
<code>serviceAccountId</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>apiKeyId</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Password">Password
</h3>
<p>
<p>Password generates a random password based on the
configuration parameters in spec.
You can specify the length, characterset and other attributes.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.PasswordSpec">
PasswordSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>length</code></br>
<em>
int
</em>
</td>
<td>
<p>Length of the password to be generated.
Defaults to 24</p>
</td>
</tr>
<tr>
<td>
<code>digits</code></br>
<em>
int
</em>
</td>
<td>
<p>Digits specifies the number of digits in the generated
password. If omitted it defaults to 25% of the length of the password</p>
</td>
</tr>
<tr>
<td>
<code>symbols</code></br>
<em>
int
</em>
</td>
<td>
<p>Symbols specifies the number of symbol characters in the generated
password. If omitted it defaults to 25% of the length of the password</p>
</td>
</tr>
<tr>
<td>
<code>symbolCharacters</code></br>
<em>
string
</em>
</td>
<td>
<p>SymbolCharacters specifies the special characters that should be used
in the generated password.</p>
</td>
</tr>
<tr>
<td>
<code>noUpper</code></br>
<em>
bool
</em>
</td>
<td>
<p>Set NoUpper to disable uppercase characters</p>
</td>
</tr>
<tr>
<td>
<code>allowRepeat</code></br>
<em>
bool
</em>
</td>
<td>
<p>set AllowRepeat to true to allow repeating characters.</p>
</td>
</tr>
<tr>
<td>
<code>encoding</code></br>
<em>
string
</em>
</td>
<td>
<p>Encoding specifies the encoding of the generated password.
Valid values are:
- &ldquo;raw&rdquo; (default): no encoding
- &ldquo;base64&rdquo;: standard base64 encoding
- &ldquo;base64url&rdquo;: base64url encoding
- &ldquo;base32&rdquo;: base32 encoding
- &ldquo;hex&rdquo;: hexadecimal encoding</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PasswordSpec">PasswordSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.BasicAuthSpec">BasicAuthSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Password">Password</a>)
</p>
<p>
<p>PasswordSpec controls the behavior of the password generator.</p>
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
<code>length</code></br>
<em>
int
</em>
</td>
<td>
<p>Length of the password to be generated.
Defaults to 24</p>
</td>
</tr>
<tr>
<td>
<code>digits</code></br>
<em>
int
</em>
</td>
<td>
<p>Digits specifies the number of digits in the generated
password. If omitted it defaults to 25% of the length of the password</p>
</td>
</tr>
<tr>
<td>
<code>symbols</code></br>
<em>
int
</em>
</td>
<td>
<p>Symbols specifies the number of symbol characters in the generated
password. If omitted it defaults to 25% of the length of the password</p>
</td>
</tr>
<tr>
<td>
<code>symbolCharacters</code></br>
<em>
string
</em>
</td>
<td>
<p>SymbolCharacters specifies the special characters that should be used
in the generated password.</p>
</td>
</tr>
<tr>
<td>
<code>noUpper</code></br>
<em>
bool
</em>
</td>
<td>
<p>Set NoUpper to disable uppercase characters</p>
</td>
</tr>
<tr>
<td>
<code>allowRepeat</code></br>
<em>
bool
</em>
</td>
<td>
<p>set AllowRepeat to true to allow repeating characters.</p>
</td>
</tr>
<tr>
<td>
<code>encoding</code></br>
<em>
string
</em>
</td>
<td>
<p>Encoding specifies the encoding of the generated password.
Valid values are:
- &ldquo;raw&rdquo; (default): no encoding
- &ldquo;base64&rdquo;: standard base64 encoding
- &ldquo;base64url&rdquo;: base64url encoding
- &ldquo;base32&rdquo;: base32 encoding
- &ldquo;hex&rdquo;: hexadecimal encoding</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQL">PostgreSQL
</h3>
<p>
<p>PostgreSQL generates a PostgreSQL user based on the configuration parameters in spec.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLSpec">
PostgreSQLSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>database</code></br>
<em>
string
</em>
</td>
<td>
<p>Database is the name of the database to connect to.
If not specified, the &ldquo;postgres&rdquo; database will be used.</p>
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
<p>Host is the server where the database is hosted.</p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
string
</em>
</td>
<td>
<p>Port is the port of the database to connect to.
If not specified, the &ldquo;5432&rdquo; port will be used.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLAuth">
PostgreSQLAuth
</a>
</em>
</td>
<td>
<p>Auth contains the credentials or auth configuration</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLUser">
PostgreSQLUser
</a>
</em>
</td>
<td>
<p>User is the data of the user to be created.</p>
</td>
</tr>
<tr>
<td>
<code>cleanupPolicy</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLCleanupPolicy">
PostgreSQLCleanupPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQLAuth">PostgreSQLAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLSpec">PostgreSQLSpec</a>)
</p>
<p>
<p>PostgreSQLAuth defines PostgreSQL authentication configuration.</p>
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
string
</em>
</td>
<td>
<p>A basic auth username used to authenticate against the PostgreSQL instance.</p>
</td>
</tr>
<tr>
<td>
<code>password</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>A basic auth password used to authenticate against the PostgreSQL instance.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQLCleanupPolicy">PostgreSQLCleanupPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLSpec">PostgreSQLSpec</a>)
</p>
<p>
<p>PostgreSQLCleanupPolicy controls the cleanup policy for the postgreSQL generator.</p>
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
<code>CleanupPolicy</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.CleanupPolicy">
CleanupPolicy
</a>
</em>
</td>
<td>
<p>
(Members of <code>CleanupPolicy</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>activityTrackingInterval</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ActivityTrackingInterval is the cron expression to run the user activity tracking</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQLSpec">PostgreSQLSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQL">PostgreSQL</a>)
</p>
<p>
<p>PostgreSQLSpec controls the behavior of the postgreSQL generator.</p>
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
<code>database</code></br>
<em>
string
</em>
</td>
<td>
<p>Database is the name of the database to connect to.
If not specified, the &ldquo;postgres&rdquo; database will be used.</p>
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
<p>Host is the server where the database is hosted.</p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
string
</em>
</td>
<td>
<p>Port is the port of the database to connect to.
If not specified, the &ldquo;5432&rdquo; port will be used.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLAuth">
PostgreSQLAuth
</a>
</em>
</td>
<td>
<p>Auth contains the credentials or auth configuration</p>
</td>
</tr>
<tr>
<td>
<code>user</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLUser">
PostgreSQLUser
</a>
</em>
</td>
<td>
<p>User is the data of the user to be created.</p>
</td>
</tr>
<tr>
<td>
<code>cleanupPolicy</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLCleanupPolicy">
PostgreSQLCleanupPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQLUser">PostgreSQLUser
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLSpec">PostgreSQLSpec</a>)
</p>
<p>
<p>PostgreSQLUser defines a PostgreSQL user.</p>
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
string
</em>
</td>
<td>
<p>The username of the user to be created.</p>
</td>
</tr>
<tr>
<td>
<code>suffixSize</code></br>
<em>
int
</em>
</td>
<td>
<p>SuffixSize define the size of the random suffix added after the defined username.
If not specified, a random suffix of size 8 will be used.
If set to 0, no suffix will be added.</p>
</td>
</tr>
<tr>
<td>
<code>attributes</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLUserAttribute">
[]PostgreSQLUserAttribute
</a>
</em>
</td>
<td>
<p>Attributes is the list of PostgreSQL role attributes assigned to this user.</p>
</td>
</tr>
<tr>
<td>
<code>roles</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Roles is the list of existing roles that will be granted to this user.
If a role does not exist, it will be created without any attributes.</p>
</td>
</tr>
<tr>
<td>
<code>destructiveCleanup</code></br>
<em>
bool
</em>
</td>
<td>
<p>If set to true, the generator will drop all objects owned by the user
before deleting the user during cleanup.
If false (default), ownership of all objects will be reassigned
to the role specified in <code>spec.user.reassignTo</code>.</p>
</td>
</tr>
<tr>
<td>
<code>reassignTo</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the role to which all owned objects should be reassigned
during cleanup (if DestructiveCleanup is false).
If not specified, the role from <code>spec.auth.username</code> will be used.
If the role does not exist, it will be created with no attributes or roles..</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQLUserAttribute">PostgreSQLUserAttribute
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.PostgreSQLUser">PostgreSQLUser</a>)
</p>
<p>
<p>PostgreSQLUserAttribute defines a PostgreSQL user attribute.</p>
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
<p>Attribute is the name of the PostgreSQL role attribute to be set for the user.
Valid values: SUPERUSER, CREATEDB, CREATEROLE, REPLICATION, NOINHERIT, BYPASSRLS, CONNECTION_LIMIT.</p>
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
<p>Optional value for the attribute (e.g., connection limit)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQLUserAttributesEnum">PostgreSQLUserAttributesEnum
(<code>string</code> alias)</p></h3>
<p>
<p>PostgreSQLUserAttributesEnum represents PostgreSQL user attributes.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;BYPASSRLS&#34;</p></td>
<td><p>PostgreSQLUserByPassRls grants the ability to bypass row-level security.</p>
</td>
</tr><tr><td><p>&#34;CONNECTION LIMIT&#34;</p></td>
<td><p>PostgreSQLUserConnectionLimit grants the ability to limit the number of connections.</p>
</td>
</tr><tr><td><p>&#34;CREATEDB&#34;</p></td>
<td><p>PostgreSQLUserCreateDb grants the ability to create databases.</p>
</td>
</tr><tr><td><p>&#34;CREATEROLE&#34;</p></td>
<td><p>PostgreSQLUserCreateRole grants the ability to create roles.</p>
</td>
</tr><tr><td><p>&#34;LOGIN&#34;</p></td>
<td><p>PostgreSQLUserLogin grants the ability to login.</p>
</td>
</tr><tr><td><p>&#34;NOINHERIT&#34;</p></td>
<td><p>PostgreSQLUserNoInherit grants the ability to inherit privileges.</p>
</td>
</tr><tr><td><p>&#34;PASSWORD&#34;</p></td>
<td><p>PostgreSQLUserPassword grants the ability to set a password.</p>
</td>
</tr><tr><td><p>&#34;REPLICATION&#34;</p></td>
<td><p>PostgreSQLUserReplication grants the ability to replicate data.</p>
</td>
</tr><tr><td><p>&#34;SUPERUSER&#34;</p></td>
<td><p>PostgreSQLUserSuperUser grants superuser privileges.</p>
</td>
</tr></tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.PostgreSQLUserState">PostgreSQLUserState
</h3>
<p>
<p>PostgreSQLUserState represents the state of a PostgreSQL user.</p>
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
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.QuayAccessToken">QuayAccessToken
</h3>
<p>
<p>QuayAccessToken generates Quay oauth token for pulling/pushing images</p>
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
<a href="#generators.external-secrets.io/v1alpha1.QuayAccessTokenSpec">
QuayAccessTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL configures the Quay instance URL. Defaults to quay.io.</p>
</td>
</tr>
<tr>
<td>
<code>robotAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the robot account you are federating with</p>
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
<p>Name of the service account you are federating with</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.QuayAccessTokenSpec">QuayAccessTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.QuayAccessToken">QuayAccessToken</a>)
</p>
<p>
<p>QuayAccessTokenSpec defines the desired state to generate a Quay access token.</p>
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
<p>URL configures the Quay instance URL. Defaults to quay.io.</p>
</td>
</tr>
<tr>
<td>
<code>robotAccount</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the robot account you are federating with</p>
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
<p>Name of the service account you are federating with</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQ">RabbitMQ
</h3>
<p>
<p>RabbitMQ generates a random password based on the.
configuration parameters in spec.
You can specify the length, characterset and other attributes.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQSpec">
RabbitMQSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQServer">
RabbitMQServer
</a>
</em>
</td>
<td>
<p>Server defines the RabbitMQ Server Parameters to connect to.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQAuth">
RabbitMQAuth
</a>
</em>
</td>
<td>
<p>Auth defines the RabbitMQ authentication parameters.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQConfig">
RabbitMQConfig
</a>
</em>
</td>
<td>
<p>Config defines how to rotate the Secret within RabbitMQ.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQAuth">RabbitMQAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQSpec">RabbitMQSpec</a>)
</p>
<p>
<p>RabbitMQAuth defines the RabbitMQ authentication parameters.</p>
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
<code>basicAuth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQBasicAuth">
RabbitMQBasicAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BasicAuth contains basic authentication credentials.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQBasicAuth">RabbitMQBasicAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQAuth">RabbitMQAuth</a>)
</p>
<p>
<p>RabbitMQBasicAuth contains basic authentication credentials.</p>
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
string
</em>
</td>
<td>
<p>Username is the RabbitMQ username to connect to.
Must have sufficient permissions for administration tasks.</p>
</td>
</tr>
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
<p>PasswordSecretRef is a reference to a secret containing the password.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQConfig">RabbitMQConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQSpec">RabbitMQSpec</a>)
</p>
<p>
<p>RabbitMQConfig contains the configuration for password rotation.</p>
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
string
</em>
</td>
<td>
<p>Username contains the target username to rotate passwords for.</p>
</td>
</tr>
<tr>
<td>
<code>passwordPolicy</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQPasswordPolicy">
RabbitMQPasswordPolicy
</a>
</em>
</td>
<td>
<p>PasswordPolicy contains the password policy to apply.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQPasswordGeneratorRef">RabbitMQPasswordGeneratorRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQPasswordPolicy">RabbitMQPasswordPolicy</a>)
</p>
<p>
<p>RabbitMQPasswordGeneratorRef is a reference to a password generator.</p>
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
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQPasswordPolicy">RabbitMQPasswordPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQConfig">RabbitMQConfig</a>)
</p>
<p>
<p>RabbitMQPasswordPolicy contains the password policy to apply.</p>
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
<code>passwordGeneratorRef</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQPasswordGeneratorRef">
RabbitMQPasswordGeneratorRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PasswordGeneratorRef is a reference to a password generator.</p>
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
<p>SecretRef is a reference to a Secret Key containing the Password.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQServer">RabbitMQServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQSpec">RabbitMQSpec</a>)
</p>
<p>
<p>RabbitMQServer defines the RabbitMQ Server Parameters to connect to.</p>
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
<p>Host is the hostname of the RabbitMQ server.</p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Port is the port of the RabbitMQ server.</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLS indicates whether to use TLS.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RabbitMQSpec">RabbitMQSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQ">RabbitMQ</a>)
</p>
<p>
<p>RabbitMQSpec user generation behavior for rabbitMQ.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQServer">
RabbitMQServer
</a>
</em>
</td>
<td>
<p>Server defines the RabbitMQ Server Parameters to connect to.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQAuth">
RabbitMQAuth
</a>
</em>
</td>
<td>
<p>Auth defines the RabbitMQ authentication parameters.</p>
</td>
</tr>
<tr>
<td>
<code>config</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RabbitMQConfig">
RabbitMQConfig
</a>
</em>
</td>
<td>
<p>Config defines how to rotate the Secret within RabbitMQ.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.RequestParameters">RequestParameters
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.STSSessionTokenSpec">STSSessionTokenSpec</a>)
</p>
<p>
<p>RequestParameters contains parameters that can be passed to the STS service.</p>
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
<code>sessionDuration</code></br>
<em>
int32
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>serialNumber</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SerialNumber is the identification number of the MFA device that is associated with the IAM user who is making
the GetSessionToken call.
Possible values: hardware device (such as GAHT12345678) or an Amazon Resource Name (ARN) for a virtual device
(such as arn:aws:iam::123456789012:mfa/user)</p>
</td>
</tr>
<tr>
<td>
<code>tokenCode</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TokenCode is the value provided by the MFA device, if MFA is required.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.SSHKey">SSHKey
</h3>
<p>
<p>SSHKey generates SSH key pairs.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.SSHKeySpec">
SSHKeySpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>keyType</code></br>
<em>
string
</em>
</td>
<td>
<p>KeyType specifies the SSH key type (rsa, ecdsa, ed25519)</p>
</td>
</tr>
<tr>
<td>
<code>keySize</code></br>
<em>
int
</em>
</td>
<td>
<p>KeySize specifies the key size for RSA keys (default: 2048) and ECDSA keys (default: 256).
For RSA keys: 2048, 3072, 4096
For ECDSA keys: 256, 384, 521
Ignored for ed25519 keys</p>
</td>
</tr>
<tr>
<td>
<code>comment</code></br>
<em>
string
</em>
</td>
<td>
<p>Comment specifies an optional comment for the SSH key</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.SSHKeySpec">SSHKeySpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.SSHKey">SSHKey</a>)
</p>
<p>
<p>SSHKeySpec controls the behavior of the ssh key generator.</p>
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
<code>keyType</code></br>
<em>
string
</em>
</td>
<td>
<p>KeyType specifies the SSH key type (rsa, ecdsa, ed25519)</p>
</td>
</tr>
<tr>
<td>
<code>keySize</code></br>
<em>
int
</em>
</td>
<td>
<p>KeySize specifies the key size for RSA keys (default: 2048) and ECDSA keys (default: 256).
For RSA keys: 2048, 3072, 4096
For ECDSA keys: 256, 384, 521
Ignored for ed25519 keys</p>
</td>
</tr>
<tr>
<td>
<code>comment</code></br>
<em>
string
</em>
</td>
<td>
<p>Comment specifies an optional comment for the SSH key</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.STSSessionToken">STSSessionToken
</h3>
<p>
<p>STSSessionToken uses the GetSessionToken API to retrieve an authorization token.
The authorization token is valid for 12 hours.
The authorizationToken returned is a base64 encoded string that can be decoded.
For more information, see GetSessionToken (<a href="https://docs.aws.amazon.com/STS/latest/APIReference/API_GetSessionToken.html">https://docs.aws.amazon.com/STS/latest/APIReference/API_GetSessionToken.html</a>).</p>
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
<a href="#generators.external-secrets.io/v1alpha1.STSSessionTokenSpec">
STSSessionTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region specifies the region to operate in.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">
AWSAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines how to authenticate with AWS</p>
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
<p>You can assume a role before making calls to the
desired AWS service.</p>
</td>
</tr>
<tr>
<td>
<code>requestParameters</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RequestParameters">
RequestParameters
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RequestParameters contains parameters that can be passed to the STS service.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.STSSessionTokenSpec">STSSessionTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.STSSessionToken">STSSessionToken</a>)
</p>
<p>
<p>STSSessionTokenSpec defines the desired state to generate an AWS STS session token.</p>
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
<p>Region specifies the region to operate in.</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.AWSAuth">
AWSAuth
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Auth defines how to authenticate with AWS</p>
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
<p>You can assume a role before making calls to the
desired AWS service.</p>
</td>
</tr>
<tr>
<td>
<code>requestParameters</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.RequestParameters">
RequestParameters
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>RequestParameters contains parameters that can be passed to the STS service.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.SecretKeySelector">SecretKeySelector
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GrafanaAuth">GrafanaAuth</a>, 
<a href="#generators.external-secrets.io/v1alpha1.GrafanaBasicAuth">GrafanaBasicAuth</a>, 
<a href="#generators.external-secrets.io/v1alpha1.WebhookSecret">WebhookSecret</a>)
</p>
<p>
<p>SecretKeySelector defines a reference to a specific key within a Kubernetes Secret.</p>
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
<p>The name of the Secret resource being referred to.</p>
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
<p>The key where the token is found.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.SendgridAuth">SendgridAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.SendgridTokenSpec">SendgridTokenSpec</a>)
</p>
<p>
<p>SendgridAuth contains the authentication information for the Sendgrid API.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.SendgridAuthSecretRef">
SendgridAuthSecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.SendgridAuthSecretRef">SendgridAuthSecretRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.SendgridAuth">SendgridAuth</a>)
</p>
<p>
<p>SendgridAuthSecretRef contains a reference to a secret key selector.</p>
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
<code>apiKeySecretRef</code></br>
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
<h3 id="generators.external-secrets.io/v1alpha1.SendgridAuthorizationToken">SendgridAuthorizationToken
</h3>
<p>
<p>SendgridAuthorizationToken generates sendgrid api keys</p>
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
<a href="#generators.external-secrets.io/v1alpha1.SendgridTokenSpec">
SendgridTokenSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>dataResidency</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>scopes</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.SendgridAuth">
SendgridAuth
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.SendgridTokenSpec">SendgridTokenSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.SendgridAuthorizationToken">SendgridAuthorizationToken</a>)
</p>
<p>
<p>SendgridTokenSpec contains the specification for the Sendgrid API token.</p>
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
<code>dataResidency</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>scopes</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.SendgridAuth">
SendgridAuth
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.StatefulResource">StatefulResource
</h3>
<p>
<p>StatefulResource represents a Kubernetes resource that has state which can be tracked.</p>
</p>
<h3 id="generators.external-secrets.io/v1alpha1.UUID">UUID
</h3>
<p>
<p>UUID generates a version 1 UUID (e56657e3-764f-11ef-a397-65231a88c216).</p>
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
<a href="#generators.external-secrets.io/v1alpha1.UUIDSpec">
UUIDSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.UUIDSpec">UUIDSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.UUID">UUID</a>)
</p>
<p>
<p>UUIDSpec controls the behavior of the uuid generator.</p>
</p>
<h3 id="generators.external-secrets.io/v1alpha1.UsernameSpec">UsernameSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.BasicAuthSpec">BasicAuthSpec</a>)
</p>
<p>
<p>UsernameSpec controls the behavior of the username generated.</p>
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
<code>length</code></br>
<em>
int
</em>
</td>
<td>
<p>Length of each word of the username to be generated.
Defaults to 8</p>
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
<p>Prefix specifies a prefix to be added to the generated
username. If omitted it defaults to empty</p>
</td>
</tr>
<tr>
<td>
<code>sufix</code></br>
<em>
string
</em>
</td>
<td>
<p>Sufix specifies a sufix to be added to the generated
username. If omitted it defaults to empty</p>
</td>
</tr>
<tr>
<td>
<code>wordCount</code></br>
<em>
int
</em>
</td>
<td>
<p>WordCount specifies the number of words in the generated
username. If omitted it defaults to 1</p>
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
<p>Separator specifies the separator character that should be used
in the generated username. If omitted it defaults to &ldquo;_&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>includeNumbers</code></br>
<em>
bool
</em>
</td>
<td>
<p>set IncludeNumbers to add 4 numbers at the end of the username after the sufix.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.VaultDynamicSecret">VaultDynamicSecret
</h3>
<p>
<p>VaultDynamicSecret represents a generator that can create dynamic secrets from HashiCorp Vault.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecretSpec">
VaultDynamicSecretSpec
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
The ESO controller is instantiated with a specific controller name and filters VDS based on this property</p>
</td>
</tr>
<tr>
<td>
<code>method</code></br>
<em>
string
</em>
</td>
<td>
<p>Vault API method to use (GET/POST/other)</p>
</td>
</tr>
<tr>
<td>
<code>parameters</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<p>Parameters to pass to Vault write (for non-GET methods)</p>
</td>
</tr>
<tr>
<td>
<code>resultType</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecretResultType">
VaultDynamicSecretResultType
</a>
</em>
</td>
<td>
<p>Result type defines which data is returned from the generator.
By default, it is the &ldquo;data&rdquo; section of the Vault API response.
When using e.g. /auth/token/create the &ldquo;data&rdquo; section is empty but
the &ldquo;auth&rdquo; section contains the generated token.
Please refer to the vault docs regarding the result data structure.
Additionally, accessing the raw response is possibly by using &ldquo;Raw&rdquo; result type.</p>
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
<code>provider</code></br>
<em>
<a href="#external-secrets.io/v1.VaultProvider">
VaultProvider
</a>
</em>
</td>
<td>
<p>Vault provider common spec</p>
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
<p>Vault path to obtain the dynamic secret from</p>
</td>
</tr>
<tr>
<td>
<code>allowEmptyResponse</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Do not fail if no secrets are found. Useful for requests where no data is expected.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.VaultDynamicSecretResultType">VaultDynamicSecretResultType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecretSpec">VaultDynamicSecretSpec</a>)
</p>
<p>
<p>VaultDynamicSecretResultType defines which part of the Vault API response should be returned.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Auth&#34;</p></td>
<td><p>VaultDynamicSecretResultTypeAuth specifies to return the &ldquo;auth&rdquo; section of the Vault API response.</p>
</td>
</tr><tr><td><p>&#34;Data&#34;</p></td>
<td><p>VaultDynamicSecretResultTypeData specifies to return the &ldquo;data&rdquo; section of the Vault API response.</p>
</td>
</tr><tr><td><p>&#34;Raw&#34;</p></td>
<td><p>VaultDynamicSecretResultTypeRaw specifies to return the raw response from the Vault API.</p>
</td>
</tr></tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.VaultDynamicSecretSpec">VaultDynamicSecretSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecret">VaultDynamicSecret</a>)
</p>
<p>
<p>VaultDynamicSecretSpec defines the desired spec of VaultDynamicSecret.</p>
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
The ESO controller is instantiated with a specific controller name and filters VDS based on this property</p>
</td>
</tr>
<tr>
<td>
<code>method</code></br>
<em>
string
</em>
</td>
<td>
<p>Vault API method to use (GET/POST/other)</p>
</td>
</tr>
<tr>
<td>
<code>parameters</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<p>Parameters to pass to Vault write (for non-GET methods)</p>
</td>
</tr>
<tr>
<td>
<code>resultType</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.VaultDynamicSecretResultType">
VaultDynamicSecretResultType
</a>
</em>
</td>
<td>
<p>Result type defines which data is returned from the generator.
By default, it is the &ldquo;data&rdquo; section of the Vault API response.
When using e.g. /auth/token/create the &ldquo;data&rdquo; section is empty but
the &ldquo;auth&rdquo; section contains the generated token.
Please refer to the vault docs regarding the result data structure.
Additionally, accessing the raw response is possibly by using &ldquo;Raw&rdquo; result type.</p>
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
<code>provider</code></br>
<em>
<a href="#external-secrets.io/v1.VaultProvider">
VaultProvider
</a>
</em>
</td>
<td>
<p>Vault provider common spec</p>
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
<p>Vault path to obtain the dynamic secret from</p>
</td>
</tr>
<tr>
<td>
<code>allowEmptyResponse</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Do not fail if no secrets are found. Useful for requests where no data is expected.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.Webhook">Webhook
</h3>
<p>
<p>Webhook connects to a third party API server to handle the secrets generation
configuration parameters in spec.
You can specify the server, the token, and additional body parameters.
See documentation for the full API specification for requests and responses.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookSpec">
WebhookSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
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
<a href="#generators.external-secrets.io/v1alpha1.AuthorizationProtocol">
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookResult">
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookSecret">
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookCAProvider">
WebhookCAProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for the CA bundle to use to validate webhook server certificate.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorStatus">
GeneratorStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.WebhookCAProvider">WebhookCAProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.WebhookSpec">WebhookSpec</a>)
</p>
<p>
<p>WebhookCAProvider defines a location to fetch the cert for the webhook provider from.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookCAProviderType">
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
<h3 id="generators.external-secrets.io/v1alpha1.WebhookCAProviderType">WebhookCAProviderType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.WebhookCAProvider">WebhookCAProvider</a>)
</p>
<p>
<p>WebhookCAProviderType defines the type of provider for webhook CA certificates.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ConfigMap&#34;</p></td>
<td><p>WebhookCAProviderTypeConfigMap indicates the CA provider is a ConfigMap resource.</p>
</td>
</tr><tr><td><p>&#34;Secret&#34;</p></td>
<td><p>WebhookCAProviderTypeSecret indicates the CA provider is a Secret resource.</p>
</td>
</tr></tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.WebhookResult">WebhookResult
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.WebhookSpec">WebhookSpec</a>)
</p>
<p>
<p>WebhookResult defines how to format and extract results from the webhook response.</p>
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
<h3 id="generators.external-secrets.io/v1alpha1.WebhookSecret">WebhookSecret
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.WebhookSpec">WebhookSpec</a>)
</p>
<p>
<p>WebhookSecret defines a secret reference that will be used in webhook templates.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.SecretKeySelector">
SecretKeySelector
</a>
</em>
</td>
<td>
<p>Secret ref to fill in credentials</p>
</td>
</tr>
</tbody>
</table>
<h3 id="generators.external-secrets.io/v1alpha1.WebhookSpec">WebhookSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">GeneratorSpec</a>, 
<a href="#generators.external-secrets.io/v1alpha1.Webhook">Webhook</a>)
</p>
<p>
<p>WebhookSpec controls the behavior of the external generator. Any body parameters should be passed to the server through the parameters field.</p>
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
<a href="#generators.external-secrets.io/v1alpha1.AuthorizationProtocol">
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookResult">
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookSecret">
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
<a href="#generators.external-secrets.io/v1alpha1.WebhookCAProvider">
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
<hr/>
<h2 id="scan.external-secrets.io/v1alpha1">scan.external-secrets.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains resources for scan</p>
</p>
Resource Types:
<ul></ul>
<h3 id="scan.external-secrets.io/v1alpha1.Consumer">Consumer
</h3>
<p>
<p>Consumer is the schema to store duplicate findings from a job</p>
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
<a href="#scan.external-secrets.io/v1alpha1.ConsumerSpec">
ConsumerSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.TargetReference">
TargetReference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<p>Type discriminates which payload below is populated.</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>A stable ID for correlation across scans.</p>
</td>
</tr>
<tr>
<td>
<code>displayName</code></br>
<em>
string
</em>
</td>
<td>
<p>Human readable name for UIs.</p>
</td>
</tr>
<tr>
<td>
<code>attributes</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerAttrs">
ConsumerAttrs
</a>
</em>
</td>
<td>
<p>Exactly one of the following should be set according to Type.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerStatus">
ConsumerStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.ConsumerAttrs">ConsumerAttrs
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerFinding">ConsumerFinding</a>, 
<a href="#scan.external-secrets.io/v1alpha1.ConsumerSpec">ConsumerSpec</a>)
</p>
<p>
<p>ConsumerAttrs defines the attributes of a Consumer.</p>
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
<code>vmProcess</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.VMProcessSpec">
VMProcessSpec
</a>
</em>
</td>
<td>
<p>VMProcess defines the attributes of a VM process.</p>
</td>
</tr>
<tr>
<td>
<code>gitHubActor</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.GitHubActorSpec">
GitHubActorSpec
</a>
</em>
</td>
<td>
<p>GitHubActor defines the attributes of a GitHub actor.</p>
</td>
</tr>
<tr>
<td>
<code>k8sWorkload</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.K8sWorkloadSpec">
K8sWorkloadSpec
</a>
</em>
</td>
<td>
<p>K8sWorkload defines the attributes of a Kubernetes workload.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.ConsumerConditionType">ConsumerConditionType
(<code>string</code> alias)</p></h3>
<p>
<p>ConsumerConditionType defines the type of a Consumer condition.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;UsingLatestVersion&#34;</p></td>
<td><p>ConsumerLatestVersion indicates that the consumer is using the latest version of the external-secrets-enterprise.</p>
</td>
</tr><tr><td><p>&#34;LocationsOutOfDate&#34;</p></td>
<td><p>ConsumerLocationsOutOfDate indicates that the consumer is using the latest version of the external-secrets-enterprise.</p>
</td>
</tr><tr><td><p>&#34;LocationsUpToDate&#34;</p></td>
<td><p>ConsumerLocationsUpToDate indicates that the consumer is using the latest version of the external-secrets-enterprise.</p>
</td>
</tr><tr><td><p>&#34;ConsumerNotReady&#34;</p></td>
<td><p>ConsumerNotReady indicates that the consumer is using the latest version of the external-secrets-enterprise.</p>
</td>
</tr><tr><td><p>&#34;WorkloadNotReady&#34;</p></td>
<td><p>ConsumerWorkloadNotReady indicates that the consumer is using the latest version of the external-secrets-enterprise.</p>
</td>
</tr><tr><td><p>&#34;WorkloadReady&#34;</p></td>
<td><p>ConsumerWorkloadReady indicates that the consumer is using the latest version of the external-secrets-enterprise.</p>
</td>
</tr></tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.ConsumerFinding">ConsumerFinding
</h3>
<p>
<p>ConsumerFinding defines a finding from a consumer.</p>
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
<code>observedIndex</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.SecretUpdateRecord">
SecretUpdateRecord
</a>
</em>
</td>
<td>
<p>ObservedIndex is a map of secret names to SecretUpdateRecord.</p>
</td>
</tr>
<tr>
<td>
<code>location</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.SecretInStoreRef">
SecretInStoreRef
</a>
</em>
</td>
<td>
<p>Location is a SecretInStoreRef.</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<p>Type is the type of the finding.</p>
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
<p>ID is the external ID of the finding.</p>
</td>
</tr>
<tr>
<td>
<code>displayName</code></br>
<em>
string
</em>
</td>
<td>
<p>DisplayName is the display name of the finding.</p>
</td>
</tr>
<tr>
<td>
<code>attributes</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerAttrs">
ConsumerAttrs
</a>
</em>
</td>
<td>
<p>Attributes are the attributes of the finding.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.ConsumerSpec">ConsumerSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.Consumer">Consumer</a>)
</p>
<p>
<p>ConsumerSpec defines the desired state of Consumer.</p>
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
<code>target</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.TargetReference">
TargetReference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<p>Type discriminates which payload below is populated.</p>
</td>
</tr>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>A stable ID for correlation across scans.</p>
</td>
</tr>
<tr>
<td>
<code>displayName</code></br>
<em>
string
</em>
</td>
<td>
<p>Human readable name for UIs.</p>
</td>
</tr>
<tr>
<td>
<code>attributes</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerAttrs">
ConsumerAttrs
</a>
</em>
</td>
<td>
<p>Exactly one of the following should be set according to Type.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.ConsumerStatus">ConsumerStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.Consumer">Consumer</a>)
</p>
<p>
<p>ConsumerStatus defines the observed state of Consumer.</p>
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
<code>observedIndex</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.SecretUpdateRecord">
map[string]github.com/external-secrets/external-secrets/apis/scan/v1alpha1.SecretUpdateRecord
</a>
</em>
</td>
<td>
<p>ObservedIndex is a map of secret names to SecretUpdateRecord.</p>
</td>
</tr>
<tr>
<td>
<code>locations</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.SecretInStoreRef">
[]SecretInStoreRef
</a>
</em>
</td>
<td>
<p>Locations is a list of SecretInStoreRef.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<p>Conditions is a list of metav1.Condition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.Finding">Finding
</h3>
<p>
<p>Finding is the schema to store duplicate findings from a job</p>
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
<a href="#scan.external-secrets.io/v1alpha1.FindingSpec">
FindingSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the external ID of the finding.</p>
</td>
</tr>
<tr>
<td>
<code>displayName</code></br>
<em>
string
</em>
</td>
<td>
<p>DisplayName is the display name of the finding.</p>
</td>
</tr>
<tr>
<td>
<code>hash</code></br>
<em>
string
</em>
</td>
<td>
<p>Hash is the hash of the finding (salted).</p>
</td>
</tr>
<tr>
<td>
<code>runTemplateRef</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.RunTemplateReference">
RunTemplateReference
</a>
</em>
</td>
<td>
<p>RunTemplateRef is a reference to the run template.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.FindingStatus">
FindingStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.FindingSpec">FindingSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.Finding">Finding</a>)
</p>
<p>
<p>FindingSpec defines the desired state of Finding.</p>
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
<code>id</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is the external ID of the finding.</p>
</td>
</tr>
<tr>
<td>
<code>displayName</code></br>
<em>
string
</em>
</td>
<td>
<p>DisplayName is the display name of the finding.</p>
</td>
</tr>
<tr>
<td>
<code>hash</code></br>
<em>
string
</em>
</td>
<td>
<p>Hash is the hash of the finding (salted).</p>
</td>
</tr>
<tr>
<td>
<code>runTemplateRef</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.RunTemplateReference">
RunTemplateReference
</a>
</em>
</td>
<td>
<p>RunTemplateRef is a reference to the run template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.FindingStatus">FindingStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.Finding">Finding</a>)
</p>
<p>
<p>FindingStatus defines the observed state of Finding.</p>
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
<code>locations</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.SecretInStoreRef">
[]SecretInStoreRef
</a>
</em>
</td>
<td>
<p>Locations is a list of SecretInStoreRef.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.GitHubActorSpec">GitHubActorSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerAttrs">ConsumerAttrs</a>)
</p>
<p>
<p>GitHubActorSpec describes who/what is interacting with a repo.</p>
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
<code>repository</code></br>
<em>
string
</em>
</td>
<td>
<p>Repo slug &ldquo;owner/name&rdquo; for context (e.g., &ldquo;acme/api&rdquo;).</p>
</td>
</tr>
<tr>
<td>
<code>actorType</code></br>
<em>
string
</em>
</td>
<td>
<p>ActorType: &ldquo;User&rdquo; | &ldquo;App&rdquo; | &ldquo;Bot&rdquo; (GitHub notions)</p>
</td>
</tr>
<tr>
<td>
<code>actorLogin</code></br>
<em>
string
</em>
</td>
<td>
<p>ActorLogin is the login of the actor.</p>
</td>
</tr>
<tr>
<td>
<code>actorID</code></br>
<em>
string
</em>
</td>
<td>
<p>ActorID is the stable numeric id of the actor if known.</p>
</td>
</tr>
<tr>
<td>
<code>event</code></br>
<em>
string
</em>
</td>
<td>
<p>Optional context that led to detection (push/clone/workflow).</p>
</td>
</tr>
<tr>
<td>
<code>workflowRunID</code></br>
<em>
string
</em>
</td>
<td>
<p>Optional: workflow/job id when usage came from Actions</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.Job">Job
</h3>
<p>
<p>Job is the schema to run a scan job over targets</p>
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
<a href="#scan.external-secrets.io/v1alpha1.JobSpec">
JobSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>constraints</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.JobConstraints">
JobConstraints
</a>
</em>
</td>
<td>
<p>Constrains this job to a given set of SecretStores / Targets.
By default it will run against all SecretStores / Targets on the Job namespace.</p>
</td>
</tr>
<tr>
<td>
<code>runPolicy</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.JobRunPolicy">
JobRunPolicy
</a>
</em>
</td>
<td>
<p>Defines the RunPolicy for this job (Poll/OnChange/Once)</p>
</td>
</tr>
<tr>
<td>
<code>interval</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>Defines the interval for this job if Policy is Poll(Poll/OnChange/Once)</p>
</td>
</tr>
<tr>
<td>
<code>jobTimeout</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>TODO - also implement Cron Schedulingf
Define the interval to wait before forcing reconcile if job froze at running state</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.JobStatus">
JobStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.JobConstraints">JobConstraints
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.JobSpec">JobSpec</a>)
</p>
<p>
<p>JobConstraints defines the constraints for a job.</p>
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
<code>secretStoreConstraints</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.SecretStoreConstraint">
[]SecretStoreConstraint
</a>
</em>
</td>
<td>
<p>SecretStoreConstraints defines the constraints for a job.</p>
</td>
</tr>
<tr>
<td>
<code>targetConstraints</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.TargetConstraint">
[]TargetConstraint
</a>
</em>
</td>
<td>
<p>TargetConstraints defines the constraints for a job.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.JobRunPolicy">JobRunPolicy
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.JobSpec">JobSpec</a>)
</p>
<p>
<p>JobRunPolicy defines the run policy for a job.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;OnChange&#34;</p></td>
<td><p>JobRunPolicyOnChange defines the run policy for a job.</p>
</td>
</tr><tr><td><p>&#34;Once&#34;</p></td>
<td><p>JobRunPolicyOnce defines the run policy for a job.</p>
</td>
</tr><tr><td><p>&#34;Poll&#34;</p></td>
<td><p>JobRunPolicyPull defines the run policy for a job.</p>
</td>
</tr></tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.JobRunStatus">JobRunStatus
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.JobStatus">JobStatus</a>)
</p>
<p>
<p>JobRunStatus defines the status of a job.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Failed&#34;</p></td>
<td><p>JobRunStatusFailed defines the status of a job.</p>
</td>
</tr><tr><td><p>&#34;Running&#34;</p></td>
<td><p>JobRunStatusRunning defines the status of a job.</p>
</td>
</tr><tr><td><p>&#34;Succeeded&#34;</p></td>
<td><p>JobRunStatusSucceeded defines the status of a job.</p>
</td>
</tr></tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.JobSpec">JobSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.Job">Job</a>)
</p>
<p>
<p>JobSpec defines the desired state of Job.</p>
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
<code>constraints</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.JobConstraints">
JobConstraints
</a>
</em>
</td>
<td>
<p>Constrains this job to a given set of SecretStores / Targets.
By default it will run against all SecretStores / Targets on the Job namespace.</p>
</td>
</tr>
<tr>
<td>
<code>runPolicy</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.JobRunPolicy">
JobRunPolicy
</a>
</em>
</td>
<td>
<p>Defines the RunPolicy for this job (Poll/OnChange/Once)</p>
</td>
</tr>
<tr>
<td>
<code>interval</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>Defines the interval for this job if Policy is Poll(Poll/OnChange/Once)</p>
</td>
</tr>
<tr>
<td>
<code>jobTimeout</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>TODO - also implement Cron Schedulingf
Define the interval to wait before forcing reconcile if job froze at running state</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.JobStatus">JobStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.Job">Job</a>)
</p>
<p>
<p>JobStatus defines the observed state of a Job.</p>
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
<code>observedSecretStoresDigest</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObservedSecretStoresDigest is a digest of the SecretStores that were used in the last run.</p>
</td>
</tr>
<tr>
<td>
<code>observedTargetsDigest</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObservedTargetsDigest is a digest of the Targets that were used in the last run.</p>
</td>
</tr>
<tr>
<td>
<code>lastRunTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastRunTime is the time when the job was last run.</p>
</td>
</tr>
<tr>
<td>
<code>runStatus</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.JobRunStatus">
JobRunStatus
</a>
</em>
</td>
<td>
<p>RunStatus is the status of the job.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<p>Conditions is a list of metav1.Condition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.K8sWorkloadSpec">K8sWorkloadSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerAttrs">ConsumerAttrs</a>)
</p>
<p>
<p>K8sWorkloadSpec describes the workload that is interacting with a kubernetes target.</p>
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
<code>clusterName</code></br>
<em>
string
</em>
</td>
<td>
<p>ClusterName is the name of the cluster.</p>
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
<p>Namespace is the namespace of the workload.</p>
</td>
</tr>
<tr>
<td>
<code>workloadKind</code></br>
<em>
string
</em>
</td>
<td>
<p>Workload identity (top controller or naked Pod as fallback)
e.g., Kind=&ldquo;Deployment&rdquo;, Group=&ldquo;apps&rdquo;, Version=&ldquo;v1&rdquo;, Name=&ldquo;api&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>workloadGroup</code></br>
<em>
string
</em>
</td>
<td>
<p>WorkloadGroup is the group of the workload.</p>
</td>
</tr>
<tr>
<td>
<code>workloadVersion</code></br>
<em>
string
</em>
</td>
<td>
<p>WorkloadVersion is the version of the workload.</p>
</td>
</tr>
<tr>
<td>
<code>workloadName</code></br>
<em>
string
</em>
</td>
<td>
<p>WorkloadName is the name of the workload.</p>
</td>
</tr>
<tr>
<td>
<code>workloadUID</code></br>
<em>
string
</em>
</td>
<td>
<p>WorkloadUID is the UID of the workload.</p>
</td>
</tr>
<tr>
<td>
<code>controller</code></br>
<em>
string
</em>
</td>
<td>
<p>Convenience string for UIs: &ldquo;deployment.apps/api&rdquo;</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.RemoteRef">RemoteRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.SecretInStoreRef">SecretInStoreRef</a>)
</p>
<p>
<p>RemoteRef defines a reference to a remote secret.</p>
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
<code>property</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>startIndex</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>endIndex</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.RunTemplateReference">RunTemplateReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.FindingSpec">FindingSpec</a>)
</p>
<p>
<p>RunTemplateReference defines a reference to a run template.</p>
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
<p>Name is the name of the run template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.SecretInStoreRef">SecretInStoreRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerFinding">ConsumerFinding</a>, 
<a href="#scan.external-secrets.io/v1alpha1.ConsumerStatus">ConsumerStatus</a>, 
<a href="#scan.external-secrets.io/v1alpha1.FindingStatus">FindingStatus</a>)
</p>
<p>
<p>SecretInStoreRef defines a reference to a secret in a secret store.</p>
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
<code>remoteRef</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.RemoteRef">
RemoteRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.SecretStoreConstraint">SecretStoreConstraint
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.JobConstraints">JobConstraints</a>)
</p>
<p>
<p>SecretStoreConstraint defines the constraints for a job.</p>
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
<code>matchExpression</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
[]Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>MatchExpressions defines the constraints for a job.</p>
</td>
</tr>
<tr>
<td>
<code>matchLabels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>MatchLabels defines the constraints for a job.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.SecretUpdateRecord">SecretUpdateRecord
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerFinding">ConsumerFinding</a>, 
<a href="#scan.external-secrets.io/v1alpha1.ConsumerStatus">ConsumerStatus</a>)
</p>
<p>
<p>SecretUpdateRecord defines the timestamp when a PushSecret was applied to a secret.</p>
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
<code>timestamp</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>secretHash</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.TargetConstraint">TargetConstraint
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.JobConstraints">JobConstraints</a>)
</p>
<p>
<p>TargetConstraint defines the constraints for a job.</p>
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
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<p>Kind defines the kind of the target.</p>
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
<p>APIVersion defines the API version of the target.</p>
</td>
</tr>
<tr>
<td>
<code>matchExpression</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
[]Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>MatchExpressions defines the constraints for a job.</p>
</td>
</tr>
<tr>
<td>
<code>matchLabels</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>MatchLabels defines the constraints for a job.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.TargetReference">TargetReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerSpec">ConsumerSpec</a>)
</p>
<p>
<p>TargetReference defines a reference to a target.</p>
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
<p>Name is the name of the target.</p>
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
<p>Namespace is the namespace of the target.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scan.external-secrets.io/v1alpha1.VMProcessSpec">VMProcessSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scan.external-secrets.io/v1alpha1.ConsumerAttrs">ConsumerAttrs</a>)
</p>
<p>
<p>VMProcessSpec describes a process on a VM/host.</p>
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
<code>ruid</code></br>
<em>
int64
</em>
</td>
<td>
<p>RUID is the real user ID.</p>
</td>
</tr>
<tr>
<td>
<code>euid</code></br>
<em>
int64
</em>
</td>
<td>
<p>EUID is the effective user ID.</p>
</td>
</tr>
<tr>
<td>
<code>executable</code></br>
<em>
string
</em>
</td>
<td>
<p>Executable is the path to the executable.</p>
</td>
</tr>
<tr>
<td>
<code>cmdline</code></br>
<em>
string
</em>
</td>
<td>
<p>Cmdline is the command line arguments.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<h2 id="target.external-secrets.io/v1alpha1">target.external-secrets.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains resources for targets</p>
</p>
Resource Types:
<ul></ul>
<h3 id="target.external-secrets.io/v1alpha1.CertAuth">CertAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesAuth">KubernetesAuth</a>)
</p>
<p>
<p>CertAuth contains the client certificate and key for certificate authentication.</p>
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
<h3 id="target.external-secrets.io/v1alpha1.GenericTarget">GenericTarget
</h3>
<p>
<p>GenericTarget is a common interface for interacting with Targets.</p>
</p>
<h3 id="target.external-secrets.io/v1alpha1.GithubAppAuth">GithubAppAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.GithubTargetAuth">GithubTargetAuth</a>)
</p>
<p>
<p>GithubAppAuth contains the Github App authentication spec.</p>
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
<code>appID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>installID</code></br>
<em>
string
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
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.GithubRepository">GithubRepository
</h3>
<p>
<p>GithubRepository is the schema for a GitHub target.</p>
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
<a href="#target.external-secrets.io/v1alpha1.GithubRepositorySpec">
GithubRepositorySpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>owner</code></br>
<em>
string
</em>
</td>
<td>
<p>Owner of the repository (user or organization).</p>
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
<p>Repository name.</p>
</td>
</tr>
<tr>
<td>
<code>branch</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Branch to target (optional, defaults to default branch).</p>
</td>
</tr>
<tr>
<td>
<code>enterpriseUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>GitHub Enterprise endpoint. Format should be http(s)://[hostname]/api/v3/
or it will always return the 406 status code
If empty, default GitHub client will be configured</p>
</td>
</tr>
<tr>
<td>
<code>uploadUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>GitHub Enterprise upload endpoint. The upload URL format should be http(s)://[hostname]/api/uploads/
or it will always return the 406 status code
If empty, default GitHub client will be configured</p>
</td>
</tr>
<tr>
<td>
<code>paths</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Paths to scan or push secrets to (relative to repo root).</p>
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
<p>CABundle is an optional PEM encoded CA bundle for HTTPS verification (for GitHub Enterprise).</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.GithubTargetAuth">
GithubTargetAuth
</a>
</em>
</td>
<td>
<p>Auth method to access the repository.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.TargetStatus">
TargetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.GithubRepositorySpec">GithubRepositorySpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.GithubRepository">GithubRepository</a>)
</p>
<p>
<p>GithubRepositorySpec contains the GithubRepository spec.</p>
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
<code>owner</code></br>
<em>
string
</em>
</td>
<td>
<p>Owner of the repository (user or organization).</p>
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
<p>Repository name.</p>
</td>
</tr>
<tr>
<td>
<code>branch</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Branch to target (optional, defaults to default branch).</p>
</td>
</tr>
<tr>
<td>
<code>enterpriseUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>GitHub Enterprise endpoint. Format should be http(s)://[hostname]/api/v3/
or it will always return the 406 status code
If empty, default GitHub client will be configured</p>
</td>
</tr>
<tr>
<td>
<code>uploadUrl</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>GitHub Enterprise upload endpoint. The upload URL format should be http(s)://[hostname]/api/uploads/
or it will always return the 406 status code
If empty, default GitHub client will be configured</p>
</td>
</tr>
<tr>
<td>
<code>paths</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Paths to scan or push secrets to (relative to repo root).</p>
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
<p>CABundle is an optional PEM encoded CA bundle for HTTPS verification (for GitHub Enterprise).</p>
</td>
</tr>
<tr>
<td>
<code>auth</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.GithubTargetAuth">
GithubTargetAuth
</a>
</em>
</td>
<td>
<p>Auth method to access the repository.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.GithubTargetAuth">GithubTargetAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.GithubRepositorySpec">GithubRepositorySpec</a>)
</p>
<p>
<p>GithubTargetAuth contains the Github target auth spec.</p>
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
<code>token</code></br>
<em>
<a href="https://pkg.go.dev/github.com/external-secrets/external-secrets/apis/meta/v1#SecretKeySelector">
External Secrets meta/v1.SecretKeySelector
</a>
</em>
</td>
<td>
<p>Use a personal access token.</p>
</td>
</tr>
<tr>
<td>
<code>appAuth</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.GithubAppAuth">
GithubAppAuth
</a>
</em>
</td>
<td>
<p>GitHub App authentication (JWT).</p>
</td>
</tr>
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.KubernetesAuth">KubernetesAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesClusterSpec">KubernetesClusterSpec</a>)
</p>
<p>
<p>KubernetesAuth contains the Kubernetes auth spec.</p>
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
<a href="#target.external-secrets.io/v1alpha1.CertAuth">
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
<a href="#target.external-secrets.io/v1alpha1.TokenAuth">
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
<h3 id="target.external-secrets.io/v1alpha1.KubernetesCluster">KubernetesCluster
</h3>
<p>
<p>KubernetesCluster is the schema for a Kubernetes target.</p>
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
<a href="#target.external-secrets.io/v1alpha1.KubernetesClusterSpec">
KubernetesClusterSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>server</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesServer">
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
<a href="#target.external-secrets.io/v1alpha1.KubernetesAuth">
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
<code>namespaces</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.NamespacesMatcher">
NamespacesMatcher
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>namespaces controls which namespaces are in scope during scans.
If both include and exclude are empty, all namespaces are included.
Include/exclude support glob-like patterns (implementation detail in provider).</p>
</td>
</tr>
<tr>
<td>
<code>selector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>selector filters workloads/pods by labels before scan evaluation.
If empty, all labeled/unlabeled workloads are considered.</p>
</td>
</tr>
<tr>
<td>
<code>scan</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesScanOptions">
KubernetesScanOptions
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>scan toggles specific binding paths to consider as &ldquo;manifest bindings&rdquo;.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.TargetStatus">
TargetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.KubernetesClusterSpec">KubernetesClusterSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesCluster">KubernetesCluster</a>)
</p>
<p>
<p>KubernetesClusterSpec contains the KubernetesCluster spec.</p>
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
<a href="#target.external-secrets.io/v1alpha1.KubernetesServer">
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
<a href="#target.external-secrets.io/v1alpha1.KubernetesAuth">
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
<code>namespaces</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.NamespacesMatcher">
NamespacesMatcher
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>namespaces controls which namespaces are in scope during scans.
If both include and exclude are empty, all namespaces are included.
Include/exclude support glob-like patterns (implementation detail in provider).</p>
</td>
</tr>
<tr>
<td>
<code>selector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>selector filters workloads/pods by labels before scan evaluation.
If empty, all labeled/unlabeled workloads are considered.</p>
</td>
</tr>
<tr>
<td>
<code>scan</code></br>
<em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesScanOptions">
KubernetesScanOptions
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>scan toggles specific binding paths to consider as &ldquo;manifest bindings&rdquo;.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.KubernetesScanOptions">KubernetesScanOptions
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesClusterSpec">KubernetesClusterSpec</a>)
</p>
<p>
<p>KubernetesScanOptions selects which manifest bindings are recognized.
All fields default to true except IncludeImagePullSecrets (default false).</p>
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
<code>includeImagePullSecrets</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>IncludeImagePullSecrets: consider spec.imagePullSecrets and SA.imagePullSecrets.</p>
</td>
</tr>
<tr>
<td>
<code>includeEnvFrom</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>IncludeEnvFrom: consider containers[*].envFrom[].secretRef and initContainers[] equivalents.</p>
</td>
</tr>
<tr>
<td>
<code>includeEnvSecretKeyRefs</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>IncludeEnvSecretKeyRefs: consider containers[*].env[].valueFrom.secretKeyRef (and initContainers).</p>
</td>
</tr>
<tr>
<td>
<code>includeVolumeSecrets</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>IncludeVolumeSecrets: consider volumes[].secret.secretName.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.KubernetesServer">KubernetesServer
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesClusterSpec">KubernetesClusterSpec</a>)
</p>
<p>
<p>KubernetesServer contains the Kubernetes server spec.</p>
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
<h3 id="target.external-secrets.io/v1alpha1.NamespacesMatcher">NamespacesMatcher
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesClusterSpec">KubernetesClusterSpec</a>)
</p>
<p>
<p>NamespacesMatcher selects namespaces by include/exclude pattern lists.
Empty/omitted fields mean &ldquo;no constraint&rdquo;.</p>
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
<code>include</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>exclude</code></br>
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
<h3 id="target.external-secrets.io/v1alpha1.ScanTarget">ScanTarget
</h3>
<p>
<p>ScanTarget is an interface for scanning a Target.</p>
</p>
<h3 id="target.external-secrets.io/v1alpha1.TargetCapabilities">TargetCapabilities
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.TargetStatus">TargetStatus</a>)
</p>
<p>
<p>TargetCapabilities defines the possible operations a Target can do.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ReadOnly&#34;</p></td>
<td><p>TargetReadOnly defines a Target that can only read secrets.</p>
</td>
</tr><tr><td><p>&#34;ReadWrite&#34;</p></td>
<td><p>TargetReadWrite defines a Target that can read and write secrets.</p>
</td>
</tr><tr><td><p>&#34;WriteOnly&#34;</p></td>
<td><p>TargetWriteOnly defines a Target that can only write secrets.</p>
</td>
</tr></tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.TargetConditionType">TargetConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.TargetStatusCondition">TargetStatusCondition</a>)
</p>
<p>
<p>TargetConditionType defines the possible conditions a Target can have.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>TargetReady defines a Target that is ready to be used.</p>
</td>
</tr></tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.TargetProvider">TargetProvider
</h3>
<p>
<p>TargetProvider is an interface for creating a ScanTarget.</p>
</p>
<h3 id="target.external-secrets.io/v1alpha1.TargetStatus">TargetStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.GithubRepository">GithubRepository</a>, 
<a href="#target.external-secrets.io/v1alpha1.KubernetesCluster">KubernetesCluster</a>)
</p>
<p>
<p>TargetStatus defines the observed state of the Target.</p>
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
<a href="#target.external-secrets.io/v1alpha1.TargetStatusCondition">
[]TargetStatusCondition
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
<a href="#target.external-secrets.io/v1alpha1.TargetCapabilities">
TargetCapabilities
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>pushIndex</code></br>
<em>
<a href="#scan.external-secrets.io/v1alpha1.[]github.com/external-secrets/external-secrets/apis/scan/v1alpha1.SecretUpdateRecord">
map[string][]github.com/external-secrets/external-secrets/apis/scan/v1alpha1.SecretUpdateRecord
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="target.external-secrets.io/v1alpha1.TargetStatusCondition">TargetStatusCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.TargetStatus">TargetStatus</a>)
</p>
<p>
<p>TargetStatusCondition defines the status of a Target.</p>
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
<a href="#target.external-secrets.io/v1alpha1.TargetConditionType">
TargetConditionType
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
<h3 id="target.external-secrets.io/v1alpha1.TokenAuth">TokenAuth
</h3>
<p>
(<em>Appears on:</em>
<a href="#target.external-secrets.io/v1alpha1.KubernetesAuth">KubernetesAuth</a>)
</p>
<p>
<p>TokenAuth contains the bearer token for token authentication.</p>
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
<hr/>
<h2 id="workflows.external-secrets.io/v1alpha1">workflows.external-secrets.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains resources for workflows</p>
</p>
Resource Types:
<ul></ul>
<h3 id="workflows.external-secrets.io/v1alpha1.ConverterFunc">ConverterFunc
</h3>
<p>
<p>ConverterFunc is a function that converts a value to a specific type.</p>
</p>
<h3 id="workflows.external-secrets.io/v1alpha1.DebugStep">DebugStep
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">Step</a>)
</p>
<p>
<p>DebugStep defines a step that outputs debug information.</p>
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
<code>message</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.DestinationRef">DestinationRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.PushStep">PushStep</a>)
</p>
<p>
<p>DestinationRef allows you to override the SecretStore destination
where the secret will be pushed to.</p>
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
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.FindingParameterType">FindingParameterType
</h3>
<p>
<p>FindingParameterType defines a parameter to be passed to a finding type.</p>
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
<p>Name is the name of the secretstore.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.GeneratorParameterType">GeneratorParameterType
</h3>
<p>
<p>GeneratorParameterType defines a parameter to be passed to a generator type.</p>
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
<p>Name is the name of the generator.</p>
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
<p>Kind defines the kind of the generator. It can be &lsquo;any&rsquo;</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.GeneratorStep">GeneratorStep
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">Step</a>)
</p>
<p>
<p>GeneratorStep defines a step that generates secrets using a configured generator.</p>
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
<tr>
<td>
<code>kind</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorKind">
GeneratorKind
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Kind specifies the kind of generator to use when using inline generator configuration.
Required when using inline generator configuration.</p>
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#generators.external-secrets.io/v1alpha1.GeneratorSpec">
GeneratorSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Generator contains the inline generator configuration.
Required when using inline generator configuration.</p>
<br/>
<br/>
<table>
</table>
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
<p>Rewrite contains rules for rewriting the generated keys.</p>
</td>
</tr>
<tr>
<td>
<code>autoCleanup</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>AutoCleanup indicates whether to delete the old generated secrets at when creating a new one.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.JavaScriptStep">JavaScriptStep
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">Step</a>)
</p>
<p>
<p>JavaScriptStep defines a step that executes JavaScript code with access to step input data.</p>
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
<code>script</code></br>
<em>
string
</em>
</td>
<td>
<p>Script contains the JavaScript code to execute</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.Job">Job
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowSpec">WorkflowSpec</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowTemplateSpec">WorkflowTemplateSpec</a>)
</p>
<p>
<p>Job defines a unit of work in a workflow.</p>
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
<code>dependsOn</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>variables</code></br>
<em>
map[string]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>standard</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.StandardJob">
StandardJob
</a>
</em>
</td>
<td>
<p>Standard job configuration</p>
</td>
</tr>
<tr>
<td>
<code>loop</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.LoopJob">
LoopJob
</a>
</em>
</td>
<td>
<p>Loop job configuration</p>
</td>
</tr>
<tr>
<td>
<code>switch</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.SwitchJob">
SwitchJob
</a>
</em>
</td>
<td>
<p>Switch job configuration</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.JobPhase">JobPhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.JobStatus">JobStatus</a>)
</p>
<p>
<p>JobPhase types for job state machine.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Failed&#34;</p></td>
<td><p>JobPhaseFailed indicates the job has failed.</p>
</td>
</tr><tr><td><p>&#34;Pending&#34;</p></td>
<td><p>JobPhasePending indicates the job is pending execution.</p>
</td>
</tr><tr><td><p>&#34;Running&#34;</p></td>
<td><p>JobPhaseRunning indicates the job is running.</p>
</td>
</tr><tr><td><p>&#34;Succeeded&#34;</p></td>
<td><p>JobPhaseSucceeded indicates the job has succeeded.</p>
</td>
</tr></tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.JobStatus">JobStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowStatus">WorkflowStatus</a>)
</p>
<p>
<p>JobStatus defines the observed state of a Job.</p>
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
<code>phase</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.JobPhase">
JobPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>stepStatuses</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.StepStatus">
map[string]github.com/external-secrets/external-secrets/apis/workflows/v1alpha1.StepStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>startTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>completionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>executionTimeNanos</code></br>
<em>
int64
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.LoopJob">LoopJob
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Job">Job</a>)
</p>
<p>
<p>LoopJob defines a job that executes its steps in a loop.</p>
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
<code>concurrency</code></br>
<em>
int
</em>
</td>
<td>
<p>Concurrency specifies how many iterations can run in parallel</p>
</td>
</tr>
<tr>
<td>
<code>range</code></br>
<em>
string
</em>
</td>
<td>
<p>Range is a template string that resolves to a map of strings
The steps will be executed for each key-value pair in the map</p>
</td>
</tr>
<tr>
<td>
<code>steps</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">
[]Step
</a>
</em>
</td>
<td>
<p>Steps contains the list of steps to execute in each iteration</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.NamedWorkflowRunStatus">NamedWorkflowRunStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateStatus">WorkflowRunTemplateStatus</a>)
</p>
<p>
<p>NamedWorkflowRunStatus represents a named workflow run status.</p>
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
<code>runName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunStatus">
WorkflowRunStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.OutputDefinition">OutputDefinition
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">Step</a>)
</p>
<p>
<p>OutputDefinition defines an output variable from a workflow step
This allows workflow authors to explicitly define what outputs a step provides,
including the name, type, and sensitivity of each output.</p>
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
<p>Name is the name of the output variable
This is the key that will be used to access the output in templates</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.OutputType">
OutputType
</a>
</em>
</td>
<td>
<p>Type is the data type of the output variable
Supported types are: bool, number, time, and map</p>
</td>
</tr>
<tr>
<td>
<code>sensitive</code></br>
<em>
bool
</em>
</td>
<td>
<p>Sensitive indicates whether the output should be masked in the workflow status
If true, the output value will be replaced with asterisks (********)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.OutputType">OutputType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.OutputDefinition">OutputDefinition</a>)
</p>
<p>
<p>OutputType defines the type of an output variable</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;bool&#34;</p></td>
<td><p>OutputTypeBool represents a boolean output type
This is used for true/false values.</p>
</td>
</tr><tr><td><p>&#34;map&#34;</p></td>
<td><p>OutputTypeMap represents a map output type
This is used for complex data structures like JSON objects.</p>
</td>
</tr><tr><td><p>&#34;number&#34;</p></td>
<td><p>OutputTypeNumber represents a float64 output type
This is used for numeric values.</p>
</td>
</tr><tr><td><p>&#34;string&#34;</p></td>
<td><p>OutputTypeString represents a string output type
This is used for text values.</p>
</td>
</tr><tr><td><p>&#34;time&#34;</p></td>
<td><p>OutputTypeTime represents a time.Time output type
This is used for timestamp values.</p>
</td>
</tr></tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.Parameter">Parameter
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.ParameterGroup">ParameterGroup</a>)
</p>
<p>
<p>Parameter defines a parameter that can be overridden when creating a workflow.</p>
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
<p>Name of the parameter</p>
</td>
</tr>
<tr>
<td>
<code>description</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Description is a human-readable description of the parameter</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.ParameterType">
ParameterType
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Type specifies the data type of the parameter
For array/multi-value parameters, use allowMultiple: true with the appropriate type</p>
</td>
</tr>
<tr>
<td>
<code>required</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Required indicates whether the parameter must be provided</p>
</td>
</tr>
<tr>
<td>
<code>default</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Default value to use if not provided</p>
</td>
</tr>
<tr>
<td>
<code>allowMultiple</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>AllowMultiple indicates if multiple values can be selected
When true, the parameter accepts an array of values of the specified type</p>
</td>
</tr>
<tr>
<td>
<code>resourceConstraints</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.ResourceConstraints">
ResourceConstraints
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ResourceConstraints for Kubernetes resource types</p>
</td>
</tr>
<tr>
<td>
<code>validation</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.ParameterValidation">
ParameterValidation
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Validation constraints
MinItems and MaxItems apply when allowMultiple is true</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.ParameterGroup">ParameterGroup
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowTemplateSpec">WorkflowTemplateSpec</a>)
</p>
<p>
<p>ParameterGroup defines a group of parameters with a name and description.</p>
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
<p>Name of the parameter group</p>
</td>
</tr>
<tr>
<td>
<code>description</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Description is a human-readable description of the parameter group</p>
</td>
</tr>
<tr>
<td>
<code>parameters</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Parameter">
[]Parameter
</a>
</em>
</td>
<td>
<p>Parameters contained in this group</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.ParameterType">ParameterType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Parameter">Parameter</a>)
</p>
<p>
<p>ParameterType represents the data type of a parameter</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;bool&#34;</p></td>
<td><p>ParameterTypeBool represents a boolean parameter type.</p>
</td>
</tr><tr><td><p>&#34;clustersecretstore&#34;</p></td>
<td><p>ParameterTypeClusterSecretStore represents a cluster secret store parameter type.</p>
</td>
</tr><tr><td><p>&#34;^object\\[([a-zA-Z0-9_-]&#43;)\\](namespace|secretstore|externalsecret|clustersecretstore|secretlocation|finding|generator\\[[a-zA-Z0-9_-]&#43;\\]|array\\[(?:secretstore|secretlocation|finding|generator\\[[a-zA-Z0-9_-]&#43;\\])\\])$&#34;</p></td>
<td><p>ParameterTypeCustomObject represents a custom object parameter type.</p>
</td>
</tr><tr><td><p>&#34;externalsecret&#34;</p></td>
<td><p>ParameterTypeExternalSecret represents an external secret parameter type.</p>
</td>
</tr><tr><td><p>&#34;finding&#34;</p></td>
<td><p>ParameterTypeFinding represents a finding parameter type.</p>
</td>
</tr><tr><td><p>&#34;array[finding]&#34;</p></td>
<td><p>ParameterTypeFindingArray represents an array of findings.</p>
</td>
</tr><tr><td><p>&#34;^generator\\[([a-zA-Z0-9_-]&#43;)\\]$&#34;</p></td>
<td><p>ParameterTypeGenerator represents a generator parameter type.</p>
</td>
</tr><tr><td><p>&#34;^array\\[generator\\[([a-zA-Z0-9_-]&#43;)\\]\\]$&#34;</p></td>
<td><p>ParameterTypeGeneratorArray represents an array of generators.</p>
</td>
</tr><tr><td><p>&#34;namespace&#34;</p></td>
<td><p>ParameterTypeNamespace Kubernetes resource types.</p>
</td>
</tr><tr><td><p>&#34;number&#34;</p></td>
<td><p>ParameterTypeNumber represents a number parameter type.</p>
</td>
</tr><tr><td><p>&#34;object&#34;</p></td>
<td><p>ParameterTypeObject represents an object parameter type.</p>
</td>
</tr><tr><td><p>&#34;secret&#34;</p></td>
<td><p>ParameterTypeSecret represents a secret parameter type.</p>
</td>
</tr><tr><td><p>&#34;secretlocation&#34;</p></td>
<td><p>ParameterTypeSecretLocation represents a secret location parameter type.</p>
</td>
</tr><tr><td><p>&#34;array[secretlocation]&#34;</p></td>
<td><p>ParameterTypeSecretLocationArray represents an array of secret locations.</p>
</td>
</tr><tr><td><p>&#34;secretstore&#34;</p></td>
<td><p>ParameterTypeSecretStore represents a secret store parameter type.</p>
</td>
</tr><tr><td><p>&#34;array[secretstore]&#34;</p></td>
<td><p>ParameterTypeSecretStoreArray represents an array of secret stores.</p>
</td>
</tr><tr><td><p>&#34;string&#34;</p></td>
<td><p>ParameterTypeString represents a string parameter type.</p>
</td>
</tr><tr><td><p>&#34;time&#34;</p></td>
<td><p>ParameterTypeTime represents a time parameter type.</p>
</td>
</tr></tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.ParameterValidation">ParameterValidation
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Parameter">Parameter</a>)
</p>
<p>
<p>ParameterValidation defines validation rules for parameters.</p>
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
<code>minItems</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>MinItems minimum number of items for multi-select (only when AllowMultiple=true)</p>
</td>
</tr>
<tr>
<td>
<code>maxItems</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>MaxItems maximum number of items for multi-select (only when AllowMultiple=true)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.Phase">Phase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunStatus">WorkflowRunStatus</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowStatus">WorkflowStatus</a>)
</p>
<p>
<p>Phase types for workflow state machine.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Failed&#34;</p></td>
<td><p>PhaseFailed indicates the workflow has failed.</p>
</td>
</tr><tr><td><p>&#34;Pending&#34;</p></td>
<td><p>PhasePending indicates the workflow is pending execution.</p>
</td>
</tr><tr><td><p>&#34;Running&#34;</p></td>
<td><p>PhaseRunning indicates the workflow is running.</p>
</td>
</tr><tr><td><p>&#34;Succeeded&#34;</p></td>
<td><p>PhaseSucceeded indicates the workflow has succeeded.</p>
</td>
</tr></tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.PullStep">PullStep
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">Step</a>)
</p>
<p>
<p>PullStep defines a step that pulls secrets from a secret store.</p>
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
<a href="#external-secrets.io/v1.StoreSourceRef">
StoreSourceRef
</a>
</em>
</td>
<td>
<p>Source allows you to fetch secrets from a SecretStore.</p>
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
<p>Data allows you to fetch specific data from the secret.</p>
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
<p>DataFrom allows you to find multiple secrets in a store or extract structured data from a secret.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.PushStep">PushStep
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">Step</a>)
</p>
<p>
<p>PushStep defines a step that pushes secrets to a destination.</p>
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
<code>secretSource</code></br>
<em>
string
</em>
</td>
<td>
<p>SecretSource defines the source map in the workflow context,
indicating where to retrieve the secret values</p>
</td>
</tr>
<tr>
<td>
<code>destination</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.DestinationRef">
DestinationRef
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
<a href="#external-secrets.io/v1alpha1.PushSecretData">
[]PushSecretData
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.ResourceConstraints">ResourceConstraints
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Parameter">Parameter</a>)
</p>
<p>
<p>ResourceConstraints defines constraints for Kubernetes resource selection.</p>
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
<p>Namespace restricts resource selection to specific namespace(s)</p>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LabelSelector for filtering resources</p>
</td>
</tr>
<tr>
<td>
<code>allowCrossNamespace</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>AllowCrossNamespace indicates if resources from other namespaces can be selected
Only applies to cluster-scoped selections</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.RunPolicy">RunPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateSpec">WorkflowRunTemplateSpec</a>)
</p>
<p>
<p>RunPolicy defines the policy for running workflow runs.</p>
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
<code>once</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicyOnce">
RunPolicyOnce
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>scheduled</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicyScheduled">
RunPolicyScheduled
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>onChange</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicyOnChange">
RunPolicyOnChange
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.RunPolicyOnChange">RunPolicyOnChange
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicy">RunPolicy</a>)
</p>
<p>
<p>RunPolicyOnChange specifies that the workflow should run when changes are detected.</p>
</p>
<h3 id="workflows.external-secrets.io/v1alpha1.RunPolicyOnce">RunPolicyOnce
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicy">RunPolicy</a>)
</p>
<p>
<p>RunPolicyOnce specifies that the workflow should run only once.</p>
</p>
<h3 id="workflows.external-secrets.io/v1alpha1.RunPolicyScheduled">RunPolicyScheduled
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicy">RunPolicy</a>)
</p>
<p>
<p>RunPolicyScheduled defines a scheduled policy for running workflow runs.</p>
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
<code>every</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Duration">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cron</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.SecretLocationParameterType">SecretLocationParameterType
</h3>
<p>
<p>SecretLocationParameterType defines a parameter to be passed to a secret location type.</p>
</p>
<h3 id="workflows.external-secrets.io/v1alpha1.SecretStoreParameterType">SecretStoreParameterType
</h3>
<p>
<p>SecretStoreParameterType defines a parameter to be passed to a secret store type.</p>
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
<p>Name is the name of the secretstore.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.StandardJob">StandardJob
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Job">Job</a>)
</p>
<p>
<p>StandardJob is the default job type that executes a sequence of steps.</p>
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
<code>steps</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">
[]Step
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.Step">Step
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.LoopJob">LoopJob</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.StandardJob">StandardJob</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.SwitchCase">SwitchCase</a>)
</p>
<p>
<p>Step defines a single step in a workflow job.</p>
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
</td>
</tr>
<tr>
<td>
<code>pull</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.PullStep">
PullStep
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>push</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.PushStep">
PushStep
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>debug</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.DebugStep">
DebugStep
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>transform</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.TransformStep">
TransformStep
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>generator</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.GeneratorStep">
GeneratorStep
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>javascript</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.JavaScriptStep">
JavaScriptStep
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>outputs</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.OutputDefinition">
[]OutputDefinition
</a>
</em>
</td>
<td>
<p>Outputs defines the expected outputs from this step
Only values explicitly defined here will be saved in the step outputs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.StepPhase">StepPhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.StepStatus">StepStatus</a>)
</p>
<p>
<p>StepPhase types for step state machine.</p>
</p>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Failed&#34;</p></td>
<td><p>StepPhaseFailed indicates the step has failed.</p>
</td>
</tr><tr><td><p>&#34;Pending&#34;</p></td>
<td><p>StepPhasePending indicates the step is pending execution.</p>
</td>
</tr><tr><td><p>&#34;Running&#34;</p></td>
<td><p>StepPhaseRunning indicates the step is running.</p>
</td>
</tr><tr><td><p>&#34;Succeeded&#34;</p></td>
<td><p>StepPhaseSucceeded indicates the step has succeeded.</p>
</td>
</tr></tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.StepStatus">StepStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.JobStatus">JobStatus</a>)
</p>
<p>
<p>StepStatus defines the observed state of a Step.</p>
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
<code>phase</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.StepPhase">
StepPhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>outputs</code></br>
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
<code>message</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>startTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>completionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>executionTimeNanos</code></br>
<em>
int64
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.SwitchCase">SwitchCase
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.SwitchJob">SwitchJob</a>)
</p>
<p>
<p>SwitchCase defines a case of steps with a condition.</p>
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
<code>condition</code></br>
<em>
string
</em>
</td>
<td>
<p>Condition is a template string that resolves to a boolean value
If the condition evaluates to true, this branch will be executed</p>
</td>
</tr>
<tr>
<td>
<code>steps</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">
[]Step
</a>
</em>
</td>
<td>
<p>Steps contains the list of steps to execute if the condition is true</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.SwitchJob">SwitchJob
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Job">Job</a>)
</p>
<p>
<p>SwitchJob defines a job that executes different cases of steps based on conditions.</p>
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
<code>cases</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.SwitchCase">
[]SwitchCase
</a>
</em>
</td>
<td>
<p>Cases contains the different cases of steps to execute based on conditions</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.TemplateRef">TemplateRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunSpec">WorkflowRunSpec</a>)
</p>
<p>
<p>TemplateRef is a reference to a WorkflowTemplate.</p>
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
<p>Name of the template</p>
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
<p>Namespace of the template
If not specified, the namespace of the WorkflowRun is used</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.TransformStep">TransformStep
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Step">Step</a>)
</p>
<p>
<p>TransformStep defines a step that transforms data.</p>
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
<code>mappings</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>Transform is a map of key-value pairs, where the value is a template string
that will be dynamically resolved at runtime against the workflow data.</p>
</td>
</tr>
<tr>
<td>
<code>template</code></br>
<em>
string
</em>
</td>
<td>
<p>Full YAML template to be generated during transformation</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.Workflow">Workflow
</h3>
<p>
<p>Workflow is the Schema for the workflows API.</p>
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
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowSpec">
WorkflowSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
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
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>variables</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>jobs</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Job">
map[string]github.com/external-secrets/external-secrets/apis/workflows/v1alpha1.Job
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowStatus">
WorkflowStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRef">WorkflowRef
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunStatus">WorkflowRunStatus</a>)
</p>
<p>
<p>WorkflowRef is a reference to a Workflow.</p>
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
<p>Name of the workflow</p>
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
<p>Namespace of the workflow</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRun">WorkflowRun
</h3>
<p>
<p>WorkflowRun is the Schema for the workflowruns API.</p>
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
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunSpec">
WorkflowRunSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>templateRef</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.TemplateRef">
TemplateRef
</a>
</em>
</td>
<td>
<p>TemplateRef is a reference to the template to use</p>
</td>
</tr>
<tr>
<td>
<code>arguments</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Arguments are the values for template parameters
Each argument corresponds to a parameter defined in the template</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunStatus">
WorkflowRunStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRunSpec">WorkflowRunSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRun">WorkflowRun</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateSpec">WorkflowRunTemplateSpec</a>)
</p>
<p>
<p>WorkflowRunSpec defines the desired state of WorkflowRun.</p>
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
<code>templateRef</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.TemplateRef">
TemplateRef
</a>
</em>
</td>
<td>
<p>TemplateRef is a reference to the template to use</p>
</td>
</tr>
<tr>
<td>
<code>arguments</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Arguments are the values for template parameters
Each argument corresponds to a parameter defined in the template</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRunStatus">WorkflowRunStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.NamedWorkflowRunStatus">NamedWorkflowRunStatus</a>, 
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRun">WorkflowRun</a>)
</p>
<p>
<p>WorkflowRunStatus defines the observed state of WorkflowRun.</p>
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
<code>workflowRef</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRef">
WorkflowRef
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>WorkflowRef is a reference to the created workflow</p>
</td>
</tr>
<tr>
<td>
<code>sensitiveValuesSecrets</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SensitiveValuesSecrets is a list of secret names containing sensitive values</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions represent the latest available observations of the WorkflowRun&rsquo;s state</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Phase">
Phase
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Phase represents the current phase of the WorkflowRun</p>
</td>
</tr>
<tr>
<td>
<code>startTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StartTime represents when the WorkflowRun started</p>
</td>
</tr>
<tr>
<td>
<code>completionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CompletionTime represents when the WorkflowRun completed</p>
</td>
</tr>
<tr>
<td>
<code>executionTimeNanos</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExecutionTimeNanos represents the duration between the start and completion of the WorkflowRun, in nanoseconds</p>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRunTemplate">WorkflowRunTemplate
</h3>
<p>
<p>WorkflowRunTemplate is the Schema for the workflowruntemplates API.</p>
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
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateSpec">
WorkflowRunTemplateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>runSpec</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunSpec">
WorkflowRunSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>runPolicy</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicy">
RunPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>revisionHistoryLimit</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateStatus">
WorkflowRunTemplateStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateSpec">WorkflowRunTemplateSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunTemplate">WorkflowRunTemplate</a>)
</p>
<p>
<p>WorkflowRunTemplateSpec defines the desired state of WorkflowRunTemplate.</p>
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
<code>runSpec</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunSpec">
WorkflowRunSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>runPolicy</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.RunPolicy">
RunPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>revisionHistoryLimit</code></br>
<em>
int
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateStatus">WorkflowRunTemplateStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowRunTemplate">WorkflowRunTemplate</a>)
</p>
<p>
<p>WorkflowRunTemplateStatus defines the observed state of WorkflowRunTemplate.</p>
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
<code>lastRunTime</code></br>
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
<tr>
<td>
<code>runs</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.NamedWorkflowRunStatus">
[]NamedWorkflowRunStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Conditions represent the latest available observations of the WorkflowRun&rsquo;s state</p>
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
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRunTemplateValidator">WorkflowRunTemplateValidator
</h3>
<p>
<p>WorkflowRunTemplateValidator validates WorkflowRunTemplate resources.</p>
</p>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowRunValidator">WorkflowRunValidator
</h3>
<p>
<p>WorkflowRunValidator validates WorkflowRun resources.</p>
</p>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowSpec">WorkflowSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Workflow">Workflow</a>)
</p>
<p>
<p>WorkflowSpec defines the desired state of Workflow.</p>
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
<code>version</code></br>
<em>
string
</em>
</td>
<td>
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
</td>
</tr>
<tr>
<td>
<code>variables</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1#JSON">
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>jobs</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Job">
map[string]github.com/external-secrets/external-secrets/apis/workflows/v1alpha1.Job
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowStatus">WorkflowStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.Workflow">Workflow</a>)
</p>
<p>
<p>WorkflowStatus defines the observed state of Workflow.</p>
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#condition-v1-meta">
[]Kubernetes meta/v1.Condition
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Phase">
Phase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>jobStatuses</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.JobStatus">
map[string]github.com/external-secrets/external-secrets/apis/workflows/v1alpha1.JobStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>executionOrder</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>startTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>completionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>executionTimeNanos</code></br>
<em>
int64
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowTemplate">WorkflowTemplate
</h3>
<p>
<p>WorkflowTemplate is the Schema for the workflowtemplates API.</p>
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
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowTemplateSpec">
WorkflowTemplateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version of the workflow template</p>
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
<p>Name is a human-readable name for the workflow template</p>
</td>
</tr>
<tr>
<td>
<code>parameterGroups</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.ParameterGroup">
[]ParameterGroup
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ParameterGroups that can be overridden when creating a workflow</p>
</td>
</tr>
<tr>
<td>
<code>jobs</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Job">
map[string]github.com/external-secrets/external-secrets/apis/workflows/v1alpha1.Job
</a>
</em>
</td>
<td>
<p>Jobs is a map of job names to job definitions</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="workflows.external-secrets.io/v1alpha1.WorkflowTemplateSpec">WorkflowTemplateSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#workflows.external-secrets.io/v1alpha1.WorkflowTemplate">WorkflowTemplate</a>)
</p>
<p>
<p>WorkflowTemplateSpec defines the desired state of WorkflowTemplate.</p>
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
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version of the workflow template</p>
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
<p>Name is a human-readable name for the workflow template</p>
</td>
</tr>
<tr>
<td>
<code>parameterGroups</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.ParameterGroup">
[]ParameterGroup
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ParameterGroups that can be overridden when creating a workflow</p>
</td>
</tr>
<tr>
<td>
<code>jobs</code></br>
<em>
<a href="#workflows.external-secrets.io/v1alpha1.Job">
map[string]github.com/external-secrets/external-secrets/apis/workflows/v1alpha1.Job
</a>
</em>
</td>
<td>
<p>Jobs is a map of job names to job definitions</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>.
</em></p>
