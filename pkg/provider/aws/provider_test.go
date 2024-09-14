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

package aws

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pointer "k8s.io/utils/ptr"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager"
)

func TestProvider(t *testing.T) {
	cl := clientfake.NewClientBuilder().Build()
	p := Provider{}

	// inject fake static credentials because we test
	// if we are able to get credentials when constructing the client
	// see #415
	t.Setenv("AWS_ACCESS_KEY_ID", "1234")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "1234")

	tbl := []struct {
		test    string
		store   esv1beta1.GenericStore
		expType any
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
			test:    "should create parameter store client",
			expErr:  false,
			expType: &parameterstore.ParameterStore{},
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{
							Service: esv1beta1.AWSServiceParameterStore,
						},
					},
				},
			},
		},
		{
			test:    "should create secretsmanager client",
			expErr:  false,
			expType: &secretsmanager.SecretsManager{},
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{
							Service: esv1beta1.AWSServiceSecretsManager,
						},
					},
				},
			},
		},
		{
			test:   "invalid service should return an error",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{
							Service: "HIHIHIHHEHEHEHEHEHE",
						},
					},
				},
			},
		},
		{
			test:   "newSession error should be returned",
			expErr: true,
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						AWS: &esv1beta1.AWSProvider{
							Service: esv1beta1.AWSServiceParameterStore,
							Auth: esv1beta1.AWSAuth{
								SecretRef: &esv1beta1.AWSAuthSecretRef{
									AccessKeyID: esmeta.SecretKeySelector{
										Name:      "foo",
										Namespace: aws.String("NOOP"),
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

const (
	validRegion                  = "eu-central-1"
	validFipsSecretManagerRegion = "us-east-1-fips"
	validFipsSsmRegion           = "fips-us-east-1"
)

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
			name:    "invalid region",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region: "noop.",
							},
						},
					},
				},
			},
		},
		{
			name: "valid region secrets manager",
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
							},
						},
					},
				},
			},
		},
		{
			name: "valid fips region secrets manager",
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validFipsSecretManagerRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
							},
						},
					},
				},
			},
		},
		{
			name: "valid fips region parameter store",
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validFipsSsmRegion,
								Service: esv1beta1.AWSServiceParameterStore,
							},
						},
					},
				},
			},
		},
		{
			name: "valid secretsmanager config: force delete without recovery",
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								SecretsManager: &esv1beta1.SecretsManager{
									ForceDeleteWithoutRecovery: true,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "valid secretsmanager config: recovery window",
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								SecretsManager: &esv1beta1.SecretsManager{
									RecoveryWindowInDays: 30,
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "invalid static creds auth / AccessKeyID",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								Auth: esv1beta1.AWSAuth{
									SecretRef: &esv1beta1.AWSAuthSecretRef{
										AccessKeyID: esmeta.SecretKeySelector{
											Name:      "foobar",
											Namespace: pointer.To("unacceptable"),
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
			name:    "invalid static creds auth / SecretAccessKey",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								Auth: esv1beta1.AWSAuth{
									SecretRef: &esv1beta1.AWSAuthSecretRef{
										SecretAccessKey: esmeta.SecretKeySelector{
											Name:      "foobar",
											Namespace: pointer.To("unacceptable"),
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
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								Auth: esv1beta1.AWSAuth{
									SecretRef: &esv1beta1.AWSAuthSecretRef{
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
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								Auth: esv1beta1.AWSAuth{
									SecretRef: &esv1beta1.AWSAuthSecretRef{
										AccessKeyID: esmeta.SecretKeySelector{
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
		{
			name:    "referentAuth jwt: sa selector without namespace",
			wantErr: false,
			args: args{
				store: &esv1beta1.ClusterSecretStore{
					TypeMeta: v1.TypeMeta{
						Kind: esv1beta1.ClusterSecretStoreKind,
					},
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								Auth: esv1beta1.AWSAuth{
									JWTAuth: &esv1beta1.AWSJWTAuth{
										ServiceAccountRef: &esmeta.ServiceAccountSelector{
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
		{
			name:    "invalid jwt auth: not allowed sa selector namespace",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								Auth: esv1beta1.AWSAuth{
									JWTAuth: &esv1beta1.AWSJWTAuth{
										ServiceAccountRef: &esmeta.ServiceAccountSelector{
											Name:      "foobar",
											Namespace: pointer.To("unacceptable"),
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
			name:    "invalid SecretsManager config: conflicting settings",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								SecretsManager: &esv1beta1.SecretsManager{
									ForceDeleteWithoutRecovery: true,
									RecoveryWindowInDays:       7,
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "invalid SecretsManager config: recovery window too small",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								SecretsManager: &esv1beta1.SecretsManager{
									RecoveryWindowInDays: 6,
								},
							},
						},
					},
				},
			},
		},
		{
			name:    "invalid SecretsManager config: recovery window too big",
			wantErr: true,
			args: args{
				store: &esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							AWS: &esv1beta1.AWSProvider{
								Region:  validRegion,
								Service: esv1beta1.AWSServiceSecretsManager,
								SecretsManager: &esv1beta1.SecretsManager{
									RecoveryWindowInDays: 31,
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
				AWS: &esv1beta1.AWSProvider{
					Service: "ParameterStore",
					Region:  validRegion,
					Auth: esv1beta1.AWSAuth{
						SecretRef: &esv1beta1.AWSAuthSecretRef{
							SecretAccessKey: esmeta.SecretKeySelector{
								Name: "creds",
								Key:  "sak",
							},
							AccessKeyID: esmeta.SecretKeySelector{
								Name: "creds",
								Key:  "ak",
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

	expected := fmt.Sprintf("unable to initialize aws provider: time: invalid duration %q", invalid)
	ctx := context.TODO()

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "creds",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"sak": []byte("OK"),
			"ak":  []byte("OK"),
		},
	}).Build()
	provider := func(*session.Session) stsiface.STSAPI { return nil }

	_, err := newClient(ctx, spec, kube, "default", provider)

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
