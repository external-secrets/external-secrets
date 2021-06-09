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
	"testing"

	tassert "github.com/stretchr/testify/assert"
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
	tassert.EqualError(t, err, "secrets \"user\" not found")
	tassert.Nil(t, secretClient)
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
	rf := esv1alpha1.ExternalSecretDataRemoteRef{}
	azureMock.AddSecret(testAzure.vaultURL, "testName", "My Secret", true)
	azureMock.ExpectsGetSecretsComplete(ctx, testAzure.vaultURL, nil)
	azureMock.ExpectsGetSecret(ctx, testAzure.vaultURL, "testName", "")
	secretMap, err := testAzure.GetSecretMap(ctx, rf)
	azureMock.AssertExpectations(t)
	tassert.Nil(t, err, "the return err should be nil")
	tassert.Equal(t, secretMap, map[string][]byte{"testName": []byte("My Secret")})
}

func TestGetSecretMapNotEnabled(t *testing.T) {
	testAzure, azureMock := newAzure()
	ctx := context.Background()
	rf := esv1alpha1.ExternalSecretDataRemoteRef{}
	azureMock.AddSecret(testAzure.vaultURL, "testName", "My Secret", false)
	azureMock.ExpectsGetSecretsComplete(ctx, testAzure.vaultURL, nil)
	secretMap, err := testAzure.GetSecretMap(ctx, rf)
	azureMock.AssertExpectations(t)
	tassert.Nil(t, err, "the return err should be nil")
	tassert.Empty(t, secretMap)
}

func TestGetCertBundleForPKCS(t *testing.T) {
	rawCertExample := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURC" +
		"VENDQWUyZ0F3SUJBZ0lFUnIxWTdEQU5CZ2txaGtpRzl3MEJBUVVGQURBeU1Rc3d" +
		"DUVlEVlFRR0V3SkUKUlRFUU1BNEdBMVVFQ2hNSFFXMWhaR1YxY3pFUk1BOEdBMV" +
		"VFQXhNSVUwRlFJRkp2YjNRd0hoY05NVE13TWpFMApNVE15TmpRNVdoY05NelV4T" +
		"WpNeE1UTXlOalE1V2pBeU1Rc3dDUVlEVlFRR0V3SkVSVEVRTUE0R0ExVUVDaE1I" +
		"CnFWUlE3NjNGODFwWnorNXgyejJ6NmZyd0JHNUF3YUZKL1RmTE9HQzZQWnl5bW1" +
		"pSlllL2tjUDdVeUhMQnBUUVkKLzloNTF5dDB5NlRBS1JmRk1wMlhuVUZBaWdyL0" +
		"0xYVc1NjdORStQYzN5S0RWWlVHdU82UXZ0cExCZkpPS3pZSAowc3F3OElmYjRlN" +
		"0R6TkJuTmRoVDhzbGdUYkh5K3RzZUtPb0xHNi9rUktmRmRvSmRoeHAzeGNnbm56" +
		"ZkY0anUvCi9UZTRYaWsxNC9FMAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t"
	c, ok := getCertBundleForPKCS(rawCertExample)
	bundle := ""
	tassert.Nil(t, ok)
	tassert.Equal(t, c, bundle)
}
