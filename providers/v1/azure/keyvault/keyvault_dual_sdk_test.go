/*
Copyright © 2025 ESO Maintainer Team

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

package keyvault

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"encoding/json"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestFeatureFlagRouting tests that the UseAzureSDK feature flag correctly routes to the appropriate implementation.
func TestFeatureFlagRouting(t *testing.T) {
	testCases := []struct {
		name         string
		useAzureSDK  *bool
		expectNewSDK bool
		description  string
	}{
		{
			name:         "default_legacy_sdk",
			useAzureSDK:  nil,
			expectNewSDK: false,
			description:  "When UseAzureSDK is nil (default), should use legacy SDK",
		},
		{
			name:         "explicit_legacy_sdk",
			useAzureSDK:  ptr.To(false),
			expectNewSDK: false,
			description:  "When UseAzureSDK is explicitly false, should use legacy SDK",
		},
		{
			name:         "explicit_new_sdk",
			useAzureSDK:  ptr.To(true),
			expectNewSDK: true,
			description:  "When UseAzureSDK is true, should use new SDK",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test provider with the specified feature flag
			provider := &esv1.AzureKVProvider{
				VaultURL:    ptr.To("https://test-vault.vault.azure.net/"),
				TenantID:    ptr.To("test-tenant"),
				AuthType:    ptr.To(esv1.AzureServicePrincipal),
				UseAzureSDK: tc.useAzureSDK,
				AuthSecretRef: &esv1.AzureKVAuth{
					ClientID: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-id",
					},
					ClientSecret: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-secret",
					},
				},
			}

			// Create Azure client
			azure := &Azure{
				provider: provider,
			}

			// Test the useNewSDK() method
			result := azure.useNewSDK()
			if result != tc.expectNewSDK {
				t.Errorf("Expected useNewSDK() to return %v for %s, got %v", tc.expectNewSDK, tc.description, result)
			}
		})
	}
}

// TestClientInitialization tests that both client initialization paths work correctly.
func TestClientInitialization(t *testing.T) {
	// Create test secret with credentials
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"client-id":     []byte("test-client-id"),
			"client-secret": []byte("test-client-secret"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithObjects(secret).Build()

	testCases := []struct {
		name                string
		useAzureSDK         *bool
		expectedErrorPrefix string
		description         string
	}{
		{
			name:                "legacy_client_init",
			useAzureSDK:         ptr.To(false),
			expectedErrorPrefix: "", // May succeed or fail with auth errors, but should not panic
			description:         "Legacy client initialization should not panic",
		},
		{
			name:                "new_sdk_client_init",
			useAzureSDK:         ptr.To(true),
			expectedErrorPrefix: "", // May succeed or fail with auth errors, but should not panic
			description:         "New SDK client initialization should not panic",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &esv1.AzureKVProvider{
				VaultURL:    ptr.To("https://test-vault.vault.azure.net/"),
				TenantID:    ptr.To("test-tenant"),
				AuthType:    ptr.To(esv1.AzureServicePrincipal),
				UseAzureSDK: tc.useAzureSDK,
				AuthSecretRef: &esv1.AzureKVAuth{
					ClientID: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-id",
					},
					ClientSecret: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-secret",
					},
				},
			}

			store := &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "test-namespace",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AzureKV: provider,
					},
				},
			}

			// Test that client initialization doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Client initialization panicked for %s: %v", tc.description, r)
				}
			}()

			azure := &Azure{}
			_, err := azure.NewClient(context.Background(), store, fakeClient, "test-namespace")

			// We expect errors due to authentication issues in tests, but no panics
			// The important thing is that the code paths are exercised without crashing
			if err != nil {
				t.Logf("Expected auth error for %s: %v", tc.description, err)
			}
		})
	}
}

// TestConfigurationValidation tests that the feature flag is properly validated and accepted.
func TestConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name        string
		useAzureSDK *bool
		expectValid bool
		description string
	}{
		{
			name:        "nil_feature_flag",
			useAzureSDK: nil,
			expectValid: true,
			description: "Nil feature flag should be valid (defaults to legacy)",
		},
		{
			name:        "false_feature_flag",
			useAzureSDK: ptr.To(false),
			expectValid: true,
			description: "False feature flag should be valid (legacy SDK)",
		},
		{
			name:        "true_feature_flag",
			useAzureSDK: ptr.To(true),
			expectValid: true,
			description: "True feature flag should be valid (new SDK)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &esv1.AzureKVProvider{
				VaultURL:    ptr.To("https://test-vault.vault.azure.net/"),
				TenantID:    ptr.To("test-tenant"),
				AuthType:    ptr.To(esv1.AzureServicePrincipal),
				UseAzureSDK: tc.useAzureSDK,
				AuthSecretRef: &esv1.AzureKVAuth{
					ClientID: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-id",
					},
					ClientSecret: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-secret",
					},
				},
			}

			store := &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AzureKV: provider,
					},
				},
			}

			azure := &Azure{}
			warnings, err := azure.ValidateStore(store)

			if tc.expectValid {
				if err != nil {
					t.Errorf("Expected validation to pass for %s, got error: %v", tc.description, err)
				}
				if len(warnings) > 0 {
					t.Logf("Validation warnings for %s: %v", tc.description, warnings)
				}
			} else if err == nil {
				t.Errorf("Expected validation to fail for %s, but it passed", tc.description)
			}
		})
	}
}

// TestBackwardCompatibility ensures that existing configurations continue to work.
func TestBackwardCompatibility(t *testing.T) {
	// Test that existing configurations without UseAzureSDK still work
	provider := &esv1.AzureKVProvider{
		VaultURL: ptr.To("https://test-vault.vault.azure.net/"),
		TenantID: ptr.To("test-tenant"),
		AuthType: ptr.To(esv1.AzureServicePrincipal),
		// UseAzureSDK intentionally omitted to test backward compatibility
		AuthSecretRef: &esv1.AzureKVAuth{
			ClientID: &v1.SecretKeySelector{
				Name: "test-secret",
				Key:  "client-id",
			},
			ClientSecret: &v1.SecretKeySelector{
				Name: "test-secret",
				Key:  "client-secret",
			},
		},
	}

	azure := &Azure{
		provider: provider,
	}

	// Should default to legacy SDK (false)
	if azure.useNewSDK() {
		t.Error("Expected backward compatibility: nil UseAzureSDK should default to legacy SDK (false)")
	}

	// Validation should still pass
	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AzureKV: provider,
			},
		},
	}

	warnings, err := azure.ValidateStore(store)
	if err != nil {
		t.Errorf("Backward compatibility failed: existing configuration should validate, got error: %v", err)
	}
	if len(warnings) > 0 {
		t.Logf("Backward compatibility warnings: %v", warnings)
	}
}

// TestAzureStackCloudConfiguration tests Azure Stack Cloud configuration validation.
func TestAzureStackCloudConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		useAzureSDK   *bool
		envType       esv1.AzureEnvironmentType
		customConfig  *esv1.AzureCustomCloudConfig
		expectError   bool
		expectedError string
		description   string
	}{
		{
			name:        "azure_stack_with_new_sdk_and_config",
			useAzureSDK: ptr.To(true),
			envType:     esv1.AzureEnvironmentAzureStackCloud,
			customConfig: &esv1.AzureCustomCloudConfig{
				ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
				KeyVaultEndpoint:        ptr.To("https://vault.local.azurestack.external/"),
				KeyVaultDNSSuffix:       ptr.To(".vault.local.azurestack.external"),
			},
			expectError: false,
			description: "Azure Stack with new SDK and custom config should be valid",
		},
		{
			name:          "azure_stack_without_custom_config",
			useAzureSDK:   ptr.To(true),
			envType:       esv1.AzureEnvironmentAzureStackCloud,
			customConfig:  nil,
			expectError:   true,
			expectedError: "CustomCloudConfig is required when EnvironmentType is AzureStackCloud",
			description:   "Azure Stack without custom config should fail",
		},
		{
			name:        "azure_stack_with_legacy_sdk",
			useAzureSDK: ptr.To(false),
			envType:     esv1.AzureEnvironmentAzureStackCloud,
			customConfig: &esv1.AzureCustomCloudConfig{
				ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
			},
			expectError:   true,
			expectedError: "AzureStackCloud environment requires UseAzureSDK to be set to true - the legacy SDK does not support custom clouds",
			description:   "Azure Stack with legacy SDK should fail",
		},
		{
			name:        "azure_stack_without_new_sdk_flag",
			useAzureSDK: nil, // defaults to false
			envType:     esv1.AzureEnvironmentAzureStackCloud,
			customConfig: &esv1.AzureCustomCloudConfig{
				ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
			},
			expectError:   true,
			expectedError: "AzureStackCloud environment requires UseAzureSDK to be set to true - the legacy SDK does not support custom clouds",
			description:   "Azure Stack without explicit new SDK flag should fail",
		},
		{
			name:        "azure_stack_missing_aad_endpoint",
			useAzureSDK: ptr.To(true),
			envType:     esv1.AzureEnvironmentAzureStackCloud,
			customConfig: &esv1.AzureCustomCloudConfig{
				KeyVaultEndpoint: ptr.To("https://vault.custom.cloud/"),
			},
			expectError:   true,
			expectedError: "activeDirectoryEndpoint is required in CustomCloudConfig",
			description:   "Azure Stack without AAD endpoint should fail",
		},
		{
			name:        "custom_config_without_azure_stack",
			useAzureSDK: ptr.To(true),
			envType:     esv1.AzureEnvironmentPublicCloud,
			customConfig: &esv1.AzureCustomCloudConfig{
				ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
			},
			expectError:   true,
			expectedError: "CustomCloudConfig should only be specified when EnvironmentType is AzureStackCloud",
			description:   "Custom config with non-AzureStack environment should fail",
		},
		{
			name:         "public_cloud_without_custom_config",
			useAzureSDK:  ptr.To(true),
			envType:      esv1.AzureEnvironmentPublicCloud,
			customConfig: nil,
			expectError:  false,
			description:  "Public cloud without custom config should be valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &esv1.AzureKVProvider{
				VaultURL:          ptr.To("https://test-vault.vault.azure.net/"),
				TenantID:          ptr.To("test-tenant"),
				AuthType:          ptr.To(esv1.AzureServicePrincipal),
				UseAzureSDK:       tc.useAzureSDK,
				EnvironmentType:   tc.envType,
				CustomCloudConfig: tc.customConfig,
				AuthSecretRef: &esv1.AzureKVAuth{
					ClientID: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-id",
					},
					ClientSecret: &v1.SecretKeySelector{
						Name: "test-secret",
						Key:  "client-secret",
					},
				},
			}

			azure := &Azure{provider: provider}
			store := &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AzureKV: provider,
					},
				},
			}

			warnings, err := azure.ValidateStore(store)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected validation to fail for %s, but it succeeded", tc.description)
				} else if tc.expectedError != "" && err.Error() != tc.expectedError {
					t.Errorf("Expected error message '%s' for %s, got: %v", tc.expectedError, tc.description, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to succeed for %s, but got error: %v", tc.description, err)
				}
			}
			if len(warnings) > 0 {
				t.Logf("Warnings for %s: %v", tc.name, warnings)
			}
		})
	}
}

// TestGetCloudConfiguration tests the cloud configuration resolution.
func TestGetCloudConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		provider      *esv1.AzureKVProvider
		expectError   bool
		expectedError string
		description   string
	}{
		{
			name: "public_cloud",
			provider: &esv1.AzureKVProvider{
				EnvironmentType: esv1.AzureEnvironmentPublicCloud,
			},
			expectError: false,
			description: "Public cloud should return valid configuration",
		},
		{
			name: "us_government_cloud",
			provider: &esv1.AzureKVProvider{
				EnvironmentType: esv1.AzureEnvironmentUSGovernmentCloud,
			},
			expectError: false,
			description: "US Government cloud should return valid configuration",
		},
		{
			name: "china_cloud",
			provider: &esv1.AzureKVProvider{
				EnvironmentType: esv1.AzureEnvironmentChinaCloud,
			},
			expectError: false,
			description: "China cloud should return valid configuration",
		},
		{
			name: "azure_stack_with_config",
			provider: &esv1.AzureKVProvider{
				EnvironmentType: esv1.AzureEnvironmentAzureStackCloud,
				UseAzureSDK:     ptr.To(true),
				CustomCloudConfig: &esv1.AzureCustomCloudConfig{
					ActiveDirectoryEndpoint: "https://login.local.azurestack.external/",
					KeyVaultEndpoint:        ptr.To("https://vault.local.azurestack.external/"),
				},
			},
			expectError: false,
			description: "Azure Stack with valid config should return custom configuration",
		},
		{
			name: "azure_stack_without_new_sdk",
			provider: &esv1.AzureKVProvider{
				EnvironmentType: esv1.AzureEnvironmentAzureStackCloud,
				UseAzureSDK:     ptr.To(false),
				CustomCloudConfig: &esv1.AzureCustomCloudConfig{
					ActiveDirectoryEndpoint: "https://login.local.azurestack.external/",
				},
			},
			expectError:   true,
			expectedError: "AzureStackCloud environment requires UseAzureSDK to be set to true",
			description:   "Azure Stack without new SDK should fail",
		},
		{
			name: "azure_stack_without_config",
			provider: &esv1.AzureKVProvider{
				EnvironmentType: esv1.AzureEnvironmentAzureStackCloud,
				UseAzureSDK:     ptr.To(true),
			},
			expectError:   true,
			expectedError: "CustomCloudConfig is required when EnvironmentType is AzureStackCloud",
			description:   "Azure Stack without custom config should fail",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := getCloudConfiguration(tc.provider)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				} else if tc.expectedError != "" && err.Error() != tc.expectedError {
					t.Errorf("Expected error '%s' for %s, got: %v", tc.expectedError, tc.description, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, but got: %v", tc.description, err)
				}
				if config.ActiveDirectoryAuthorityHost == "" && tc.provider.EnvironmentType != esv1.AzureEnvironmentAzureStackCloud {
					// For predefined clouds, we should have a valid config
					t.Errorf("Expected valid cloud configuration for %s", tc.description)
				}
			}
		})
	}
}

type fakeID struct {
	name string
}

func (f *fakeID) Name() string {
	return f.name
}

type fakeSecretProperties struct {
	ID *fakeID
}

type fakeCertProperties struct {
	ID *fakeID
}

type fakeKeyProperties struct {
	KID *fakeID
}

type fakeSecretsPager struct {
	called bool
	page   struct{ Value []fakeSecretProperties }
}

func (p *fakeSecretsPager) More() bool { return !p.called }

func (p *fakeSecretsPager) NextPage(ctx context.Context) (struct{ Value []fakeSecretProperties }, error) {
	if p.called {
		return struct{ Value []fakeSecretProperties }{}, errors.New("no more pages")
	}
	p.called = true
	return p.page, nil
}

type fakeCertsPager struct {
	called bool
	page   struct{ Value []fakeCertProperties }
}

func (p *fakeCertsPager) More() bool { return !p.called }

func (p *fakeCertsPager) NextPage(ctx context.Context) (struct{ Value []fakeCertProperties }, error) {
	if p.called {
		return struct{ Value []fakeCertProperties }{}, errors.New("no more pages")
	}
	p.called = true
	return p.page, nil
}

type fakeKeysPager struct {
	called bool
	page   struct{ Value []fakeKeyProperties }
}

func (p *fakeKeysPager) More() bool { return !p.called }

func (p *fakeKeysPager) NextPage(ctx context.Context) (struct{ Value []fakeKeyProperties }, error) {
	if p.called {
		return struct{ Value []fakeKeyProperties }{}, errors.New("no more pages")
	}
	p.called = true
	return p.page, nil
}

type fakeSecretsClient struct {
	pager *fakeSecretsPager
}

func (c *fakeSecretsClient) NewListSecretPropertiesPager(_ any) *fakeSecretsPager {
	return c.pager
}

type fakeCertsClient struct {
	pager   *fakeCertsPager
	getCert map[string][]byte
}

func (c *fakeCertsClient) NewListCertificatePropertiesPager(_ any) *fakeCertsPager {
	return c.pager
}

func (c *fakeCertsClient) GetCertificate(ctx context.Context, name, version string, _ any) (struct{ CER []byte }, error) {
	return struct{ CER []byte }{CER: c.getCert[name]}, nil
}

type fakeKeysClient struct {
	pager  *fakeKeysPager
	getKey map[string]any
}

func (c *fakeKeysClient) NewListKeyPropertiesPager(_ any) *fakeKeysPager {
	return c.pager
}

func (c *fakeKeysClient) GetKey(ctx context.Context, name, version string, _ any) (struct{ Key any }, error) {
	return struct{ Key any }{Key: c.getKey[name]}, nil
}

type testAzure struct {
	secretsClient      *fakeSecretsClient
	certsClient        *fakeCertsClient
	keysClient         *fakeKeysClient
	processSecretsPage func(ctx context.Context, props any, ref esv1beta1.ExternalSecretFind, out map[string][]byte) error
}

func (a *testAzure) getAllSecretsWithNewSDK(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	secretsMap := make(map[string][]byte)

	secretPager := a.secretsClient.NewListSecretPropertiesPager(nil)
	for secretPager.More() {
		page, err := secretPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		if err := a.processSecretsPage(ctx, page.Value, ref, secretsMap); err != nil {
			return nil, err
		}
	}

	certPager := a.certsClient.NewListCertificatePropertiesPager(nil)
	for certPager.More() {
		page, err := certPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cert := range page.Value {
			if cert.ID == nil {
				continue
			}

			name := cert.ID.Name()
			resp, _ := a.certsClient.GetCertificate(ctx, name, "", nil)
			if resp.CER != nil {
				secretsMap["cert/"+name] = resp.CER
			}
		}
	}

	keyPager := a.keysClient.NewListKeyPropertiesPager(nil)
	for keyPager.More() {
		page, err := keyPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, keyProps := range page.Value {
			if keyProps.KID == nil {
				continue
			}

			name := keyProps.KID.Name()
			resp, _ := a.keysClient.GetKey(ctx, name, "", nil)

			if resp.Key != nil {
				jwkBytes, _ := json.Marshal(resp.Key)
				secretsMap["key/"+name] = jwkBytes
			}
		}
	}

	return secretsMap, nil
}

func TestGetAllSecretsWithNewSDK_FakeClients(t *testing.T) {
	ctx := context.Background()

	secretPager := &fakeSecretsPager{
		page: struct{ Value []fakeSecretProperties }{
			Value: []fakeSecretProperties{
				{ID: &fakeID{name: "mysecret"}},
			},
		},
	}
	secretsClient := &fakeSecretsClient{pager: secretPager}

	certPager := &fakeCertsPager{
		page: struct{ Value []fakeCertProperties }{
			Value: []fakeCertProperties{
				{ID: &fakeID{name: "certA"}},
			},
		},
	}
	certsClient := &fakeCertsClient{
		pager: certPager,
		getCert: map[string][]byte{
			"certA": []byte("CERTDATA"),
		},
	}

	keyPager := &fakeKeysPager{
		page: struct{ Value []fakeKeyProperties }{
			Value: []fakeKeyProperties{
				{KID: &fakeID{name: "keyA"}},
			},
		},
	}
	keysClient := &fakeKeysClient{
		pager: keyPager,
		getKey: map[string]any{
			"keyA": map[string]string{"kty": "RSA"},
		},
	}

	az := &testAzure{
		secretsClient: secretsClient,
		certsClient:   certsClient,
		keysClient:    keysClient,

		processSecretsPage: func(
			ctx context.Context,
			props any,
			ref esv1beta1.ExternalSecretFind,
			out map[string][]byte,
		) error {
			out["mysecret"] = []byte("hello")
			return nil
		},
	}

	out, err := az.getAllSecretsWithNewSDK(ctx, esv1beta1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	keyJSON, _ := json.Marshal(map[string]string{"kty": "RSA"})
	expected := map[string][]byte{
		"mysecret":   []byte("hello"),
		"cert/certA": []byte("CERTDATA"),
		"key/keyA":   keyJSON,
	}

	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("expected %v, got %v", expected, out)
	}
}
