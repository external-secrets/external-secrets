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
							SecretRef: &esmeta.SecretKeySelector{
								Name: "vault-secret",
								Key:  "key",
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

func TestNewVault(t *testing.T) {
	errBoom := errors.New("boom")
	secretData := []byte("some-creds")

	type args struct {
		newClientFunc func(c *vault.Config) (Client, error)
		store         esv1alpha1.GenericStore
		kube          kclient.Client
		ns            string
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
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
		"GetKubeSecretError": {
			reason: "Should return error if fetching kubernetes secret fails.",
			args: args{
				store: makeSecretStore(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			want: want{
				err: fmt.Errorf(errGetKubeSecret, makeSecretStore().Spec.Provider.Vault.Auth.Kubernetes.SecretRef.Name, errBoom),
			},
		},
		"SuccessfulVaultStore": {
			reason: "Should return a Vault provider successfully",
			args: args{
				store: makeSecretStore(),
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
						if o, ok := obj.(*corev1.Secret); ok {
							o.Data = map[string][]byte{
								"key": secretData,
							}
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
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
		})
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
