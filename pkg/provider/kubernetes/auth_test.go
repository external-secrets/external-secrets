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
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	pointer "k8s.io/utils/ptr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	fclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	utilfake "github.com/external-secrets/external-secrets/pkg/provider/util/fake"
)

func TestSetAuth(t *testing.T) {
	type fields struct {
		kube          kclient.Client
		kubeclientset typedcorev1.CoreV1Interface
		store         *esv1beta1.KubernetesProvider
		namespace     string
		storeKind     string
	}
	type want struct {
		Certificate []byte
		Key         []byte
		CA          []byte
		BearerToken []byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    want
		wantErr bool
	}{
		{
			name: "should return err if no ca provided",
			fields: fields{
				store: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{},
				},
			},
			want:    want{},
			wantErr: true,
		},
		{
			name: "should return err if no auth provided",
			fields: fields{
				store: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{
						CABundle: []byte("1234"),
					},
				},
			},
			want: want{
				CA: []byte("1234"),
			},
			wantErr: true,
		},
		{
			name: "should fetch ca from Secret",
			fields: fields{
				namespace: "default",
				kube: fclient.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foobar",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"cert": []byte("1234"),
					},
				}).Build(),
				store: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{
						CAProvider: &esv1beta1.CAProvider{
							Type: esv1beta1.CAProviderTypeSecret,
							Name: "foobar",
							Key:  "cert",
						},
					},
				},
			},
			want: want{
				CA: []byte("1234"),
			},
			wantErr: true,
		},
		{
			name: "should fetch ca from ConfigMap",
			fields: fields{
				namespace: "default",
				kube: fclient.NewClientBuilder().WithObjects(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foobar",
						Namespace: "default",
					},
					Data: map[string]string{
						"cert": "1234",
					},
				}).Build(),
				store: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{
						CAProvider: &esv1beta1.CAProvider{
							Type: esv1beta1.CAProviderTypeConfigMap,
							Name: "foobar",
							Key:  "cert",
						},
					},
				},
			},
			want: want{
				CA: []byte("1234"),
			},
			wantErr: true,
		},
		{
			name: "should set token from secret",
			fields: fields{
				namespace: "default",
				kube: fclient.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foobar",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"token": []byte("mytoken"),
					},
				}).Build(),
				store: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{
						CABundle: []byte("1234"),
					},
					Auth: esv1beta1.KubernetesAuth{
						Token: &esv1beta1.TokenAuth{
							BearerToken: v1.SecretKeySelector{
								Name:      "foobar",
								Namespace: pointer.To("shouldnotberelevant"),
								Key:       "token",
							},
						},
					},
				},
			},
			want: want{
				CA:          []byte("1234"),
				BearerToken: []byte("mytoken"),
			},
			wantErr: false,
		},
		{
			name: "should set client cert from secret",
			fields: fields{
				namespace: "default",
				kube: fclient.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mycert",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"cert": []byte("my-cert"),
						"key":  []byte("my-key"),
					},
				}).Build(),
				store: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{
						CABundle: []byte("1234"),
					},
					Auth: esv1beta1.KubernetesAuth{
						Cert: &esv1beta1.CertAuth{
							ClientCert: v1.SecretKeySelector{
								Name: "mycert",
								Key:  "cert",
							},
							ClientKey: v1.SecretKeySelector{
								Name: "mycert",
								Key:  "key",
							},
						},
					},
				},
			},
			want: want{
				CA:          []byte("1234"),
				Certificate: []byte("my-cert"),
				Key:         []byte("my-key"),
			},
			wantErr: false,
		},
		{
			name: "should set token from service account",
			fields: fields{
				namespace: "default",
				kube: fclient.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-sa",
						Namespace: "default",
					},
				}).Build(),
				kubeclientset: utilfake.NewCreateTokenMock().WithToken("my-sa-token"),
				store: &esv1beta1.KubernetesProvider{
					Server: esv1beta1.KubernetesServer{
						CABundle: []byte("1234"),
					},
					Auth: esv1beta1.KubernetesAuth{
						ServiceAccount: &v1.ServiceAccountSelector{
							Name:      "my-sa",
							Namespace: pointer.To("shouldnotberelevant"),
						},
					},
				},
			},
			want: want{
				CA:          []byte("1234"),
				BearerToken: []byte("my-sa-token"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Client{
				ctrlClientset: tt.fields.kubeclientset,
				ctrlClient:    tt.fields.kube,
				store:         tt.fields.store,
				namespace:     tt.fields.namespace,
				storeKind:     tt.fields.storeKind,
			}
			if err := k.setAuth(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("BaseClient.setAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
			w := want{
				Certificate: k.Certificate,
				Key:         k.Key,
				CA:          k.CA,
				BearerToken: k.BearerToken,
			}
			if !cmp.Equal(w, tt.want) {
				t.Errorf("unexpected value: expected %#v, got %#v", tt.want, w)
			}
		})
	}
}
