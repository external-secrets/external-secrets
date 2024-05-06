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
package fortanix

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func pointer[T any](d T) *T {
	return &d
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(data)
}

func createMockKubernetesClient(t *testing.T) kubeclient.Client {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1":
			respondJSON(w, metav1.APIResourceList{
				APIResources: []metav1.APIResource{
					{
						Name:       "secrets",
						Namespaced: true,
						Kind:       "Secret",
						Verbs: metav1.Verbs{
							"get",
						},
					},
				},
			})
		case "/api/v1/namespaces/test/secrets/secret-name":
			respondJSON(w, corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret-name",
				},
				Data: map[string][]byte{
					"apiKey": []byte("apiKey"),
				},
			})
		case "/api/v1/namespaces/test/secrets/missing-secret":
			w.WriteHeader(404)
			respondJSON(w, metav1.Status{
				Code: 404,
			})
		}
	}))
	t.Cleanup(server.Close)

	clientConfig := clientcmd.NewDefaultClientConfig(clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"test": {
				Server: server.URL,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test": {
				Token: "token",
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test": {
				Cluster:  "test",
				AuthInfo: "test",
			},
		},
		CurrentContext: "test",
	}, &clientcmd.ConfigOverrides{})

	restConfig, err := clientConfig.ClientConfig()
	assert.Nil(t, err)
	c, err := kubeclient.New(restConfig, kubeclient.Options{})
	assert.Nil(t, err)

	return c
}

func TestNewClient(t *testing.T) {
	t.Run("should create new client", func(t *testing.T) {
		ctx := context.Background()
		p := &Provider{}
		c := createMockKubernetesClient(t)
		s := esv1beta1.SecretStore{
			Spec: esv1beta1.SecretStoreSpec{
				Provider: &esv1beta1.SecretStoreProvider{
					Fortanix: &esv1beta1.FortanixProvider{
						APIKey: &esv1beta1.FortanixProviderSecretRef{
							SecretRef: &v1.SecretKeySelector{
								Name: "secret-name",
								Key:  "apiKey",
							},
						},
					},
				},
			},
		}

		_, err := p.NewClient(ctx, &s, c, "test")

		assert.Nil(t, err)
	})

	t.Run("should fail to create new client if secret is missing", func(t *testing.T) {
		ctx := context.Background()
		p := &Provider{}
		c := createMockKubernetesClient(t)
		s := esv1beta1.SecretStore{
			Spec: esv1beta1.SecretStoreSpec{
				Provider: &esv1beta1.SecretStoreProvider{
					Fortanix: &esv1beta1.FortanixProvider{
						APIKey: &esv1beta1.FortanixProviderSecretRef{
							SecretRef: &v1.SecretKeySelector{
								Name: "missing-secret",
								Key:  "apiKey",
							},
						},
					},
				},
			},
		}

		_, err := p.NewClient(ctx, &s, c, "test")

		assert.ErrorContains(t, err, "cannot resolve secret key ref")
	})
}

func TestValidateStore(t *testing.T) {
	tests := map[string]struct {
		cfg  esv1beta1.FortanixProvider
		want error
	}{
		"missing api key": {
			cfg:  esv1beta1.FortanixProvider{},
			want: errors.New("apiKey is required"),
		},
		"missing api key secret ref": {
			cfg: esv1beta1.FortanixProvider{
				APIKey: &esv1beta1.FortanixProviderSecretRef{},
			},
			want: errors.New("apiKey.secretRef is required"),
		},
		"missing api key secret ref name": {
			cfg: esv1beta1.FortanixProvider{
				APIKey: &esv1beta1.FortanixProviderSecretRef{
					SecretRef: &v1.SecretKeySelector{
						Key: "key",
					},
				},
			},
			want: errors.New("apiKey.secretRef.name is required"),
		},
		"missing api key secret ref key": {
			cfg: esv1beta1.FortanixProvider{
				APIKey: &esv1beta1.FortanixProviderSecretRef{
					SecretRef: &v1.SecretKeySelector{
						Name: "name",
					},
				},
			},
			want: errors.New("apiKey.secretRef.key is required"),
		},
		"disallowed namespace in store ref": {
			cfg: esv1beta1.FortanixProvider{
				APIKey: &esv1beta1.FortanixProviderSecretRef{
					SecretRef: &v1.SecretKeySelector{
						Key:       "key",
						Name:      "name",
						Namespace: pointer("namespace"),
					},
				},
			},
			want: errors.New("namespace not allowed with namespaced SecretStore"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Fortanix: &tc.cfg,
					},
				},
			}
			p := &Provider{}
			_, got := p.ValidateStore(&s)
			assert.Equal(t, tc.want, got)
		})
	}
}
