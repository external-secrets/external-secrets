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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	vault "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/fake"
)

func makeValidSecretStore() *esv1alpha1.SecretStore {
	return &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-store",
			Namespace: "default",
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				Vault: &esv1alpha1.VaultProvider{
					Server:  "vault.example.com",
					Path:    "secret",
					Version: esv1alpha1.VaultKVStoreV2,
					Auth: esv1alpha1.VaultAuth{
						Kubernetes: &esv1alpha1.VaultKubernetesAuth{
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

func makeValidSecretStoreWithCerts() *esv1alpha1.SecretStore {
	return &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vault-store",
			Namespace: "default",
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				Vault: &esv1alpha1.VaultProvider{
					Server:  "vault.example.com",
					Path:    "secret",
					Version: esv1alpha1.VaultKVStoreV2,
					Auth: esv1alpha1.VaultAuth{
						Cert: &esv1alpha1.VaultCertAuth{
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

type secretStoreTweakFn func(s *esv1alpha1.SecretStore)

func makeSecretStore(tweaks ...secretStoreTweakFn) *esv1alpha1.SecretStore {
	store := makeValidSecretStore()

	for _, fn := range tweaks {
		fn(store)
	}

	return store
}

func newVaultResponse(data *vault.Secret) *vault.Response {
	jsonData, _ := json.Marshal(data)
	return &vault.Response{
		Response: &http.Response{
			Body: ioutil.NopCloser(bytes.NewReader(jsonData)),
		},
	}
}

func newVaultTokenIDResponse(token string) *vault.Response {
	return newVaultResponse(&vault.Secret{
		Data: map[string]interface{}{
			"id": token,
		},
	})
}

type args struct {
	newClientFunc func(c *vault.Config) (Client, error)
	store         esv1alpha1.GenericStore
	kube          kclient.Client
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
	secretData := []byte("some-creds")
	secretClientKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEArfZ4HV1obFVlVNiA24tX/UOakqRnEtWXpIvaOsMaPGvvODgGe4XnyJGO32idPv85sIr7vDH9p+OhactVlJV1fu5SZoZ7pg4jTCLqVDCb3IRD++yik2Sw58YayNe3HiaCTsJQWeMXLzfaqOeyk6bEpBCJo09+3QxUWxijgJ7YZCb+Gi8pf3ZWeSZG+rGNNvXHmTs1Yu1H849SYXu+uJOd/R3ZSTw8CxFe4eTLgbCnPf6tgA8Sg2hc+CAZxunPP2JLZWbiJXxjNRoypso6MAJ1FRkx5sTJiLg6UoLvd95/S/lCVOR2PDlM1hg7ox8VEd4QHky7tLx7gji/5hHQKJQSTwIDAQABAoIBAQCYPICQ8hVX+MNcpLrfZenycR7sBYNOMC0silbH5cUn6yzFfgHuRxi3pOnrCJnTb3cE0BvMbdMVAVdYReD2znSsR9NEdZvvjZ/GGSgH1SIQsI7t//+mDQ/jRLJb4KsXb4vJcLLwdpLrd22bMmhMXjzndrF8gSz8NLX9omozPM8RlLxjzPzYOdlX/Zw8V68qQH2Ic04KbtnCwyAUIgAJxYtn/uYB8lzILBkyzQqwhQKkDDZQ0wbZT0hP6z+HgsdifwQvHG1GZAgCuzzyXrL/4TgDaDhYdMVoBA4+HPmzqm5MkBvjH4oqroxjRofUroVix0OGXZJMI1OJ0z/ubzmwCq5BAoGBANqbwzAydUJs0P+GFL94K/Y6tXULKA2c9N0crbxoxheobRpuJvhpW1ZE/9UGpaYX1Rw3nW4x+Jwvt83YkgHAlR4LgEwDvdJPZobybfqifQDiraUO0t62Crn8mSxOsFCugtRIFniwnX67w3uKxiSdCZYbJGs9JEDTpxRG/PSWq3QlAoGBAMu3zOv1PJAhOky7VcxFxWQPEMY+t2PA/sneD01/qgGuhlTwL4QlpywmBqXcI070dcvcBkP0flnWI7y5cnuE1+55twmsrvfaS8s1+AYje0b35DsaF2vtKuJrXC0AGKP+/eiycd9cbvVW2GWOxE7Ui76Mj95MARK8ZNjt0wJagQhjAoGASm9dD80uhhadN1RFPkjB1054OMk6sx/tdFhug8e9I5MSyzwUguME2aQW5EcmIh7dToVVUo8rUqsgz7NdS8FyRM+vuLJRcQneJDbp4bxwCdwlOh2JCZI8psVutlp4yJATNgrxs9iXV+7BChDflNnvyK+nP+iKrpQiwNHHEdU3vg0CgYEAvEpwD4+loJn1psJn9NxwK6F5IaMKIhtZ4/9pKXpcCh3jb1JouL2MnFOxRVAJGor87aW57Mlol2RDt8W4OM56PqMlOL3xIokUEQka66GT6e5pdu8QwuJ9BrWwhq9WFw4yZQe6FHb836qbbJLegvYVC9QjjZW2UDjtBUwcAkrghH0CgYBUMmMOCwIfMEtMaWxZRGdxRabazLhn7TXhBpVTuv7WouPaXYd7ZGjCTMKAuVa/E4afBlxgemnqBuX90gHpK/dDmn9l+lp8GZey0grJ7G0x5HEMiKziaX5PrgAcKbQ70m9ZNZ1deYhsC05X8rHNexZB6ns7Yms9L7qnlAy51ZH2zw==
-----END RSA PRIVATE KEY-----`)
	clientCrt := []byte(`-----BEGIN CERTIFICATE-----
MIICsTCCAZkCFEJJ4daz5sxkFlzq9n1djLEuG7bmMA0GCSqGSIb3DQEBCwUAMBMxETAPBgNVBAMMCHZhdWx0LWNhMB4XDTIxMDcyMDA4MTQxM1oXDTIyMDcyMDA4MTQxM1owFzEVMBMGA1UEAwwMdmF1bHQtY2xpZW50MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArfZ4HV1obFVlVNiA24tX/UOakqRnEtWXpIvaOsMaPGvvODgGe4XnyJGO32idPv85sIr7vDH9p+OhactVlJV1fu5SZoZ7pg4jTCLqVDCb3IRD++yik2Sw58YayNe3HiaCTsJQWeMXLzfaqOeyk6bEpBCJo09+3QxUWxijgJ7YZCb+Gi8pf3ZWeSZG+rGNNvXHmTs1Yu1H849SYXu+uJOd/R3ZSTw8CxFe4eTLgbCnPf6tgA8Sg2hc+CAZxunPP2JLZWbiJXxjNRoypso6MAJ1FRkx5sTJiLg6UoLvd95/S/lCVOR2PDlM1hg7ox8VEd4QHky7tLx7gji/5hHQKJQSTwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQAsDYKtzScIA7bqIOmqF8rr+oLSjRhPt5OfT+KGNdXk8G3VAy1ED2tyCHaRNC7dPLq4EvcxbIXQnXPy1iZMofriGbFPAcQ2fyWUesAD6bYSpI+bYxwz6Ebb93hU5nc/FyXg8yh0kgiGbY3MrACPjxqP2+z5kcOC3u3hx3SZylgW7TeOXDTdqSbNfH1b+1rR/bVNgQQshjhU9d+c4Yv/t0u07uykBhHLWZDSnYiAeOZ8+mWuOSDkcZHE1zznx74fWgtN0zRDtr0L0w9evT9R2CnNSZGxXcEQxAlQ7SL/Jyw82TFCGEw0L4jj7jjvx0N5J8KX/DulUDE9vuVyQEJ88Epe
-----END CERTIFICATE-----
`)

	cases := map[string]testCase{
		"InvalidVaultStore": {
			reason: "Should return error if given an invalid vault store.",
			args: args{
				store: &esv1alpha1.SecretStore{},
			},
			want: want{
				err: errors.New(errVaultStore),
			},
		},
		"AddVaultStoreCertsError": {
			reason: "Should return error if given an invalid CA certificate.",
			args: args{
				store: makeSecretStore(func(s *esv1alpha1.SecretStore) {
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
				store: makeSecretStore(func(s *esv1alpha1.SecretStore) {
					s.Spec.Provider.Vault.Auth = esv1alpha1.VaultAuth{}
				}),
			},
			want: want{
				err: errors.New(errAuthFormat),
			},
		},
		"GetKubeServiceAccountError": {
			reason: "Should return error if fetching kubernetes secret fails.",
			args: args{
				store: makeSecretStore(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errGetKubeSA, "example-sa", errBoom),
			},
		},
		"GetKubeSecretError": {
			reason: "Should return error if fetching kubernetes secret fails.",
			args: args{
				store: makeSecretStore(func(s *esv1alpha1.SecretStore) {
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
		"SuccessfulVaultStore": {
			reason: "Should return a Vault provider successfully",
			args: args{
				store: makeSecretStore(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.ServiceAccount); ok {
							o.Secrets = []corev1.ObjectReference{
								{
									Name: "example-secret-token",
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
				newClientFunc: func(c *vault.Config) (Client, error) {
					return &fake.VaultClient{
						MockNewRequest: fake.NewMockNewRequestFn(&vault.Request{}),
						MockRawRequestWithContext: fake.NewMockRawRequestWithContextFn(
							newVaultTokenIDResponse("test-token"), nil, func(got *vault.Request) error {
								kubeRole := makeValidSecretStore().Spec.Provider.Vault.Auth.Kubernetes.Role
								want := kubeParameters(kubeRole, string(secretData))
								if diff := cmp.Diff(want, got.Obj); diff != "" {
									t.Errorf("RawRequestWithContext(...): -want, +got:\n%s", diff)
								}

								return nil
							}),
						MockSetToken: fake.NewSetTokenFn(),
					}, nil
				},
			},
			want: want{
				err: nil,
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
				newClientFunc: func(c *vault.Config) (Client, error) {
					return &fake.VaultClient{
						MockNewRequest: fake.NewMockNewRequestFn(&vault.Request{}),
						MockRawRequestWithContext: fake.NewMockRawRequestWithContextFn(
							newVaultTokenIDResponse("test-token"), nil, func(got *vault.Request) error { return nil }),
						MockSetToken: fake.NewSetTokenFn(),
					}, nil
				},
			},
			want: want{
				err: nil,
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
				newClientFunc: func(c *vault.Config) (Client, error) {
					return &fake.VaultClient{
						MockNewRequest: fake.NewMockNewRequestFn(&vault.Request{}),
						MockRawRequestWithContext: fake.NewMockRawRequestWithContextFn(
							newVaultTokenIDResponse("test-token"), nil, func(got *vault.Request) error { return nil }),
						MockSetToken: fake.NewSetTokenFn(),
					}, nil
				},
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
				newClientFunc: func(c *vault.Config) (Client, error) {
					return &fake.VaultClient{
						MockNewRequest: fake.NewMockNewRequestFn(&vault.Request{}),
						MockRawRequestWithContext: fake.NewMockRawRequestWithContextFn(
							newVaultTokenIDResponse("test-token"), nil, func(got *vault.Request) error { return nil }),
						MockSetToken: fake.NewSetTokenFn(),
					}, nil
				},
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
	_, err := conn.NewClient(context.Background(), tc.args.store, tc.args.kube, tc.args.ns)
	if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
		t.Errorf("\n%s\nvault.New(...): -want error, +got error:\n%s", tc.reason, diff)
	}
}

func TestGetSecretMap(t *testing.T) {
	errBoom := errors.New("boom")

	type args struct {
		store   *esv1alpha1.VaultProvider
		kube    kclient.Client
		vClient Client
		ns      string
		data    esv1alpha1.ExternalSecretDataRemoteRef
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ReadSecretError": {
			reason: "Should return error if vault client fails to read secret.",
			args: args{
				store: makeSecretStore().Spec.Provider.Vault,
				vClient: &fake.VaultClient{
					MockNewRequest:            fake.NewMockNewRequestFn(&vault.Request{}),
					MockRawRequestWithContext: fake.NewMockRawRequestWithContextFn(nil, errBoom),
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
				client:    tc.args.vClient,
				store:     tc.args.store,
				namespace: tc.args.ns,
			}
			_, err := vStore.GetSecretMap(context.Background(), tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nvault.GetSecretMap(...): -want error, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}
