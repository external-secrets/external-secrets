/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package keyvault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/tidwall/gjson"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	defaultObjType       = "secret"
	objectTypeCert       = "cert"
	objectTypeKey        = "key"
	AzureDefaultAudience = "api://AzureADTokenExchange"
	AnnotationClientID   = "azure.workload.identity/client-id"
	AnnotationTenantID   = "azure.workload.identity/tenant-id"

	errUnexpectedStoreSpec   = "unexpected store spec"
	errMissingAuthType       = "cannot initialize Azure Client: no valid authType was specified"
	errPropNotExist          = "property %s does not exist in key %s"
	errTagNotExist           = "tag %s does not exist"
	errUnknownObjectType     = "unknown Azure Keyvault object Type for %s"
	errUnmarshalJSONData     = "error unmarshalling json data: %w"
	errDataFromCert          = "cannot get use dataFrom to get certificate secret"
	errDataFromKey           = "cannot get use dataFrom to get key secret"
	errMissingTenant         = "missing tenantID in store config"
	errMissingSecretRef      = "missing secretRef in provider config"
	errMissingClientIDSecret = "missing accessKeyID/secretAccessKey in store config"
	errFindSecret            = "could not find secret %s/%s: %w"
	errFindDataKey           = "no data for %q in secret '%s/%s'"

	errInvalidStore              = "invalid store"
	errInvalidStoreSpec          = "invalid store spec"
	errInvalidStoreProv          = "invalid store provider"
	errInvalidAzureProv          = "invalid azure keyvault provider"
	errInvalidSecRefClientID     = "invalid AuthSecretRef.ClientID: %w"
	errInvalidSecRefClientSecret = "invalid AuthSecretRef.ClientSecret: %w"
	errInvalidSARef              = "invalid ServiceAccountRef: %w"

	errMissingWorkloadEnvVars = "missing environment variables. AZURE_CLIENT_ID, AZURE_TENANT_ID and AZURE_FEDERATED_TOKEN_FILE must be set"
	errReadTokenFile          = "unable to read token file %s: %w"
	errMissingSAAnnotation    = "missing service account annotation: %s"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Azure{}
var _ esv1beta1.Provider = &Azure{}

// interface to keyvault.BaseClient.
type SecretClient interface {
	GetKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string) (result keyvault.KeyBundle, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	GetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error)
	GetCertificate(ctx context.Context, vaultBaseURL string, certificateName string, certificateVersion string) (result keyvault.CertificateBundle, err error)
}

type Azure struct {
	crClient   client.Client
	kubeClient kcorev1.CoreV1Interface
	store      esv1beta1.GenericStore
	provider   *esv1beta1.AzureKVProvider
	baseClient SecretClient
	namespace  string
}

func init() {
	esv1beta1.Register(&Azure{}, &esv1beta1.SecretStoreProvider{
		AzureKV: &esv1beta1.AzureKVProvider{},
	})
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (a *Azure) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// NewClient constructs a new secrets client based on the provided store.
func (a *Azure) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	az := &Azure{
		crClient:   kube,
		kubeClient: kubeClient.CoreV1(),
		store:      store,
		namespace:  namespace,
		provider:   provider,
	}

	var authorizer autorest.Authorizer
	switch *provider.AuthType {
	case esv1beta1.AzureManagedIdentity:
		authorizer, err = az.authorizerForManagedIdentity()
	case esv1beta1.AzureServicePrincipal:
		authorizer, err = az.authorizerForServicePrincipal(ctx)
	case esv1beta1.AzureWorkloadIdentity:
		authorizer, err = az.authorizerForWorkloadIdentity(ctx, NewTokenProvider)
	default:
		err = fmt.Errorf(errMissingAuthType)
	}

	cl := keyvault.New()
	cl.Authorizer = authorizer
	az.baseClient = &cl

	return az, err
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.AzureKVProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider.AzureKV == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}

	return spc.Provider.AzureKV, nil
}

func (a *Azure) ValidateStore(store esv1beta1.GenericStore) error {
	if store == nil {
		return fmt.Errorf(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return fmt.Errorf(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return fmt.Errorf(errInvalidStoreProv)
	}
	p := spc.Provider.AzureKV
	if p == nil {
		return fmt.Errorf(errInvalidAzureProv)
	}
	if p.AuthSecretRef != nil {
		if p.AuthSecretRef.ClientID != nil {
			if err := utils.ValidateSecretSelector(store, *p.AuthSecretRef.ClientID); err != nil {
				return fmt.Errorf(errInvalidSecRefClientID, err)
			}
		}
		if p.AuthSecretRef.ClientSecret != nil {
			if err := utils.ValidateSecretSelector(store, *p.AuthSecretRef.ClientSecret); err != nil {
				return fmt.Errorf(errInvalidSecRefClientSecret, err)
			}
		}
	}
	if p.ServiceAccountRef != nil {
		if err := utils.ValidateServiceAccountSelector(store, *p.ServiceAccountRef); err != nil {
			return fmt.Errorf(errInvalidSARef, err)
		}
	}
	return nil
}

func (a *Azure) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Not Implemented PushSecret.
func (a *Azure) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Implements store.Client.GetAllSecrets Interface.
// Retrieves a map[string][]byte with the secret names as key and the secret itself as the calue.
func (a *Azure) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	basicClient := a.baseClient
	secretsMap := make(map[string][]byte)
	checkTags := len(ref.Tags) > 0
	checkName := ref.Name != nil && len(ref.Name.RegExp) > 0

	secretListIter, err := basicClient.GetSecretsComplete(context.Background(), *a.provider.VaultURL, nil)
	if err != nil {
		return nil, err
	}

	for secretListIter.NotDone() {
		secretList := secretListIter.Response().Value
		for _, secret := range *secretList {
			ok, secretName := isValidSecret(checkTags, checkName, ref, secret)
			if !ok {
				continue
			}

			secretResp, err := basicClient.GetSecret(context.Background(), *a.provider.VaultURL, secretName, "")
			if err != nil {
				return nil, err
			}

			secretValue := *secretResp.Value
			secretsMap[secretName] = []byte(secretValue)
		}

		err = secretListIter.Next()
		if err != nil {
			return nil, err
		}
	}
	return secretsMap, nil
}

// Retrieves a tag value if specified and all tags in JSON format if not.
func getSecretTag(tags map[string]*string, property string) ([]byte, error) {
	if property == "" {
		secretTagsData := make(map[string]string)
		for k, v := range tags {
			secretTagsData[k] = *v
		}
		return json.Marshal(secretTagsData)
	}
	if val, exist := tags[property]; exist {
		return []byte(*val), nil
	}

	idx := strings.Index(property, ".")
	if idx < 0 {
		return nil, fmt.Errorf(errTagNotExist, property)
	}

	if idx > 0 {
		tagName := property[0:idx]
		if val, exist := tags[tagName]; exist {
			key := strings.Replace(property, tagName+".", "", 1)
			return getProperty(*val, key, property)
		}
	}

	return nil, fmt.Errorf(errTagNotExist, property)
}

// Retrieves a property value if specified and the secret value if not.
func getProperty(secret, property, key string) ([]byte, error) {
	if property == "" {
		return []byte(secret), nil
	}
	res := gjson.Get(secret, property)
	if !res.Exists() {
		idx := strings.Index(property, ".")
		if idx < 0 {
			return nil, fmt.Errorf(errPropNotExist, property, key)
		}
		escaped := strings.ReplaceAll(property, ".", "\\.")
		jValue := gjson.Get(secret, escaped)
		if jValue.Exists() {
			return []byte(jValue.String()), nil
		}
		return nil, fmt.Errorf(errPropNotExist, property, key)
	}
	return []byte(res.String()), nil
}

// Implements store.Client.GetSecret Interface.
// Retrieves a secret/Key/Certificate/Tag with the secret name defined in ref.Name
// The Object Type is defined as a prefix in the ref.Name , if no prefix is defined , we assume a secret is required.
func (a *Azure) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	objectType, secretName := getObjType(ref)

	switch objectType {
	case defaultObjType:
		// returns a SecretBundle with the secret value
		// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#SecretBundle
		secretResp, err := a.baseClient.GetSecret(context.Background(), *a.provider.VaultURL, secretName, ref.Version)
		if err != nil {
			return nil, err
		}
		if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(secretResp.Tags, ref.Property)
		}
		return getProperty(*secretResp.Value, ref.Property, ref.Key)
	case objectTypeCert:
		// returns a CertBundle. We return CER contents of x509 certificate
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#CertificateBundle
		certResp, err := a.baseClient.GetCertificate(context.Background(), *a.provider.VaultURL, secretName, ref.Version)
		if err != nil {
			return nil, err
		}
		if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(certResp.Tags, ref.Property)
		}
		return *certResp.Cer, nil
	case objectTypeKey:
		// returns a KeyBundle that contains a jwk
		// azure kv returns only public keys
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#KeyBundle
		keyResp, err := a.baseClient.GetKey(context.Background(), *a.provider.VaultURL, secretName, ref.Version)
		if err != nil {
			return nil, err
		}
		if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(keyResp.Tags, ref.Property)
		}
		return json.Marshal(keyResp.Key)
	}

	return nil, fmt.Errorf(errUnknownObjectType, secretName)
}

// returns a SecretBundle with the tags values.
func (a *Azure) getSecretTags(ref esv1beta1.ExternalSecretDataRemoteRef) (map[string]*string, error) {
	_, secretName := getObjType(ref)
	secretResp, err := a.baseClient.GetSecret(context.Background(), *a.provider.VaultURL, secretName, ref.Version)

	if err != nil {
		return nil, err
	}

	secretTagsData := make(map[string]*string)

	for tagname, tagval := range secretResp.Tags {
		name := secretName + "_" + tagname
		kv := make(map[string]string)
		err = json.Unmarshal([]byte(*tagval), &kv)
		// if the tagvalue is not in JSON format then we added to secretTagsData we added as it is
		if err != nil {
			secretTagsData[name] = tagval
		} else {
			for k, v := range kv {
				value := v
				secretTagsData[name+"_"+k] = &value
			}
		}
	}
	return secretTagsData, nil
}

// Implements store.Client.GetSecretMap Interface.
// New version of GetSecretMap.
func (a *Azure) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	objectType, secretName := getObjType(ref)

	switch objectType {
	case defaultObjType:
		data, err := a.GetSecret(ctx, ref)
		if err != nil {
			return nil, err
		}

		if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
			tags, _ := a.getSecretTags(ref)
			return getSecretMapProperties(tags, ref.Key, ref.Property), nil
		}

		return getSecretMapMap(data)

	case objectTypeCert:
		return nil, fmt.Errorf(errDataFromCert)
	case objectTypeKey:
		return nil, fmt.Errorf(errDataFromKey)
	}
	return nil, fmt.Errorf(errUnknownObjectType, secretName)
}

func getSecretMapMap(data []byte) (map[string][]byte, error) {
	kv := make(map[string]json.RawMessage)
	err := json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalJSONData, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		err = json.Unmarshal(v, &strVal)
		if err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}
	return secretData, nil
}

func getSecretMapProperties(tags map[string]*string, key, property string) map[string][]byte {
	tagByteArray := make(map[string][]byte)
	if property != "" {
		keyPropertyName := key + "_" + property
		singleTag, _ := getSecretTag(tags, keyPropertyName)
		tagByteArray[keyPropertyName] = singleTag

		return tagByteArray
	}

	for k, v := range tags {
		tagByteArray[k] = []byte(*v)
	}

	return tagByteArray
}

func (a *Azure) authorizerForWorkloadIdentity(ctx context.Context, tokenProvider tokenProviderFunc) (autorest.Authorizer, error) {
	aadEndpoint := AadEndpointForType(a.provider.EnvironmentType)
	kvResource := kvResourceForProviderConfig(a.provider.EnvironmentType)
	// if no serviceAccountRef was provided
	// we expect certain env vars to be present.
	// They are set by the azure workload identity webhook.
	if a.provider.ServiceAccountRef == nil {
		clientID := os.Getenv("AZURE_CLIENT_ID")
		tenantID := os.Getenv("AZURE_TENANT_ID")
		tokenFilePath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
		if clientID == "" || tenantID == "" || tokenFilePath == "" {
			return nil, errors.New(errMissingWorkloadEnvVars)
		}
		token, err := os.ReadFile(tokenFilePath)
		if err != nil {
			return nil, fmt.Errorf(errReadTokenFile, tokenFilePath, err)
		}
		tp, err := tokenProvider(ctx, string(token), clientID, tenantID, aadEndpoint, kvResource)
		if err != nil {
			return nil, err
		}
		return autorest.NewBearerAuthorizer(tp), nil
	}
	ns := a.namespace
	if a.store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		ns = *a.provider.ServiceAccountRef.Namespace
	}
	var sa corev1.ServiceAccount
	err := a.crClient.Get(ctx, types.NamespacedName{
		Name:      a.provider.ServiceAccountRef.Name,
		Namespace: ns,
	}, &sa)
	if err != nil {
		return nil, err
	}
	clientID, ok := sa.ObjectMeta.Annotations[AnnotationClientID]
	if !ok {
		return nil, fmt.Errorf(errMissingSAAnnotation, AnnotationClientID)
	}
	tenantID, ok := sa.ObjectMeta.Annotations[AnnotationTenantID]
	if !ok {
		return nil, fmt.Errorf(errMissingSAAnnotation, AnnotationTenantID)
	}
	audiences := []string{AzureDefaultAudience}
	if len(a.provider.ServiceAccountRef.Audiences) > 0 {
		audiences = append(audiences, a.provider.ServiceAccountRef.Audiences...)
	}
	token, err := FetchSAToken(ctx, ns, a.provider.ServiceAccountRef.Name, audiences, a.kubeClient)
	if err != nil {
		return nil, err
	}
	tp, err := tokenProvider(ctx, token, clientID, tenantID, aadEndpoint, kvResource)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(tp), nil
}

func FetchSAToken(ctx context.Context, ns, name string, audiences []string, kubeClient kcorev1.CoreV1Interface) (string, error) {
	token, err := kubeClient.ServiceAccounts(ns).CreateToken(ctx, name, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: audiences,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return token.Status.Token, nil
}

// tokenProvider satisfies the adal.OAuthTokenProvider interface.
type tokenProvider struct {
	accessToken string
}

type tokenProviderFunc func(ctx context.Context, token, clientID, tenantID, aadEndpoint, kvResource string) (adal.OAuthTokenProvider, error)

func NewTokenProvider(ctx context.Context, token, clientID, tenantID, aadEndpoint, kvResource string) (adal.OAuthTokenProvider, error) {
	// exchange token with Azure AccessToken
	cred := confidential.NewCredFromAssertionCallback(func(ctx context.Context, aro confidential.AssertionRequestOptions) (string, error) {
		return token, nil
	})
	cClient, err := confidential.New(clientID, cred, confidential.WithAuthority(
		fmt.Sprintf("%s%s/oauth2/token", aadEndpoint, tenantID),
	))
	if err != nil {
		return nil, err
	}
	scope := kvResource
	// .default needs to be added to the scope
	if !strings.Contains(kvResource, ".default") {
		scope = fmt.Sprintf("%s/.default", kvResource)
	}
	authRes, err := cClient.AcquireTokenByCredential(ctx, []string{
		scope,
	})
	if err != nil {
		return nil, err
	}
	return &tokenProvider{
		accessToken: authRes.AccessToken,
	}, nil
}

func (t *tokenProvider) OAuthToken() string {
	return t.accessToken
}

func (a *Azure) authorizerForManagedIdentity() (autorest.Authorizer, error) {
	msiConfig := kvauth.NewMSIConfig()
	msiConfig.Resource = kvResourceForProviderConfig(a.provider.EnvironmentType)
	if a.provider.IdentityID != nil {
		msiConfig.ClientID = *a.provider.IdentityID
	}
	return msiConfig.Authorizer()
}

func (a *Azure) authorizerForServicePrincipal(ctx context.Context) (autorest.Authorizer, error) {
	if a.provider.TenantID == nil {
		return nil, fmt.Errorf(errMissingTenant)
	}
	if a.provider.AuthSecretRef == nil {
		return nil, fmt.Errorf(errMissingSecretRef)
	}
	if a.provider.AuthSecretRef.ClientID == nil || a.provider.AuthSecretRef.ClientSecret == nil {
		return nil, fmt.Errorf(errMissingClientIDSecret)
	}
	clusterScoped := false
	if a.store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		clusterScoped = true
	}
	cid, err := a.secretKeyRef(ctx, a.store.GetNamespace(), *a.provider.AuthSecretRef.ClientID, clusterScoped)
	if err != nil {
		return nil, err
	}
	csec, err := a.secretKeyRef(ctx, a.store.GetNamespace(), *a.provider.AuthSecretRef.ClientSecret, clusterScoped)
	if err != nil {
		return nil, err
	}
	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(cid, csec, *a.provider.TenantID)
	clientCredentialsConfig.Resource = kvResourceForProviderConfig(a.provider.EnvironmentType)
	clientCredentialsConfig.AADEndpoint = AadEndpointForType(a.provider.EnvironmentType)
	return clientCredentialsConfig.Authorizer()
}

// secretKeyRef fetch a secret key.
func (a *Azure) secretKeyRef(ctx context.Context, namespace string, secretRef smmeta.SecretKeySelector, clusterScoped bool) (string, error) {
	var secret corev1.Secret
	ref := types.NamespacedName{
		Namespace: namespace,
		Name:      secretRef.Name,
	}
	if clusterScoped && secretRef.Namespace != nil {
		ref.Namespace = *secretRef.Namespace
	}
	err := a.crClient.Get(ctx, ref, &secret)
	if err != nil {
		return "", fmt.Errorf(errFindSecret, ref.Namespace, ref.Name, err)
	}
	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf(errFindDataKey, secretRef.Key, secretRef.Name, namespace)
	}
	value := strings.TrimSpace(string(keyBytes))
	return value, nil
}

func (a *Azure) Close(ctx context.Context) error {
	return nil
}

func (a *Azure) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func AadEndpointForType(t esv1beta1.AzureEnvironmentType) string {
	switch t {
	case esv1beta1.AzureEnvironmentPublicCloud:
		return azure.PublicCloud.ActiveDirectoryEndpoint
	case esv1beta1.AzureEnvironmentChinaCloud:
		return azure.ChinaCloud.ActiveDirectoryEndpoint
	case esv1beta1.AzureEnvironmentUSGovernmentCloud:
		return azure.USGovernmentCloud.ActiveDirectoryEndpoint
	case esv1beta1.AzureEnvironmentGermanCloud:
		return azure.GermanCloud.ActiveDirectoryEndpoint
	default:
		return azure.PublicCloud.ActiveDirectoryEndpoint
	}
}

func kvResourceForProviderConfig(t esv1beta1.AzureEnvironmentType) string {
	var res string
	switch t {
	case esv1beta1.AzureEnvironmentPublicCloud:
		res = azure.PublicCloud.KeyVaultEndpoint
	case esv1beta1.AzureEnvironmentChinaCloud:
		res = azure.ChinaCloud.KeyVaultEndpoint
	case esv1beta1.AzureEnvironmentUSGovernmentCloud:
		res = azure.USGovernmentCloud.KeyVaultEndpoint
	case esv1beta1.AzureEnvironmentGermanCloud:
		res = azure.GermanCloud.KeyVaultEndpoint
	default:
		res = azure.PublicCloud.KeyVaultEndpoint
	}
	return strings.TrimSuffix(res, "/")
}

func getObjType(ref esv1beta1.ExternalSecretDataRemoteRef) (string, string) {
	objectType := defaultObjType

	secretName := ref.Key
	nameSplitted := strings.Split(secretName, "/")

	if len(nameSplitted) > 1 {
		objectType = nameSplitted[0]
		secretName = nameSplitted[1]
		// TODO: later tokens can be used to read the secret tags
	}
	return objectType, secretName
}

func isValidSecret(checkTags, checkName bool, ref esv1beta1.ExternalSecretFind, secret keyvault.SecretItem) (bool, string) {
	if secret.ID == nil || !*secret.Attributes.Enabled {
		return false, ""
	}

	if checkTags && !okByTags(ref, secret) {
		return false, ""
	}

	secretName := path.Base(*secret.ID)
	if checkName && !okByName(ref, secretName) {
		return false, ""
	}

	return true, secretName
}

func okByName(ref esv1beta1.ExternalSecretFind, secretName string) bool {
	matches, _ := regexp.MatchString(ref.Name.RegExp, secretName)
	return matches
}

func okByTags(ref esv1beta1.ExternalSecretFind, secret keyvault.SecretItem) bool {
	tagsFound := true
	for k, v := range ref.Tags {
		if val, ok := secret.Tags[k]; !ok || *val != v {
			tagsFound = false
			break
		}
	}
	return tagsFound
}
