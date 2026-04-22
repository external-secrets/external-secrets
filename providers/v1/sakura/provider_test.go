/*
Copyright © The ESO Authors

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

package sakura_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/sakura"
)

func TestValidateStore(t *testing.T) {
	t.Parallel()

	type args struct {
		store esv1.GenericStore
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "empty VaultResourceID",
			wantErr: true,
			args: args{
				store: &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-store",
						Namespace: "default",
					},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Sakura: &esv1.SakuraProvider{
								VaultResourceID: "",
								Auth: esv1.SakuraAuth{
									SecretRef: esv1.SakuraSecretRef{
										AccessToken: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token",
										},
										AccessTokenSecret: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token-secret",
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
			name:    "AccessToken namespace mismatch",
			wantErr: true,
			args: args{
				store: &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-store",
						Namespace: "default",
					},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Sakura: &esv1.SakuraProvider{
								VaultResourceID: "123456789012",
								Auth: esv1.SakuraAuth{
									SecretRef: esv1.SakuraSecretRef{
										AccessToken: esmeta.SecretKeySelector{
											Name:      "secret-name",
											Key:       "access-token",
											Namespace: new("different-namespace"),
										},
										AccessTokenSecret: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token-secret",
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
			name:    "AccessTokenSecret namespace mismatch",
			wantErr: true,
			args: args{
				store: &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-store",
						Namespace: "default",
					},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Sakura: &esv1.SakuraProvider{
								VaultResourceID: "123456789012",
								Auth: esv1.SakuraAuth{
									SecretRef: esv1.SakuraSecretRef{
										AccessToken: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token",
										},
										AccessTokenSecret: esmeta.SecretKeySelector{
											Name:      "secret-name",
											Key:       "access-token-secret",
											Namespace: new("different-namespace"),
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
			name:    "SecretRef without namespace",
			wantErr: false,
			args: args{
				store: &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-store",
						Namespace: "default",
					},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Sakura: &esv1.SakuraProvider{
								VaultResourceID: "123456789012",
								Auth: esv1.SakuraAuth{
									SecretRef: esv1.SakuraSecretRef{
										AccessToken: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token",
										},
										AccessTokenSecret: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token-secret",
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
			name:    "SecretStore with explicit namespace",
			wantErr: false,
			args: args{
				store: &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-store",
						Namespace: "default",
					},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Sakura: &esv1.SakuraProvider{
								VaultResourceID: "123456789012",
								Auth: esv1.SakuraAuth{
									SecretRef: esv1.SakuraSecretRef{
										AccessToken: esmeta.SecretKeySelector{
											Name:      "secret-name",
											Key:       "access-token",
											Namespace: new("default"),
										},
										AccessTokenSecret: esmeta.SecretKeySelector{
											Name:      "secret-name",
											Key:       "access-token-secret",
											Namespace: new("default"),
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
			name:    "ClusterSecretStore without namespace",
			wantErr: false,
			args: args{
				store: &esv1.ClusterSecretStore{
					TypeMeta: metav1.TypeMeta{
						Kind: esv1.ClusterSecretStoreKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-store",
					},

					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Sakura: &esv1.SakuraProvider{
								VaultResourceID: "123456789012",
								Auth: esv1.SakuraAuth{
									SecretRef: esv1.SakuraSecretRef{
										AccessToken: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token",
										},
										AccessTokenSecret: esmeta.SecretKeySelector{
											Name: "secret-name",
											Key:  "access-token-secret",
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
			name:    "ClusterSecretStore with explicit namespace",
			wantErr: false,
			args: args{
				store: &esv1.ClusterSecretStore{
					TypeMeta: metav1.TypeMeta{
						Kind: esv1.ClusterSecretStoreKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-store",
					},

					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Sakura: &esv1.SakuraProvider{
								VaultResourceID: "123456789012",
								Auth: esv1.SakuraAuth{
									SecretRef: esv1.SakuraSecretRef{
										AccessToken: esmeta.SecretKeySelector{
											Name:      "secret-name",
											Key:       "access-token",
											Namespace: new("some-namespace"),
										},
										AccessTokenSecret: esmeta.SecretKeySelector{
											Name:      "secret-name",
											Key:       "access-token-secret",
											Namespace: new("some-namespace"),
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
			t.Parallel()

			_, err := new(sakura.Provider).ValidateStore(tt.args.store)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
