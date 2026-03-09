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

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/bradleyfalzon/ghinstallation/v2"
	github "github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type getSecretFn func(ctx context.Context, ref esv1.PushSecretRemoteRef) (*github.Secret, *github.Response, error)

func withGetSecretFn(secret *github.Secret, response *github.Response, err error) getSecretFn {
	return func(_ context.Context, _ esv1.PushSecretRemoteRef) (*github.Secret, *github.Response, error) {
		return secret, response, err
	}
}

type getPublicKeyFn func(ctx context.Context) (*github.PublicKey, *github.Response, error)

func withGetPublicKeyFn(key *github.PublicKey, response *github.Response, err error) getPublicKeyFn {
	return func(_ context.Context) (*github.PublicKey, *github.Response, error) {
		return key, response, err
	}
}

type createOrUpdateSecretFn func(ctx context.Context, encryptedSecret *github.EncryptedSecret) (*github.Response, error)

func withCreateOrUpdateSecretFn(response *github.Response, err error) createOrUpdateSecretFn {
	return func(_ context.Context, _ *github.EncryptedSecret) (*github.Response, error) {
		return response, err
	}
}

func TestSecretExists(t *testing.T) {
	type testCase struct {
		name        string
		prov        *esv1.GithubProvider
		remoteRef   esv1.PushSecretData
		getSecretFn getSecretFn
		wantErr     error
		exists      bool
	}
	tests := []testCase{
		{
			name:        "getSecret fail",
			getSecretFn: withGetSecretFn(nil, nil, errors.New("boom")),
			exists:      false,
			wantErr:     errors.New("error fetching secret"),
		},
		{
			name:        "no secret",
			getSecretFn: withGetSecretFn(nil, nil, nil),
			exists:      false,
		},
		{
			name:        "with secret",
			getSecretFn: withGetSecretFn(&github.Secret{}, nil, nil),
			exists:      true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := Client{
				provider: test.prov,
			}
			g.getSecretFn = test.getSecretFn
			ok, err := g.SecretExists(context.TODO(), test.remoteRef)
			assert.Equal(t, test.exists, ok)
			if test.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.wantErr.Error())
			}
		})
	}
}

func TestPushSecret(t *testing.T) {
	type testCase struct {
		name             string
		prov             *esv1.GithubProvider
		secret           *corev1.Secret
		remoteRef        esv1.PushSecretData
		getSecretFn      getSecretFn
		getPublicKeyFn   getPublicKeyFn
		createOrUpdateFn createOrUpdateSecretFn
		wantErr          error
	}
	tests := []testCase{
		{
			name:        "failGetSecretFn",
			getSecretFn: withGetSecretFn(nil, nil, errors.New("boom")),
			wantErr:     errors.New("error fetching secret"),
		},
		{
			name: "failGetPublicKey",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(nil, nil, errors.New("boom")),
			wantErr:        errors.New("error fetching public key"),
		},
		{
			name: "failDecodeKey",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("broken"),
				KeyID: ptr.To("123"),
			}, nil, nil),
			wantErr: errors.New("unable to decode public key"),
		},
		{
			name: "failSecretData",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("Cg=="),
				KeyID: ptr.To("123"),
			}, nil, nil),
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bar"),
				},
			},
			remoteRef: esv1alpha1.PushSecretData{
				Match: esv1alpha1.PushSecretMatch{
					SecretKey: "bar",
				},
			},
			wantErr: errors.New("not found in secret"),
		},
		{
			name: "failSecretData",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("Zm9vYmFyCg=="),
				KeyID: ptr.To("123"),
			}, nil, nil),
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bingg"),
				},
			},
			remoteRef: esv1alpha1.PushSecretData{
				Match: esv1alpha1.PushSecretMatch{
					SecretKey: "foo",
				},
			},
			createOrUpdateFn: withCreateOrUpdateSecretFn(nil, errors.New("boom")),
			wantErr:          errors.New("failed to create secret"),
		},
		{
			name: "Success",
			getSecretFn: withGetSecretFn(&github.Secret{
				Name: "foo",
			}, nil, nil),
			getPublicKeyFn: withGetPublicKeyFn(&github.PublicKey{
				Key:   ptr.To("Zm9vYmFyCg=="),
				KeyID: ptr.To("123"),
			}, nil, nil),
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"foo": []byte("bingg"),
				},
			},
			remoteRef: esv1alpha1.PushSecretData{
				Match: esv1alpha1.PushSecretMatch{
					SecretKey: "foo",
				},
			},
			createOrUpdateFn: withCreateOrUpdateSecretFn(nil, nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := Client{
				provider: test.prov,
			}
			g.getSecretFn = test.getSecretFn
			g.getPublicKeyFn = test.getPublicKeyFn
			g.createOrUpdateFn = test.createOrUpdateFn
			err := g.PushSecret(context.TODO(), test.secret, test.remoteRef)
			if test.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.wantErr.Error())
			}
		})
	}
}

// generateTestPrivateKey generates a PEM-encoded RSA private key for testing.
func generateTestPrivateKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return string(privateKeyPEM), nil
}

func TestAuthWithPrivateKey(t *testing.T) {
	// Generate a valid private key for testing
	privateKeyPEM, err := generateTestPrivateKey()
	require.NoError(t, err)

	tests := []struct {
		name           string
		provider       *esv1.GithubProvider
		secret         *corev1.Secret
		wantErr        bool
		wantBaseURL    string
		wantUploadURL  string
		checkTransport bool
	}{
		{
			name: "GitHub.com (default)",
			provider: &esv1.GithubProvider{
				AppID:          1,
				InstallationID: 1,
				URL:            "https://github.com/",
				Auth: esv1.GithubAppAuth{
					PrivateKey: esmeta.SecretKeySelector{
						Name: "test-secret",
						Key:  "private-key",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"private-key": []byte(privateKeyPEM),
				},
			},
			wantErr:        false,
			wantBaseURL:    "https://api.github.com/",
			checkTransport: false, // For default GitHub, we don't modify transport
		},
		{
			name: "GitHub Enterprise with custom URL",
			provider: &esv1.GithubProvider{
				AppID:          1,
				InstallationID: 1,
				URL:            "https://github.enterprise.com/",
				Auth: esv1.GithubAppAuth{
					PrivateKey: esmeta.SecretKeySelector{
						Name: "test-secret",
						Key:  "private-key",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"private-key": []byte(privateKeyPEM),
				},
			},
			wantErr:        false,
			wantBaseURL:    "https://github.enterprise.com/api/v3/",
			wantUploadURL:  "https://github.enterprise.com/api/uploads/",
			checkTransport: true,
		},
		{
			name: "GitHub Enterprise with separate upload URL",
			provider: &esv1.GithubProvider{
				AppID:          1,
				InstallationID: 1,
				URL:            "https://github.enterprise.com/api/v3",
				UploadURL:      "https://uploads.github.enterprise.com/api/v3",
				Auth: esv1.GithubAppAuth{
					PrivateKey: esmeta.SecretKeySelector{
						Name: "test-secret",
						Key:  "private-key",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"private-key": []byte(privateKeyPEM),
				},
			},
			wantErr:        false,
			wantBaseURL:    "https://github.enterprise.com/api/v3/",
			wantUploadURL:  "https://uploads.github.enterprise.com/api/v3/api/uploads/",
			checkTransport: true,
		},
		{
			name: "Empty URL (default to github.com)",
			provider: &esv1.GithubProvider{
				AppID:          1,
				InstallationID: 1,
				URL:            "",
				Auth: esv1.GithubAppAuth{
					PrivateKey: esmeta.SecretKeySelector{
						Name: "test-secret",
						Key:  "private-key",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"private-key": []byte(privateKeyPEM),
				},
			},
			wantErr:        false,
			wantBaseURL:    "https://api.github.com/",
			checkTransport: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake Kubernetes client with the secret
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.secret).
				Build()

			// Create the GitHub client
			client := &Client{
				crClient:  fakeClient,
				provider:  tt.provider,
				namespace: "default",
				storeKind: "SecretStore",
			}

			// Call AuthWithPrivateKey
			ghClient, err := client.AuthWithPrivateKey(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, ghClient)

			// Verify the BaseURL is set correctly
			assert.Equal(t, tt.wantBaseURL, ghClient.BaseURL.String())

			// If UploadURL is specified, verify it
			if tt.wantUploadURL != "" {
				assert.Equal(t, tt.wantUploadURL, ghClient.UploadURL.String())
			}

			// For GitHub Enterprise, verify the transport BaseURL is also set
			if tt.checkTransport {
				transport := ghClient.Client().Transport
				require.NotNil(t, transport)

				// Type assert to ghinstallation.Transport
				ghTransport, ok := transport.(*ghinstallation.Transport)
				require.True(t, ok, "Expected transport to be *ghinstallation.Transport")

				// Verify the BaseURL is set on the transport
				assert.Equal(t, tt.wantBaseURL, ghTransport.BaseURL,
					"Transport BaseURL should match the enterprise URL")
			}
		})
	}
}
