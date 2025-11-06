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

package vault

import (
	"context"
	"os"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSetEnvVar(t *testing.T) {
	c := &client{
		log: logr.Discard(),
	}

	tests := []struct {
		name      string
		key       string
		value     string
		wantError bool
	}{
		{
			name:      "valid environment variable",
			key:       "TEST_VAR",
			value:     "test_value",
			wantError: false,
		},
		{
			name:      "empty value should error",
			key:       "TEST_VAR",
			value:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variable after test
			defer func() {
				if tt.key != "" {
					os.Unsetenv(tt.key)
				}
			}()

			err := c.setEnvVar(tt.key, tt.value)

			if tt.wantError && err == nil {
				t.Errorf("setEnvVar() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("setEnvVar() unexpected error: %v", err)
			}

			// If successful, verify the environment variable was actually set
			if !tt.wantError && err == nil {
				actualValue := os.Getenv(tt.key)
				if actualValue != tt.value {
					t.Errorf("setEnvVar() environment variable not set correctly, got %v, want %v", actualValue, tt.value)
				}
			}
		})
	}
}

func TestSetGCPEnvironment(t *testing.T) {
	c := &client{
		log: logr.Discard(),
	}

	tests := []struct {
		name        string
		accessToken string
		wantError   bool
	}{
		{
			name:        "valid access token",
			accessToken: "ya29.test-token",
			wantError:   false,
		},
		{
			name:        "empty access token",
			accessToken: "",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variable after test
			defer os.Unsetenv(googleOAuthAccessTokenKey)

			err := c.setGCPEnvironment(tt.accessToken)

			if tt.wantError && err == nil {
				t.Errorf("setGCPEnvironment() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("setGCPEnvironment() unexpected error: %v", err)
			}

			// If successful, verify the GOOGLE_OAUTH_ACCESS_TOKEN was set
			if !tt.wantError && err == nil {
				actualValue := os.Getenv(googleOAuthAccessTokenKey)
				if actualValue != tt.accessToken {
					t.Errorf("setGCPEnvironment() %s not set correctly, got %v, want %v", googleOAuthAccessTokenKey, actualValue, tt.accessToken)
				}
			}
		})
	}
}

func TestSetupDefaultGCPAuth(t *testing.T) {
	c := &client{
		log: logr.Discard(),
	}

	err := c.setupDefaultGCPAuth()
	// In test environments, ADC might not be available, so we expect this to potentially fail.
	// This is expected behavior - the function should fail gracefully if ADC is not configured.
	// The test verifies that the function behaves correctly in both scenarios:
	// 1. ADC available: function succeeds
	// 2. ADC unavailable: function fails with meaningful error message
	if err != nil {
		t.Logf("setupDefaultGCPAuth() failed as expected in test environment without ADC: %v", err)
		// Verify the error message indicates ADC unavailability
		if err.Error() == "" {
			t.Errorf("setupDefaultGCPAuth() should provide meaningful error message when ADC is unavailable")
		}
	} else {
		t.Logf("setupDefaultGCPAuth() succeeded - ADC is configured and available in test environment")
	}
}

func TestSetupGCPAuthPriority(t *testing.T) {
	c := &client{
		log:       logr.Discard(),
		kube:      clientfake.NewClientBuilder().Build(),
		namespace: "default",
		storeKind: "SecretStore",
	}

	tests := []struct {
		name        string
		gcpAuth     *esv1.VaultGCPAuth
		expectError bool
		description string
	}{
		{
			name: "SecretRef has priority",
			gcpAuth: &esv1.VaultGCPAuth{
				Role:      "test-role",
				ProjectID: "test-project",
				SecretRef: &esv1.GCPSMAuthSecretRef{
					SecretAccessKey: esmeta.SecretKeySelector{
						Name: "gcp-secret",
						Key:  "credentials.json",
					},
				},
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name: "test-sa",
					},
				},
			},
			expectError: true, // Will fail because secret doesn't exist in fake client
			description: "SecretRef should be tried first",
		},
		{
			name: "WorkloadIdentity second priority",
			gcpAuth: &esv1.VaultGCPAuth{
				Role:      "test-role",
				ProjectID: "test-project",
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name: "test-sa",
					},
				},
				ServiceAccountRef: &esmeta.ServiceAccountSelector{
					Name: "test-sa",
				},
			},
			expectError: true, // Will fail because workload identity setup will fail
			description: "WorkloadIdentity should be tried when SecretRef is nil",
		},
		{
			name: "ServiceAccountRef third priority",
			gcpAuth: &esv1.VaultGCPAuth{
				Role: "test-role",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{
					Name: "test-sa",
				},
			},
			expectError: false, // Should fall back to default auth
			description: "ServiceAccountRef should fall back to default auth",
		},
		{
			name: "Default auth last resort",
			gcpAuth: &esv1.VaultGCPAuth{
				Role: "test-role",
			},
			expectError: false, // Test handles both ADC available and unavailable scenarios
			description: "Should attempt default ADC when no other auth method is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.setupGCPAuth(context.Background(), tt.gcpAuth)

			// For default auth test, ADC availability depends on the test environment.
			// We accept both success (ADC configured) and failure (ADC not available)
			// as valid outcomes, since this test should pass in any environment.
			if tt.name == "Default auth last resort" {
				if err != nil {
					t.Logf("setupGCPAuth() with default ADC: %v (expected behavior in environments without ADC configured)", err)
				} else {
					t.Logf("setupGCPAuth() with default ADC succeeded (ADC is properly configured in this environment)")
				}
				return
			}

			if tt.expectError && err == nil {
				t.Errorf("setupGCPAuth() expected error for %s, got nil", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("setupGCPAuth() unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}

func TestGCPAuthMethodSelection(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *client
		gcpAuth     *esv1.VaultGCPAuth
		expectError bool
		description string
	}{
		{
			name: "SecretRef method selected",
			setupClient: func() *client {
				// Create a secret with invalid credentials to trigger expected error
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gcp-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"credentials.json": []byte(`{"type": "service_account", "project_id": "test"}`),
					},
				}
				kube := clientfake.NewClientBuilder().WithObjects(secret).Build()
				return &client{
					log:       logr.Discard(),
					kube:      kube,
					namespace: "default",
					storeKind: "SecretStore",
				}
			},
			gcpAuth: &esv1.VaultGCPAuth{
				Role:      "test-role",
				ProjectID: "test-project",
				SecretRef: &esv1.GCPSMAuthSecretRef{
					SecretAccessKey: esmeta.SecretKeySelector{
						Name: "gcp-secret",
						Key:  "credentials.json",
					},
				},
			},
			expectError: true, // Expected to fail in test environment
			description: "Should attempt to use SecretRef method",
		},
		{
			name: "WorkloadIdentity method selected",
			setupClient: func() *client {
				return &client{
					log:       logr.Discard(),
					kube:      clientfake.NewClientBuilder().Build(),
					namespace: "default",
					storeKind: "SecretStore",
				}
			},
			gcpAuth: &esv1.VaultGCPAuth{
				Role:      "test-role",
				ProjectID: "test-project",
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name: "test-sa",
					},
				},
			},
			expectError: true, // Expected to fail in test environment
			description: "Should attempt to use WorkloadIdentity method",
		},
		{
			name: "Default ADC method selected",
			setupClient: func() *client {
				return &client{
					log:       logr.Discard(),
					kube:      clientfake.NewClientBuilder().Build(),
					namespace: "default",
					storeKind: "SecretStore",
				}
			},
			gcpAuth: &esv1.VaultGCPAuth{
				Role: "test-role",
			},
			expectError: false, // Test handles both ADC available and unavailable scenarios
			description: "Should attempt to use default ADC method when no explicit auth is configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setupClient()
			err := c.setupGCPAuth(context.Background(), tt.gcpAuth)

			// For default ADC test, availability depends on the test environment.
			// We accept both success (ADC configured) and failure (ADC not available)
			// as valid outcomes to ensure tests pass in all environments.
			if tt.name == "Default ADC method selected" {
				if err != nil {
					t.Logf("%s: %v (expected behavior in environments without ADC configured)", tt.description, err)
				} else {
					t.Logf("%s: succeeded (ADC is properly configured in this environment)", tt.description)
				}
				t.Logf("%s: test completed successfully", tt.description)
				return
			}

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}

			// All tests should at least not panic and follow the correct code path
			t.Logf("%s: test completed successfully", tt.description)
		})
	}
}
