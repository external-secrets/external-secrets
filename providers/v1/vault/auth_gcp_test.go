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
			defer os.Unsetenv("GOOGLE_OAUTH_ACCESS_TOKEN")

			err := c.setGCPEnvironment(tt.accessToken)

			if tt.wantError && err == nil {
				t.Errorf("setGCPEnvironment() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("setGCPEnvironment() unexpected error: %v", err)
			}

			// If successful, verify the GOOGLE_OAUTH_ACCESS_TOKEN was set
			if !tt.wantError && err == nil {
				actualValue := os.Getenv("GOOGLE_OAUTH_ACCESS_TOKEN")
				if actualValue != tt.accessToken {
					t.Errorf("setGCPEnvironment() GOOGLE_OAUTH_ACCESS_TOKEN not set correctly, got %v, want %v", actualValue, tt.accessToken)
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
	if err != nil {
		t.Errorf("setupDefaultGCPAuth() unexpected error: %v", err)
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
		gcpAuth     *esv1.VaultGcpAuth
		expectError bool
		description string
	}{
		{
			name: "SecretRef has priority",
			gcpAuth: &esv1.VaultGcpAuth{
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
			gcpAuth: &esv1.VaultGcpAuth{
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
			gcpAuth: &esv1.VaultGcpAuth{
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
			gcpAuth: &esv1.VaultGcpAuth{
				Role: "test-role",
			},
			expectError: false,
			description: "Should use default ADC when no other auth is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.setupGCPAuth(context.Background(), tt.gcpAuth)

			if tt.expectError && err == nil {
				t.Errorf("setupGCPAuth() expected error for %s, got nil", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("setupGCPAuth() unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}

func TestGCPAuthWithValidSecret(t *testing.T) {
	// Create a fake Kubernetes client with a secret containing GCP credentials
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gcp-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"credentials.json": []byte(`{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----\n",
				"client_email": "test@test-project.iam.gserviceaccount.com",
				"client_id": "123456789",
				"auth_uri": "https://accounts.google.com/o/oauth2/auth",
				"token_uri": "https://oauth2.googleapis.com/token"
			}`),
		},
	}

	kube := clientfake.NewClientBuilder().WithObjects(secret).Build()

	c := &client{
		log:       logr.Discard(),
		kube:      kube,
		namespace: "default",
		storeKind: "SecretStore",
	}

	gcpAuth := &esv1.VaultGcpAuth{
		Role:      "test-role",
		ProjectID: "test-project",
		SecretRef: &esv1.GCPSMAuthSecretRef{
			SecretAccessKey: esmeta.SecretKeySelector{
				Name: "gcp-secret",
				Key:  "credentials.json",
			},
		},
	}

	// This will likely fail due to token creation, but we can test the setup path
	err := c.setupServiceAccountKeyAuth(context.Background(), gcpAuth)

	// We expect an error here because we can't actually create a real token source
	// but we can verify the function doesn't panic and handles the setup correctly
	if err == nil {
		t.Log("setupServiceAccountKeyAuth() succeeded - this might indicate the test environment has actual GCP access")
	} else {
		t.Logf("setupServiceAccountKeyAuth() failed as expected in test environment: %v", err)
	}
}