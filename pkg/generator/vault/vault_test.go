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

package vaultdynamic

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	vaultapi "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	utilfake "github.com/external-secrets/external-secrets/pkg/provider/util/fake"
	provider "github.com/external-secrets/external-secrets/pkg/provider/vault"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
)

type args struct {
	jsonSpec      *apiextensions.JSON
	kube          kclient.Client
	corev1        typedcorev1.CoreV1Interface
	vaultClientFn func(config *vaultapi.Config) (util.Client, error)
}

type want struct {
	val        map[string][]byte
	partialVal map[string][]byte
	err        error
}

type testCase struct {
	reason string
	args   args
	want   want
}

func TestVaultDynamicSecretGenerator(t *testing.T) {
	cases := map[string]testCase{
		"NilSpec": {
			reason: "Raise an error with empty spec.",
			args: args{
				jsonSpec: nil,
			},
			want: want{
				err: errors.New("no config spec provided"),
			},
		},
		"InvalidSpec": {
			reason: "Raise an error with invalid spec.",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(``),
				},
			},
			want: want{
				err: errors.New("no Vault provider config in spec"),
			},
		},
		"MissingRoleName": {
			reason: "Raise error if incomplete k8s auth config is provided.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        serviceAccountRef:
          name: "testing"
  method: GET
  path: "github/token/example"`),
				},
				kube: clientfake.NewClientBuilder().Build(),
			},
			want: want{
				err: errors.New("unable to setup Vault client: no role name was provided"),
			},
		},
		"EmptyVaultResponse": {
			reason: "Fail on empty response from Vault.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        role: test
        serviceAccountRef:
          name: "testing"
  method: GET
  path: "github/token/example"`),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testing",
						Namespace: "testing",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "test",
						},
					},
				}).Build(),
			},
			want: want{
				err: errors.New("unable to get dynamic secret: empty response from Vault"),
			},
		},
		"EmptyVaultPOST": {
			reason: "Fail on empty response from Vault.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        role: test
        serviceAccountRef:
          name: "testing"
  method: POST
  parameters:
    foo: "bar"
  path: "github/token/example"`),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testing",
						Namespace: "testing",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "test",
						},
					},
				}).Build(),
			},
			want: want{
				err: errors.New("unable to get dynamic secret: empty response from Vault"),
			},
		},
		"AllowEmptyVaultPOST": {
			reason: "Allow empty response from Vault POST.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        role: test
        serviceAccountRef:
          name: "testing"
  method: POST
  parameters:
    foo: "bar"
  path: "github/token/example"
  allowEmptyResponse: true`),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testing",
						Namespace: "testing",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "test",
						},
					},
				}).Build(),
			},
			want: want{
				err: nil,
				val: nil,
			},
		},
		"AllowEmptyVaultGET": {
			reason: "Allow empty response from Vault GET.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        role: test
        serviceAccountRef:
          name: "testing"
  method: GET
  parameters:
    foo: "bar"
  path: "github/token/example"
  allowEmptyResponse: true`),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testing",
						Namespace: "testing",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "test",
						},
					},
				}).Build(),
			},
			want: want{
				err: nil,
				val: nil,
			},
		},
		"DataResultType": {
			reason: "Allow accessing the data section of the response from Vault API.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        role: test
        serviceAccountRef:
          name: "testing"
  path: "github/token/example"`),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testing",
						Namespace: "testing",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "test",
						},
					},
				}).Build(),
				vaultClientFn: fake.ModifiableClientWithLoginMock(
					func(cl *fake.VaultClient) {
						cl.MockLogical.ReadWithDataWithContextFn = func(ctx context.Context, path string, data map[string][]string) (*vaultapi.Secret, error) {
							return &vaultapi.Secret{
								Data: map[string]interface{}{
									"key": "value",
								},
							}, nil
						}
					},
				),
			},
			want: want{
				err: nil,
				val: map[string][]byte{"key": []byte("value")},
			},
		},
		"AuthResultType": {
			reason: "Allow accessing auth section of the response from Vault API.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        role: test
        serviceAccountRef:
          name: "testing"
  path: "github/token/example"
  resultType: "Auth"`),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testing",
						Namespace: "testing",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "test",
						},
					},
				}).Build(),
				vaultClientFn: fake.ModifiableClientWithLoginMock(
					func(cl *fake.VaultClient) {
						cl.MockLogical.ReadWithDataWithContextFn = func(ctx context.Context, path string, data map[string][]string) (*vaultapi.Secret, error) {
							return &vaultapi.Secret{
								Auth: &vaultapi.SecretAuth{
									EntityID: "123",
								},
							}, nil
						}
					},
				),
			},
			want: want{
				err:        nil,
				partialVal: map[string][]byte{"entity_id": []byte("123")},
			},
		},
		"RawResultType": {
			reason: "Allow accessing auth section of the response from Vault API.",
			args: args{
				corev1: utilfake.NewCreateTokenMock().WithToken("ok"),
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: VaultDynamicSecret
spec:
  provider:
    auth:
      kubernetes:
        role: test
        serviceAccountRef:
          name: "testing"
  path: "github/token/example"
  resultType: "Raw"`),
				},
				kube: clientfake.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testing",
						Namespace: "testing",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "test",
						},
					},
				}).Build(),
				vaultClientFn: fake.ModifiableClientWithLoginMock(
					func(cl *fake.VaultClient) {
						cl.MockLogical.ReadWithDataWithContextFn = func(ctx context.Context, path string, data map[string][]string) (*vaultapi.Secret, error) {
							return &vaultapi.Secret{
								LeaseID: "123",
								Data: map[string]interface{}{
									"key": "value",
								},
							}, nil
						}
					},
				),
			},
			want: want{
				err: nil,
				partialVal: map[string][]byte{
					"lease_id": []byte("123"),
					"data":     []byte(`{"key":"value"}`),
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			newClientFn := fake.ClientWithLoginMock
			if tc.args.vaultClientFn != nil {
				newClientFn = tc.args.vaultClientFn
			}
			c := &provider.Provider{NewVaultClient: newClientFn}
			gen := &Generator{}
			val, _, err := gen.generate(context.Background(), c, tc.args.jsonSpec, tc.args.kube, tc.args.corev1, "testing")
			if tc.want.err != nil {
				if err != nil {
					if diff := cmp.Diff(tc.want.err.Error(), err.Error()); diff != "" {
						t.Errorf("\n%s\nvault.GetSecret(...): -want error, +got error:\n%s", tc.reason, diff)
					}
				} else {
					t.Errorf("\n%s\nvault.GetSecret(...): -want error, +got val:\n%s", tc.reason, val)
				}
			} else if tc.want.partialVal != nil {
				for k, v := range tc.want.partialVal {
					if diff := cmp.Diff(v, val[k]); diff != "" {
						t.Errorf("\n%s\nvault.GetSecret(...) -> %s: -want partial, +got partial:\n%s", k, tc.reason, diff)
					}
				}
			} else {
				if diff := cmp.Diff(tc.want.val, val); diff != "" {
					t.Errorf("\n%s\nvault.GetSecret(...): -want val, +got val:\n%s", tc.reason, diff)
				}
			}
		})
	}
}
