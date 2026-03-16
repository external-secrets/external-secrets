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
			name:        "valid password store should pass validation",
			store:       makeValidSecretStore(),
			expectError: false,
		},
		{
			name:        "valid password store with explicit authType should pass",
			store:       makeSecretStoreWithExplicitPasswordAuthType(),
			expectError: false,
		},
		{
			name:        "valid appCredential store should pass validation",
			store:       makeSecretStoreWithAppCredAuth(),
			expectError: false,
		},
		{
			name:        "valid appCredential store with inline ID should pass",
			store:       makeSecretStoreWithAppCredValueID(),
			expectError: false,
		},
		{
			name:        "nil provider should return error",
			store:       makeSecretStoreWithNilProvider(),
			expectError: true,
			errorMsg:    "provider barbican is nil",
		},
		{
			name:        "nil barbican should return error",
			store:       makeSecretStoreWithNilBarbican(),
			expectError: true,
			errorMsg:    "provider barbican is nil",
		},
		{
			name:        "missing authURL should return error",
			store:       makeSecretStoreWithMissingAuthURL(),
			expectError: true,
			errorMsg:    "authURL is required",
		},
		{
			name:        "password auth missing username should return error",
			store:       makeSecretStorePasswordNoUsername(),
			expectError: true,
			errorMsg:    "username must specify either value or secretRef",
		},
		{
			name:        "password auth missing password should return error",
			store:       makeSecretStorePasswordNoPassword(),
			expectError: true,
			errorMsg:    "password secretRef is required",
		},
		{
			name:        "appCredential auth missing ID should return error",
			store:       makeSecretStoreAppCredNoID(),
			expectError: true,
			errorMsg:    "applicationCredentialID is required",
		},
		{
			name:        "appCredential auth ID with no value or secretRef should return error",
			store:       makeSecretStoreAppCredEmptyID(),
			expectError: true,
			errorMsg:    "applicationCredentialID must specify either value or secretRef",
		},
		{
			name:        "appCredential auth missing secret should return error",
			store:       makeSecretStoreAppCredNoSecret(),
			expectError: true,
			errorMsg:    "applicationCredentialSecret secretRef is required",
		},
		{
			name:        "unsupported auth type should return error",
			store:       makeSecretStoreWithUnsupportedAuthType(),
			expectError: true,
			errorMsg:    "unsupported auth type",
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
				// Without an OpenStack mock we still get a connection error,
				// but it must NOT be a config/validation error.
				assert.Error(t, err)
				assert.NotContains(t, err.Error(), "missing required field",
					"happy-path case should not fail on config validation")
				assert.NotContains(t, err.Error(), "unsupported auth type",
					"happy-path case should not fail on auth type selection")
				assert.NotContains(t, err.Error(), "provider barbican is nil",
					"happy-path case should not fail on provider validation")
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
		expectedAuthType *esv1.BarbicanAuthType
	}{
		{
			name:             "password store with no authType set (backward compat)",
			store:            makeValidSecretStore(),
			expectedAuthType: nil,
		},
		{
			name:             "password store with explicit authType",
			store:            makeSecretStoreWithExplicitPasswordAuthType(),
			expectedAuthType: barbicanAuthTypePtr(esv1.BarbicanAuthTypePassword),
		},
		{
			name:             "appCredential store with authType",
			store:            makeSecretStoreWithAppCredAuth(),
			expectedAuthType: barbicanAuthTypePtr(esv1.BarbicanAuthTypeApplicationCredential),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := getProvider(tc.store)

			assert.NoError(t, err)
			assert.NotNil(t, provider)
			if tc.expectedAuthType == nil {
				assert.Nil(t, provider.Auth.AuthType)
			} else {
				assert.Equal(t, *tc.expectedAuthType, *provider.Auth.AuthType)
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

func TestBuildPasswordAuthOpts(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		store       *esv1.SecretStore
		kube        *clientfake.ClientBuilder
		expectError bool
		errorMsg    string
		wantUser    string
		wantPass    string
	}{
		{
			name:        "resolve username and password from secret",
			store:       makeValidSecretStore(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecret()),
			expectError: false,
			wantUser:    testUsername,
			wantPass:    testPassword,
		},
		{
			name:        "inline username value is preferred over secretRef",
			store:       makeSecretStoreWithValueUsername(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidSecretWithNoUsername()),
			expectError: false,
			wantUser:    testUsername,
			wantPass:    testPassword,
		},
		{
			name:        "missing username secret returns error",
			store:       makeValidSecretStore(),
			kube:        clientfake.NewClientBuilder(),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "missing password key in secret returns error",
			store:       makeValidSecretStore(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeSecretWithMissingPassword()),
			expectError: true,
			errorMsg:    "missing required field",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prov := tc.store.Spec.Provider.Barbican
			opts, err := buildPasswordAuthOpts(ctx, tc.store, tc.kube.Build(), testNamespace, prov)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantUser, opts.Username)
				assert.Equal(t, tc.wantPass, opts.Password)
				assert.Equal(t, testAuthURL, opts.IdentityEndpoint)
				assert.Equal(t, testTenantName, opts.TenantName)
				assert.Equal(t, testDomainName, opts.DomainName)
			}
		})
	}
}

func TestBuildAppCredAuthOpts(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		store       *esv1.SecretStore
		kube        *clientfake.ClientBuilder
		expectError bool
		errorMsg    string
		wantCredID  string
		wantCredSec string
	}{
		{
			name:        "resolve appCredID and appCredSecret from secret",
			store:       makeSecretStoreWithAppCredAuth(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeValidAppCredSecret()),
			expectError: false,
			wantCredID:  testAppCredID,
			wantCredSec: testAppCredSecret,
		},
		{
			name:        "inline appCredID value is preferred over secretRef",
			store:       makeSecretStoreWithAppCredValueID(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeAppCredSecretWithNoID()),
			expectError: false,
			wantCredID:  testAppCredID,
			wantCredSec: testAppCredSecret,
		},
		{
			name:        "nil applicationCredentialID returns error",
			store:       makeSecretStoreAppCredNoID(),
			kube:        clientfake.NewClientBuilder(),
			expectError: true,
			errorMsg:    "applicationCredentialID is required",
		},
		{
			name:        "nil applicationCredentialSecret returns error",
			store:       makeSecretStoreAppCredNoSecret(),
			kube:        clientfake.NewClientBuilder(),
			expectError: true,
			errorMsg:    "applicationCredentialSecret is required",
		},
		{
			name:        "appCredID with no value and no secretRef returns error",
			store:       makeSecretStoreAppCredEmptyID(),
			kube:        clientfake.NewClientBuilder(),
			expectError: true,
			errorMsg:    "applicationCredentialID.secretRef is required when value is empty",
		},
		{
			name:        "missing appCredID secret object returns error",
			store:       makeSecretStoreWithAppCredAuth(),
			kube:        clientfake.NewClientBuilder(),
			expectError: true,
			errorMsg:    "missing required field",
		},
		{
			name:        "missing appCredSecret key in secret returns error",
			store:       makeSecretStoreWithAppCredAuth(),
			kube:        clientfake.NewClientBuilder().WithObjects(makeAppCredSecretWithMissingSecret()),
			expectError: true,
			errorMsg:    "missing required field",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prov := tc.store.Spec.Provider.Barbican
			opts, err := buildAppCredAuthOpts(ctx, tc.store, tc.kube.Build(), testNamespace, prov)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantCredID, opts.ApplicationCredentialID)
				assert.Equal(t, tc.wantCredSec, opts.ApplicationCredentialSecret)
				assert.Equal(t, testAuthURL, opts.IdentityEndpoint)
			}
		})
	}
}

// Helper: password auth store with no username (empty value, nil secretRef).
func makeSecretStorePasswordNoUsername() *esv1.SecretStore {
	store := makeValidSecretStore()
	store.Spec.Provider.Barbican.Auth.Username = esv1.BarbicanProviderUsernameRef{}
	return store
}

// Helper: password auth store with no password secretRef.
func makeSecretStorePasswordNoPassword() *esv1.SecretStore {
	store := makeValidSecretStore()
	store.Spec.Provider.Barbican.Auth.Password = esv1.BarbicanProviderPasswordRef{}
	return store
}

// Helper: appCredential auth with nil ApplicationCredentialID.
func makeSecretStoreAppCredNoID() *esv1.SecretStore {
	store := makeSecretStoreWithAppCredAuth()
	store.Spec.Provider.Barbican.Auth.ApplicationCredentialID = nil
	return store
}

// Helper: appCredential auth with ApplicationCredentialID present but empty (no value, no secretRef).
func makeSecretStoreAppCredEmptyID() *esv1.SecretStore {
	store := makeSecretStoreWithAppCredAuth()
	store.Spec.Provider.Barbican.Auth.ApplicationCredentialID = &esv1.BarbicanProviderAppCredIDRef{}
	return store
}

// Helper: appCredential auth with nil ApplicationCredentialSecret.
func makeSecretStoreAppCredNoSecret() *esv1.SecretStore {
	store := makeSecretStoreWithAppCredAuth()
	store.Spec.Provider.Barbican.Auth.ApplicationCredentialSecret = nil
	return store
}
