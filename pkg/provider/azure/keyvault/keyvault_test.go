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
	context "context"
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fake "github.com/external-secrets/external-secrets/pkg/provider/azure/keyvault/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

func newAzure() (Azure, *fake.AzureMock) {
	azureMock := &fake.AzureMock{}
	testAzure := Azure{
		baseClient: azureMock,
		vaultURL:   "https://local.vault/",
	}
	return testAzure, azureMock
}

func TestNewClientNoCreds(t *testing.T) {
	namespace := "internal"
	vaultURL := "https://local.vault.url"
	tenantID := "1234"
	store := esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1alpha1.SecretStoreSpec{Provider: &esv1alpha1.SecretStoreProvider{AzureKV: &esv1alpha1.AzureKVProvider{
			VaultURL: &vaultURL,
			TenantID: &tenantID,
		}}},
	}
	provider, err := schema.GetProvider(&store)
	tassert.Nil(t, err, "the return err should be nil")
	k8sClient := clientfake.NewClientBuilder().Build()
	secretClient, err := provider.NewClient(context.Background(), &store, k8sClient, namespace)
	tassert.EqualError(t, err, "missing clientID/clientSecret in store config")
	tassert.Nil(t, secretClient)

	store.Spec.Provider.AzureKV.AuthSecretRef = &esv1alpha1.AzureKVAuth{}
	secretClient, err = provider.NewClient(context.Background(), &store, k8sClient, namespace)
	tassert.EqualError(t, err, "missing accessKeyID/secretAccessKey in store config")
	tassert.Nil(t, secretClient)

	store.Spec.Provider.AzureKV.AuthSecretRef.ClientID = &v1.SecretKeySelector{Name: "user"}
	secretClient, err = provider.NewClient(context.Background(), &store, k8sClient, namespace)
	tassert.EqualError(t, err, "missing accessKeyID/secretAccessKey in store config")
	tassert.Nil(t, secretClient)

	store.Spec.Provider.AzureKV.AuthSecretRef.ClientSecret = &v1.SecretKeySelector{Name: "password"}
	secretClient, err = provider.NewClient(context.Background(), &store, k8sClient, namespace)
	tassert.EqualError(t, err, "could not find secret internal/user: secrets \"user\" not found")
	tassert.Nil(t, secretClient)
}

const (
	jwkPubRSA = `{"kid":"ex","kty":"RSA","key_ops":["sign","verify","wrapKey","unwrapKey","encrypt","decrypt"],"n":"p2VQo8qCfWAZmdWBVaYuYb-a-tWWm78K6Sr9poCvNcmv8rUPSLACxitQWR8gZaSH1DklVkqz-Ed8Cdlf8lkDg4Ex5tkB64jRdC1Uvn4CDpOH6cp-N2s8hTFLqy9_YaDmyQS7HiqthOi9oVjil1VMeWfaAbClGtFt6UnKD0Vb_DvLoWYQSqlhgBArFJi966b4E1pOq5Ad02K8pHBDThlIIx7unibLehhDU6q3DCwNH_OOLx6bgNtmvGYJDd1cywpkLQ3YzNCUPWnfMBJRP3iQP_WI21uP6cvo0DqBPBM4wvVzHbCT0vnIflwkbgEWkq1FprqAitZlop9KjLqzjp9vyQ","e":"AQAB"}`
	jwkPubEC  = `{"kid":"https://example.vault.azure.net/keys/ec-p-521/e3d0e9c179b54988860c69c6ae172c65","kty":"EC","key_ops":["sign","verify"],"crv":"P-521","x":"AedOAtb7H7Oz1C_cPKI_R4CN_eai5nteY6KFW07FOoaqgQfVCSkQDK22fCOiMT_28c8LZYJRsiIFz_IIbQUW7bXj","y":"AOnchHnmBphIWXvanmMAmcCDkaED6ycW8GsAl9fQ43BMVZTqcTkJYn6vGnhn7MObizmkNSmgZYTwG-vZkIg03HHs"}`
)

func TestGetKey(t *testing.T) {
	testAzure, azureMock := newAzure()
	ctx := context.Background()

	tbl := []struct {
		name   string
		kvName string
		jwk    *keyvault.JSONWebKey
		out    string
	}{
		{
			name:   "test public rsa key",
			kvName: "my-rsa",
			jwk:    newKVJWK([]byte(jwkPubRSA)),
			out:    jwkPubRSA,
		},
		{
			name:   "test public ec key",
			kvName: "my-ec",
			jwk:    newKVJWK([]byte(jwkPubEC)),
			out:    jwkPubEC,
		},
	}

	for _, row := range tbl {
		t.Run(row.name, func(t *testing.T) {
			azureMock.AddKey(testAzure.vaultURL, row.kvName, row.jwk, true)
			azureMock.ExpectsGetKey(ctx, testAzure.vaultURL, row.kvName, "")

			rf := esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "key/" + row.kvName,
			}
			secret, err := testAzure.GetSecret(ctx, rf)
			azureMock.AssertExpectations(t)
			tassert.Nil(t, err, "the return err should be nil")
			tassert.Equal(t, []byte(row.out), secret)
		})
	}
}

func TestGetSecretWithVersion(t *testing.T) {
	testAzure, azureMock := newAzure()
	ctx := context.Background()
	version := "v1"

	rf := esv1alpha1.ExternalSecretDataRemoteRef{
		Key:     "testName",
		Version: version,
	}
	azureMock.AddSecretWithVersion(testAzure.vaultURL, "testName", version, "My Secret", true)
	azureMock.ExpectsGetSecret(ctx, testAzure.vaultURL, "testName", version)

	secret, err := testAzure.GetSecret(ctx, rf)
	azureMock.AssertExpectations(t)
	tassert.Nil(t, err, "the return err should be nil")
	tassert.Equal(t, []byte("My Secret"), secret)
}

func TestGetSecretWithoutVersion(t *testing.T) {
	testAzure, azureMock := newAzure()
	ctx := context.Background()

	rf := esv1alpha1.ExternalSecretDataRemoteRef{
		Key: "testName",
	}
	azureMock.AddSecret(testAzure.vaultURL, "testName", "My Secret", true)
	azureMock.ExpectsGetSecret(ctx, testAzure.vaultURL, "testName", "")

	secret, err := testAzure.GetSecret(ctx, rf)
	azureMock.AssertExpectations(t)
	tassert.Nil(t, err, "the return err should be nil")
	tassert.Equal(t, []byte("My Secret"), secret)
}

func TestGetSecretMap(t *testing.T) {
	testAzure, azureMock := newAzure()
	ctx := context.Background()
	rf := esv1alpha1.ExternalSecretDataRemoteRef{
		Key: "testName",
	}
	azureMock.AddSecret(testAzure.vaultURL, "testName", "{\"username\": \"user1\", \"pass\": \"123\"}", true)
	azureMock.ExpectsGetSecret(ctx, testAzure.vaultURL, "testName", "")
	secretMap, err := testAzure.GetSecretMap(ctx, rf)
	azureMock.AssertExpectations(t)
	tassert.Nil(t, err, "the return err should be nil")
	tassert.Equal(t, secretMap, map[string][]byte{"username": []byte("user1"), "pass": []byte("123")})
}

func newKVJWK(b []byte) *keyvault.JSONWebKey {
	var key keyvault.JSONWebKey
	err := json.Unmarshal(b, &key)
	if err != nil {
		panic(err)
	}
	return &key
}
