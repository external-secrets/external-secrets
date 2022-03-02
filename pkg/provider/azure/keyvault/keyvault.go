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
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

const (
	defaultObjType = "secret"
	objectTypeCert = "cert"
	objectTypeKey  = "key"
	vaultResource  = "https://vault.azure.net"

	errUnexpectedStoreSpec   = "unexpected store spec"
	errMissingAuthType       = "cannot initialize Azure Client: no valid authType was specified"
	errPropNotExist          = "property %s does not exist in key %s"
	errUnknownObjectType     = "unknown Azure Keyvault object Type for %s"
	errUnmarshalJSONData     = "error unmarshalling json data: %w"
	errDataFromCert          = "cannot get use dataFrom to get certificate secret"
	errDataFromKey           = "cannot get use dataFrom to get key secret"
	errMissingTenant         = "missing tenantID in store config"
	errMissingSecretRef      = "missing secretRef in provider config"
	errMissingClientIDSecret = "missing accessKeyID/secretAccessKey in store config"
	errFindSecret            = "could not find secret %s/%s: %w"
	errFindDataKey           = "no data for %q in secret '%s/%s'"
)

// interface to keyvault.BaseClient.
type SecretClient interface {
	GetKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string) (result keyvault.KeyBundle, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	GetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error)
	GetCertificate(ctx context.Context, vaultBaseURL string, certificateName string, certificateVersion string) (result keyvault.CertificateBundle, err error)
}

type Azure struct {
	kube       client.Client
	store      esv1beta1.GenericStore
	provider   *esv1beta1.AzureKVProvider
	baseClient SecretClient
	namespace  string
}

func init() {
	schema.Register(&Azure{}, &esv1beta1.SecretStoreProvider{
		AzureKV: &esv1beta1.AzureKVProvider{},
	})
}

// NewClient constructs a new secrets client based on the provided store.
func (a *Azure) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	az := &Azure{
		kube:      kube,
		store:     store,
		namespace: namespace,
		provider:  provider,
	}

	ok, err := az.setAzureClientWithManagedIdentity()
	if ok {
		return az, err
	}

	ok, err = az.setAzureClientWithServicePrincipal(ctx)
	if ok {
		return az, err
	}

	return nil, fmt.Errorf(errMissingAuthType)
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.AzureKVProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider.AzureKV == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}

	return spc.Provider.AzureKV, nil
}

// Empty GetAllSecrets.
func (a *Azure) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// Implements store.Client.GetSecret Interface.
// Retrieves a secret/Key/Certificate with the secret name defined in ref.Name
// The Object Type is defined as a prefix in the ref.Name , if no prefix is defined , we assume a secret is required.
func (a *Azure) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	version := ""
	objectType, secretName := getObjType(ref)

	if ref.Version != "" {
		version = ref.Version
	}

	switch objectType {
	case defaultObjType:
		// returns a SecretBundle with the secret value
		// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#SecretBundle
		secretResp, err := a.baseClient.GetSecret(context.Background(), *a.provider.VaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		if ref.Property == "" {
			return []byte(*secretResp.Value), nil
		}
		res := gjson.Get(*secretResp.Value, ref.Property)
		if !res.Exists() {
			return nil, fmt.Errorf(errPropNotExist, ref.Property, ref.Key)
		}
		return []byte(res.String()), err
	case objectTypeCert:
		// returns a CertBundle. We return CER contents of x509 certificate
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#CertificateBundle
		secretResp, err := a.baseClient.GetCertificate(context.Background(), *a.provider.VaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		return *secretResp.Cer, nil
	case objectTypeKey:
		// returns a KeyBundle that contains a jwk
		// azure kv returns only public keys
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#KeyBundle
		keyResp, err := a.baseClient.GetKey(context.Background(), *a.provider.VaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		return json.Marshal(keyResp.Key)
	}

	return nil, fmt.Errorf(errUnknownObjectType, secretName)
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

		kv := make(map[string]string)
		err = json.Unmarshal(data, &kv)
		if err != nil {
			return nil, fmt.Errorf(errUnmarshalJSONData, err)
		}

		secretData := make(map[string][]byte)
		for k, v := range kv {
			secretData[k] = []byte(v)
		}

		return secretData, nil
	case objectTypeCert:
		return nil, fmt.Errorf(errDataFromCert)
	case objectTypeKey:
		return nil, fmt.Errorf(errDataFromKey)
	}

	return nil, fmt.Errorf(errUnknownObjectType, secretName)
}

func (a *Azure) setAzureClientWithManagedIdentity() (bool, error) {
	if *a.provider.AuthType != esv1beta1.ManagedIdentity {
		return false, nil
	}

	msiConfig := kvauth.NewMSIConfig()
	msiConfig.Resource = vaultResource
	if a.provider.IdentityID != nil {
		msiConfig.ClientID = *a.provider.IdentityID
	}
	authorizer, err := msiConfig.Authorizer()
	if err != nil {
		return true, err
	}

	cl := keyvault.New()
	cl.Authorizer = authorizer
	a.baseClient = &cl
	return true, nil
}

func (a *Azure) setAzureClientWithServicePrincipal(ctx context.Context) (bool, error) {
	if *a.provider.AuthType != esv1beta1.ServicePrincipal {
		return false, nil
	}

	if a.provider.TenantID == nil {
		return true, fmt.Errorf(errMissingTenant)
	}
	if a.provider.AuthSecretRef == nil {
		return true, fmt.Errorf(errMissingSecretRef)
	}
	if a.provider.AuthSecretRef.ClientID == nil || a.provider.AuthSecretRef.ClientSecret == nil {
		return true, fmt.Errorf(errMissingClientIDSecret)
	}
	clusterScoped := false
	if a.store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		clusterScoped = true
	}
	cid, err := a.secretKeyRef(ctx, a.store.GetNamespace(), *a.provider.AuthSecretRef.ClientID, clusterScoped)
	if err != nil {
		return true, err
	}
	csec, err := a.secretKeyRef(ctx, a.store.GetNamespace(), *a.provider.AuthSecretRef.ClientSecret, clusterScoped)
	if err != nil {
		return true, err
	}

	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(cid, csec, *a.provider.TenantID)
	clientCredentialsConfig.Resource = vaultResource
	authorizer, err := clientCredentialsConfig.Authorizer()
	if err != nil {
		return true, err
	}

	cl := keyvault.New()
	cl.Authorizer = authorizer
	a.baseClient = &cl
	return true, nil
}

func (a *Azure) secretKeyRef(ctx context.Context, namespace string, secretRef smmeta.SecretKeySelector, clusterScoped bool) (string, error) {
	var secret corev1.Secret
	ref := types.NamespacedName{
		Namespace: namespace,
		Name:      secretRef.Name,
	}
	if clusterScoped && secretRef.Namespace != nil {
		ref.Namespace = *secretRef.Namespace
	}
	err := a.kube.Get(ctx, ref, &secret)
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

func (a *Azure) Validate() error {
	return nil
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
