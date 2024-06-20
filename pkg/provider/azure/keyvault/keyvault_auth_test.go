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
	pointer "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	utilfake "github.com/external-secrets/external-secrets/pkg/provider/util/fake"
)

var vaultURL = "https://local.vault.url"

var mockCertificate = `
-----BEGIN CERTIFICATE-----
MIICBzCCAbGgAwIBAgIUSoCD1fgywDbmeRaGrkYzGWUd1wMwDQYJKoZIhvcNAQEL
BQAwcTELMAkGA1UEBhMCQVoxGTAXBgNVBAgMEE1vY2sgQ2VydGlmaWNhdGUxMzAx
BgNVBAoMKkV4dGVybmFsIFNlY3JldHMgT3BlcmF0b3IgTW9jayBDZXJ0aWZpY2F0
ZTESMBAGA1UEAwwJTW9jayBDZXJ0MB4XDTI0MDUwODA4NDkzMFoXDTI1MDUwODA4
NDkzMFowcTELMAkGA1UEBhMCQVoxGTAXBgNVBAgMEE1vY2sgQ2VydGlmaWNhdGUx
MzAxBgNVBAoMKkV4dGVybmFsIFNlY3JldHMgT3BlcmF0b3IgTW9jayBDZXJ0aWZp
Y2F0ZTESMBAGA1UEAwwJTW9jayBDZXJ0MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJB
ALkU1YgMk1Dk149F/HsHA0TjzLwfDa9tT0cfqA1u0hoJkb2r9jdWUyiugGaEz/PU
TGWrvp8aiXPrGuu5Y6PY27ECAwEAAaMhMB8wHQYDVR0OBBYEFAMB0YwnYjUm00og
kGce8Yhr4I03MA0GCSqGSIb3DQEBCwUAA0EAr0BMs/3hIOdZc0WHZUCTZ0GGor3G
ViYUPHOw8z6UZGPGN6qiAejmkT6uP3LkkSW+7TIIQ1pkQxcn5xfFJXBexw==
-----END CERTIFICATE-----
-----BEGIN PRIVATE KEY-----
MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEAuRTViAyTUOTXj0X8
ewcDROPMvB8Nr21PRx+oDW7SGgmRvav2N1ZTKK6AZoTP89RMZau+nxqJc+sa67lj
o9jbsQIDAQABAkA35CnDpwCJykGqW5kuUeTT1fMK0FnioyDwuoeWXuQFxmB6Md89
+ABxyjAt3nmwRRVBrVFdNibb9asR5KFHwn1NAiEA4NlrSnJrY1xODIjEXf0fLTwu
wpyUO1lX585OjYDiOYsCIQDSuP4ttH/1Hg3f9veEE4RgDEk+QcisrzF8q4Oa5sDP
MwIgfejiTtcR0ZsPza8Mn0EuIyuPV8VMsItQUWtSy6R/ig8CIQC86cBmNUXp+HGz
8fLg46ZvfVREjjFcLwwMmq83tdvxZQIgPAbezuRCrduH19xgMO8BXndS5DAovgvE
/MpQnEyQtVA=
-----END PRIVATE KEY-----
`

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
		secretName    = "mi-spec"
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
		prep       func(*testing.T)
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
			prep: func(t *testing.T) {
				t.Setenv("AZURE_CLIENT_ID", clientID)
				t.Setenv("AZURE_TENANT_ID", tenantID)
				t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "invalid file")
			},
			expErr: "unable to read token file invalid file: open invalid file: no such file or directory",
		},
		{
			name:     "correct workload identity",
			provider: &esv1beta1.AzureKVProvider{},
			prep: func(t *testing.T) {
				t.Setenv("AZURE_CLIENT_ID", clientID)
				t.Setenv("AZURE_TENANT_ID", tenantID)
				t.Setenv("AZURE_FEDERATED_TOKEN_FILE", tokenFile)
			},
		},
		{
			name:     "missing sa annotations, tenantID, and clientId/tenantId AuthSecretRef",
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
			expErr: "missing clientID: either serviceAccountRef or service account annotation 'azure.workload.identity/client-id' is missing",
		},
		{
			name: "duplicated clientId",
			provider: &esv1beta1.AzureKVProvider{
				VaultURL: &vaultURL,
				AuthType: &authType,
				TenantID: pointer.To(tenantID),
				ServiceAccountRef: &v1.ServiceAccountSelector{
					Name: saName,
				},
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientID: &v1.SecretKeySelector{Name: secretName, Namespace: pointer.To(namespace), Key: clientID},
					TenantID: &v1.SecretKeySelector{Name: secretName, Namespace: pointer.To(namespace), Key: tenantID},
				},
			},
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
						Annotations: map[string]string{
							AnnotationClientID: clientID,
							AnnotationTenantID: tenantID,
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						clientID: []byte("clientid"),
						tenantID: []byte("tenantid"),
					},
				},
			},
			expErr: "multiple clientID found. Check secretRef and serviceAccountRef",
		},
		{
			name: "duplicated tenantId",
			provider: &esv1beta1.AzureKVProvider{
				VaultURL: &vaultURL,
				AuthType: &authType,
				TenantID: pointer.To(tenantID),
				ServiceAccountRef: &v1.ServiceAccountSelector{
					Name: saName,
				},
			},
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
						Annotations: map[string]string{
							AnnotationClientID: clientID,
							AnnotationTenantID: tenantID,
						},
					},
				},
			},
			expErr: "multiple tenantID found. Check secretRef, 'spec.provider.azurekv.tenantId', and serviceAccountRef",
		},
		{
			name:     "successful case #1: ClientID, TenantID from ServiceAccountRef",
			provider: defaultProvider,
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      saName,
						Namespace: namespace,
						Annotations: map[string]string{
							AnnotationClientID: clientID,
							AnnotationTenantID: tenantID,
						},
					},
				},
			},
		},
		{
			name: "successful case #2: ClientID, TenantID from AuthSecretRef",
			provider: &esv1beta1.AzureKVProvider{
				VaultURL: &vaultURL,
				AuthType: &authType,
				ServiceAccountRef: &v1.ServiceAccountSelector{
					Name: saName,
				},
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientID: &v1.SecretKeySelector{Name: secretName, Namespace: pointer.To(namespace), Key: clientID},
					TenantID: &v1.SecretKeySelector{Name: secretName, Namespace: pointer.To(namespace), Key: tenantID},
				},
			},
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:        saName,
						Namespace:   namespace,
						Annotations: map[string]string{},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						clientID: []byte("clientid"),
						tenantID: []byte("tenantid"),
					},
				},
			},
		},
		{
			name: "successful case #3: ClientID from AuthSecretRef, TenantID from provider",
			provider: &esv1beta1.AzureKVProvider{
				VaultURL: &vaultURL,
				AuthType: &authType,
				TenantID: pointer.To(tenantID),
				ServiceAccountRef: &v1.ServiceAccountSelector{
					Name: saName,
				},
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientID: &v1.SecretKeySelector{Name: secretName, Namespace: pointer.To(namespace), Key: clientID},
				},
			},
			k8sObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:        saName,
						Namespace:   namespace,
						Annotations: map[string]string{},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: namespace,
					},
					Data: map[string][]byte{
						clientID: []byte("clientid"),
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
				kubeClient: utilfake.NewCreateTokenMock().WithToken(saToken),
				provider:   store.Spec.Provider.AzureKV,
			}
			tokenProvider := func(ctx context.Context, token, clientID, tenantID, aadEndpoint, kvResource string) (adal.OAuthTokenProvider, error) {
				tassert.Equal(t, token, saToken)
				tassert.Equal(t, clientID, clientID)
				tassert.Equal(t, tenantID, tenantID)
				return &tokenProvider{accessToken: azAccessToken}, nil
			}
			if row.prep != nil {
				row.prep(t)
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
				TenantID: pointer.To("mytenant"),
			},
		},
		{
			name:   "bad config",
			expErr: "missing accessKeyID/secretAccessKey in store config",
			store:  &defaultStore,
			provider: &esv1beta1.AzureKVProvider{
				AuthType:      &authType,
				VaultURL:      &vaultURL,
				TenantID:      pointer.To("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{},
			},
		},
		{
			name:   "bad config: missing secret",
			expErr: "cannot get Kubernetes secret \"password\": secrets \"password\" not found",
			store:  &defaultStore,
			provider: &esv1beta1.AzureKVProvider{
				AuthType: &authType,
				VaultURL: &vaultURL,
				TenantID: pointer.To("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientSecret: &v1.SecretKeySelector{Name: "password"},
					ClientID:     &v1.SecretKeySelector{Name: "password"},
				},
			},
		},
		{
			name:   "cluster secret store",
			expErr: "cannot get Kubernetes secret \"password\": secrets \"password\" not found",
			store: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: esv1beta1.ClusterSecretStoreKind,
				},
				Spec: esv1beta1.SecretStoreSpec{Provider: &esv1beta1.SecretStoreProvider{}},
			},
			provider: &esv1beta1.AzureKVProvider{
				AuthType: &authType,
				VaultURL: &vaultURL,
				TenantID: pointer.To("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientSecret: &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo")},
					ClientID:     &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo")},
				},
			},
		},
		{
			name: "correct cluster secret store with ClientSecret",
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
				TenantID: pointer.To("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientSecret: &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "secret"},
					ClientID:     &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "id"},
				},
			},
		},
		{
			name:   "bad config: both clientSecret and clientCredentials are configured",
			expErr: "both clientSecret and clientCredentials set",
			objects: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "password",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"id":          []byte("foo"),
					"certificate": []byte("bar"),
					"secret":      []byte("bar"),
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
				TenantID: pointer.To("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientID:          &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "id"},
					ClientCertificate: &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "certificate"},
					ClientSecret:      &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "secret"},
				},
			},
		},
		{
			name:   "bad config: no valid client certificate in pem file",
			expErr: "failed to get oauth token from certificate auth: failed to decode certificate: no certificate found in PEM file",
			objects: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "password",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"id":          []byte("foo"),
					"certificate": []byte("bar"),
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
				TenantID: pointer.To("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientID:          &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "id"},
					ClientCertificate: &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "certificate"},
				},
			},
		},
		{
			name: "correct configuration with certificate authentication",
			objects: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "password",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"id":          []byte("foo"),
					"certificate": []byte(mockCertificate),
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
				TenantID: pointer.To("mytenant"),
				AuthSecretRef: &esv1beta1.AzureKVAuth{
					ClientID:          &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "id"},
					ClientCertificate: &v1.SecretKeySelector{Name: "password", Namespace: pointer.To("foo"), Key: "certificate"},
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
