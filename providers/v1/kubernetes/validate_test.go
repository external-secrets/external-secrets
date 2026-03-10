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

package kubernetes

import (
	"context"
	"errors"
	"reflect"
	"testing"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pointer "k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type fakeReviewClient struct {
	authReview *authv1.SelfSubjectRulesReview
}

func (fk fakeReviewClient) Create(_ context.Context, _ *authv1.SelfSubjectRulesReview, _ metav1.CreateOptions) (*authv1.SelfSubjectRulesReview, error) {
	if fk.authReview == nil {
		return nil, errors.New(errSomethingWentWrong)
	}
	return fk.authReview, nil
}

type fakeAccessReviewClient struct {
	accessReview *authv1.SelfSubjectAccessReview
}

func (fk fakeAccessReviewClient) Create(_ context.Context, _ *authv1.SelfSubjectAccessReview, _ metav1.CreateOptions) (*authv1.SelfSubjectAccessReview, error) {
	if fk.accessReview == nil {
		return nil, errors.New(errSomethingWentWrong)
	}
	return fk.accessReview, nil
}

func TestValidateStore(t *testing.T) {
	type fields struct {
		Client       KClient
		ReviewClient RClient
		Namespace    string
	}

	tests := []struct {
		name        string
		fields      fields
		store       esv1.GenericStore
		wantErr     bool
		wantWarning bool
	}{
		{
			name: "empty ca returns warning for system roots",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{},
					},
				},
			},
			wantErr:     false,
			wantWarning: true,
		},
		{
			name: "authRef suppresses no-ca warning",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							AuthRef: &v1.SecretKeySelector{
								Name: "my-kubeconfig",
								Key:  "config",
							},
						},
					},
				},
			},
			wantErr:     false,
			wantWarning: false,
		},
		{
			name: "token auth without ca returns warning only",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Auth: &esv1.KubernetesAuth{
								Token: &esv1.TokenAuth{
									BearerToken: v1.SecretKeySelector{
										Name: "my-token",
										Key:  "token",
									},
								},
							},
						},
					},
				},
			},
			wantErr:     false,
			wantWarning: true,
		},
		{
			name: "no ca with other validation error still returns warning",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Auth: &esv1.KubernetesAuth{
								Cert: &esv1.CertAuth{
									ClientCert: v1.SecretKeySelector{
										Name: "",
									},
								},
							},
						},
					},
				},
			},
			wantErr:     true,
			wantWarning: true,
		},
		{
			name: "invalid client cert name",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								Cert: &esv1.CertAuth{
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
			name: "caprovider with empty namespace for cluster secret store",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "ClusterSecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CAProvider: &esv1.CAProvider{
									Name: "foobar",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "caprovider with non empty namespace for secret store",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CAProvider: &esv1.CAProvider{
									Name:      "foobar",
									Namespace: pointer.To("noop"),
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								Cert: &esv1.CertAuth{
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								Cert: &esv1.CertAuth{
									ClientCert: v1.SecretKeySelector{
										Name:      "foobar",
										Key:       "foobar",
										Namespace: pointer.To("noop"),
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								Token: &esv1.TokenAuth{
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								Token: &esv1.TokenAuth{
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								Token: &esv1.TokenAuth{
									BearerToken: v1.SecretKeySelector{
										Name:      "foobar",
										Key:       "foobar",
										Namespace: pointer.To("nop"),
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								ServiceAccount: &v1.ServiceAccountSelector{
									Name:      "foobar",
									Namespace: pointer.To("foobar"),
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
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Kubernetes: &esv1.KubernetesProvider{
							Server: esv1.KubernetesServer{
								CABundle: []byte("1234"),
							},
							Auth: &esv1.KubernetesAuth{
								ServiceAccount: &v1.ServiceAccountSelector{
									Name: "foobar",
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
			k := &Provider{}
			warnings, err := k.ValidateStore(tt.store)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProviderKubernetes.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantWarning {
				if len(warnings) != 1 {
					t.Fatalf("ProviderKubernetes.ValidateStore() expected exactly 1 warning, got %d: %v", len(warnings), warnings)
				}
				if warnings[0] != warnNoCAConfigured {
					t.Errorf("ProviderKubernetes.ValidateStore() warning = %q, want %q", warnings[0], warnNoCAConfigured)
				}
			} else if len(warnings) > 0 {
				t.Errorf("ProviderKubernetes.ValidateStore() unexpected warnings: %v", warnings)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	successReview := authv1.SelfSubjectRulesReview{
		Status: authv1.SubjectRulesReviewStatus{
			ResourceRules: []authv1.ResourceRule{
				{
					Verbs:     []string{"get"},
					Resources: []string{"secrets"},
				},
			},
		},
	}
	failReview := authv1.SelfSubjectRulesReview{
		Status: authv1.SubjectRulesReviewStatus{
			ResourceRules: []authv1.ResourceRule{
				{
					Verbs:     []string{"update"},
					Resources: []string{"secrets"},
				},
			},
		},
	}
	successWildcardReview := authv1.SelfSubjectRulesReview{
		Status: authv1.SubjectRulesReviewStatus{
			ResourceRules: []authv1.ResourceRule{
				{
					Verbs:     []string{"*"},
					Resources: []string{"*"},
					APIGroups: []string{"*"},
				},
			},
		},
	}
	successAccessReview := authv1.SelfSubjectAccessReview{
		Status: authv1.SubjectAccessReviewStatus{
			Allowed: true,
		},
	}
	failAccessReview := authv1.SelfSubjectAccessReview{
		Status: authv1.SubjectAccessReviewStatus{
			Allowed: false,
		},
	}

	type fields struct {
		Client             KClient
		ReviewClient       RClient
		AccessReviewClient AClient
		Namespace          string
		store              *esv1.KubernetesProvider
		storeKind          string
	}
	tests := []struct {
		name    string
		fields  fields
		want    esv1.ValidationResult
		wantErr bool
	}{
		{
			name: "empty ns should return unknown for referent auth",
			fields: fields{
				storeKind: esv1.ClusterSecretStoreKind,
				store: &esv1.KubernetesProvider{
					Auth: &esv1.KubernetesAuth{
						ServiceAccount: &v1.ServiceAccountSelector{
							Name: "foobar",
						},
					},
				},
				ReviewClient:       fakeReviewClient{authReview: &successReview},
				AccessReviewClient: fakeAccessReviewClient{accessReview: &successAccessReview},
			},
			want:    esv1.ValidationResultUnknown,
			wantErr: false,
		},
		{
			name: "review results in unknown",
			fields: fields{
				Namespace:          "default",
				ReviewClient:       fakeReviewClient{},
				AccessReviewClient: fakeAccessReviewClient{},
				store:              &esv1.KubernetesProvider{},
			},
			want:    esv1.ValidationResultUnknown,
			wantErr: true,
		},
		{
			name: "rules & access review not allowed results in error",
			fields: fields{
				Namespace:          "default",
				ReviewClient:       fakeReviewClient{authReview: &failReview},
				AccessReviewClient: fakeAccessReviewClient{accessReview: &failAccessReview},
				store:              &esv1.KubernetesProvider{},
			},
			want:    esv1.ValidationResultError,
			wantErr: true,
		},
		{
			name: "rules review allowed results in no error",
			fields: fields{
				Namespace:          "default",
				ReviewClient:       fakeReviewClient{authReview: &successReview},
				AccessReviewClient: fakeAccessReviewClient{accessReview: &failAccessReview},
				store:              &esv1.KubernetesProvider{},
			},
			want:    esv1.ValidationResultReady,
			wantErr: false,
		},
		{
			name: "rules review allowed results in no error",
			fields: fields{
				Namespace:          "default",
				ReviewClient:       fakeReviewClient{authReview: &successWildcardReview},
				AccessReviewClient: fakeAccessReviewClient{accessReview: &failAccessReview},
				store:              &esv1.KubernetesProvider{},
			},
			want:    esv1.ValidationResultReady,
			wantErr: false,
		},
		{
			name: "access review allowed results in no error",
			fields: fields{
				Namespace:          "default",
				ReviewClient:       fakeReviewClient{authReview: &failReview},
				AccessReviewClient: fakeAccessReviewClient{accessReview: &successAccessReview},
				store:              &esv1.KubernetesProvider{},
			},
			want:    esv1.ValidationResultReady,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Client{
				userSecretClient:       tt.fields.Client,
				userReviewClient:       tt.fields.ReviewClient,
				userAccessReviewClient: tt.fields.AccessReviewClient,
				namespace:              tt.fields.Namespace,
				store:                  tt.fields.store,
				storeKind:              tt.fields.storeKind,
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
