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
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

const (
	defaultObjType = "secret"
)

// Provider satisfies the provider interface.
type Provider struct{}

// interface to keyvault.BaseClient.
type SecretClient interface {
	GetKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string) (result keyvault.KeyBundle, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	GetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error)
	GetCertificate(ctx context.Context, vaultBaseURL string, certificateName string, certificateVersion string) (result keyvault.CertificateBundle, err error)
}

// Azure satisfies the provider.SecretsClient interface.
type Azure struct {
	kube       client.Client
	store      esv1alpha1.GenericStore
	baseClient SecretClient
	vaultURL   string
	namespace  string
}

func init() {
	schema.Register(&Provider{}, &esv1alpha1.SecretStoreProvider{
		AzureKV: &esv1alpha1.AzureKVProvider{},
	})
}

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	anAzure := &Azure{
		kube:      kube,
		store:     store,
		namespace: namespace,
	}
	azClient, vaultURL, err := anAzure.newAzureClient(ctx)

	if err != nil {
		return nil, err
	}

	anAzure.baseClient = azClient
	anAzure.vaultURL = vaultURL
	return anAzure, nil
}

// Implements store.Client.GetSecret Interface.
// Retrieves a secret/Key/Certificate with the secret name defined in ref.Name
// The Object Type is defined as a prefix in the ref.Name , if no prefix is defined , we assume a secret is required.
func (a *Azure) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	version := ""
	basicClient := a.baseClient
	objectType, secretName := getObjType(ref)

	if ref.Version != "" {
		version = ref.Version
	}

	switch objectType {
	case defaultObjType:
		// returns a SecretBundle with the secret value
		// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#SecretBundle
		secretResp, err := basicClient.GetSecret(context.Background(), a.vaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		if ref.Property == "" {
			return []byte(*secretResp.Value), nil
		}
		res := gjson.Get(*secretResp.Value, ref.Property)
		if !res.Exists() {
			return nil, fmt.Errorf("property %s does not exist in key %s", ref.Property, ref.Key)
		}
		return []byte(res.String()), err
	case "cert":
		// returns a CertBundle. We return CER contents of x509 certificate
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#CertificateBundle
		secretResp, err := basicClient.GetCertificate(context.Background(), a.vaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		return *secretResp.Cer, nil
	case "key":
		// returns a KeyBundla that contains a jwk
		// azure kv returns only public keys
		// see: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault#KeyBundle
		keyResp, err := basicClient.GetKey(context.Background(), a.vaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		return json.Marshal(keyResp.Key)
	}

	return nil, fmt.Errorf("unknown Azure Keyvault object Type for %s", secretName)
}

// Implements store.Client.GetSecretMap Interface.
// New version of GetSecretMap.
func (a *Azure) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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
			return nil, fmt.Errorf("error unmarshalling json data: %w", err)
		}

		secretData := make(map[string][]byte)
		for k, v := range kv {
			secretData[k] = []byte(v)
		}

		return secretData, nil
	case "cert":
		return nil, fmt.Errorf("cannot get use dataFrom to get certificate secret")
	case "key":
		return nil, fmt.Errorf("cannot get use dataFrom to get key secret")
	}

	return nil, fmt.Errorf("unknown Azure Keyvault object Type for %s", secretName)
}

func (a *Azure) newAzureClient(ctx context.Context) (*keyvault.BaseClient, string, error) {
	spec := *a.store.GetSpec().Provider.AzureKV
	tenantID := *spec.TenantID
	vaultURL := *spec.VaultURL

	if spec.AuthSecretRef == nil {
		return nil, "", fmt.Errorf("missing clientID/clientSecret in store config")
	}
	clusterScoped := false
	if a.store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
		clusterScoped = true
	}
	if spec.AuthSecretRef.ClientID == nil || spec.AuthSecretRef.ClientSecret == nil {
		return nil, "", fmt.Errorf("missing accessKeyID/secretAccessKey in store config")
	}
	cid, err := a.secretKeyRef(ctx, a.store.GetNamespace(), *spec.AuthSecretRef.ClientID, clusterScoped)
	if err != nil {
		return nil, "", err
	}
	csec, err := a.secretKeyRef(ctx, a.store.GetNamespace(), *spec.AuthSecretRef.ClientSecret, clusterScoped)
	if err != nil {
		return nil, "", err
	}

	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(cid, csec, tenantID)
	// the default resource api is the management URL and not the vault URL which we need for keyvault operations
	clientCredentialsConfig.Resource = "https://vault.azure.net"
	authorizer, err := clientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, "", err
	}

	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	return &basicClient, vaultURL, nil
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
		return "", fmt.Errorf("could not find secret %s/%s: %w", ref.Namespace, ref.Name, err)
	}
	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf("no data for %q in secret '%s/%s'", secretRef.Key, secretRef.Name, namespace)
	}
	value := strings.TrimSpace(string(keyBytes))
	return value, nil
}

func (a *Azure) Close(ctx context.Context) error {
	return nil
}

func getObjType(ref esv1alpha1.ExternalSecretDataRemoteRef) (string, string) {
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
