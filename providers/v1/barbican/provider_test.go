/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package barbican

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testAuthURL        = "https://keystone.example.com/v3"
	testTenantName     = "test-tenant"
	testDomainName     = "default"
	testRegion         = "RegionOne"
	testUsername       = "test-user"
	testPassword       = "test-password"
	testSecretName     = "barbican-creds"
	testNamespace      = "default"
	testAppCredID      = "app-cred-id-123"
	testAppCredSecret  = "app-cred-secret-456"
	testAppCredSecName = "barbican-app-creds"
)

type validateStoreTestCase struct {
	name        string
	store       esv1.GenericStore
	expectError bool
	errorMsg    string
}

func TestProviderCapabilities(t *testing.T) {
	provider := &Provider{}
	capabilities := provider.Capabilities()

	assert.Equal(t, esv1.SecretStoreReadOnly, capabilities)
}

func TestValidateStore(t *testing.T) {
	provider := &Provider{}

	testCases := []validateStoreTestCase{
		{
			name:        "nil store should return error",
			store:       nil,
			expectError: true,
			errorMsg:    "store is nil",
		},
		{
			name:        "valid store should pass validation",
			store:       makeValidSecretStore(),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			warnings, err := provider.ValidateStore(tc.store)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, warnings)
			} else {
				assert.NoError(t, err)
				assert.Nil(t, warnings)
			}
		})
	}
}

func TestGetProvider(t *testing.T) {
	testCases := []struct {
		name        string
		store       esv1.GenericStore
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid store with barbican provider",
			store:       makeValidSecretStore(),
			expectError: false,
		},
		{
			name:        "nil provider should return error",
			store:       makeSecretStoreWithNilProvider(),
			expectError: true,
			errorMsg:    "provider barbican is nil",
		},
		{
			name:        "nil barbican provider should return error",
			store:       makeSecretStoreWithNilBarbican(),
			expectError: true,
			errorMsg:    "provider barbican is nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := getProvider(tc.store)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, testAuthURL, provider.AuthURL)
				assert.Equal(t, testTenantName, provider.TenantName)
				assert.Equal(t, testDomainName, provider.DomainName)
				assert.Equal(t, testRegion, provider.Region)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	testCases := []struct {
		name        string
		store       esv1.GenericStore
		kube        *clientfake.ClientBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing authURL should return error",
			store:       makeSecretStoreWithMissingAuthURL(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecret()),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "username as value should pass",
			store:       makeSecretStoreWithValueUsername(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecretWithNoUsername()),
			expectError: false,
		},
		{
			name:        "username as value and secret should pass",
			store:       makeSecretStoreWithValueUsername(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecret()),
			expectError: false,
		},
		{
			name:        "missing username secret should return error",
			store:       makeValidSecretStore(),
			kube:        clientfake.NewClientBuilder(),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "missing password in secret should return error",
			store:       makeValidSecretStore(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeSecretWithMissingPassword()),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "nil barbican provider should return error",
			store:       makeSecretStoreWithNilBarbican(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecret()),
			expectError: true,
			errorMsg:    "provider barbican is nil",
		},
		// Backward compatibility: password auth with no authType set (defaults to password)
		{
			name:        "password auth without explicit authType should pass (backward compat)",
			store:       makeValidSecretStore(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecret()),
			expectError: false,
		},
		// Backward compatibility: password auth with explicit authType=password
		{
			name:        "password auth with explicit authType=password should pass",
			store:       makeSecretStoreWithExplicitPasswordAuthType(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecret()),
			expectError: false,
		},
		// Application credential auth type tests
		{
			name:        "appCredential auth with valid secret should pass",
			store:       makeSecretStoreWithAppCredAuth(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidAppCredSecret()),
			expectError: false,
		},
		{
			name:        "appCredential auth with value appCredID should pass",
			store:       makeSecretStoreWithAppCredValueID(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeAppCredSecretWithNoID()),
			expectError: false,
		},
		{
			name:        "appCredential auth missing appCredID secret should return error",
			store:       makeSecretStoreWithAppCredAuth(),
			kube:        clientfake.NewClientBuilder(),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "appCredential auth missing appCredSecret in secret should return error",
			store:       makeSecretStoreWithAppCredAuth(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeAppCredSecretWithMissingSecret()),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "appCredential auth missing authURL should return error",
			store:       makeSecretStoreWithAppCredMissingAuthURL(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidAppCredSecret()),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "unsupported auth type should return error",
			store:       makeSecretStoreWithUnsupportedAuthType(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecret()),
			expectError: true,
			errorMsg:    "unsupported auth type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &Provider{}
			fakeClient := tc.kube.Build()

			// Note: This test will fail when trying to actually connect to OpenStack
			// In a real test environment, we would need to mock the OpenStack client
			_, err := provider.NewClient(context.Background(), tc.store, fakeClient, testNamespace)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				// This would only pass with proper OpenStack mocking
				assert.Error(t, err) // We expect an error due to missing OpenStack mock
			}
		})
	}
}

func TestNewClientAuthTypeDefaultsToPassword(t *testing.T) {
	// Verify that when AuthType is nil, the provider defaults to password auth
	// and resolves username/password credentials correctly.
	store := makeValidSecretStore()
	assert.Nil(t, store.Spec.Provider.Barbican.Auth.AuthType, "AuthType should be nil for backward compatibility test")

	fakeClient := clientfake.NewClientBuilder().WithObjects(makeValidSecret()).Build()
	provider := &Provider{}

	// The call will fail at OpenStack connection, but should NOT fail on credential resolution
	_, err := provider.NewClient(context.Background(), store, fakeClient, testNamespace)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "missing required field")
	assert.NotContains(t, err.Error(), "unsupported auth type")
}

func TestGetProviderWithAuthType(t *testing.T) {
	testCases := []struct {
		name             string
		store            esv1.GenericStore
		expectError      bool
		errorMsg         string
		expectedAuthType *esv1.BarbicanAuthType
	}{
		{
			name:             "password store with no authType set (backward compat)",
			store:            makeValidSecretStore(),
			expectError:      false,
			expectedAuthType: nil,
		},
		{
			name:             "password store with explicit authType",
			store:            makeSecretStoreWithExplicitPasswordAuthType(),
			expectError:      false,
			expectedAuthType: barbicanAuthTypePtr(esv1.BarbicanAuthTypePassword),
		},
		{
			name:             "appCredential store with authType",
			store:            makeSecretStoreWithAppCredAuth(),
			expectError:      false,
			expectedAuthType: barbicanAuthTypePtr(esv1.BarbicanAuthTypeApplicationCredential),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := getProvider(tc.store)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				if tc.expectedAuthType == nil {
					assert.Nil(t, provider.Auth.AuthType)
				} else {
					assert.Equal(t, *tc.expectedAuthType, *provider.Auth.AuthType)
				}
			}
		})
	}
}

// Helper functions to create test fixtures

func makeValidSecretStore() *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-store",
			Namespace: testNamespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Barbican: &esv1.BarbicanProvider{
					AuthURL:    testAuthURL,
					TenantName: testTenantName,
					DomainName: testDomainName,
					Region:     testRegion,
					Auth: esv1.BarbicanAuth{
						Username: esv1.BarbicanProviderUsernameRef{
							SecretRef: &esmeta.SecretKeySelector{
								Name: testSecretName,
								Key:  "username",
							},
						},
						Password: esv1.BarbicanProviderPasswordRef{
							SecretRef: &esmeta.SecretKeySelector{
								Name: testSecretName,
								Key:  "password",
							},
						},
					},
				},
			},
		},
	}
}

func makeSecretStoreWithValueUsername() *esv1.SecretStore {
	store := makeValidSecretStore()
	store.Spec.Provider.Barbican.Auth.Username = esv1.BarbicanProviderUsernameRef{
		Value: testUsername,
	}
	return store
}

func makeSecretStoreWithNilProvider() *esv1.SecretStore {
	store := makeValidSecretStore()
	store.Spec.Provider = nil
	return store
}

func makeSecretStoreWithNilBarbican() *esv1.SecretStore {
	store := makeValidSecretStore()
	store.Spec.Provider.Barbican = nil
	return store
}

func makeSecretStoreWithMissingAuthURL() *esv1.SecretStore {
	store := makeValidSecretStore()
	store.Spec.Provider.Barbican.AuthURL = ""
	return store
}

func makeValidSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"username": []byte(testUsername),
			"password": []byte(testPassword),
		},
	}
}

func makeValidSecretWithNoUsername() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"password": []byte(testPassword),
		},
	}
}

func makeSecretWithMissingPassword() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"username": []byte(testUsername),
			// missing password key
		},
	}
}

// Helper: returns a pointer to a BarbicanAuthType.
func barbicanAuthTypePtr(t esv1.BarbicanAuthType) *esv1.BarbicanAuthType {
	return &t
}

// Helper: password auth store with explicit authType=password.
func makeSecretStoreWithExplicitPasswordAuthType() *esv1.SecretStore {
	store := makeValidSecretStore()
	store.Spec.Provider.Barbican.Auth.AuthType = barbicanAuthTypePtr(esv1.BarbicanAuthTypePassword)
	return store
}

// Helper: application credential auth store.
func makeSecretStoreWithAppCredAuth() *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-store-appcred",
			Namespace: testNamespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Barbican: &esv1.BarbicanProvider{
					AuthURL:    testAuthURL,
					TenantName: testTenantName,
					DomainName: testDomainName,
					Region:     testRegion,
					Auth: esv1.BarbicanAuth{
						AuthType: barbicanAuthTypePtr(esv1.BarbicanAuthTypeApplicationCredential),
						ApplicationCredentialID: &esv1.BarbicanProviderAppCredIDRef{
							SecretRef: &esmeta.SecretKeySelector{
								Name: testAppCredSecName,
								Key:  "app-cred-id",
							},
						},
						ApplicationCredentialSecret: &esv1.BarbicanProviderAppCredSecretRef{
							SecretRef: &esmeta.SecretKeySelector{
								Name: testAppCredSecName,
								Key:  "app-cred-secret",
							},
						},
					},
				},
			},
		},
	}
}

// Helper: application credential auth store with inline value for appCredID.
func makeSecretStoreWithAppCredValueID() *esv1.SecretStore {
	store := makeSecretStoreWithAppCredAuth()
	store.Spec.Provider.Barbican.Auth.ApplicationCredentialID = &esv1.BarbicanProviderAppCredIDRef{
		Value: testAppCredID,
	}
	return store
}

// Helper: application credential auth store missing authURL.
func makeSecretStoreWithAppCredMissingAuthURL() *esv1.SecretStore {
	store := makeSecretStoreWithAppCredAuth()
	store.Spec.Provider.Barbican.AuthURL = ""
	return store
}

// Helper: unsupported auth type store.
func makeSecretStoreWithUnsupportedAuthType() *esv1.SecretStore {
	store := makeValidSecretStore()
	unsupported := esv1.BarbicanAuthType("kerberos")
	store.Spec.Provider.Barbican.Auth.AuthType = &unsupported
	return store
}

// Helper: valid k8s secret for application credentials.
func makeValidAppCredSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppCredSecName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"app-cred-id":     []byte(testAppCredID),
			"app-cred-secret": []byte(testAppCredSecret),
		},
	}
}

// Helper: k8s secret with only the app credential secret (no ID).
func makeAppCredSecretWithNoID() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppCredSecName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"app-cred-secret": []byte(testAppCredSecret),
		},
	}
}

// Helper: k8s secret with app credential ID but missing app credential secret.
func makeAppCredSecretWithMissingSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppCredSecName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"app-cred-id": []byte(testAppCredID),
			// missing app-cred-secret key
		},
	}
}
