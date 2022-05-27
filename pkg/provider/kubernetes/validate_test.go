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
package kubernetes

import (
	"context"
	"errors"
	"reflect"
	"testing"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type fakeReviewClient struct {
	authReview *authv1.SelfSubjectAccessReview
}

func (fk fakeReviewClient) Create(ctx context.Context, selfSubjectAccessReview *authv1.SelfSubjectAccessReview, opts metav1.CreateOptions) (*authv1.SelfSubjectAccessReview, error) {
	if fk.authReview == nil {
		return nil, errors.New(errSomethingWentWrong)
	}
	return fk.authReview, nil
}

func TestValidateStore(t *testing.T) {
	type fields struct {
		Client       KClient
		ReviewClient RClient
		Namespace    string
	}

	tests := []struct {
		name    string
		fields  fields
		store   esv1beta1.GenericStore
		wantErr bool
	}{
		{
			name: "empty ca",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid client cert name",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								Cert: &esv1beta1.CertAuth{
									ClientCert: v1.SecretKeySelector{
										Name: "",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid client cert key",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								Cert: &esv1beta1.CertAuth{
									ClientCert: v1.SecretKeySelector{
										Name: "foobar",
										Key:  "",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid client cert secretRef",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								Cert: &esv1beta1.CertAuth{
									ClientCert: v1.SecretKeySelector{
										Name:      "foobar",
										Key:       "foobar",
										Namespace: pointer.String("noop"),
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid client token auth name",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								Token: &esv1beta1.TokenAuth{
									BearerToken: v1.SecretKeySelector{
										Name: "",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid client token auth key",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								Token: &esv1beta1.TokenAuth{
									BearerToken: v1.SecretKeySelector{
										Name: "foobar",
										Key:  "",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid client token auth namespace",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								Token: &esv1beta1.TokenAuth{
									BearerToken: v1.SecretKeySelector{
										Name:      "foobar",
										Key:       "foobar",
										Namespace: pointer.String("nop"),
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid service account auth name",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								ServiceAccount: &esv1beta1.ServiceAccountAuth{
									ServiceAccountRef: v1.ServiceAccountSelector{
										Name:      "foobar",
										Namespace: pointer.String("foobar"),
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid auth",
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Kubernetes: &esv1beta1.KubernetesProvider{
							Server: esv1beta1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: esv1beta1.KubernetesAuth{
								ServiceAccount: &esv1beta1.ServiceAccountAuth{
									ServiceAccountRef: v1.ServiceAccountSelector{
										Name: "foobar",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &ProviderKubernetes{
				Client:       tt.fields.Client,
				ReviewClient: tt.fields.ReviewClient,
				Namespace:    tt.fields.Namespace,
			}
			if err := k.ValidateStore(tt.store); (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	type fields struct {
		Client       KClient
		ReviewClient RClient
		Namespace    string
	}
	tests := []struct {
		name    string
		fields  fields
		want    esv1beta1.ValidationResult
		wantErr bool
	}{
		{
			name:    "empty ns should return unknown for referent auth",
			fields:  fields{},
			want:    esv1beta1.ValidationResultUnknown,
			wantErr: false,
		},
		{
			name: "review results in unknown",
			fields: fields{
				Namespace:    "default",
				ReviewClient: fakeReviewClient{},
			},
			want:    esv1beta1.ValidationResultUnknown,
			wantErr: true,
		},
		{
			name: "not allowed results in error",
			fields: fields{
				Namespace: "default",
				ReviewClient: fakeReviewClient{authReview: &authv1.SelfSubjectAccessReview{
					Status: authv1.SubjectAccessReviewStatus{Allowed: false},
				}},
			},
			want:    esv1beta1.ValidationResultError,
			wantErr: true,
		},
		{
			name: "allowed results in no error",
			fields: fields{
				Namespace: "default",
				ReviewClient: fakeReviewClient{authReview: &authv1.SelfSubjectAccessReview{
					Status: authv1.SubjectAccessReviewStatus{Allowed: true},
				}},
			},
			want:    esv1beta1.ValidationResultReady,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &ProviderKubernetes{
				Client:       tt.fields.Client,
				ReviewClient: tt.fields.ReviewClient,
				Namespace:    tt.fields.Namespace,
			}
			got, err := k.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProviderKubernetes.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
