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
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"golang.org/x/crypto/pkcs12"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

// Provider satisfies the provider interface.
type Provider struct{}

// interface to keyvault.BaseClient.
type SecretClient interface {
	GetKey(ctx context.Context, vaultBaseURL string, keyName string, keyVersion string) (result keyvault.KeyBundle, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	GetSecretsComplete(ctx context.Context, vaultBaseURL string, maxresults *int32) (result keyvault.SecretListResultIterator, err error)
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
	objectType := "secret"
	basicClient := a.baseClient
	secretValue := ""

	if ref.Version != "" {
		version = ref.Version
	}

	secretName := ref.Key
	nameSplitted := strings.Split(secretName, "/")

	if len(nameSplitted) > 1 {
		objectType = nameSplitted[0]
		secretName = nameSplitted[1]
		// Shall we neglect any later tokens or raise an error ??
	}

	switch objectType {
	case "secret":
		secretResp, err := basicClient.GetSecret(context.Background(), a.vaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		secretValue = *secretResp.Value

	case "cert":
		secretResp, err := basicClient.GetSecret(context.Background(), a.vaultURL, secretName, version)
		if err != nil {
			return nil, err
		}

		if secretResp.ContentType != nil && *secretResp.ContentType == "application/x-pkcs12" {
			secretValue, err = getCertBundleForPKCS(*secretResp.Value)
			// Do we really need to decode PKCS raw value to PEM ? or will that be achieved by the templating features ?
			if err != nil {
				return nil, err
			}
		} else {
			secretValue = *secretResp.Value
		}

	case "key":
		keyResp, err := basicClient.GetKey(context.Background(), a.vaultURL, secretName, version)
		if err != nil {
			return nil, err
		}
		jwk := *keyResp.Key
		// Do we really need to decode JWK raw value to PEM ? or will that be achieved by the templating features ?
		secretValue, err = getPublicKeyFromJwk(jwk)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unknown Azure Keyvault object Type for %s", secretName)
	}

	return []byte(secretValue), nil
}

// Implements store.Client.GetSecretMap Interface.
// retrieve ALL secrets in a specific keyvault.
// ExternalSecretDataRemoteRef Key is mandatory, but with current model we do not use its content.
func (a *Azure) GetSecretMap(ctx context.Context, _ esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	basicClient := a.baseClient
	secretsMap := make(map[string][]byte)

	secretListIter, err := basicClient.GetSecretsComplete(context.Background(), a.vaultURL, nil)
	if err != nil {
		return nil, err
	}
	for secretListIter.NotDone() {
		secretList := secretListIter.Response().Value
		for _, secret := range *secretList {
			if !*secret.Attributes.Enabled {
				continue
			}
			secretName := path.Base(*secret.ID)
			secretResp, err := basicClient.GetSecret(context.Background(), a.vaultURL, secretName, "")
			secretValue := *secretResp.Value

			if err != nil {
				return nil, err
			}
			secretsMap[secretName] = []byte(secretValue)
		}
		err = secretListIter.Next()
		if err != nil {
			return nil, err
		}
	}
	return secretsMap, nil
}

// getCertBundle returns the certificate bundle.
func getCertBundleForPKCS(certificateRawVal string) (bundle string, err error) {
	pfx, err := base64.StdEncoding.DecodeString(certificateRawVal)

	if err != nil {
		return bundle, err
	}
	blocks, _ := pkcs12.ToPEM(pfx, "")

	for _, block := range blocks {
		// no headers
		if block.Type == "PRIVATE KEY" {
			pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				panic(err)
			}
			derStream := x509.MarshalPKCS1PrivateKey(pkey)
			block = &pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: derStream,
			}
		}
		block.Headers = nil
		bundle += string(pem.EncodeToMemory(block))
	}
	return bundle, nil
}

func getPublicKeyFromJwk(jwk keyvault.JSONWebKey) (bundle string, err error) {
	if jwk.Kty != "RSA" {
		return "", fmt.Errorf("invalid key type: %s", jwk.Kty)
	}
	// decode the base64 bytes for n
	nb, err := base64.RawURLEncoding.DecodeString(*jwk.N)
	if err != nil {
		return "", err
	}
	e := 0
	// The default exponent is usually 65537, so just compare the
	// base64 for [1,0,1] or [0,1,0,1]
	if *jwk.E == "AQAB" || *jwk.E == "AAEAAQ" {
		e = 65537
	} else {
		// need to decode "e" as a big-endian int
		return "", fmt.Errorf("need to deocde e: %s", *jwk.E)
	}

	pk := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nb),
		E: e,
	}

	der, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return "", err
	}
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: der,
	}
	var out bytes.Buffer
	err = pem.Encode(&out, block)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func (a *Azure) newAzureClient(ctx context.Context) (*keyvault.BaseClient, string, error) {
	spec := *a.store.GetSpec().Provider.AzureKV
	tenantID := *spec.TenantID
	vaultURL := *spec.VaultURL

	if spec.AuthSecretRef == nil {
		return nil, "", fmt.Errorf("missing clientID/clientSecret in store config")
	}
	scoped := true
	if a.store.GetObjectMeta().String() == "ClusterSecretStore" {
		scoped = false
	}
	if spec.AuthSecretRef.ClientID == nil || spec.AuthSecretRef.ClientSecret == nil {
		return nil, "", fmt.Errorf("missing accessKeyID/secretAccessKey in store config")
	}
	cid, err := a.secretKeyRef(ctx, a.store.GetNamespacedName(), *spec.AuthSecretRef.ClientID, scoped)
	if err != nil {
		return nil, "", err
	}
	csec, err := a.secretKeyRef(ctx, a.store.GetNamespacedName(), *spec.AuthSecretRef.ClientSecret, scoped)
	if err != nil {
		return nil, "", err
	}
	os.Setenv("AZURE_TENANT_ID", tenantID)
	os.Setenv("AZURE_CLIENT_ID", cid)
	os.Setenv("AZURE_CLIENT_SECRET", csec)

	authorizer, err := kvauth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, "", err
	}
	os.Unsetenv("AZURE_TENANT_ID")
	os.Unsetenv("AZURE_CLIENT_ID")
	os.Unsetenv("AZURE_CLIENT_SECRET")

	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	return &basicClient, vaultURL, nil
}

func (a *Azure) secretKeyRef(ctx context.Context, namespace string, secretRef smmeta.SecretKeySelector, scoped bool) (string, error) {
	var secret corev1.Secret
	ref := types.NamespacedName{
		Namespace: namespace,
		Name:      secretRef.Name,
	}
	if !scoped && secretRef.Namespace != nil {
		ref.Namespace = *secretRef.Namespace
	}
	err := a.kube.Get(ctx, ref, &secret)
	if err != nil {
		return "", err
	}
	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf("no data for %q in secret '%s/%s'", secretRef.Key, secretRef.Name, namespace)
	}
	value := strings.TrimSpace(string(keyBytes))
	return value, nil
}
