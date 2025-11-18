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

package secretserver

import (
	"context"
	"math/rand"
	"testing"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kubeErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

func TestDoesConfigDependOnNamespace(t *testing.T) {
	tests := map[string]struct {
		cfg  esv1.SecretServerProvider
		want bool
	}{
		"true when Username references a secret without explicit namespace": {
			cfg: esv1.SecretServerProvider{
				Username: &esv1.SecretServerProviderRef{
					SecretRef: &v1.SecretKeySelector{Name: "foo"},
				},
				Password: &esv1.SecretServerProviderRef{SecretRef: nil},
			},
			want: true,
		},
		"true when password references a secret without explicit namespace": {
			cfg: esv1.SecretServerProvider{
				Username: &esv1.SecretServerProviderRef{SecretRef: nil},
				Password: &esv1.SecretServerProviderRef{
					SecretRef: &v1.SecretKeySelector{Name: "foo"},
				},
			},
			want: true,
		},
		"false when neither Username or Password reference a secret": {
			cfg: esv1.SecretServerProvider{
				Username: &esv1.SecretServerProviderRef{SecretRef: nil},
				Password: &esv1.SecretServerProviderRef{SecretRef: nil},
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
	ambiguousSecretRef := &esv1.SecretServerProviderRef{
		SecretRef: &v1.SecretKeySelector{Name: "foo"}, Value: "foo",
	}
	testURL := "https://example.com"

	tests := map[string]struct {
		cfg  esv1.SecretServerProvider
		want error
	}{
		"invalid without username": {
			cfg: esv1.SecretServerProvider{
				Username:  nil,
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: errEmptyUserName,
		},
		"invalid without password": {
			cfg: esv1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  nil,
				ServerURL: testURL,
			},
			want: errEmptyPassword,
		},
		"invalid without serverURL": {
			cfg: esv1.SecretServerProvider{
				Username: validSecretRefUsingValue,
				Password: validSecretRefUsingValue,
				/*ServerURL: testURL,*/
			},
			want: errEmptyServerURL,
		},
		"invalid with ambiguous Username": {
			cfg: esv1.SecretServerProvider{
				Username:  ambiguousSecretRef,
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: errSecretRefAndValueConflict,
		},
		"invalid with ambiguous Password": {
			cfg: esv1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  ambiguousSecretRef,
				ServerURL: testURL,
			},
			want: errSecretRefAndValueConflict,
		},
		"invalid with invalid Username": {
			cfg: esv1.SecretServerProvider{
				Username:  makeSecretRefUsingValue(""),
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: errSecretRefAndValueMissing,
		},
		"invalid with invalid Password": {
			cfg: esv1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  makeSecretRefUsingValue(""),
				ServerURL: testURL,
			},
			want: errSecretRefAndValueMissing,
		},
		"valid with tenant/clientID/clientSecret": {
			cfg: esv1.SecretServerProvider{
				Username:  validSecretRefUsingValue,
				Password:  validSecretRefUsingValue,
				ServerURL: testURL,
			},
			want: nil,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
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
	domain := "domain1"

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
		Data: map[string][]byte{
			userNameKey: []byte(userNameValue),
			passwordKey: []byte(passwordValue),
		},
	}

	validProvider := &esv1.SecretServerProvider{
		Username:  makeSecretRefUsingRef(clientSecret.Name, userNameKey),
		Password:  makeSecretRefUsingRef(clientSecret.Name, passwordKey),
		ServerURL: "https://example.com",
	}

	clientSecretWithDomain := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "with-domain", Namespace: "default"},
		Data: map[string][]byte{
			userNameKey: []byte(userNameValue),
			passwordKey: []byte(passwordValue),
			domain:      []byte(domain),
		},
	}

	validProviderWithDomain := &esv1.SecretServerProvider{
		Username:  makeSecretRefUsingRef(clientSecretWithDomain.Name, userNameKey),
		Password:  makeSecretRefUsingRef(clientSecretWithDomain.Name, passwordKey),
		Domain:    domain,
		ServerURL: "https://example.com",
	}

	// Valid test CA certificate
	testCABundle := []byte(`-----BEGIN CERTIFICATE-----
MIIDHTCCAgWgAwIBAgIRAKC4yxy9QGocND+6avTf7BgwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAeFw0yMTAzMjAyMDA4MDhaFw0yMTAzMjAyMDM4
MDhaMBIxEDAOBgNVBAoTB0FjbWUgQ28wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQC3o6/JdZEqNbqNRkopHhJtJG5c4qS5d0tQ/kZYpfD/v/izAYum4Nzj
aG15owr92/11W0pxPUliRLti3y6iScTs+ofm2D7p4UXj/Fnho/2xoWSOoWAodgvW
Y8jh8A0LQALZiV/9QsrJdXZdS47DYZLsQ3z9yFC/CdXkg1l7AQ3fIVGKdrQBr9kE
1gEDqnKfRxXI8DEQKXr+CKPUwCAytegmy0SHp53zNAvY+kopHytzmJpXLoEhxq4e
ugHe52vXHdh/HJ9VjNp0xOH1waAgAGxHlltCW0PVd5AJ0SXROBS/a3V9sZCbCrJa
YOOonQSEswveSv6PcG9AHvpNPot2Xs6hAgMBAAGjbjBsMA4GA1UdDwEB/wQEAwIC
pDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQW
BBR00805mrpoonp95RmC3B6oLl+cGTAVBgNVHREEDjAMggpnb29ibGUuY29tMA0G
CSqGSIb3DQEBCwUAA4IBAQAipc1b6JrEDayPjpz5GM5krcI8dCWVd8re0a9bGjjN
ioWGlu/eTr5El0ffwCNZ2WLmL9rewfHf/bMvYz3ioFZJ2OTxfazqYXNggQz6cMfa
lbedDCdt5XLVX2TyerGvFram+9Uyvk3l0uM7rZnwAmdirG4Tv94QRaD3q4xTj/c0
mv+AggtK0aRFb9o47z/BypLdk5mhbf3Mmr88C8XBzEnfdYyf4JpTlZrYLBmDCu5d
9RLLsjXxhag8xqMtd1uLUM8XOTGzVWacw8iGY+CTtBKqyA+AE6/bDwZvEwVtsKtC
QJ85ioEpy00NioqcF0WyMZH80uMsPycfpnl5uF7RkW8u
-----END CERTIFICATE-----`)

	caSecretName := "ca-secret"
	caSecretKey := "ca.crt"
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: caSecretName, Namespace: "default"},
		Data: map[string][]byte{
			caSecretKey: testCABundle,
		},
	}

	caConfigMapName := "ca-configmap"
	caConfigMapKey := "ca.crt"
	caConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: caConfigMapName, Namespace: "default"},
		Data: map[string]string{
			caConfigMapKey: string(testCABundle),
		},
	}

	tests := map[string]struct {
		store    esv1.GenericStore          // leave nil for namespaced store
		provider *esv1.SecretServerProvider // discarded when store is set
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
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						SecretServer: validProvider,
					},
				},
			},
			errCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errClusterStoreRequiresNamespace)
			},
		},
		"dangling password ref": {
			provider: &esv1.SecretServerProvider{
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
			provider: &esv1.SecretServerProvider{
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
			provider: &esv1.SecretServerProvider{
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
			provider: &esv1.SecretServerProvider{
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
			provider: &esv1.SecretServerProvider{
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
			provider: &esv1.SecretServerProvider{
				Username:  makeSecretRefUsingValue(userNameValue),
				Password:  makeSecretRefUsingValue(passwordValue),
				ServerURL: validProvider.ServerURL,
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
		},
		"cluster secret store": {
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						SecretServer: &esv1.SecretServerProvider{
							Username:  makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, userNameKey),
							Password:  makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, passwordKey),
							ServerURL: validProvider.ServerURL,
						},
					},
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
		},
		"cluster secret store with domain": {
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						SecretServer: &esv1.SecretServerProvider{
							Username:  makeSecretRefUsingNamespacedRef(clientSecretWithDomain.Namespace, clientSecretWithDomain.Name, userNameKey),
							Password:  makeSecretRefUsingNamespacedRef(clientSecretWithDomain.Namespace, clientSecretWithDomain.Name, passwordKey),
							Domain:    validProviderWithDomain.Domain,
							ServerURL: validProviderWithDomain.ServerURL,
						},
					},
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret, clientSecretWithDomain).Build(),
		},
		"valid with CABundle and CAProvider using Secret": {
			provider: &esv1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
				CABundle:  testCABundle,
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: caSecretName,
					Key:  caSecretKey,
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret, caSecret).Build(),
		},
		"valid with CABundle and CAProvider using ConfigMap": {
			provider: &esv1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
				CABundle:  testCABundle,
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeConfigMap,
					Name: caConfigMapName,
					Key:  caConfigMapKey,
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret, caConfigMap).Build(),
		},
		"CABundle without CAProvider is ignored": {
			provider: &esv1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
				CABundle:  testCABundle,
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
		},
		"CAProvider without CABundle is ignored": {
			provider: &esv1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: caSecretName,
					Key:  caSecretKey,
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret, caSecret).Build(),
		},
		"invalid CABundle format with CAProvider": {
			provider: &esv1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
				CABundle:  []byte("invalid certificate data"),
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: caSecretName,
					Key:  caSecretKey,
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret, caSecret).Build(),
			errCheck: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to decode ca bundle")
			},
		},
		"missing CAProvider Secret with valid CABundle": {
			provider: &esv1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
				CABundle:  testCABundle,
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: "non-existent-secret",
					Key:  caSecretKey,
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret).Build(),
			// CABundle takes precedence, so even if the secret doesn't exist, CABundle is used
		},
		"only CAProvider without CABundle is ignored": {
			provider: &esv1.SecretServerProvider{
				Username:  validProvider.Username,
				Password:  validProvider.Password,
				ServerURL: validProvider.ServerURL,
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: caSecretName,
					Key:  caSecretKey,
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret, caSecret).Build(),
			// No error expected because both CABundle AND CAProvider must be set for TLS config
		},
		"cluster secret store with CABundle and CAProvider": {
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						SecretServer: &esv1.SecretServerProvider{
							Username:  makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, userNameKey),
							Password:  makeSecretRefUsingNamespacedRef(clientSecret.Namespace, clientSecret.Name, passwordKey),
							ServerURL: validProvider.ServerURL,
							CABundle:  testCABundle,
							CAProvider: &esv1.CAProvider{
								Type:      esv1.CAProviderTypeSecret,
								Name:      caSecretName,
								Key:       caSecretKey,
								Namespace: esutils.Ptr("default"),
							},
						},
					},
				},
			},
			kube: clientfake.NewClientBuilder().WithObjects(clientSecret, caSecret).Build(),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := &Provider{}
			store := tc.store
			if store == nil {
				store = &esv1.SecretStore{
					TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
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
				expectedCredentials := server.UserCredential{
					Username: userNameValue,
					Password: passwordValue,
				}
				if name == "cluster secret store with domain" {
					expectedCredentials.Domain = domain
				}
				assert.Equal(t, expectedCredentials, secretServerClient.Configuration.Credentials)
			} else {
				assert.Nil(t, sc)
				tc.errCheck(t, err)
			}
		})
	}
}

func makeSecretRefUsingNamespacedRef(namespace, name, key string) *esv1.SecretServerProviderRef {
	return &esv1.SecretServerProviderRef{
		SecretRef: &v1.SecretKeySelector{Namespace: esutils.Ptr(namespace), Name: name, Key: key},
	}
}

func makeSecretRefUsingValue(val string) *esv1.SecretServerProviderRef {
	return &esv1.SecretServerProviderRef{Value: val}
}

func makeSecretRefUsingRef(name, key string) *esv1.SecretServerProviderRef {
	return &esv1.SecretServerProviderRef{
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
