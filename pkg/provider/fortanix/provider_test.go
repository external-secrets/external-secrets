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
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestNewClient(t *testing.T) {
	t.Run("should create new client", func(t *testing.T) {
		ctx := context.Background()
		p := &Provider{}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-name",
				Namespace: "test",
			},
			Data: map[string][]byte{
				"apiKey": []byte("apiKey"),
			},
		}
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
		scheme := runtime.NewScheme()
		require.NoError(t, esv1beta1.AddToScheme(scheme))
		require.NoError(t, corev1.AddToScheme(scheme))
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret, &s).Build()
		_, err := p.NewClient(ctx, &s, fakeClient, "test")

		assert.Nil(t, err)
	})

	t.Run("should fail to create new client if secret is missing", func(t *testing.T) {
		ctx := context.Background()
		p := &Provider{}
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
		scheme := runtime.NewScheme()
		require.NoError(t, esv1beta1.AddToScheme(scheme))
		require.NoError(t, corev1.AddToScheme(scheme))
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&s).Build()
		_, err := p.NewClient(ctx, &s, fakeClient, "test")

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
						Namespace: to.Ptr("namespace"),
					},
				},
			},
			want: errors.New("namespace should either be empty or match the namespace of the SecretStore for a namespaced SecretStore"),
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
