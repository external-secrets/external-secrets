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

package barbican

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestProvider(t *testing.T) {
	cl := clientfake.NewClientBuilder().Build()
	p := Provider{}

	tbl := []struct {
		test    string
		store   esv1beta1.GenericStore
		expType interface{}
		expErr  bool
	}{
		{
			test:   "should not create provider due to nil store",
			store:  nil,
			expErr: true,
		},
		{
			test:   "should not create provider due to missing provider",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{},
			},
		},
		{
			test:   "should not create provider due to missing provider field",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{},
				},
			},
		},
		{
			test:   "newSession error should be returned",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Barbican: &esv1beta1.BarbicanProvider{
							Auth: esv1beta1.BarbicanAuth{
								UserPass: &esv1beta1.BarbicanAuthUserPass{
									UserName: "foo",
									PasswordRef: &esv1beta1.BarbicanAuthSecretRef{
										SecretAccessKey: esmeta.SecretKeySelector{
											Name:      "foo",
											Namespace: StringPtr("NOOP"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for i := range tbl {
		row := tbl[i]
		t.Run(row.test, func(t *testing.T) {
			sc, err := p.NewClient(context.TODO(), row.store, cl, "foo")
			if row.expErr {
				assert.Error(t, err)
				assert.Nil(t, sc)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, sc)
				assert.IsType(t, row.expType, sc)
			}
		})
	}
}

func TestValidateStore(t *testing.T) {
	type args struct {
		store esv1beta1.GenericStore
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "invalid static creds auth / AccessKeyID",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Barbican: &esv1beta1.BarbicanProvider{
								Auth: esv1beta1.BarbicanAuth{
									UserPass: &esv1beta1.BarbicanAuthUserPass{
										UserName: "foo",
										PasswordRef: &esv1beta1.BarbicanAuthSecretRef{
											SecretAccessKey: esmeta.SecretKeySelector{
												Name:      "foobar",
												Namespace: StringPtr("unacceptable"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "referentAuth static creds / SecretAccessKey without namespace",
			wantErr: false,
			args: args{
				store: &esv1beta1.ClusterSecretStore{
					TypeMeta: v1.TypeMeta{
						Kind: esv1beta1.ClusterSecretStoreKind,
					},
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Barbican: &esv1beta1.BarbicanProvider{
								Auth: esv1beta1.BarbicanAuth{
									UserPass: &esv1beta1.BarbicanAuthUserPass{
										UserName: "foobar",
										PasswordRef: &esv1beta1.BarbicanAuthSecretRef{
											SecretAccessKey: esmeta.SecretKeySelector{
												Name: "foobar",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "referentAuth static creds / AccessKeyID without namespace",
			wantErr: false,
			args: args{
				store: &esv1beta1.ClusterSecretStore{
					TypeMeta: v1.TypeMeta{
						Kind: esv1beta1.ClusterSecretStoreKind,
					},
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Barbican: &esv1beta1.BarbicanProvider{
								Auth: esv1beta1.BarbicanAuth{
									UserPass: &esv1beta1.BarbicanAuthUserPass{
										UserName: "foobar",
										PasswordRef: &esv1beta1.BarbicanAuthSecretRef{
											SecretAccessKey: esmeta.SecretKeySelector{
												Name: "foobar",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{}
			if _, err := p.ValidateStore(tt.args.store); (err != nil) != tt.wantErr {
				t.Errorf("Provider.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidRetryInput(t *testing.T) {
	invalid := "Invalid"
	spec := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Barbican: &esv1beta1.BarbicanProvider{
					Auth: esv1beta1.BarbicanAuth{
						UserPass: &esv1beta1.BarbicanAuthUserPass{
							UserName: "foobar",
							PasswordRef: &esv1beta1.BarbicanAuthSecretRef{
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: "foobar",
								},
							},
						},
					},
				},
			},
			RetrySettings: &esv1beta1.SecretStoreRetrySettings{
				RetryInterval: &invalid,
			},
		},
	}

	expected := "invalid barbican config"
	ctx := context.TODO()

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "creds",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"sak": []byte("OK"),
		},
	}).Build()

	_, err := newClient(ctx, spec, kube, "default")

	if !ErrorContains(err, expected) {
		t.Errorf("CheckValidRetryInput unexpected error: %s, expected: '%s'", err.Error(), expected)
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
