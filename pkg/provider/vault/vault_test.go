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

package vault

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	vault "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/pointer"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	utilfake "github.com/external-secrets/external-secrets/pkg/provider/util/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/fake"
)

const (
	tokenSecretName  = "example-secret-token"
	secretDataString = "some-creds"
)

var (
	secretStorePath = "secret"
)

func makeValidSecretStoreWithVersion(v esv1beta1.VaultKVStoreVersion) *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-store",
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Vault: &esv1beta1.VaultProvider{
					Server:  "vault.example.com",
					Path:    &secretStorePath,
					Version: v,
					Auth: esv1beta1.VaultAuth{
						Kubernetes: &esv1beta1.VaultKubernetesAuth{
							Path: "kubernetes",
							Role: "kubernetes-auth-role",
							ServiceAccountRef: &esmeta.ServiceAccountSelector{
								Name: "example-sa",
							},
						},
					},
				},
			},
		},
	}
}

func makeValidSecretStore() *esv1beta1.SecretStore {
	return makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2)
}

func makeValidSecretStoreWithCerts() *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-store",
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Vault: &esv1beta1.VaultProvider{
					Server:  "vault.example.com",
					Path:    &secretStorePath,
					Version: esv1beta1.VaultKVStoreV2,
					Auth: esv1beta1.VaultAuth{
						Cert: &esv1beta1.VaultCertAuth{
							ClientCert: esmeta.SecretKeySelector{
								Name: "tls-auth-certs",
								Key:  "tls.crt",
							},
							SecretRef: esmeta.SecretKeySelector{
								Name: "tls-auth-certs",
								Key:  "tls.key",
							},
						},
					},
				},
			},
		},
	}
}

func makeValidSecretStoreWithK8sCerts(isSecret bool) *esv1beta1.SecretStore {
	store := makeSecretStore()
	caProvider := &esv1beta1.CAProvider{
		Name: "vault-cert",
		Key:  "cert",
	}

	if isSecret {
		caProvider.Type = "Secret"
	} else {
		caProvider.Type = "ConfigMap"
	}

	store.Spec.Provider.Vault.CAProvider = caProvider
	return store
}

func makeInvalidClusterSecretStoreWithK8sCerts() *esv1beta1.ClusterSecretStore {
	return &esv1beta1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-store",
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Vault: &esv1beta1.VaultProvider{
					Server:  "vault.example.com",
					Path:    &secretStorePath,
					Version: "v2",
					Auth: esv1beta1.VaultAuth{
						Kubernetes: &esv1beta1.VaultKubernetesAuth{
							Path: "kubernetes",
							Role: "kubernetes-auth-role",
							ServiceAccountRef: &esmeta.ServiceAccountSelector{
								Name: "example-sa",
							},
						},
					},
					CAProvider: &esv1beta1.CAProvider{
						Name: "vault-cert",
						Key:  "cert",
						Type: "Secret",
					},
				},
			},
		},
	}
}

type secretStoreTweakFn func(s *esv1beta1.SecretStore)

func makeSecretStore(tweaks ...secretStoreTweakFn) *esv1beta1.SecretStore {
	store := makeValidSecretStore()

	for _, fn := range tweaks {
		fn(store)
	}

	return store
}

type args struct {
	newClientFunc func(c *vault.Config) (Client, error)
	store         esv1beta1.GenericStore
	kube          kclient.Client
	corev1        typedcorev1.CoreV1Interface
	ns            string
}

type want struct {
	err error
}

type testCase struct {
	reason string
	args   args
	want   want
}

func clientWithLoginMock(c *vault.Config) (Client, error) {
	cl := fake.VaultClient{
		MockSetToken: fake.NewSetTokenFn(),
		MockAuth:     fake.NewVaultAuth(),
		MockLogical:  fake.NewVaultLogical(),
	}
	auth := cl.Auth()
	token := cl.AuthToken()
	logical := cl.Logical()
	out := VClient{
		setToken:     cl.SetToken,
		token:        cl.Token,
		clearToken:   cl.ClearToken,
		auth:         auth,
		authToken:    token,
		logical:      logical,
		setNamespace: cl.SetNamespace,
		addHeader:    cl.AddHeader,
	}
	return out, nil
}

func kubeMockWithSecretTokenAndServiceAcc(obj kclient.Object) error {
	if o, ok := obj.(*corev1.ServiceAccount); ok {
		o.Secrets = []corev1.ObjectReference{
			{
				Name: tokenSecretName,
			},
		}
		return nil
	}
	if o, ok := obj.(*corev1.Secret); ok {
		o.Data = map[string][]byte{
			"token": []byte(secretDataString),
		}
		return nil
	}
	return nil
}

func TestNewVault(t *testing.T) {
	errBoom := errors.New("boom")
	secretClientKey := []byte(`-----BEGIN PRIVATE KEY-----
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
-----END PRIVATE KEY-----`)
	clientCrt := []byte(`-----BEGIN CERTIFICATE-----
MIIFkTCCA3mgAwIBAgIUBEUg3m/WqAsWHG4Q/II3IePFfuowDQYJKoZIhvcNAQELBQAwWDELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDERMA8GA1UEAwwIdmF1bHQtY2EwHhcNMjIwNzI5MjEyMjE4WhcNMzkwMTAxMjEyMjE4WjBYMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMREwDwYDVQQDDAh2YXVsdC1jYTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAKLhwbYLEd6M5dpZbReuLg9tbK5qKe6sJm5Qs8sLSMmwkXTOSrvvw7smV5bOHtCGpUEooiFbos1RVL22YYd8cLu59oev1i1p3NWSIPNAq0DQKBxyj3wJ9ftvOfmlmf/J+dtE39pPLFPnffkMWEdzT8UImfZD/kf1S1fotHcffNtP0ijfZPQHo4BQooH/e7PEAQIXDBIrxSoZR57wkoYGtTXRdEAwbE5zUE5ZikIdZlEfYVEYiBL5vBgc5dSM+/oUfurH7AIzOXGp30sa+JzYKkUTQGISw4siJj+oSqjO93Zo4tWEE+NP/0tNe9FnBf+oDuHhgiyerk3yU3jK2vJvXc3191FWDKZXKQaRoCaHnMDgMiDjxQtbM590/BFDWRWrZNGaPciVwmw2foyAclNtGq8J0kzIDn0VRkL35Ad8LfTuxXZpY+hVPC5TIPek4W7grQtX44ohVDw5M6oClucv104tXHjAVVgTfaVMLi1F3FRTwwqI8ShPX0D5ssw+tkeepcIhHMvDB97AgKBM1g24MdX3c6aokrghdwXxQbTS/6lHD66apc3E2STTHeCt9fnDSwEZlvK6iNgedBDHrBl9SXeYTMnbOWYIHX/YYq1efq0gOZHHN4nZuBopOMb+4K/04a2nVUuafdv/eoL6F7h+/+UaH/N3+LkkxWAxcz0w9GVZAgMBAAGjUzBRMB0GA1UdDgQWBBQuIVwmjMZvkq+jf6ViTelH5KDBVDAfBgNVHSMEGDAWgBQuIVwmjMZvkq+jf6ViTelH5KDBVDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4ICAQAk4kNyFzmiKnREmi5PPj7xGAtv2aJIdMEfcZJ9e+H0Nb2aCvMvZsDodduXu6G5+1opd45v0AeTjLBkXDO6/8vnyM32VZEEKCAwMCLcOLD1z0+r+gaurDYMOGU5qr8hQadHKFsxEDYnR/9KdHhBg6A8qE2cOQa1ryu34DnWQ3m0CBApClf1YBRp/4T8BmHumfH6odD96H30HVzINrd9WM2hR9GRE3xqQyfwlvqmGn9S6snSVa+mcJ6w2wNE2LPGx0kOtBeOIUdfSsEgvSRjbowSHz9lohFZ0LxJYyizCA5vnMmYyhhkfJqm7YtjHkGWgXmqpH9BFt0D3gfORlIh787nuWfxtZ+554rDyQmPjYQG/qF4+Awehr4RxiGWTox1C67G/RzA6TOXX09xuFY+3U1ich90/KffvhoHvRVfhzxx+HUUY2qSU3HqQDzgieQQBaMuOhd1i6pua+/kPSXkuXqnIs8daao/goR5iU/lPLs7M8Dy7xZ9adzbIPuNuzHir2UuvtPlW+x/sSvOnVL9r/7TrAuWhdScglQ70EInPDVX7BgDWKrZUh86N4d7fu2f/T+6VoUSGEjq8obCj3BQ61mNEoftKVECUO4MMUdat6pY/4Xh6Dwc+FnbvR2+sX7IzI7FtgOrfO6abT+LCAR0R+UXyvnqZcjK2zkHz4DfXFbCQg==
-----END CERTIFICATE-----`)
	secretData := []byte(secretDataString)

	cases := map[string]testCase{
		"InvalidVaultStore": {
			reason: "Should return error if given an invalid vault store.",
			args: args{
				store: &esv1beta1.SecretStore{},
			},
			want: want{
				err: errors.New(errVaultStore),
			},
		},
		"AddVaultStoreCertsError": {
			reason: "Should return error if given an invalid CA certificate.",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.CABundle = []byte("badcertdata")
				}),
			},
			want: want{
				err: errors.New(errVaultCert),
			},
		},
		"VaultAuthFormatError": {
			reason: "Should return error if no valid authentication method is given.",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{}
				}),
			},
			want: want{
				err: errors.New(errAuthFormat),
			},
		},
		"GetKubeServiceAccountError": {
			reason: "Should return error if fetching kubernetes secret fails.",
			args: args{
				newClientFunc: clientWithLoginMock,
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
				store:  makeSecretStore(),
				corev1: utilfake.NewCreateTokenMock().WithError(errBoom),
			},
			want: want{
				err: fmt.Errorf(errGetKubeSATokenRequest, "example-sa", errBoom),
			},
		},
		"GetKubeSecretError": {
			reason: "Should return error if fetching kubernetes secret fails.",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.Auth.Kubernetes.ServiceAccountRef = nil
					s.Spec.Provider.Vault.Auth.Kubernetes.SecretRef = &esmeta.SecretKeySelector{
						Name: "vault-secret",
						Key:  "key",
					}
				}),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errGetKubeSecret, "vault-secret", errBoom),
			},
		},
		"SuccessfulVaultStoreWithCertAuth": {
			reason: "Should return a Vault provider successfully",
			args: args{
				store: makeValidSecretStoreWithCerts(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.Secret); ok {
							o.Data = map[string][]byte{
								"tls.key": secretClientKey,
								"tls.crt": clientCrt,
							}
							return nil
						}
						return nil
					}),
				},
				newClientFunc: clientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"SuccessfulVaultStoreWithK8sCertSecret": {
			reason: "Should return a Vault prodvider with the cert from k8s",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(true),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.Secret); ok {
							o.Data = map[string][]byte{
								"cert":  clientCrt,
								"token": secretData,
							}
							return nil
						}
						return nil
					}),
				},
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: clientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"GetCertNamespaceMissingError": {
			reason: "Should return an error if namespace is missing and is a ClusterSecretStore",
			args: args{
				store: makeInvalidClusterSecretStoreWithK8sCerts(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, kubeMockWithSecretTokenAndServiceAcc),
				},
			},
			want: want{
				err: errors.New(errCANamespace),
			},
		},
		"GetCertSecretKeyMissingError": {
			reason: "Should return an error if the secret key is missing",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(true),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, kubeMockWithSecretTokenAndServiceAcc),
				},
				newClientFunc: clientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf(errVaultCert, errors.New(`cannot find secret data for key: "cert"`)),
			},
		},
		"SuccessfulVaultStoreWithK8sCertConfigMap": {
			reason: "Should return a Vault prodvider with the cert from k8s",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(false),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.ConfigMap); ok {
							o.Data = map[string]string{
								"cert": string(clientCrt),
							}
							return nil
						}

						return nil
					}),
				},
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: clientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"GetCertConfigMapMissingError": {
			reason: "Should return an error if the config map key is missing",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(false),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.ServiceAccount); ok {
							o.Secrets = []corev1.ObjectReference{
								{
									Name: tokenSecretName,
								},
							}
							return nil
						}

						if o, ok := obj.(*corev1.Secret); ok {
							o.Data = map[string][]byte{
								"token": secretData,
							}
							return nil
						}

						return nil
					}),
				},
				newClientFunc: clientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf(errConfigMapFmt, "cert"),
			},
		},
		"GetCertificateFormatError": {
			reason: "Should return error if client certificate is in wrong format.",
			args: args{
				store: makeValidSecretStoreWithCerts(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.Secret); ok {
							o.Data = map[string][]byte{
								"tls.key": secretClientKey,
								"tls.crt": []byte("cert with mistak"),
							}
							return nil
						}
						return nil
					}),
				},
				newClientFunc: clientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf(errClientTLSAuth, "tls: failed to find any PEM data in certificate input"),
			},
		},
		"GetKeyFormatError": {
			reason: "Should return error if client key is in wrong format.",
			args: args{
				store: makeValidSecretStoreWithCerts(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.Secret); ok {
							o.Data = map[string][]byte{
								"tls.key": []byte("key with mistake"),
								"tls.crt": clientCrt,
							}
							return nil
						}
						return nil
					}),
				},
				newClientFunc: clientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf(errClientTLSAuth, "tls: failed to find any PEM data in key input"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vaultTest(t, name, tc)
		})
	}
}

func vaultTest(t *testing.T, name string, tc testCase) {
	conn := &connector{
		newVaultClient: tc.args.newClientFunc,
	}
	if tc.args.newClientFunc == nil {
		conn.newVaultClient = newVaultClient
	}
	_, err := conn.newClient(context.Background(), tc.args.store, tc.args.kube, tc.args.corev1, tc.args.ns)
	if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
		t.Errorf("\n%s\nvault.New(...): -want error, +got error:\n%s", tc.reason, diff)
	}
}

func TestGetSecret(t *testing.T) {
	errBoom := errors.New("boom")
	secret := map[string]interface{}{
		"access_key":    "access_key",
		"access_secret": "access_secret",
	}
	secretWithNilVal := map[string]interface{}{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"token":         nil,
	}
	secretWithNestedVal := map[string]interface{}{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"nested.bar":    "something different",
		"nested": map[string]string{
			"foo": "oke",
			"bar": "also ok?",
		},
	}

	type args struct {
		store    *esv1beta1.VaultProvider
		kube     kclient.Client
		vLogical Logical
		ns       string
		data     esv1beta1.ExternalSecretDataRemoteRef
	}

	type want struct {
		err error
		val []byte
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ReadSecret": {
			reason: "Should return the secret with property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "access_key",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("access_key"),
			},
		},
		"ReadSecretWithNil": {
			reason: "Should return the secret with property if it has a nil val",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "access_key",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNilVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("access_key"),
			},
		},
		"ReadSecretWithoutProperty": {
			reason: "Should return the json encoded secret without property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data:  esv1beta1.ExternalSecretDataRemoteRef{},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte(`{"access_key":"access_key","access_secret":"access_secret"}`),
			},
		},
		"ReadSecretWithNestedValue": {
			reason: "Should return a nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nested.foo",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("oke"),
			},
		},
		"ReadSecretWithNestedValueFromData": {
			reason: "Should return a nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					//
					Property: "nested.bar",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: nil,
				val: []byte("something different"),
			},
		},
		"NonexistentProperty": {
			reason: "Should return error property does not exist.",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nop.doesnt.exist",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNestedVal, nil),
				},
			},
			want: want{
				err: fmt.Errorf(errSecretKeyFmt, "nop.doesnt.exist"),
			},
		},
		"ReadSecretError": {
			reason: "Should return error if vault client fails to read secret.",
			args: args{
				store: makeSecretStore().Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errReadSecret, errBoom),
			},
		},
		"ReadSecretNotFound": {
			reason: "Secret doesn't exist",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "access_key",
				},
				vLogical: &fake.Logical{
					ReadWithDataWithContextFn: func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
						return nil, nil
					},
				},
			},
			want: want{
				err: errors.New(errNotFound),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				kube:      tc.args.kube,
				logical:   tc.args.vLogical,
				store:     tc.args.store,
				namespace: tc.args.ns,
			}
			val, err := vStore.GetSecret(context.Background(), tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecret(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(string(tc.want.val), string(val)); diff != "" {
				t.Errorf("\n%s\nvault.GetSecret(...): -want val, +got val:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	errBoom := errors.New("boom")
	secret := map[string]interface{}{
		"access_key":    "access_key",
		"access_secret": "access_secret",
	}
	secretWithNilVal := map[string]interface{}{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"token":         nil,
	}
	secretWithNestedVal := map[string]interface{}{
		"access_key":    "access_key",
		"access_secret": "access_secret",
		"nested": map[string]interface{}{
			"foo": map[string]string{
				"oke":    "yup",
				"mhkeih": "yada yada",
			},
		},
	}
	secretWithTypes := map[string]interface{}{
		"access_secret": "access_secret",
		"f32":           float32(2.12),
		"f64":           float64(2.1234534153423423),
		"int":           42,
		"bool":          true,
		"bt":            []byte("foobar"),
	}

	type args struct {
		store   *esv1beta1.VaultProvider
		kube    kclient.Client
		vClient Logical
		ns      string
		data    esv1beta1.ExternalSecretDataRemoteRef
	}

	type want struct {
		err error
		val map[string][]byte
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ReadSecretKV1": {
			reason: "Should map the secret even if it has a nil value",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secret, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
				},
			},
		},
		"ReadSecretKV2": {
			reason: "Should map the secret even if it has a nil value",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]interface{}{
						"data": secret,
					}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
				},
			},
		},
		"ReadSecretWithNilValueKV1": {
			reason: "Should map the secret even if it has a nil value",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(secretWithNilVal, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
					"token":         []byte(nil),
				},
			},
		},
		"ReadSecretWithNilValueKV2": {
			reason: "Should map the secret even if it has a nil value",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]interface{}{
						"data": secretWithNilVal}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_key":    []byte("access_key"),
					"access_secret": []byte("access_secret"),
					"token":         []byte(nil),
				},
			},
		},
		"ReadSecretWithTypesKV2": {
			reason: "Should map the secret even if it has other types",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]interface{}{
						"data": secretWithTypes}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"access_secret": []byte("access_secret"),
					"f32":           []byte("2.12"),
					"f64":           []byte("2.1234534153423423"),
					"int":           []byte("42"),
					"bool":          []byte("true"),
					"bt":            []byte("Zm9vYmFy"), // base64
				},
			},
		},
		"ReadNestedSecret": {
			reason: "Should map the secret for deeply nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nested",
				},
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]interface{}{
						"data": secretWithNestedVal}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"foo": []byte(`{"mhkeih":"yada yada","oke":"yup"}`),
				},
			},
		},
		"ReadDeeplyNestedSecret": {
			reason: "Should map the secret for deeply nested property",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				data: esv1beta1.ExternalSecretDataRemoteRef{
					Property: "nested.foo",
				},
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(map[string]interface{}{
						"data": secretWithNestedVal}, nil),
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"oke":    []byte("yup"),
					"mhkeih": []byte("yada yada"),
				},
			},
		},
		"ReadSecretError": {
			reason: "Should return error if vault client fails to read secret.",
			args: args{
				store: makeSecretStore().Spec.Provider.Vault,
				vClient: &fake.Logical{
					ReadWithDataWithContextFn: fake.NewReadWithContextFn(nil, errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errReadSecret, errBoom),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				kube:      tc.args.kube,
				logical:   tc.args.vClient,
				store:     tc.args.store,
				namespace: tc.args.ns,
			}
			val, err := vStore.GetSecretMap(context.Background(), tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.val, val); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want val, +got val:\n%s", tc.reason, diff)
			}
		})
	}
}

func newListWithContextFn(secrets map[string]interface{}) func(ctx context.Context, path string) (*vault.Secret, error) {
	return func(ctx context.Context, path string) (*vault.Secret, error) {
		path = strings.TrimPrefix(path, "secret/metadata/")
		if path == "" {
			path = "default"
		}
		data, ok := secrets[path]
		if !ok {
			return nil, errors.New("Secret not found")
		}
		meta := data.(map[string]interface{})
		ans := meta["metadata"].(map[string]interface{})
		secret := &vault.Secret{
			Data: map[string]interface{}{
				"keys": ans["keys"],
			},
		}
		return secret, nil
	}
}

func newReadtWithContextFn(secrets map[string]interface{}) func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
	return func(ctx context.Context, path string, d map[string][]string) (*vault.Secret, error) {
		path = strings.TrimPrefix(path, "secret/data/")
		path = strings.TrimPrefix(path, "secret/metadata/")
		if path == "" {
			path = "default"
		}
		data, ok := secrets[path]
		if !ok {
			return nil, errors.New("Secret not found")
		}
		meta := data.(map[string]interface{})
		metadata := meta["metadata"].(map[string]interface{})
		content := map[string]interface{}{
			"data":            meta["data"],
			"custom_metadata": metadata["custom_metadata"],
		}
		secret := &vault.Secret{
			Data: content,
		}
		return secret, nil
	}
}
func TestGetAllSecrets(t *testing.T) {
	secret1Bytes := []byte("{\"access_key\":\"access_key\",\"access_secret\":\"access_secret\"}")
	secret2Bytes := []byte("{\"access_key\":\"access_key2\",\"access_secret\":\"access_secret2\"}")
	path1Bytes := []byte("{\"access_key\":\"path1\",\"access_secret\":\"path1\"}")
	path2Bytes := []byte("{\"access_key\":\"path2\",\"access_secret\":\"path2\"}")
	tagBytes := []byte("{\"access_key\":\"unfetched\",\"access_secret\":\"unfetched\"}")
	path := "path"
	secret := map[string]interface{}{
		"secret1": map[string]interface{}{
			"metadata": map[string]interface{}{
				"custom_metadata": map[string]interface{}{
					"foo": "bar",
				},
			},
			"data": map[string]interface{}{
				"access_key":    "access_key",
				"access_secret": "access_secret",
			},
		},
		"secret2": map[string]interface{}{
			"metadata": map[string]interface{}{
				"custom_metadata": map[string]interface{}{
					"foo": "baz",
				},
			},
			"data": map[string]interface{}{
				"access_key":    "access_key2",
				"access_secret": "access_secret2",
			},
		},
		"secret3": map[string]interface{}{
			"metadata": map[string]interface{}{
				"custom_metadata": map[string]interface{}{
					"foo": "baz",
				},
			},
			"data": nil,
		},
		"tag": map[string]interface{}{
			"metadata": map[string]interface{}{
				"custom_metadata": map[string]interface{}{
					"foo": "baz",
				},
			},
			"data": map[string]interface{}{
				"access_key":    "unfetched",
				"access_secret": "unfetched",
			},
		},
		"path/1": map[string]interface{}{
			"metadata": map[string]interface{}{
				"custom_metadata": map[string]interface{}{
					"foo": "path",
				},
			},
			"data": map[string]interface{}{
				"access_key":    "path1",
				"access_secret": "path1",
			},
		},
		"path/2": map[string]interface{}{
			"metadata": map[string]interface{}{
				"custom_metadata": map[string]interface{}{
					"foo": "path",
				},
			},
			"data": map[string]interface{}{
				"access_key":    "path2",
				"access_secret": "path2",
			},
		},
		"default": map[string]interface{}{
			"data": map[string]interface{}{
				"empty": "true",
			},
			"metadata": map[string]interface{}{
				"keys": []interface{}{"secret1", "secret2", "secret3", "tag", "path/"},
			},
		},
		"path/": map[string]interface{}{
			"data": map[string]interface{}{
				"empty": "true",
			},
			"metadata": map[string]interface{}{
				"keys": []interface{}{"1", "2"},
			},
		},
	}
	type args struct {
		store    *esv1beta1.VaultProvider
		kube     kclient.Client
		vLogical Logical
		ns       string
		data     esv1beta1.ExternalSecretFind
	}

	type want struct {
		err error
		val map[string][]byte
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"FindByName": {
			reason: "should map multiple secrets matching name",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Name: &esv1beta1.FindName{
						RegExp: "secret.*",
					},
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"secret1": secret1Bytes,
					"secret2": secret2Bytes,
				},
			},
		},
		"FindByTag": {
			reason: "should map multiple secrets matching tags",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"foo": "baz",
					},
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"tag":     tagBytes,
					"secret2": secret2Bytes,
				},
			},
		},
		"FilterByPath": {
			reason: "should filter secrets based on path",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Path: &path,
					Tags: map[string]string{
						"foo": "path",
					},
				},
			},
			want: want{
				err: nil,
				val: map[string][]byte{
					"path/1": path1Bytes,
					"path/2": path2Bytes,
				},
			},
		},
		"FailIfKv1": {
			reason: "should not work if using kv1 store",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn:         newListWithContextFn(secret),
					ReadWithDataWithContextFn: newReadtWithContextFn(secret),
				},
				data: esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"foo": "baz",
					},
				},
			},
			want: want{
				err: errors.New(errUnsupportedKvVersion),
			},
		},
		"MetadataNotFound": {
			reason: "metadata secret not found",
			args: args{
				store: makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV2).Spec.Provider.Vault,
				vLogical: &fake.Logical{
					ListWithContextFn: newListWithContextFn(secret),
					ReadWithDataWithContextFn: func(ctx context.Context, path string, d map[string][]string) (*vault.Secret, error) {
						return nil, nil
					},
				},
				data: esv1beta1.ExternalSecretFind{
					Tags: map[string]string{
						"foo": "baz",
					},
				},
			},
			want: want{
				err: errors.New(errNotFound),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				kube:      tc.args.kube,
				logical:   tc.args.vLogical,
				store:     tc.args.store,
				namespace: tc.args.ns,
			}
			val, err := vStore.GetAllSecrets(context.Background(), tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.val, val); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want val, +got val:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGetSecretPath(t *testing.T) {
	storeV2 := makeValidSecretStore()
	storeV2NoPath := storeV2.DeepCopy()
	storeV2NoPath.Spec.Provider.Vault.Path = nil

	storeV1 := makeValidSecretStoreWithVersion(esv1beta1.VaultKVStoreV1)
	storeV1NoPath := storeV1.DeepCopy()
	storeV1NoPath.Spec.Provider.Vault.Path = nil

	type args struct {
		store    *esv1beta1.VaultProvider
		path     string
		expected string
	}
	cases := map[string]struct {
		reason string
		args   args
	}{
		"PathWithoutFormatV2": {
			reason: "Data needs to be found in path",
			args: args{
				store:    storeV2.Spec.Provider.Vault,
				path:     "secret/test",
				expected: "secret/data/test",
			},
		},
		"PathWithDataV2": {
			reason: "Data needs to be found only once in path",
			args: args{
				store:    storeV2.Spec.Provider.Vault,
				path:     "secret/data/test",
				expected: "secret/data/test",
			},
		},
		"PathWithoutFormatV2_NoPath": {
			reason: "Data needs to be found in path and correct mountpoint is set",
			args: args{
				store:    storeV2NoPath.Spec.Provider.Vault,
				path:     "secret/test",
				expected: "secret/data/test",
			},
		},
		"PathWithoutFormatV1": {
			reason: "Data needs to be found in path",
			args: args{
				store:    storeV1.Spec.Provider.Vault,
				path:     "secret/test",
				expected: "secret/test",
			},
		},
		"PathWithoutFormatV1_NoPath": {
			reason: "Data needs to be found in path and correct mountpoint is set",
			args: args{
				store:    storeV1NoPath.Spec.Provider.Vault,
				path:     "secret/test",
				expected: "secret/test",
			},
		},
		"WithoutPathButMountpointV2": {
			reason: "Mountpoint needs to be set in addition to data",
			args: args{
				store:    storeV2.Spec.Provider.Vault,
				path:     "test",
				expected: "secret/data/test",
			},
		},
		"WithoutPathButMountpointV1": {
			reason: "Mountpoint needs to be set in addition to data",
			args: args{
				store:    storeV1.Spec.Provider.Vault,
				path:     "test",
				expected: "secret/test",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vStore := &client{
				store: tc.args.store,
			}
			want := vStore.buildPath(tc.args.path)
			if diff := cmp.Diff(want, tc.args.expected); diff != "" {
				t.Errorf("\n%s\nvault.buildPath(...): -want expected, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestValidateStore(t *testing.T) {
	type args struct {
		auth esv1beta1.VaultAuth
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty auth",
			args: args{},
		},

		{
			name: "invalid approle with namespace",
			args: args{
				auth: esv1beta1.VaultAuth{
					AppRole: &esv1beta1.VaultAppRole{
						SecretRef: esmeta.SecretKeySelector{
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid clientcert",
			args: args{
				auth: esv1beta1.VaultAuth{
					Cert: &esv1beta1.VaultCertAuth{
						ClientCert: esmeta.SecretKeySelector{
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid cert secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Cert: &esv1beta1.VaultCertAuth{
						SecretRef: esmeta.SecretKeySelector{
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid jwt secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Jwt: &esv1beta1.VaultJwtAuth{
						SecretRef: &esmeta.SecretKeySelector{
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kubernetes sa",
			args: args{
				auth: esv1beta1.VaultAuth{
					Kubernetes: &esv1beta1.VaultKubernetesAuth{
						ServiceAccountRef: &esmeta.ServiceAccountSelector{
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid kubernetes secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Kubernetes: &esv1beta1.VaultKubernetesAuth{
						SecretRef: &esmeta.SecretKeySelector{
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ldap secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					Ldap: &esv1beta1.VaultLdapAuth{
						SecretRef: esmeta.SecretKeySelector{
							Namespace: pointer.StringPtr("invalid"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid token secret",
			args: args{
				auth: esv1beta1.VaultAuth{
					TokenSecretRef: &esmeta.SecretKeySelector{
						Namespace: pointer.StringPtr("invalid"),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &connector{
				newVaultClient: nil,
			}
			store := &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Vault: &esv1beta1.VaultProvider{
							Auth: tt.args.auth,
						},
					},
				},
			}
			if err := c.ValidateStore(store); (err != nil) != tt.wantErr {
				t.Errorf("connector.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
