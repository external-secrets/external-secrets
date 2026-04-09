//go:build integration

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

package vaultwarden

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	integrationSecretName = "vaultwarden-integration-creds"
	integrationNamespace  = "default"

	envVaultwardenURL            = "VAULTWARDEN_URL"
	envVaultwardenClientID       = "VAULTWARDEN_CLIENT_ID"
	envVaultwardenClientSecret   = "VAULTWARDEN_CLIENT_SECRET"
	envVaultwardenMasterPassword = "VAULTWARDEN_MASTER_PASSWORD"
)

// fakePushSecretData is a local implementation of esv1.PushSecretData for tests.
type fakePushSecretData struct {
	secretKey string
	remoteKey string
	property  string
}

func (f fakePushSecretData) GetMetadata() *apiextensionsv1.JSON { return nil }
func (f fakePushSecretData) GetSecretKey() string               { return f.secretKey }
func (f fakePushSecretData) GetRemoteKey() string               { return f.remoteKey }
func (f fakePushSecretData) GetProperty() string                { return f.property }

// fakePushSecretRemoteRef is a local implementation of esv1.PushSecretRemoteRef for tests.
type fakePushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (f fakePushSecretRemoteRef) GetRemoteKey() string { return f.remoteKey }
func (f fakePushSecretRemoteRef) GetProperty() string  { return f.property }

// buildIntegrationClient sets up a *Client wired to a fake K8s client that holds the
// Vaultwarden credentials from environment variables. No real Kubernetes cluster is needed.
func buildIntegrationClient(t *testing.T) *Client {
	t.Helper()

	vwURL := requireEnv(t, envVaultwardenURL)
	clientID := requireEnv(t, envVaultwardenClientID)
	clientSecret := requireEnv(t, envVaultwardenClientSecret)
	masterPassword := requireEnv(t, envVaultwardenMasterPassword)

	// Build a fake K8s secret holding all three credentials under well-known keys.
	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integrationSecretName,
			Namespace: integrationNamespace,
		},
		Data: map[string][]byte{
			"clientID":       []byte(clientID),
			"clientSecret":   []byte(clientSecret),
			"masterPassword": []byte(masterPassword),
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	kubeClient := clientfake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(k8sSecret).
		Build()

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vaultwarden-test-store",
			Namespace: integrationNamespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Vaultwarden: &esv1.VaultwardenProvider{
					URL: vwURL,
					Auth: esv1.VaultwardenAuth{
						SecretRef: esv1.VaultwardenSecretRef{
							ClientID: esmeta.SecretKeySelector{
								Name: integrationSecretName,
								Key:  "clientID",
							},
							ClientSecret: esmeta.SecretKeySelector{
								Name: integrationSecretName,
								Key:  "clientSecret",
							},
							MasterPassword: esmeta.SecretKeySelector{
								Name: integrationSecretName,
								Key:  "masterPassword",
							},
						},
					},
				},
			},
		},
	}

	// TODO: support custom CA via provider config.
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}

	return &Client{
		httpClient: httpClient,
		provider:   store.Spec.Provider.Vaultwarden,
		crClient:   kubeClient,
		namespace:  integrationNamespace,
		store:      store,
	}
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skipf("skipping integration test: %s not set", name)
	}
	return v
}

func uniqueName() string {
	return fmt.Sprintf("ESO_VW_TEST_%d", time.Now().UnixNano())
}

// sharedClient is a package-level client shared across all integration tests
// to avoid repeated token fetches hitting Vaultwarden's rate limiter.
var sharedClient *Client

func TestMain(m *testing.M) {
	// Build the shared client only if env vars are set; otherwise tests will skip themselves.
	if os.Getenv(envVaultwardenURL) != "" &&
		os.Getenv(envVaultwardenClientID) != "" &&
		os.Getenv(envVaultwardenClientSecret) != "" &&
		os.Getenv(envVaultwardenMasterPassword) != "" {
		// We can't call t.Helper() here, so build directly.
		vwURL := os.Getenv(envVaultwardenURL)
		clientID := os.Getenv(envVaultwardenClientID)
		clientSecret := os.Getenv(envVaultwardenClientSecret)
		masterPassword := os.Getenv(envVaultwardenMasterPassword)

		k8sSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: integrationSecretName, Namespace: integrationNamespace},
			Data: map[string][]byte{
				"clientID":       []byte(clientID),
				"clientSecret":   []byte(clientSecret),
				"masterPassword": []byte(masterPassword),
			},
		}
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		kubeClient := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(k8sSecret).Build()
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{Name: "vaultwarden-test-store", Namespace: integrationNamespace},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Vaultwarden: &esv1.VaultwardenProvider{
						URL: vwURL,
						Auth: esv1.VaultwardenAuth{
							SecretRef: esv1.VaultwardenSecretRef{
								ClientID:       esmeta.SecretKeySelector{Name: integrationSecretName, Key: "clientID"},
								ClientSecret:   esmeta.SecretKeySelector{Name: integrationSecretName, Key: "clientSecret"},
								MasterPassword: esmeta.SecretKeySelector{Name: integrationSecretName, Key: "masterPassword"},
							},
						},
					},
				},
			},
		}
		sharedClient = &Client{
			httpClient: &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}, //nolint:gosec
			provider:   store.Spec.Provider.Vaultwarden,
			crClient:   kubeClient,
			namespace:  integrationNamespace,
			store:      store,
		}
	}
	os.Exit(m.Run())
}

// getSharedClient returns the shared client, or skips the test if env vars are missing.
func getSharedClient(t *testing.T) *Client {
	t.Helper()
	if sharedClient == nil {
		t.Skip("skipping integration test: VAULTWARDEN_* env vars not set")
	}
	return sharedClient
}

// TestIntegration_PushExistsDelete pushes a secret, verifies it exists, then deletes it and confirms it's gone.
func TestIntegration_PushExistsDelete(t *testing.T) {
	c := getSharedClient(t)
	ctx := context.Background()

	name := uniqueName()
	value := "hello-from-eso"

	k8sSecret := &corev1.Secret{
		Data: map[string][]byte{
			"data": []byte(value),
		},
	}
	data := fakePushSecretData{secretKey: "data", remoteKey: name}

	// Push
	require.NoError(t, c.PushSecret(ctx, k8sSecret, data))

	// Exists
	exists, err := c.SecretExists(ctx, fakePushSecretRemoteRef{remoteKey: name})
	require.NoError(t, err)
	assert.True(t, exists, "secret should exist after push")

	// GetSecret round-trip
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: name})
	require.NoError(t, err)
	assert.Equal(t, value, string(got))

	// Delete
	require.NoError(t, c.DeleteSecret(ctx, fakePushSecretRemoteRef{remoteKey: name}))

	// Gone
	exists, err = c.SecretExists(ctx, fakePushSecretRemoteRef{remoteKey: name})
	require.NoError(t, err)
	assert.False(t, exists, "secret should be gone after delete")

	// Idempotent second delete should not error.
	require.NoError(t, c.DeleteSecret(ctx, fakePushSecretRemoteRef{remoteKey: name}))
}

// TestIntegration_PushUpdate pushes the same secret name twice with different values and confirms the update.
func TestIntegration_PushUpdate(t *testing.T) {
	c := getSharedClient(t)
	ctx := context.Background()

	name := uniqueName()

	push := func(value string) {
		t.Helper()
		k8sSecret := &corev1.Secret{
			Data: map[string][]byte{"data": []byte(value)},
		}
		require.NoError(t, c.PushSecret(ctx, k8sSecret, fakePushSecretData{secretKey: "data", remoteKey: name}))
	}

	t.Cleanup(func() {
		_ = c.DeleteSecret(ctx, fakePushSecretRemoteRef{remoteKey: name})
	})

	push("first-value")
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: name})
	require.NoError(t, err)
	assert.Equal(t, "first-value", string(got))

	push("second-value")
	got, err = c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: name})
	require.NoError(t, err)
	assert.Equal(t, "second-value", string(got))
}

// TestIntegration_GetSecretMap pushes a JSON-valued secret and retrieves it back as a map.
func TestIntegration_GetSecretMap(t *testing.T) {
	c := getSharedClient(t)
	ctx := context.Background()

	name := uniqueName()

	payload := map[string]string{
		"username": "alice",
		"password": "s3cr3t",
		"host":     "db.example.com",
	}
	jsonBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	k8sSecret := &corev1.Secret{
		Data: map[string][]byte{"data": jsonBytes},
	}

	require.NoError(t, c.PushSecret(ctx, k8sSecret, fakePushSecretData{secretKey: "data", remoteKey: name}))
	t.Cleanup(func() {
		_ = c.DeleteSecret(ctx, fakePushSecretRemoteRef{remoteKey: name})
	})

	m, err := c.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: name})
	require.NoError(t, err)

	assert.Equal(t, "alice", string(m["username"]))
	assert.Equal(t, "s3cr3t", string(m["password"]))
	assert.Equal(t, "db.example.com", string(m["host"]))
}

// TestIntegration_Validate checks that Validate() returns Ready for a working configuration.
func TestIntegration_Validate(t *testing.T) {
	c := getSharedClient(t)

	result, err := c.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
}
