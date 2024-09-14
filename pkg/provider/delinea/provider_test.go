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

package delinea

import (
	"context"
	"testing"

	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kubeErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

func TestDoesConfigDependOnNamespace(t *testing.T) {
	tests := map[string]struct {
		cfg  esv1beta1.DelineaProvider
		want bool
	}{
		"true when client ID references a secret without explicit namespace": {
			cfg: esv1beta1.DelineaProvider{
				ClientID: &esv1beta1.DelineaProviderSecretRef{
					SecretRef: &v1.SecretKeySelector{Name: "foo"},
				},
				ClientSecret: &esv1beta1.DelineaProviderSecretRef{SecretRef: nil},
			},
			want: true,
		},
		"true when client secret references a secret without explicit namespace": {
			cfg: esv1beta1.DelineaProvider{
				ClientID: &esv1beta1.DelineaProviderSecretRef{SecretRef: nil},
				ClientSecret: &esv1beta1.DelineaProviderSecretRef{
					SecretRef: &v1.SecretKeySelector{Name: "foo"},
				},
			},
			want: true,
		},
		"false when neither client ID nor secret reference a secret": {
			cfg: esv1beta1.DelineaProvider{
				ClientID:     &esv1beta1.DelineaProviderSecretRef{SecretRef: nil},
				ClientSecret: &esv1beta1.DelineaProviderSecretRef{SecretRef: nil},
			},
			want: false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := doesConfigDependOnNamespace(&tc.cfg)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateStore(t *testing.T) {
	validSecretRefUsingValue := makeSecretRefUsingValue("foo")
	ambiguousSecretRef := &esv1beta1.DelineaProviderSecretRef{
		SecretRef: &v1.SecretKeySelector{Name: "foo"}, Value: "foo",
	}
	tests := map[string]struct {
		cfg  esv1beta1.DelineaProvider
		want error
	}{
		"invalid without tenant": {
			cfg: esv1beta1.DelineaProvider{
				Tenant:       "",
				ClientID:     validSecretRefUsingValue,
				ClientSecret: validSecretRefUsingValue,
			},
			want: errEmptyTenant,
		},
		"invalid without clientID": {
			cfg: esv1beta1.DelineaProvider{
				Tenant: "foo",
				// ClientID omitted
				ClientSecret: validSecretRefUsingValue,
			},
			want: errEmptyClientID,
		},
		"invalid without clientSecret": {
			cfg: esv1beta1.DelineaProvider{
				Tenant:   "foo",
				ClientID: validSecretRefUsingValue,
				// ClientSecret omitted
			},
			want: errEmptyClientSecret,
		},
		"invalid with ambiguous clientID": {
			cfg: esv1beta1.DelineaProvider{
				Tenant:       "foo",
				ClientID:     ambiguousSecretRef,
				ClientSecret: validSecretRefUsingValue,
			},
			want: errSecretRefAndValueConflict,
		},
		"invalid with ambiguous clientSecret": {
			cfg: esv1beta1.DelineaProvider{
				Tenant:       "foo",
				ClientID:     validSecretRefUsingValue,
				ClientSecret: ambiguousSecretRef,
			},
			want: errSecretRefAndValueConflict,
		},
		"invalid with invalid clientID": {
			cfg: esv1beta1.DelineaProvider{
				Tenant:       "foo",
				ClientID:     makeSecretRefUsingValue(""),
				ClientSecret: validSecretRefUsingValue,
			},
			want: errSecretRefAndValueMissing,
		},
		"invalid with invalid clientSecret": {
			cfg: esv1beta1.DelineaProvider{
				Tenant:       "foo",
				ClientID:     validSecretRefUsingValue,
				ClientSecret: makeSecretRefUsingValue(""),
			},
			want: errSecretRefAndValueMissing,
		},
		"valid with tenant/clientID/clientSecret": {
			cfg: esv1beta1.DelineaProvider{
				Tenant:       "foo",
				ClientID:     validSecretRefUsingValue,
				ClientSecret: validSecretRefUsingValue,
			},
			want: nil,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Delinea: &tc.cfg,
					},
				},
			}
			p := &Provider{}
			_, got := p.ValidateStore(&s)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateStoreBailsOnUnexpectedStore(t *testing.T) {
	tests := map[string]struct {
		store esv1beta1.GenericStore
		want  error
	}{
		"missing store": {nil, errMissingStore},
		"missing spec":  {&esv1beta1.SecretStore{}, errInvalidSpec},
		"missing provider": {&esv1beta1.SecretStore{
			Spec: esv1beta1.SecretStoreSpec{Provider: nil},
		}, errInvalidSpec},
		"missing delinea": {&esv1beta1.SecretStore{
			Spec: esv1beta1.SecretStoreSpec{Provider: &esv1beta1.SecretStoreProvider{
				Delinea: nil,
			}},
		}, errInvalidSpec},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := &Provider{}
			_, got := p.ValidateStore(tc.store)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNewClient(t *testing.T) {
	tenant := "foo"
	clientIDKey := "username"
	clientIDValue := "client id"
	clientSecretKey := "password"
	clientSecretValue := "client secret"

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Data: map[string][]byte{
			clientIDKey:     []byte(clientIDValue),
			clientSecretKey: []byte(clientSecretValue),
		},
	}

	validProvider := &esv1beta1.DelineaProvider{
		Tenant:       tenant,
		ClientID:     makeSecretRefUsingRef(clientSecret.Name, clientIDKey),
		ClientSecret: makeSecretRefUsingRef(clientSecret.Name, clientSecretKey),
	}

	tests := map[string]struct {
		store    esv1beta1.GenericStore     // leave nil for namespaced store
		provider *esv1beta1.DelineaProvider // discarded when store is set
		kube     kubeClient.Client
		errCheck func(t *testing.T, err error)
	}{
		"missing provider config": {
			provider: nil,
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errInvalidSpec)
			},
		},
		"namespace-dependent cluster secret store": {
			store: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1beta1.ClusterSecretStoreKind},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Delinea: validProvider,
					},
				},
			},
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errClusterStoreRequiresNamespace)
			},
		},
		"dangling client ID ref": {
			provider: &esv1beta1.DelineaProvider{
				Tenant:       tenant,
				ClientID:     makeSecretRefUsingRef("typo", clientIDKey),
				ClientSecret: makeSecretRefUsingRef(clientSecret.Name, clientSecretKey),
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.True(t, kubeErrors.IsNotFound(err))
			},
		},
		"dangling client secret ref": {
			provider: &esv1beta1.DelineaProvider{
				Tenant:       tenant,
				ClientID:     makeSecretRefUsingRef(clientSecret.Name, clientIDKey),
				ClientSecret: makeSecretRefUsingRef("typo", clientSecretKey),
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.True(t, kubeErrors.IsNotFound(err))
			},
		},
		"secret ref without name": {
			provider: &esv1beta1.DelineaProvider{
				Tenant:       tenant,
				ClientID:     makeSecretRefUsingRef("", clientIDKey),
				ClientSecret: makeSecretRefUsingRef(clientSecret.Name, clientSecretKey),
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errMissingSecretName)
			},
		},
		"secret ref without key": {
			provider: &esv1beta1.DelineaProvider{
				Tenant:       tenant,
				ClientID:     makeSecretRefUsingRef(clientSecret.Name, ""),
				ClientSecret: makeSecretRefUsingRef(clientSecret.Name, clientSecretKey),
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errMissingSecretKey)
			},
		},
		"secret ref with non-existent keys": {
			provider: &esv1beta1.DelineaProvider{
				Tenant:       tenant,
				ClientID:     makeSecretRefUsingRef(clientSecret.Name, "typo"),
				ClientSecret: makeSecretRefUsingRef(clientSecret.Name, clientSecretKey),
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.EqualError(t, err, "cannot find secret data for key: \"typo\"")
			},
		},
		"valid secret refs": {
			provider: validProvider,
			kube:     clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
		},
		"secret values": {
			provider: &esv1beta1.DelineaProvider{
				Tenant:       tenant,
				ClientID:     makeSecretRefUsingValue(clientIDValue),
				ClientSecret: makeSecretRefUsingValue(clientSecretValue),
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
		},
		"cluster secret store": {
			store: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1beta1.ClusterSecretStoreKind},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Delinea: &esv1beta1.DelineaProvider{
							Tenant:       tenant,
							ClientID:     makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, clientIDKey),
							ClientSecret: makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, clientSecretKey),
						},
					},
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := &Provider{}
			store := tc.store
			if store == nil {
				store = &esv1beta1.SecretStore{
					TypeMeta: metav1.TypeMeta{Kind: esv1beta1.SecretStoreKind},
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Delinea: tc.provider,
						},
					},
				}
			}
			sc, err := p.NewClient(context.Background(), store, tc.kube, clientSecret.Namespace)
			if tc.errCheck == nil {
				assert.NoError(t, err)
				delineaClient, ok := sc.(*client)
				assert.True(t, ok)
				dsvClient, ok := delineaClient.api.(*vault.Vault)
				assert.True(t, ok)
				assert.Equal(t, vault.Configuration{
					Credentials: vault.ClientCredential{
						ClientID:     clientIDValue,
						ClientSecret: clientSecretValue,
					},
					Tenant:      tenant,
					TLD:         "com",                                     // Default from Delinea
					URLTemplate: "https://%s.secretsvaultcloud.%s/v1/%s%s", // Default from Delinea
				}, dsvClient.Configuration)
			} else {
				assert.Nil(t, sc)
				tc.errCheck(t, err)
			}
		})
	}
}

func makeSecretRefUsingRef(name, key string) *esv1beta1.DelineaProviderSecretRef {
	return &esv1beta1.DelineaProviderSecretRef{
		SecretRef: &v1.SecretKeySelector{Name: name, Key: key},
	}
}

func makeSecretRefUsingNamespacedRef(namespace, name, key string) *esv1beta1.DelineaProviderSecretRef {
	return &esv1beta1.DelineaProviderSecretRef{
		SecretRef: &v1.SecretKeySelector{Namespace: utils.Ptr(namespace), Name: name, Key: key},
	}
}

func makeSecretRefUsingValue(val string) *esv1beta1.DelineaProviderSecretRef {
	return &esv1beta1.DelineaProviderSecretRef{Value: val}
}
