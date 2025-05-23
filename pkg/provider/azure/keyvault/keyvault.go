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
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/sha3"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	pointer "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gopkcs12 "software.sslmate.com/src/go-pkcs12"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/metadata"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	defaultObjType       = "secret"
	objectTypeCert       = "cert"
	objectTypeKey        = "key"
	AzureDefaultAudience = "api://AzureADTokenExchange"
	AnnotationClientID   = "azure.workload.identity/client-id"
	AnnotationTenantID   = "azure.workload.identity/tenant-id"
	managerLabel         = "external-secrets"
	managedBy            = "managed-by"

	errUnexpectedStoreSpec      = "unexpected store spec"
	errMissingAuthType          = "cannot initialize Azure Client: no valid authType was specified"
	errPropNotExist             = "property %s does not exist in key %s"
	errTagNotExist              = "tag %s does not exist"
	errUnknownObjectType        = "unknown Azure Keyvault object Type for %s"
	errUnmarshalJSONData        = "error unmarshalling json data: %w"
	errDataFromCert             = "cannot get use dataFrom to get certificate secret"
	errDataFromKey              = "cannot get use dataFrom to get key secret"
	errMissingTenant            = "missing tenantID in store config"
	errMissingClient            = "missing clientID: either serviceAccountRef or service account annotation '%s' is missing"
	errMissingSecretRef         = "missing secretRef in provider config"
	errMissingClientIDSecret    = "missing accessKeyID/secretAccessKey in store config"
	errInvalidClientCredentials = "both clientSecret and clientCredentials set"
	errMultipleClientID         = "multiple clientID found. Check secretRef and serviceAccountRef"
	errMultipleTenantID         = "multiple tenantID found. Check secretRef, 'spec.provider.azurekv.tenantId', and serviceAccountRef"

	errInvalidStore              = "invalid store"
	errInvalidStoreSpec          = "invalid store spec"
	errInvalidStoreProv          = "invalid store provider"
	errInvalidAzureProv          = "invalid azure keyvault provider"
	errInvalidSecRefClientID     = "invalid AuthSecretRef.ClientID: %w"
	errInvalidSecRefClientSecret = "invalid AuthSecretRef.ClientSecret: %w"
	errInvalidSARef              = "invalid ServiceAccountRef: %w"

	errMissingWorkloadEnvVars = "missing environment variables. AZURE_CLIENT_ID, AZURE_TENANT_ID and AZURE_FEDERATED_TOKEN_FILE must be set"
	errReadTokenFile          = "unable to read token file %s: %w"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Azure{}
var _ esv1.Provider = &Azure{}

// interface to keyvault.BaseClient.
type SecretClient interface {
	GetKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string) (result keyvault.KeyBundle, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	GetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error)
	GetCertificate(ctx context.Context, vaultBaseURL string, certificateName string, certificateVersion string) (result keyvault.CertificateBundle, err error)
	SetSecret(ctx context.Context, vaultBaseURL string, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error)
	ImportKey(ctx context.Context, vaultBaseURL string, keyName string, parameters keyvault.KeyImportParameters) (result keyvault.KeyBundle, err error)
	ImportCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error)
	DeleteCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.DeletedCertificateBundle, err error)
	DeleteKey(ctx context.Context, vaultBaseURL string, keyName string) (result keyvault.DeletedKeyBundle, err error)
	DeleteSecret(ctx context.Context, vaultBaseURL string, secretName string) (result keyvault.DeletedSecretBundle, err error)
}

type Azure struct {
	crClient   client.Client
	kubeClient kcorev1.CoreV1Interface
	store      esv1.GenericStore
	provider   *esv1.AzureKVProvider
	baseClient SecretClient
	namespace  string
}

type PushSecretMetadataSpec struct {
	ExpirationDate string            `json:"expirationDate,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

func init() {
	esv1.Register(&Azure{}, &esv1.SecretStoreProvider{
		AzureKV: &esv1.AzureKVProvider{},
	}, esv1.MaintenanceStatusMaintained)
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (a *Azure) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient constructs a new secrets client based on the provided store.
func (a *Azure) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
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

	// allow SecretStore controller validation to pass
	// when using referent namespace.
	if store.GetKind() == esv1.ClusterSecretStoreKind &&
		namespace == "" &&
		isReferentSpec(provider) {
		return az, nil
	}

	var authorizer autorest.Authorizer
	switch *provider.AuthType {
	case esv1.AzureManagedIdentity:
		authorizer, err = az.authorizerForManagedIdentity()
	case esv1.AzureServicePrincipal:
		authorizer, err = az.authorizerForServicePrincipal(ctx)
	case esv1.AzureWorkloadIdentity:
		authorizer, err = az.authorizerForWorkloadIdentity(ctx, NewTokenProvider)
	default:
		err = errors.New(errMissingAuthType)
	}

	cl := keyvault.New()
	cl.Authorizer = authorizer
	az.baseClient = &cl

	return az, err
}

func getProvider(store esv1.GenericStore) (*esv1.AzureKVProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider.AzureKV == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}

	return spc.Provider.AzureKV, nil
}

func (a *Azure) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, errors.New(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errInvalidStoreProv)
	}
	p := spc.Provider.AzureKV
	if p == nil {
		return nil, errors.New(errInvalidAzureProv)
	}
	if p.AuthSecretRef != nil {
		if p.AuthSecretRef.ClientID != nil {
			if err := utils.ValidateReferentSecretSelector(store, *p.AuthSecretRef.ClientID); err != nil {
				return nil, fmt.Errorf(errInvalidSecRefClientID, err)
			}
		}
		if p.AuthSecretRef.ClientSecret != nil {
			if err := utils.ValidateReferentSecretSelector(store, *p.AuthSecretRef.ClientSecret); err != nil {
				return nil, fmt.Errorf(errInvalidSecRefClientSecret, err)
			}
		}
	}
	if p.ServiceAccountRef != nil {
		if err := utils.ValidateReferentServiceAccountSelector(store, *p.ServiceAccountRef); err != nil {
			return nil, fmt.Errorf(errInvalidSARef, err)
		}
	}
	return nil, nil
}

func canDelete(tags map[string]*string, err error) (bool, error) {
	aerr := &autorest.DetailedError{}
	conv := errors.As(err, aerr)
	if err != nil && !conv {
		return false, fmt.Errorf("could not parse error: %w", err)
	}
	if conv && aerr.StatusCode != 404 { // Secret is already deleted, nothing to do.
		return false, fmt.Errorf("unexpected api error: %w", err)
	}
	if aerr.StatusCode == 404 {
		return false, nil
	}
	manager, ok := tags[managedBy]
	if !ok || manager == nil || *manager != managerLabel {
		return false, errors.New("not managed by external-secrets")
	}
	return true, nil
}

func (a *Azure) deleteKeyVaultKey(ctx context.Context, keyName string) error {
	value, err := a.baseClient.GetKey(ctx, *a.provider.VaultURL, keyName, "")
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetKey, err)
	ok, err := canDelete(value.Tags, err)
	if err != nil {
		return fmt.Errorf("error getting key %v: %w", keyName, err)
	}
	if ok {
		_, err = a.baseClient.DeleteKey(ctx, *a.provider.VaultURL, keyName)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVDeleteKey, err)
		if err != nil {
			return fmt.Errorf("error deleting key %v: %w", keyName, err)
		}
	}
	return nil
}

func (a *Azure) deleteKeyVaultSecret(ctx context.Context, secretName string) error {
	value, err := a.baseClient.GetSecret(ctx, *a.provider.VaultURL, secretName, "")
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
	ok, err := canDelete(value.Tags, err)
	if err != nil {
		return fmt.Errorf("error getting secret %v: %w", secretName, err)
	}
	if ok {
		_, err = a.baseClient.DeleteSecret(ctx, *a.provider.VaultURL, secretName)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVDeleteSecret, err)
		if err != nil {
			return fmt.Errorf("error deleting secret %v: %w", secretName, err)
		}
	}
	return nil
}

func (a *Azure) deleteKeyVaultCertificate(ctx context.Context, certName string) error {
	value, err := a.baseClient.GetCertificate(ctx, *a.provider.VaultURL, certName, "")
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetCertificate, err)
	ok, err := canDelete(value.Tags, err)
	if err != nil {
		return fmt.Errorf("error getting certificate %v: %w", certName, err)
	}
	if ok {
		_, err = a.baseClient.DeleteCertificate(ctx, *a.provider.VaultURL, certName)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVDeleteCertificate, err)
		if err != nil {
			return fmt.Errorf("error deleting certificate %v: %w", certName, err)
		}
	}
	return nil
}

func (a *Azure) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	objectType, secretName := getObjType(esv1.ExternalSecretDataRemoteRef{Key: remoteRef.GetRemoteKey()})
	switch objectType {
	case defaultObjType:
		return a.deleteKeyVaultSecret(ctx, secretName)
	case objectTypeCert:
		return a.deleteKeyVaultCertificate(ctx, secretName)
	case objectTypeKey:
		return a.deleteKeyVaultKey(ctx, secretName)
	default:
		return fmt.Errorf("secret type '%v' is not supported", objectType)
	}
}

func (a *Azure) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	objectType, secretName := getObjType(esv1.ExternalSecretDataRemoteRef{Key: remoteRef.GetRemoteKey()})

	var err error
	switch objectType {
	case defaultObjType:
		_, err = a.baseClient.GetSecret(ctx, *a.provider.VaultURL, secretName, "")
	case objectTypeCert:
		_, err = a.baseClient.GetCertificate(ctx, *a.provider.VaultURL, secretName, "")
	case objectTypeKey:
		_, err = a.baseClient.GetKey(ctx, *a.provider.VaultURL, secretName, "")
	default:
		errMsg := fmt.Sprintf("secret type '%v' is not supported", objectType)
		return false, errors.New(errMsg)
	}

	err = parseError(err)
	if err != nil {
		var noSecretErr esv1.NoSecretError
		if errors.As(err, &noSecretErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func getCertificateFromValue(value []byte) (*x509.Certificate, error) {
	// 1st: try decode pkcs12
	_, localCert, err := gopkcs12.Decode(value, "")
	if err == nil {
		return localCert, nil
	}

	// 2nd: try decode pkcs12 with chain
	_, localCert, _, err = gopkcs12.DecodeChain(value, "")
	if err == nil {
		return localCert, nil
	}

	// 3rd: try DER
	localCert, err = x509.ParseCertificate(value)
	if err == nil {
		return localCert, nil
	}

	// 4th: parse PEM blocks
	for {
		block, rest := pem.Decode(value)
		value = rest
		if block == nil {
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err == nil {
			return cert, nil
		}
	}
	return nil, errors.New("could not parse certificate value as PKCS#12, DER or PEM")
}

func getKeyFromValue(value []byte) (any, error) {
	val := value
	pemBlock, _ := pem.Decode(value)
	// if a private key regular expression doesn't match, we should consider this key to be symmetric
	if pemBlock == nil {
		return val, nil
	}
	val = pemBlock.Bytes
	switch pemBlock.Type {
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(val)
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(val)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(val)
	default:
		return nil, fmt.Errorf("key type %v is not supported", pemBlock.Type)
	}
}

func canCreate(tags map[string]*string, err error) (bool, error) {
	aerr := &autorest.DetailedError{}
	conv := errors.As(err, aerr)
	if err != nil && !conv {
		return false, fmt.Errorf("could not parse error: %w", err)
	}
	if conv && aerr.StatusCode != 404 {
		return false, fmt.Errorf("unexpected api error: %w", err)
	}
	if err == nil {
		manager, ok := tags[managedBy]
		if !ok || manager == nil || *manager != managerLabel {
			return false, errors.New("not managed by external-secrets")
		}
	}
	return true, nil
}

func (a *Azure) setKeyVaultSecret(ctx context.Context, secretName string, value []byte, expires *date.UnixTime, tags map[string]string) error {
	secret, err := a.baseClient.GetSecret(ctx, *a.provider.VaultURL, secretName, "")
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
	ok, err := canCreate(secret.Tags, err)
	if err != nil {
		return fmt.Errorf("cannot get secret %v: %w", secretName, err)
	}
	if !ok {
		return nil
	}
	val := string(value)
	if secret.Value != nil && val == *secret.Value {
		if secret.Attributes != nil {
			if (secret.Attributes.Expires == nil && expires == nil) ||
				(secret.Attributes.Expires != nil && expires != nil && *secret.Attributes.Expires == *expires) {
				return nil
			}
		}
	}

	secretParams := keyvault.SecretSetParameters{
		Value: &val,
		Tags: map[string]*string{
			managedBy: pointer.To(managerLabel),
		},
		SecretAttributes: &keyvault.SecretAttributes{
			Enabled: pointer.To(true),
		},
	}

	for k, v := range tags {
		secretParams.Tags[k] = &v
	}

	if expires != nil {
		secretParams.SecretAttributes.Expires = expires
	}

	_, err = a.baseClient.SetSecret(ctx, *a.provider.VaultURL, secretName, secretParams)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
	if err != nil {
		return fmt.Errorf("could not set secret %v: %w", secretName, err)
	}
	return nil
}

func (a *Azure) setKeyVaultCertificate(ctx context.Context, secretName string, value []byte, tags map[string]string) error {
	val := b64.StdEncoding.EncodeToString(value)
	localCert, err := getCertificateFromValue(value)
	if err != nil {
		return fmt.Errorf("value from secret is not a valid certificate: %w", err)
	}
	cert, err := a.baseClient.GetCertificate(ctx, *a.provider.VaultURL, secretName, "")
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetCertificate, err)
	ok, err := canCreate(cert.Tags, err)
	if err != nil {
		return fmt.Errorf("cannot get certificate %v: %w", secretName, err)
	}
	if !ok {
		return nil
	}
	b512 := sha3.Sum512(localCert.Raw)
	if cert.Cer != nil && b512 == sha3.Sum512(*cert.Cer) {
		return nil
	}
	params := keyvault.CertificateImportParameters{
		Base64EncodedCertificate: &val,
		Tags: map[string]*string{
			managedBy: pointer.To(managerLabel),
		},
	}

	for k, v := range tags {
		params.Tags[k] = &v
	}

	_, err = a.baseClient.ImportCertificate(ctx, *a.provider.VaultURL, secretName, params)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVImportCertificate, err)
	if err != nil {
		return fmt.Errorf("could not import certificate %v: %w", secretName, err)
	}
	return nil
}
func equalKeys(newKey, oldKey keyvault.JSONWebKey) bool {
	// checks for everything except KeyID and KeyOps
	rsaCheck := newKey.E != nil && oldKey.E != nil && *newKey.E == *oldKey.E &&
		newKey.N != nil && oldKey.N != nil && *newKey.N == *oldKey.N

	symmetricCheck := newKey.Crv == oldKey.Crv &&
		newKey.T != nil && oldKey.T != nil && *newKey.T == *oldKey.T &&
		newKey.X != nil && oldKey.X != nil && *newKey.X == *oldKey.X &&
		newKey.Y != nil && oldKey.Y != nil && *newKey.Y == *oldKey.Y

	return newKey.Kty == oldKey.Kty && (rsaCheck || symmetricCheck)
}
func (a *Azure) setKeyVaultKey(ctx context.Context, secretName string, value []byte, tags map[string]string) error {
	key, err := getKeyFromValue(value)
	if err != nil {
		return fmt.Errorf("could not load private key %v: %w", secretName, err)
	}
	jwKey, err := jwk.FromRaw(key)
	if err != nil {
		return fmt.Errorf("failed to generate a JWK from secret %v content: %w", secretName, err)
	}
	buf, err := json.Marshal(jwKey)
	if err != nil {
		return fmt.Errorf("error parsing key: %w", err)
	}
	azkey := keyvault.JSONWebKey{}
	err = json.Unmarshal(buf, &azkey)
	if err != nil {
		return fmt.Errorf("error unmarshalling key: %w", err)
	}
	keyFromVault, err := a.baseClient.GetKey(ctx, *a.provider.VaultURL, secretName, "")
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetKey, err)
	ok, err := canCreate(keyFromVault.Tags, err)
	if err != nil {
		return fmt.Errorf("cannot get key %v: %w", secretName, err)
	}
	if !ok {
		return nil
	}
	if keyFromVault.Key != nil && equalKeys(azkey, *keyFromVault.Key) {
		return nil
	}
	params := keyvault.KeyImportParameters{
		Key:           &azkey,
		KeyAttributes: &keyvault.KeyAttributes{},
		Tags: map[string]*string{
			managedBy: pointer.To(managerLabel),
		},
	}

	for k, v := range tags {
		params.Tags[k] = &v
	}
	_, err = a.baseClient.ImportKey(ctx, *a.provider.VaultURL, secretName, params)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVImportKey, err)
	if err != nil {
		return fmt.Errorf("could not import key %v: %w", secretName, err)
	}
	return nil
}

func getSecretKey(secret *corev1.Secret, data esv1.PushSecretData) ([]byte, error) {
	if data.GetSecretKey() != "" {
		return secret.Data[data.GetSecretKey()], nil
	}

	// Must convert secret values to string, otherwise data will be sent as base64 to Vault
	secretStringVal := make(map[string]string)
	for k, v := range secret.Data {
		secretStringVal[k] = string(v)
	}
	value, err := utils.JSONMarshal(secretStringVal)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize secret content as JSON: %w", err)
	}
	return value, nil
}

// PushSecret stores secrets into a Key vault instance.
func (a *Azure) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	var (
		value   []byte
		err     error
		expires *date.UnixTime
		tags    map[string]string
	)

	if value, err = getSecretKey(secret, data); err != nil {
		return err
	}

	metadata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}

	if metadata != nil && metadata.Spec.ExpirationDate != "" {
		t, err := time.Parse(time.RFC3339, metadata.Spec.ExpirationDate)
		if err != nil {
			return fmt.Errorf("error parsing expiration date in metadata: %w. Expected format: YYYY-MM-DDTHH:MM:SSZ (RFC3339). Example: 2024-12-31T20:00:00Z", err)
		}
		unixTime := date.UnixTime(t)
		expires = &unixTime
	}

	if metadata != nil && metadata.Spec.Tags != nil {
		if _, exists := metadata.Spec.Tags[managedBy]; exists {
			return fmt.Errorf("error parsing tags in metadata: Cannot specify a '%s' tag", managedBy)
		}
		tags = metadata.Spec.Tags
	}

	objectType, secretName := getObjType(esv1.ExternalSecretDataRemoteRef{Key: data.GetRemoteKey()})
	switch objectType {
	case defaultObjType:
		return a.setKeyVaultSecret(ctx, secretName, value, expires, tags)
	case objectTypeCert:
		return a.setKeyVaultCertificate(ctx, secretName, value, tags)
	case objectTypeKey:
		return a.setKeyVaultKey(ctx, secretName, value, tags)
	default:
		return fmt.Errorf("secret type %v not supported", objectType)
	}
}

// Implements store.Client.GetAllSecrets Interface.
// Retrieves a map[string][]byte with the secret names as key and the secret itself as the calue.
func (a *Azure) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	basicClient := a.baseClient
	secretsMap := make(map[string][]byte)
	checkTags := len(ref.Tags) > 0
	checkName := ref.Name != nil && ref.Name.RegExp != ""

	secretListIter, err := basicClient.GetSecretsComplete(ctx, *a.provider.VaultURL, nil)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecrets, err)
	err = parseError(err)
	if err != nil {
		return nil, err
	}

	for secretListIter.NotDone() {
		secret := secretListIter.Value()
		ok, secretName := isValidSecret(checkTags, checkName, ref, secret)
		if !ok {
			err = secretListIter.Next()
			if err != nil {
				return nil, err
			}
			continue
		}
		secretResp, err := basicClient.GetSecret(ctx, *a.provider.VaultURL, secretName, "")
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
		err = parseError(err)
		if err != nil {
			return nil, err
		}

		secretValue := *secretResp.Value
		secretsMap[secretName] = []byte(secretValue)

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

func parseError(err error) error {
	aerr := autorest.DetailedError{}
	if errors.As(err, &aerr) && aerr.StatusCode == 404 {
		return esv1.NoSecretError{}
	}
	return err
}

// GetSecret implements store.Client.GetSecret Interface.
// Retrieves a secret/Key/Certificate/Tag with the secret name defined in ref.Name
// The Object Type is defined as a prefix in the ref.Name , if no prefix is defined , we assume a secret is required.
func (a *Azure) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	objectType, secretName := getObjType(ref)

	switch objectType {
	case defaultObjType:
		// returns a SecretBundle with the secret value
		// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#SecretBundle
		secretResp, err := a.baseClient.GetSecret(ctx, *a.provider.VaultURL, secretName, ref.Version)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
		err = parseError(err)
		if err != nil {
			return nil, err
		}
		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(secretResp.Tags, ref.Property)
		}
		return getProperty(*secretResp.Value, ref.Property, ref.Key)
	case objectTypeCert:
		// returns a CertBundle. We return CER contents of x509 certificate
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#CertificateBundle
		certResp, err := a.baseClient.GetCertificate(ctx, *a.provider.VaultURL, secretName, ref.Version)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetCertificate, err)
		err = parseError(err)
		if err != nil {
			return nil, err
		}
		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(certResp.Tags, ref.Property)
		}
		return *certResp.Cer, nil
	case objectTypeKey:
		// returns a KeyBundle that contains a jwk
		// azure kv returns only public keys
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#KeyBundle
		keyResp, err := a.baseClient.GetKey(ctx, *a.provider.VaultURL, secretName, ref.Version)
		metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetKey, err)
		err = parseError(err)
		if err != nil {
			return nil, err
		}
		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			return getSecretTag(keyResp.Tags, ref.Property)
		}
		return json.Marshal(keyResp.Key)
	}

	return nil, fmt.Errorf(errUnknownObjectType, secretName)
}

// returns a SecretBundle with the tags values.
func (a *Azure) getSecretTags(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string]*string, error) {
	_, secretName := getObjType(ref)
	secretResp, err := a.baseClient.GetSecret(ctx, *a.provider.VaultURL, secretName, ref.Version)
	metrics.ObserveAPICall(constants.ProviderAzureKV, constants.CallAzureKVGetSecret, err)
	err = parseError(err)
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
func (a *Azure) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	objectType, secretName := getObjType(ref)

	switch objectType {
	case defaultObjType:
		data, err := a.GetSecret(ctx, ref)
		if err != nil {
			return nil, err
		}

		if ref.MetadataPolicy == esv1.ExternalSecretMetadataPolicyFetch {
			tags, _ := a.getSecretTags(ctx, ref)
			return getSecretMapProperties(tags, ref.Key, ref.Property), nil
		}

		return getSecretMapMap(data)

	case objectTypeCert:
		return nil, errors.New(errDataFromCert)
	case objectTypeKey:
		return nil, errors.New(errDataFromKey)
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
	// If no serviceAccountRef was provided
	// we expect certain env vars to be present.
	// They are set by the azure workload identity webhook
	// by adding the label `azure.workload.identity/use: "true"` to the external-secrets pod
	if a.provider.ServiceAccountRef == nil {
		clientID := os.Getenv("AZURE_CLIENT_ID")
		tenantID := os.Getenv("AZURE_TENANT_ID")
		tokenFilePath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
		if clientID == "" || tenantID == "" || tokenFilePath == "" {
			return nil, errors.New(errMissingWorkloadEnvVars)
		}
		token, err := os.ReadFile(filepath.Clean(tokenFilePath))
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
	if a.store.GetKind() == esv1.ClusterSecretStoreKind && a.provider.ServiceAccountRef.Namespace != nil {
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
	// Extract clientID
	var clientID string
	// First check if AuthSecretRef is set and clientID can be fetched from there
	if a.provider.AuthSecretRef != nil {
		if a.provider.AuthSecretRef.ClientID == nil {
			return nil, errors.New(errMissingClientIDSecret)
		}
		clientID, err = resolvers.SecretKeyRef(
			ctx,
			a.crClient,
			a.store.GetKind(),
			a.namespace, a.provider.AuthSecretRef.ClientID)
		if err != nil {
			return nil, err
		}
	}
	// If AuthSecretRef is not set, use default (Service Account) implementation
	// Try to get clientID from Annotations
	if len(sa.ObjectMeta.Annotations) > 0 {
		if val, found := sa.ObjectMeta.Annotations[AnnotationClientID]; found {
			// If clientID is defined in both Annotations and AuthSecretRef, return an error
			if clientID != "" {
				return nil, errors.New(errMultipleClientID)
			}
			clientID = val
		}
	}
	// Return an error if clientID is still empty
	if clientID == "" {
		return nil, fmt.Errorf(errMissingClient, AnnotationClientID)
	}
	// Extract tenantID
	var tenantID string
	// First check if AuthSecretRef is set and tenantID can be fetched from there
	if a.provider.AuthSecretRef != nil {
		// We may want to set tenantID explicitly in the `spec.provider.azurekv` section of the SecretStore object
		// So that is okay if it is not there
		if a.provider.AuthSecretRef.TenantID != nil {
			tenantID, err = resolvers.SecretKeyRef(
				ctx,
				a.crClient,
				a.store.GetKind(),
				a.namespace, a.provider.AuthSecretRef.TenantID)
			if err != nil {
				return nil, err
			}
		}
	}
	// Check if spec.provider.azurekv.tenantID is set
	if tenantID == "" && a.provider.TenantID != nil {
		tenantID = *a.provider.TenantID
	}
	// Try to get tenantID from Annotations first. Default implementation.
	if len(sa.ObjectMeta.Annotations) > 0 {
		if val, found := sa.ObjectMeta.Annotations[AnnotationTenantID]; found {
			// If tenantID is defined in both Annotations and AuthSecretRef, return an error
			if tenantID != "" {
				return nil, errors.New(errMultipleTenantID)
			}
			tenantID = val
		}
	}
	// Fallback: use the AZURE_TENANT_ID env var which is set by the azure workload identity webhook
	// https://azure.github.io/azure-workload-identity/docs/topics/service-account-labels-and-annotations.html#service-account
	if tenantID == "" {
		tenantID = os.Getenv("AZURE_TENANT_ID")
	}
	// Return an error if tenantID is still empty
	if tenantID == "" {
		return nil, errors.New(errMissingTenant)
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
	cClient, err := confidential.New(fmt.Sprintf("%s%s", aadEndpoint, tenantID), clientID, cred)
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
		return nil, errors.New(errMissingTenant)
	}
	if a.provider.AuthSecretRef == nil {
		return nil, errors.New(errMissingSecretRef)
	}
	if a.provider.AuthSecretRef.ClientID == nil || (a.provider.AuthSecretRef.ClientSecret == nil && a.provider.AuthSecretRef.ClientCertificate == nil) {
		return nil, errors.New(errMissingClientIDSecret)
	}
	if a.provider.AuthSecretRef.ClientSecret != nil && a.provider.AuthSecretRef.ClientCertificate != nil {
		return nil, errors.New(errInvalidClientCredentials)
	}

	return a.getAuthorizerFromCredentials(ctx)
}

func (a *Azure) getAuthorizerFromCredentials(ctx context.Context) (autorest.Authorizer, error) {
	clientID, err := resolvers.SecretKeyRef(
		ctx,
		a.crClient,
		a.store.GetKind(),
		a.namespace, a.provider.AuthSecretRef.ClientID,
	)

	if err != nil {
		return nil, err
	}

	if a.provider.AuthSecretRef.ClientSecret != nil {
		clientSecret, err := resolvers.SecretKeyRef(
			ctx,
			a.crClient,
			a.store.GetKind(),
			a.namespace, a.provider.AuthSecretRef.ClientSecret,
		)

		if err != nil {
			return nil, err
		}

		return getAuthorizerForClientSecret(
			clientID,
			clientSecret,
			*a.provider.TenantID,
			a.provider.EnvironmentType,
		)
	} else {
		clientCertificate, err := resolvers.SecretKeyRef(
			ctx,
			a.crClient,
			a.store.GetKind(),
			a.namespace, a.provider.AuthSecretRef.ClientCertificate,
		)

		if err != nil {
			return nil, err
		}

		return getAuthorizerForClientCertificate(
			clientID,
			[]byte(clientCertificate),
			*a.provider.TenantID,
			a.provider.EnvironmentType,
		)
	}
}

func getAuthorizerForClientSecret(clientID, clientSecret, tenantID string, environmentType esv1.AzureEnvironmentType) (autorest.Authorizer, error) {
	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	clientCredentialsConfig.Resource = kvResourceForProviderConfig(environmentType)
	clientCredentialsConfig.AADEndpoint = AadEndpointForType(environmentType)
	return clientCredentialsConfig.Authorizer()
}

func getAuthorizerForClientCertificate(clientID string, certificateBytes []byte, tenantID string, environmentType esv1.AzureEnvironmentType) (autorest.Authorizer, error) {
	clientCertificateConfig := NewClientInMemoryCertificateConfig(clientID, certificateBytes, tenantID)
	clientCertificateConfig.Resource = kvResourceForProviderConfig(environmentType)
	clientCertificateConfig.AADEndpoint = AadEndpointForType(environmentType)
	return clientCertificateConfig.Authorizer()
}

func (a *Azure) Close(_ context.Context) error {
	return nil
}

func (a *Azure) Validate() (esv1.ValidationResult, error) {
	if a.store.GetKind() == esv1.ClusterSecretStoreKind && isReferentSpec(a.provider) {
		return esv1.ValidationResultUnknown, nil
	}
	return esv1.ValidationResultReady, nil
}

func isReferentSpec(prov *esv1.AzureKVProvider) bool {
	if prov.AuthSecretRef != nil &&
		((prov.AuthSecretRef.ClientID != nil &&
			prov.AuthSecretRef.ClientID.Namespace == nil) ||
			(prov.AuthSecretRef.ClientSecret != nil &&
				prov.AuthSecretRef.ClientSecret.Namespace == nil)) {
		return true
	}
	if prov.ServiceAccountRef != nil &&
		prov.ServiceAccountRef.Namespace == nil {
		return true
	}
	return false
}

func AadEndpointForType(t esv1.AzureEnvironmentType) string {
	switch t {
	case esv1.AzureEnvironmentPublicCloud:
		return azure.PublicCloud.ActiveDirectoryEndpoint
	case esv1.AzureEnvironmentChinaCloud:
		return azure.ChinaCloud.ActiveDirectoryEndpoint
	case esv1.AzureEnvironmentUSGovernmentCloud:
		return azure.USGovernmentCloud.ActiveDirectoryEndpoint
	case esv1.AzureEnvironmentGermanCloud:
		return azure.GermanCloud.ActiveDirectoryEndpoint
	default:
		return azure.PublicCloud.ActiveDirectoryEndpoint
	}
}

func ServiceManagementEndpointForType(t esv1.AzureEnvironmentType) string {
	switch t {
	case esv1.AzureEnvironmentPublicCloud:
		return azure.PublicCloud.ServiceManagementEndpoint
	case esv1.AzureEnvironmentChinaCloud:
		return azure.ChinaCloud.ServiceManagementEndpoint
	case esv1.AzureEnvironmentUSGovernmentCloud:
		return azure.USGovernmentCloud.ServiceManagementEndpoint
	case esv1.AzureEnvironmentGermanCloud:
		return azure.GermanCloud.ServiceManagementEndpoint
	default:
		return azure.PublicCloud.ServiceManagementEndpoint
	}
}

func kvResourceForProviderConfig(t esv1.AzureEnvironmentType) string {
	var res string
	switch t {
	case esv1.AzureEnvironmentPublicCloud:
		res = azure.PublicCloud.KeyVaultEndpoint
	case esv1.AzureEnvironmentChinaCloud:
		res = azure.ChinaCloud.KeyVaultEndpoint
	case esv1.AzureEnvironmentUSGovernmentCloud:
		res = azure.USGovernmentCloud.KeyVaultEndpoint
	case esv1.AzureEnvironmentGermanCloud:
		res = azure.GermanCloud.KeyVaultEndpoint
	default:
		res = azure.PublicCloud.KeyVaultEndpoint
	}
	return strings.TrimSuffix(res, "/")
}

func getObjType(ref esv1.ExternalSecretDataRemoteRef) (string, string) {
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

func isValidSecret(checkTags, checkName bool, ref esv1.ExternalSecretFind, secret keyvault.SecretItem) (bool, string) {
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

func okByName(ref esv1.ExternalSecretFind, secretName string) bool {
	matches, _ := regexp.MatchString(ref.Name.RegExp, secretName)
	return matches
}

func okByTags(ref esv1.ExternalSecretFind, secret keyvault.SecretItem) bool {
	tagsFound := true
	for k, v := range ref.Tags {
		if val, ok := secret.Tags[k]; !ok || *val != v {
			tagsFound = false
			break
		}
	}
	return tagsFound
}
