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

package secretserver

import (
	"context"
	"math/rand"
	"testing"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
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
		cfg  esv1beta1.SecretServerProvider
		want bool
	}{
		"true when Username references a secret without explicit namespace": {
			cfg: esv1beta1.SecretServerProvider{
				Username: &esv1beta1.SecretServerProviderRef{
					SecretRef: &v1.SecretKeySelector{Name: "foo"},
				},
				Password: &esv1beta1.SecretServerProviderRef{SecretRef: nil},
			},
			want: true,
		},
		"true when password references a secret without explicit namespace": {
			cfg: esv1beta1.SecretServerProvider{
				Username: &esv1beta1.SecretServerProviderRef{SecretRef: nil},
				Password: &esv1beta1.SecretServerProviderRef{
					SecretRef: &v1.SecretKeySelector{Name: "foo"},
				},
			},
			want: true,
		},
		"false when neither Username or Password reference a secret": {
			cfg: esv1beta1.SecretServerProvider{
				Username: &esv1beta1.SecretServerProviderRef{SecretRef: nil},
				Password: &esv1beta1.SecretServerProviderRef{SecretRef: nil},
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
	ambiguousSecretRef := &esv1beta1.SecretServerProviderRef{
		SecretRef: &v1.SecretKeySelector{Name: "foo"}, Value: "foo",
	}
	testURL := "https://example.com"

	tests := map[string]struct {
		cfg  esv1beta1.SecretServerProvider
		want error
	}{
		"invalid without username": {
			cfg: esv1beta1.SecretServerProvider{
				Username:  nil,
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: errEmptyUserName,
		},
		"invalid without password": {
			cfg: esv1beta1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  nil,
				ServerURL: testURL,
			},
			want: errEmptyPassword,
		},
		"invalid without serverURL": {
			cfg: esv1beta1.SecretServerProvider{
				Username: validSecretRefUsingValue,
				Password: validSecretRefUsingValue,
				/*ServerURL: testURL,*/
			},
			want: errEmptyServerURL,
		},
		"invalid with ambiguous Username": {
			cfg: esv1beta1.SecretServerProvider{
				Username:  ambiguousSecretRef,
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: errSecretRefAndValueConflict,
		},
		"invalid with ambiguous Password": {
			cfg: esv1beta1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  ambiguousSecretRef,
				ServerURL: testURL,
			},
			want: errSecretRefAndValueConflict,
		},
		"invalid with invalid Username": {
			cfg: esv1beta1.SecretServerProvider{
				Username:  makeSecretRefUsingValue(""),
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: errSecretRefAndValueMissing,
		},
		"invalid with invalid Password": {
			cfg: esv1beta1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  makeSecretRefUsingValue(""),
				ServerURL: testURL,
			},
			want: errSecretRefAndValueMissing,
		},
		"valid with tenant/clientID/clientSecret": {
			cfg: esv1beta1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: nil,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						SecretServer: &tc.cfg,
					},
				},
			}
			p := &Provider{}
			_, got := p.ValidateStore(&s)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNewClient(t *testing.T) {
	userNameKey := "username"
	userNameValue := "foo"
	passwordKey := "password"
	passwordValue := generateRandomString()

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Data: map[string][]byte{
			userNameKey: []byte(userNameValue),
			passwordKey: []byte(passwordValue),
		},
	}

	validProvider := &esv1beta1.SecretServerProvider{
		Username:  makeSecretRefUsingRef(clientSecret.Name, userNameKey),
		Password:  makeSecretRefUsingRef(clientSecret.Name, passwordKey),
		ServerURL: "https://example.com",
	}

	tests := map[string]struct {
		store    esv1beta1.GenericStore          // leave nil for namespaced store
		provider *esv1beta1.SecretServerProvider // discarded when store is set
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
						SecretServer: validProvider,
					},
				},
			},
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errClusterStoreRequiresNamespace)
			},
		},
		"dangling password ref": {
			provider: &esv1beta1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  makeSecretRefUsingRef("typo", passwordKey),
				ServerURL: validProvider.ServerURL,
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.True(t, kubeErrors.IsNotFound(err))
			},
		},
		"dangling username ref": {
			provider: &esv1beta1.SecretServerProvider{
				Username:  makeSecretRefUsingRef("typo", userNameKey),
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.True(t, kubeErrors.IsNotFound(err))
			},
		},
		"secret ref without name": {
			provider: &esv1beta1.SecretServerProvider{
				Username:  makeSecretRefUsingRef("", userNameKey),
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errMissingSecretName)
			},
		},
		"secret ref without key": {
			provider: &esv1beta1.SecretServerProvider{
				Username:  validProvider.Password,
				Password:  makeSecretRefUsingRef(clientSecret.Name, ""),
				ServerURL: validProvider.ServerURL,
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errMissingSecretKey)
			},
		},
		"secret ref with non-existent keys": {
			provider: &esv1beta1.SecretServerProvider{
				Username:  makeSecretRefUsingRef(clientSecret.Name, "typo"),
				Password:  makeSecretRefUsingRef(clientSecret.Name, passwordKey),
				ServerURL: validProvider.ServerURL,
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
			provider: &esv1beta1.SecretServerProvider{
				Username:  makeSecretRefUsingValue(userNameValue),
				Password:  makeSecretRefUsingValue(passwordValue),
				ServerURL: validProvider.ServerURL,
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
		},
		"cluster secret store": {
			store: &esv1beta1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1beta1.ClusterSecretStoreKind},
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						SecretServer: &esv1beta1.SecretServerProvider{
							Username:  makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, userNameKey),
							Password:  makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, passwordKey),
							ServerURL: validProvider.ServerURL,
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
							SecretServer: tc.provider,
						},
					},
				}
			}
			sc, err := p.NewClient(context.Background(), store, tc.kube, clientSecret.Namespace)
			if tc.errCheck == nil {
				assert.NoError(t, err)
				delineaClient, ok := sc.(*client)
				assert.True(t, ok)
				secretServerClient, ok := delineaClient.api.(*server.Server)
				assert.True(t, ok)
				assert.Equal(t, server.UserCredential{
					Username: userNameValue,
					Password: passwordValue,
				}, secretServerClient.Configuration.Credentials)
			} else {
				assert.Nil(t, sc)
				tc.errCheck(t, err)
			}
		})
	}
}

func makeSecretRefUsingNamespacedRef(namespace, name, key string) *esv1beta1.SecretServerProviderRef {
	return &esv1beta1.SecretServerProviderRef{
		SecretRef: &v1.SecretKeySelector{Namespace: utils.Ptr(namespace), Name: name, Key: key},
	}
}

func makeSecretRefUsingValue(val string) *esv1beta1.SecretServerProviderRef {
	return &esv1beta1.SecretServerProviderRef{Value: val}
}

func makeSecretRefUsingRef(name, key string) *esv1beta1.SecretServerProviderRef {
	return &esv1beta1.SecretServerProviderRef{
		SecretRef: &v1.SecretKeySelector{Name: name, Key: key},
	}
}

func generateRandomString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}
