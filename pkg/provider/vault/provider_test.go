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
	"testing"

	"github.com/google/go-cmp/cmp"
	vault "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/ptr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	utilfake "github.com/external-secrets/external-secrets/pkg/provider/util/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
)

const (
	tokenSecretName  = "example-secret-token"
	secretDataString = "some-creds"
	tlsAuthCerts     = "tls-auth-certs"
	tlsKey           = "tls.key"
	tlsCrt           = "tls.crt"
	vaultCert        = "vault-cert"
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
								Name: tlsAuthCerts,
								Key:  tlsCrt,
							},
							SecretRef: esmeta.SecretKeySelector{
								Name: tlsAuthCerts,
								Key:  tlsKey,
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
		Name: vaultCert,
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
						Name: vaultCert,
						Key:  "cert",
						Type: "Secret",
					},
				},
			},
		},
	}
}

func makeValidSecretStoreWithIamAuthSecret() *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-store",
			Namespace: "default",
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Vault: &esv1beta1.VaultProvider{
					Server:  "https://vault.example.com:8200",
					Path:    &secretStorePath,
					Version: esv1beta1.VaultKVStoreV2,
					Auth: esv1beta1.VaultAuth{
						Iam: &esv1beta1.VaultIamAuth{
							Path:   "aws",
							Region: "us-east-1",
							Role:   "vault-role",
							SecretRef: &esv1beta1.VaultAwsAuthSecretRef{
								AccessKeyID: esmeta.SecretKeySelector{
									Name: "vault-iam-creds-secret",
									Key:  "access-key",
								},
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: "vault-iam-creds-secret",
									Key:  "secret-access-key",
								},
								SessionToken: &esmeta.SecretKeySelector{
									Name: "vault-iam-creds-secret",
									Key:  "secret-session-token",
								},
							},
						},
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

func makeClusterSecretStore(tweaks ...secretStoreTweakFn) *esv1beta1.ClusterSecretStore {
	store := makeValidSecretStore()

	for _, fn := range tweaks {
		fn(store)
	}

	return &esv1beta1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: esv1beta1.ClusterSecretStoreKind,
		},
		ObjectMeta: store.ObjectMeta,
		Spec:       store.Spec,
	}
}

type args struct {
	newClientFunc func(c *vault.Config) (util.Client, error)
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
		"InvalidRetrySettings": {
			reason: "Should return error if given an invalid Retry Interval.",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.RetrySettings = &esv1beta1.SecretStoreRetrySettings{
						MaxRetries:    ptr.To(int32(3)),
						RetryInterval: ptr.To("not-an-interval"),
					}
				}),
			},
			want: want{
				err: errors.New("time: invalid duration \"not-an-interval\""),
			},
		},
		"ValidRetrySettings": {
			reason: "Should return a Vault provider with custom retry settings",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.RetrySettings = &esv1beta1.SecretStoreRetrySettings{
						MaxRetries:    ptr.To(int32(3)),
						RetryInterval: ptr.To("10m"),
					}
				}),
				ns:            "default",
				kube:          clientfake.NewClientBuilder().Build(),
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: nil,
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
				err: fmt.Errorf("failed to decode ca bundle: %w", errors.New("failed to parse the new certificate, not valid pem data")),
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
				newClientFunc: fake.ClientWithLoginMock,
				ns:            "default",
				kube:          clientfake.NewClientBuilder().Build(),
				store:         makeSecretStore(),
				corev1:        utilfake.NewCreateTokenMock().WithError(errBoom),
			},
			want: want{
				err: fmt.Errorf(errGetKubeSATokenRequest, "example-sa", errBoom),
			},
		},
		"GetKubeSecretError": {
			reason: "Should return error if fetching kubernetes secret fails.",
			args: args{
				ns: "default",
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.Auth.Kubernetes.ServiceAccountRef = nil
					s.Spec.Provider.Vault.Auth.Kubernetes.SecretRef = &esmeta.SecretKeySelector{
						Name: "vault-secret",
						Key:  "key",
					}
				}),
				kube: clientfake.NewClientBuilder().Build(),
			},
			want: want{
				err: fmt.Errorf(`cannot get Kubernetes secret "vault-secret": %w`, errors.New(`secrets "vault-secret" not found`)),
			},
		},
		"SuccessfulVaultStoreWithCertAuth": {
			reason: "Should return a Vault provider successfully",
			args: args{
				store: makeValidSecretStoreWithCerts(),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tlsAuthCerts,
						Namespace: "default",
					},
					Data: map[string][]byte{
						tlsKey: secretClientKey,
						tlsCrt: clientCrt,
					},
				}).Build(),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"SuccessfulVaultStoreWithK8sCertSecret": {
			reason: "Should return a Vault provider with the cert from k8s",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(true),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vaultCert,
						Namespace: "default",
					},
					Data: map[string][]byte{
						"cert":  clientCrt,
						"token": secretData,
					},
				}).Build(),
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"GetCertNamespaceMissingError": {
			reason: "Should return an error if namespace is missing and is a ClusterSecretStore",
			args: args{
				store: makeInvalidClusterSecretStoreWithK8sCerts(),
				ns:    "default",
				kube:  clientfake.NewClientBuilder().Build(),
			},
			want: want{
				err: errors.New(errCANamespace),
			},
		},
		"GetCertSecretKeyMissingError": {
			reason: "Should return an error if the secret key is missing",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(true),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vaultCert,
						Namespace: "default",
					},
					Data: map[string][]byte{},
				}).Build(),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf("failed to get cert from secret: %w", fmt.Errorf("failed to resolve secret key ref: %w", errors.New("cannot find secret data for key: \"cert\""))),
			},
		},
		"SuccessfulVaultStoreWithIamAuthSecret": {
			reason: "Should return a Vault provider successfully",
			args: args{
				store: makeValidSecretStoreWithIamAuthSecret(),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vault-iam-creds-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"access-key":           []byte("TESTING"),
						"secret-access-key":    []byte("ABCDEF"),
						"secret-session-token": []byte("c2VjcmV0LXNlc3Npb24tdG9rZW4K"),
					},
				}).Build(),
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"SuccessfulVaultStoreWithK8sCertConfigMap": {
			reason: "Should return a Vault prodvider with the cert from k8s",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(false),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vaultCert,
						Namespace: "default",
					},
					Data: map[string]string{
						"cert": string(clientCrt),
					},
				}).Build(),
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"GetCertConfigMapMissingError": {
			reason: "Should return an error if the config map key is missing",
			args: args{
				store: makeValidSecretStoreWithK8sCerts(false),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-sa",
						Namespace: "default",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: tokenSecretName,
						},
					},
				}, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vaultCert,
						Namespace: "default",
					},
					Data: map[string]string{},
				}).Build(),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf("failed to get cert from configmap: %w", errors.New("failed to get caProvider configMap vault-cert -> cert")),
			},
		},
		"GetCertificateFormatError": {
			reason: "Should return error if client certificate is in wrong format.",
			args: args{
				store: makeValidSecretStoreWithCerts(),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tlsAuthCerts,
						Namespace: "default",
					},
					Data: map[string][]byte{
						tlsKey: secretClientKey,
						tlsCrt: []byte("cert with mistake"),
					},
				}).Build(),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf(errClientTLSAuth, "tls: failed to find any PEM data in certificate input"),
			},
		},
		"GetKeyFormatError": {
			reason: "Should return error if client key is in wrong format.",
			args: args{
				store: makeValidSecretStoreWithCerts(),
				ns:    "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tlsAuthCerts,
						Namespace: "default",
					},
					Data: map[string][]byte{
						tlsKey: []byte("key with mistake"),
						tlsCrt: clientCrt,
					},
				}).Build(),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf(errClientTLSAuth, "tls: failed to find any PEM data in key input"),
			},
		},
		"ClientTlsInvalidCertificatesError": {
			reason: "Should return error if client key is in wrong format.",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.ClientTLS = esv1beta1.VaultClientTLS{
						CertSecretRef: &esmeta.SecretKeySelector{
							Name: tlsAuthCerts,
						},
						KeySecretRef: &esmeta.SecretKeySelector{
							Name: tlsAuthCerts,
						},
					}
				}),
				ns: "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tlsAuthCerts,
						Namespace: "default",
					},
					Data: map[string][]byte{
						tlsKey: []byte("key with mistake"),
						tlsCrt: clientCrt,
					},
				}).Build(),
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: fmt.Errorf(errClientTLSAuth, "tls: failed to find any PEM data in key input"),
			},
		},
		"SuccessfulVaultStoreValidClientTls": {
			reason: "Should return a Vault provider with the cert from k8s",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.ClientTLS = esv1beta1.VaultClientTLS{
						CertSecretRef: &esmeta.SecretKeySelector{
							Name: tlsAuthCerts,
						},
						KeySecretRef: &esmeta.SecretKeySelector{
							Name: tlsAuthCerts,
						},
					}
				}),
				ns: "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tlsAuthCerts,
						Namespace: "default",
					},
					Data: map[string][]byte{
						tlsKey: secretClientKey,
						tlsCrt: clientCrt,
					},
				}).Build(),
				corev1:        utilfake.NewCreateTokenMock().WithToken("ok"),
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{
				err: nil,
			},
		},
		"SuccessfulVaultStoreWithSecretRef": {
			reason: "Should return a Vault provider with secret ref auth",
			args: args{
				store: makeClusterSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.Auth.Kubernetes = nil
					s.Spec.Provider.Vault.Auth.TokenSecretRef = &esmeta.SecretKeySelector{
						Name:      "vault-token",
						Namespace: ptr.To("default"),
						Key:       "token",
					}
				}),
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vault-token",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"token": []byte("token"),
					},
				}).Build(),
				// no need to mock the secret as it is not used
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{},
		},
		"SuccessfulVaultStoreWithApproleRef": {
			reason: "Should return a Vault provider with approle auth",
			args: args{
				store: makeSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.Auth.Kubernetes = nil
					s.Spec.Provider.Vault.Auth.AppRole = &esv1beta1.VaultAppRole{
						SecretRef: esmeta.SecretKeySelector{
							Name: "vault-secret-id",
							Key:  "secret-id",
						},
						RoleRef: &esmeta.SecretKeySelector{
							Name: "vault-secret-id",
							Key:  "approle",
						},
					}
				}),
				ns: "default",
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vault-secret-id",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"secret-id": []byte("myid"),
						"approle":   []byte("myrole"),
					},
				}).Build(),
				// no need to mock the secret as it is not used
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{},
		},
		"SuccessfulVaultStoreWithSecretRefAndReferentSpec": {
			reason: "Should return a Vault provider with secret ref auth",
			args: args{
				store: makeClusterSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.Auth.TokenSecretRef = &esmeta.SecretKeySelector{
						Name: "vault-token",
						Key:  "token",
					}
				}),
				// no need to mock the secret as it is not used
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{},
		},
		"SuccessfulVaultStoreWithJwtAuthAndReferentSpec": {
			reason: "Should return a Vault provider with jwt auth",
			args: args{
				store: makeClusterSecretStore(func(s *esv1beta1.SecretStore) {
					s.Spec.Provider.Vault.Auth.Kubernetes = nil
					s.Spec.Provider.Vault.Auth.Jwt = &esv1beta1.VaultJwtAuth{
						Role: "test-role",
						SecretRef: &esmeta.SecretKeySelector{
							Name: "vault-token",
						},
					}
				}),
				// no need to mock the secret as it is not used
				newClientFunc: fake.ClientWithLoginMock,
			},
			want: want{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vaultTest(t, name, tc)
		})
	}
}

func vaultTest(t *testing.T, _ string, tc testCase) {
	prov := &Provider{
		NewVaultClient: tc.args.newClientFunc,
	}
	if tc.args.newClientFunc == nil {
		prov.NewVaultClient = NewVaultClient
	}
	_, err := prov.newClient(context.Background(), tc.args.store, tc.args.kube, tc.args.corev1, tc.args.ns)
	if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
		t.Errorf("\n%s\nvault.New(...): -want error, +got error:\n%s", tc.reason, diff)
	}
}
