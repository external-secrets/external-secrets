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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	pointer "k8s.io/utils/ptr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	fclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	utilfake "github.com/external-secrets/external-secrets/runtime/util/fake"
)

const (
	caCert = `-----BEGIN CERTIFICATE-----
MIICGTCCAZ+gAwIBAgIQCeCTZaz32ci5PhwLBCou8zAKBggqhkjOPQQDAzBOMQsw
CQYDVQQGEwJVUzEXMBUGA1UEChMORGlnaUNlcnQsIEluYy4xJjAkBgNVBAMTHURp
Z2lDZXJ0IFRMUyBFQ0MgUDM4NCBSb290IEc1MB4XDTIxMDExNTAwMDAwMFoXDTQ2
MDExNDIzNTk1OVowTjELMAkGA1UEBhMCVVMxFzAVBgNVBAoTDkRpZ2lDZXJ0LCBJ
bmMuMSYwJAYDVQQDEx1EaWdpQ2VydCBUTFMgRUNDIFAzODQgUm9vdCBHNTB2MBAG
ByqGSM49AgEGBSuBBAAiA2IABMFEoc8Rl1Ca3iOCNQfN0MsYndLxf3c1TzvdlHJS
7cI7+Oz6e2tYIOyZrsn8aLN1udsJ7MgT9U7GCh1mMEy7H0cKPGEQQil8pQgO4CLp
0zVozptjn4S1mU1YoI71VOeVyaNCMEAwHQYDVR0OBBYEFMFRRVBZqz7nLFr6ICIS
B4CIfBFqMA4GA1UdDwEB/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MAoGCCqGSM49
BAMDA2gAMGUCMQCJao1H5+z8blUD2WdsJk6Dxv3J+ysTvLd6jLRl0mlpYxNjOyZQ
LgGheQaRnUi/wr4CMEfDFXuxoJGZSZOoPHzoRgaLLPIxAJSdYsiJvRmEFOml+wG4
DXZDjC5Ty3zfDBeWUA==
-----END CERTIFICATE-----
`
	authTestKubeConfig = `apiVersion: v1
clusters:
- cluster:
    server: https://api.my-domain.tld
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNHVENDQVorZ0F3SUJBZ0lRQ2VDVFphejMyY2k1UGh3TEJDb3U4ekFLQmdncWhrak9QUVFEQXpCT01Rc3cKQ1FZRFZRUUdFd0pWVXpFWE1CVUdBMVVFQ2hNT1JHbG5hVU5sY25Rc0lFbHVZeTR4SmpBa0JnTlZCQU1USFVScApaMmxEWlhKMElGUk1VeUJGUTBNZ1VETTROQ0JTYjI5MElFYzFNQjRYRFRJeE1ERXhOVEF3TURBd01Gb1hEVFEyCk1ERXhOREl6TlRrMU9Wb3dUakVMTUFrR0ExVUVCaE1DVlZNeEZ6QVZCZ05WQkFvVERrUnBaMmxEWlhKMExDQkoKYm1NdU1TWXdKQVlEVlFRREV4MUVhV2RwUTJWeWRDQlVURk1nUlVORElGQXpPRFFnVW05dmRDQkhOVEIyTUJBRwpCeXFHU000OUFnRUdCU3VCQkFBaUEySUFCTUZFb2M4UmwxQ2EzaU9DTlFmTjBNc1luZEx4ZjNjMVR6dmRsSEpTCjdjSTcrT3o2ZTJ0WUlPeVpyc244YUxOMXVkc0o3TWdUOVU3R0NoMW1NRXk3SDBjS1BHRVFRaWw4cFFnTzRDTHAKMHpWb3pwdGpuNFMxbVUxWW9JNzFWT2VWeWFOQ01FQXdIUVlEVlIwT0JCWUVGTUZSUlZCWnF6N25MRnI2SUNJUwpCNENJZkJGcU1BNEdBMVVkRHdFQi93UUVBd0lCaGpBUEJnTlZIUk1CQWY4RUJUQURBUUgvTUFvR0NDcUdTTTQ5CkJBTURBMmdBTUdVQ01RQ0phbzFINSt6OGJsVUQyV2RzSms2RHh2M0oreXNUdkxkNmpMUmwwbWxwWXhOak95WlEKTGdHaGVRYVJuVWkvd3I0Q01FZkRGWHV4b0pHWlNaT29QSHpvUmdhTExQSXhBSlNkWXNpSnZSbUVGT21sK3dHNApEWFpEakM1VHkzemZEQmVXVUE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  name: mycluster
contexts:
- context:
    cluster: mycluster
    user: myuser
  name: mycontext
current-context: mycontext
kind: Config
preferences: {}
users:
- name: myuser
  user:
    token: eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE3MTkzOTY4OTksImV4cCI6MTc1MDkzMjg4NywiYXVkIjoid3d3LmV4YW1wbGUuY29tIiwic3ViIjoianJvY2tldEBleGFtcGxlLmNvbSIsIkdpdmVuTmFtZSI6IkpvaG5ueSIsIlN1cm5hbWUiOiJSb2NrZXQiLCJFbWFpbCI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJSb2xlIjpbIk1hbmFnZXIiLCJQcm9qZWN0IEFkbWluaXN0cmF0b3IiXX0.xXrfIl0akhfjWU_BDl7Ad54SXje0YlJdnugzwh96VmM
`
	serverURL = "https://my.test.tld"
)

func TestSetAuth(t *testing.T) {
	type fields struct {
		kube          kclient.Client
		kubeclientset typedcorev1.CoreV1Interface
		store         *esv1.KubernetesProvider
		namespace     string
		storeKind     string
	}
	type want = rest.Config
	tests := []struct {
		name    string
		fields  fields
		want    *want
		wantErr bool
	}{
		{
			name: "should return err if no ca provided",
			fields: fields{
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return err if no auth provided",
			fields: fields{
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						CABundle: []byte(caCert),
					},
				},
			},
			want:    nil,
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
						"cert":  []byte(caCert),
						"token": []byte("mytoken"),
					},
				}).Build(),
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						URL: serverURL,
						CAProvider: &esv1.CAProvider{
							Type: esv1.CAProviderTypeSecret,
							Name: "foobar",
							Key:  "cert",
						},
					},
					Auth: &esv1.KubernetesAuth{
						Token: &esv1.TokenAuth{
							BearerToken: v1.SecretKeySelector{
								Name:      "foobar",
								Namespace: pointer.To("shouldnotberelevant"),
								Key:       "token",
							},
						},
					},
				},
			},
			want: &want{
				Host:        serverURL,
				BearerToken: "mytoken",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte(caCert),
				},
			},
			wantErr: false,
		},
		{
			name: "should fetch ca from ConfigMap",
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
				}, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foobar",
						Namespace: "default",
					},
					Data: map[string]string{
						"cert": "1234",
					},
				}).Build(),
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						URL: serverURL,
						CAProvider: &esv1.CAProvider{
							Type: esv1.CAProviderTypeConfigMap,
							Name: "foobar",
							Key:  "cert",
						},
					},
					Auth: &esv1.KubernetesAuth{
						Token: &esv1.TokenAuth{
							BearerToken: v1.SecretKeySelector{
								Name:      "foobar",
								Namespace: pointer.To("shouldnotberelevant"),
								Key:       "token",
							},
						},
					},
				},
			},
			want: &want{
				Host:        serverURL,
				BearerToken: "mytoken",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte("1234"),
				},
			},
			wantErr: false,
		},
		{
			name: "should use system ca roots when no ca configured",
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
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						URL: serverURL,
					},
					Auth: &esv1.KubernetesAuth{
						Token: &esv1.TokenAuth{
							BearerToken: v1.SecretKeySelector{
								Name: "foobar",
								Key:  "token",
							},
						},
					},
				},
			},
			want: &want{
				Host:        serverURL,
				BearerToken: "mytoken",
				TLSClientConfig: rest.TLSClientConfig{
					Insecure: false,
					CAData:   nil,
				},
			},
			wantErr: false,
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
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						URL:      serverURL,
						CABundle: []byte(caCert),
					},
					Auth: &esv1.KubernetesAuth{
						Token: &esv1.TokenAuth{
							BearerToken: v1.SecretKeySelector{
								Name:      "foobar",
								Namespace: pointer.To("shouldnotberelevant"),
								Key:       "token",
							},
						},
					},
				},
			},
			want: &want{
				Host:        serverURL,
				BearerToken: "mytoken",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte(caCert),
				},
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
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						URL:      serverURL,
						CABundle: []byte(caCert),
					},
					Auth: &esv1.KubernetesAuth{
						Cert: &esv1.CertAuth{
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
			want: &want{
				Host: serverURL,
				TLSClientConfig: rest.TLSClientConfig{
					CAData:   []byte(caCert),
					CertData: []byte("my-cert"),
					KeyData:  []byte("my-key"),
				},
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
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						URL:      serverURL,
						CABundle: []byte(caCert),
					},
					Auth: &esv1.KubernetesAuth{
						ServiceAccount: &v1.ServiceAccountSelector{
							Name:      "my-sa",
							Namespace: pointer.To("shouldnotberelevant"),
						},
					},
				},
			},
			want: &want{
				Host:        serverURL,
				BearerToken: "my-sa-token",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte(caCert),
				},
			},
			wantErr: false,
		},
		{
			name: "should fail with missing URL",
			fields: fields{
				namespace: "default",
				kube: fclient.NewClientBuilder().WithObjects(&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-sa",
						Namespace: "default",
					},
				}).Build(),
				kubeclientset: utilfake.NewCreateTokenMock().WithToken("my-sa-token"),
				store: &esv1.KubernetesProvider{
					Server: esv1.KubernetesServer{
						CABundle: []byte(caCert),
					},
					Auth: &esv1.KubernetesAuth{
						ServiceAccount: &v1.ServiceAccountSelector{
							Name:      "my-sa",
							Namespace: pointer.To("shouldnotberelevant"),
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should read config from secret",
			fields: fields{
				namespace: "default",
				kube: fclient.NewClientBuilder().WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foobar",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"config": []byte(authTestKubeConfig),
					},
				}).Build(),
				store: &esv1.KubernetesProvider{
					AuthRef: &v1.SecretKeySelector{
						Name:      "foobar",
						Namespace: pointer.To("default"),
						Key:       "config",
					},
				},
			},
			want: &want{
				Host:        "https://api.my-domain.tld",
				BearerToken: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE3MTkzOTY4OTksImV4cCI6MTc1MDkzMjg4NywiYXVkIjoid3d3LmV4YW1wbGUuY29tIiwic3ViIjoianJvY2tldEBleGFtcGxlLmNvbSIsIkdpdmVuTmFtZSI6IkpvaG5ueSIsIlN1cm5hbWUiOiJSb2NrZXQiLCJFbWFpbCI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJSb2xlIjpbIk1hbmFnZXIiLCJQcm9qZWN0IEFkbWluaXN0cmF0b3IiXX0.xXrfIl0akhfjWU_BDl7Ad54SXje0YlJdnugzwh96VmM",
				TLSClientConfig: rest.TLSClientConfig{
					CAData: []byte(caCert),
				},
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
			cfg, err := k.getAuth(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("BaseClient.setAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, cfg)
		})
	}
}
