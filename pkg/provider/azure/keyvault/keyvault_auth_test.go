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
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	awsauthfake "github.com/external-secrets/external-secrets/pkg/provider/aws/auth/fake"
)

var vaultURL = "https://local.vault.url"

func TestNewClientManagedIdentityNoNeedForCredentials(t *testing.T) {
	namespace := "internal"
	identityID := "1234"
	authType := esv1beta1.AzureManagedIdentity
	store := esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1beta1.SecretStoreSpec{Provider: &esv1beta1.SecretStoreProvider{AzureKV: &esv1beta1.AzureKVProvider{
			AuthType:   &authType,
			IdentityID: &identityID,
			VaultURL:   &vaultURL,
		}}},
	}
	k8sClient := clientfake.NewClientBuilder().Build()
	az := &Azure{
		crClient:  k8sClient,
		namespace: namespace,
		provider:  store.Spec.Provider.AzureKV,
		store:     &store,
	}
	authorizer, err := az.authorizerForManagedIdentity()
	if err != nil {
		// On non Azure environment, MSI auth not available, so this error should be returned
		tassert.EqualError(t, err, "failed to get oauth token from MSI: MSI not available")
	} else {
		// On Azure (where GitHub Actions are running) a secretClient is returned, as only an Authorizer is configured, but no token is requested for MI
		tassert.NotNil(t, authorizer)
	}
}

func TestGetAuthorizorForWorkloadIdentity(t *testing.T) {
	const (
		tenantID      = "my-tenant-id"
		clientID      = "my-client-id"
		azAccessToken = "my-access-token"
		saToken       = "FAKETOKEN"
		saName        = "az-wi"
		namespace     = "default"
	)

	// create a temporary file to imitate
	// azure workload identity webhook
	// see AZURE_FEDERATED_TOKEN_FILE
	tf, err := os.CreateTemp("", "")
	tassert.Nil(t, err)
	defer os.RemoveAll(tf.Name())
	_, err = tf.WriteString(saToken)
	tassert.Nil(t, err)
	tokenFile := tf.Name()

	authType := esv1beta1.AzureWorkloadIdentity
	defaultProvider := &esv1beta1.AzureKVProvider{
		VaultURL: &vaultURL,
		AuthType: &authType,
		ServiceAccountRef: &v1.ServiceAccountSelector{
			Name: saName,
		},
	}

	type testCase struct {
		name       string
		provider   *esv1beta1.AzureKVProvider
		k8sObjects []client.Object
		prep       func()
		cleanup    func()
		expErr     string
	}

	for _, row := range []testCase{
		{
			name:     "missing service account",
			provider: defaultProvider,
			expErr:   "serviceaccounts \"" + saName + "\" not found",
		},
		{
			name:     "missing webhook env vars",
			provider: &esv1beta1.AzureKVProvider{},
			expErr:   "missing environment variables. AZURE_CLIENT_ID, AZURE_TENANT_ID and AZURE_FEDERATED_TOKEN_FILE must be set",
		},
		{
			name:     "missing workload identity token file",
			provider: &esv1beta1.AzureKVProvider{},
			prep: func() {
				os.Setenv("AZURE_CLIENT_ID", clientID)
				os.Setenv("AZURE_TENANT_ID", tenantID)
				os.Setenv("AZURE_FEDERATED_TOKEN_FILE", "invalid file")
			},
			cleanup: func() {
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_TENANT_ID")
				os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")
			},
			expErr: "unable to read token file invalid file: open invalid file: no such file or directory",
		},
		{
			name:     "correct workload identity",
			provider: &esv1beta1.AzureKVProvider{},
			prep: func() {
				os.Setenv("AZURE_CLIENT_ID", clientID)
				os.Setenv("AZURE_TENANT_ID", tenantID)
				os.Setenv("AZURE_FEDERATED_TOKEN_FILE", tokenFile)
			},
			cleanup: func() {
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_TENANT_ID")
				os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")
			},
		},
		{
			name:     "missing sa annotations",
			provider: defaultProvider,
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:        saName,
						Namespace:   namespace,
						Annotations: map[string]string{},
					},
				},
			},
			expErr: "missing service account annotation: azure.workload.identity/client-id",
		},
		{
			name:     "successful case",
			provider: defaultProvider,
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
						Annotations: map[string]string{
							annotationClientID: clientID,
							annotationTenantID: tenantID,
						},
					},
				},
			},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			store := esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{Provider: &esv1beta1.SecretStoreProvider{
					AzureKV: row.provider,
				}},
			}
			k8sClient := clientfake.NewClientBuilder().
				WithObjects(row.k8sObjects...).
				Build()
			az := &Azure{
				store:      &store,
				namespace:  namespace,
				crClient:   k8sClient,
				kubeClient: awsauthfake.NewCreateTokenMock(saToken),
				provider:   store.Spec.Provider.AzureKV,
			}
			tokenProvider := func(ctx context.Context, token, clientID, tenantID string) (adal.OAuthTokenProvider, error) {
				tassert.Equal(t, token, saToken)
				tassert.Equal(t, clientID, clientID)
				tassert.Equal(t, tenantID, tenantID)
				return &tokenProvider{accessToken: azAccessToken}, nil
			}
			if row.prep != nil {
				row.prep()
			}
			if row.cleanup != nil {
				defer row.cleanup()
			}
			authorizer, err := az.authorizerForWorkloadIdentity(context.Background(), tokenProvider)
			if row.expErr == "" {
				tassert.NotNil(t, authorizer)
				tassert.Equal(t, getTokenFromAuthorizer(t, authorizer), azAccessToken)
			} else {
				tassert.EqualError(t, err, row.expErr)
			}
		})
	}
}

func TestAuth(t *testing.T) {
	defaultStore := esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{},
		},
	}
	authType := esv1beta1.AzureServicePrincipal

	type testCase struct {
		name     string
		provider *esv1beta1.AzureKVProvider
		store    esv1beta1.GenericStore
		objects  []client.Object
		expErr   string
	}
	for _, row := range []testCase{
		{
			name:   "bad config",
			expErr: "missing secretRef in provider config",
			store:  &defaultStore,
			provider: &esv1beta1.AzureKVProvider{
				AuthType: &authType,
				VaultURL: &vaultURL,
				TenantID: pointer.StringPtr("mytenant"),
			},
		},
		{
			name:   "bad config",
			expErr: "missing accessKeyID/secretAccessKey in store config",
			store:  &defaultStore,
			provider: &esv1beta1.AzureKVProvider{
				AuthType:      &authType,
				VaultURL:      &vaultURL,
				TenantID:      pointer.StringPtr("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{},
			},
		},
		{
			name:   "bad config: missing secret",
			expErr: "could not find secret default/password: secrets \"password\" not found",
			store:  &defaultStore,
			provider: &esv1beta1.AzureKVProvider{
				AuthType: &authType,
				VaultURL: &vaultURL,
				TenantID: pointer.StringPtr("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientSecret: &v1.SecretKeySelector{Name: "password"},
					ClientID:     &v1.SecretKeySelector{Name: "password"},
				},
			},
		},
		{
			name:   "cluster secret store",
			expErr: "could not find secret foo/password: secrets \"password\" not found",
			store: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1beta1.ClusterSecretStoreKind,
				},
				Spec: esv1beta1.SecretStoreSpec{Provider: &esv1beta1.SecretStoreProvider{}},
			},
			provider: &esv1beta1.AzureKVProvider{
				AuthType: &authType,
				VaultURL: &vaultURL,
				TenantID: pointer.StringPtr("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientSecret: &v1.SecretKeySelector{Name: "password", Namespace: pointer.StringPtr("foo")},
					ClientID:     &v1.SecretKeySelector{Name: "password", Namespace: pointer.StringPtr("foo")},
				},
			},
		},
		{
			name: "correct cluster secret store",
			objects: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "password",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"id":     []byte("foo"),
					"secret": []byte("bar"),
				},
			}},
			store: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1beta1.ClusterSecretStoreKind,
				},
				Spec: esv1beta1.SecretStoreSpec{Provider: &esv1beta1.SecretStoreProvider{}},
			},
			provider: &esv1beta1.AzureKVProvider{
				AuthType: &authType,
				VaultURL: &vaultURL,
				TenantID: pointer.StringPtr("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientSecret: &v1.SecretKeySelector{Name: "password", Namespace: pointer.StringPtr("foo"), Key: "secret"},
					ClientID:     &v1.SecretKeySelector{Name: "password", Namespace: pointer.StringPtr("foo"), Key: "id"},
				},
			},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			k8sClient := clientfake.NewClientBuilder().WithObjects(row.objects...).Build()
			spec := row.store.GetSpec()
			spec.Provider.AzureKV = row.provider
			az := &Azure{
				crClient:  k8sClient,
				namespace: "default",
				provider:  spec.Provider.AzureKV,
				store:     row.store,
			}
			authorizer, err := az.authorizerForServicePrincipal(context.Background())
			if row.expErr == "" {
				tassert.Nil(t, err)
				tassert.NotNil(t, authorizer)
			} else {
				tassert.EqualError(t, err, row.expErr)
			}
		})
	}
}

func getTokenFromAuthorizer(t *testing.T, authorizer autorest.Authorizer) string {
	rq, _ := http.NewRequest("POST", "http://example.com", http.NoBody)
	_, err := authorizer.WithAuthorization()(
		autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			return rq, nil
		})).Prepare(rq)
	tassert.Nil(t, err)
	return strings.TrimPrefix(rq.Header.Get("Authorization"), "Bearer ")
}
