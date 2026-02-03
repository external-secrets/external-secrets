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

package ovh

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeBuilder "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/ovh/fake"
)

var (
	namespace = "namespace"
	scheme    = runtime.NewScheme()
	_         = corev1.AddToScheme(scheme)
	kube      = fakeBuilder.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"key": []byte("value"),
			},
		}).Build()
	okmsId          = "11111111-1111-1111-1111-111111111111"
	validTokenAuth  = "Valid token auth"
	validClientCert = "Valid mtls client certificate"
	validClientKey  = "Valid mtls client key"
	fillingStr      = "string"
)

func TestNewClient(t *testing.T) {
	tests := map[string]struct {
		errshould string
		kube      kclient.Client
		store     *esv1.SecretStore
	}{
		"Nil store": {
			errshould: "store is nil",
			kube:      kube,
		},
		"Nil provider": {
			errshould: "store provider is nil",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: nil,
				},
			},
		},
		"Nil ovh provider": {
			errshould: "ovh store provider is nil",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
		},
		"Nil controller-runtime client": {
			errshould: "failed to create new ovh provider client: controller-runtime client is nil",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Auth: esv1.OvhAuth{
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      validTokenAuth,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
							Server: fillingStr,
							OkmsID: okmsId,
						},
					},
				},
			},
		},
		"Authentication method conflict": {
			errshould: "only one authentication method allowed (mtls | token)",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Authentication method empty": {
			errshould: "missing authentication method",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth:   esv1.OvhAuth{},
						},
					},
				},
			},
		},
		"Valid token auth": {
			errshould: "",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      validTokenAuth,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty token auth": {
			errshould: "failed to create new ovh provider client: ovh store auth.token.tokenSecretRef cannot be empty",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{},
								},
							},
						},
					},
				},
			},
		},
		"Valid mtls auth": {
			errshould: "",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      validClientCert,
										Namespace: &namespace,
										Key:       fillingStr,
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      validClientKey,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty mtls client certificate": {
			errshould: "missing tls certificate or key",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientKey: &esmeta.SecretKeySelector{
										Name:      validClientKey,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty mtls client key": {
			errshould: "missing tls certificate or key",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      validClientCert,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ctx := context.Background()
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			provider := Provider{
				secretKeyResolver: &fake.FakeSecretKeyResolver{},
			}
			_, err := provider.NewClient(ctx, testCase.store, testCase.kube, "namespace")
			if testCase.errshould != "" {
				if err == nil {
					t.Errorf("\nexpected error: %s\nactual error:   <nil>\n\n", testCase.errshould)
				} else if err.Error() != testCase.errshould {
					t.Errorf("\nexpected error: %s\nactual error:   %v\n\n", testCase.errshould, err)
				}
			}
		})
	}
}

func TestValidateStore(t *testing.T) {
	var namespace string = "namespace"
	tests := map[string]struct {
		errshould string
		kube      kclient.Client
		store     *esv1.SecretStore
	}{
		"Nil store": {
			errshould: "store provider is nil",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: nil,
				},
			},
		},
		"Nil ovh provider": {
			errshould: "ovh store provider is nil",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
		},
		"Authentication method conflict": {
			errshould: "only one authentication method allowed (mtls | token)",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Valid token auth": {
			errshould: "",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Valid mtls auth": {
			errshould: "",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Invalid mtls auth: missing client certificate": {
			errshould: "missing tls certificate or key",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientKey: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Invalid mtls auth: missing key certificate": {
			errshould: "missing tls certificate or key",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      fillingStr,
										Namespace: &namespace,
										Key:       fillingStr,
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty auth": {
			errshould: "missing authentication method",
			kube:      kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: fillingStr,
							OkmsID: okmsId,
						},
					},
				},
			},
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			provider := Provider{}
			_, err := provider.ValidateStore(testCase.store)
			if testCase.errshould != "" {
				if err == nil {
					t.Errorf("\nexpected error: %s\nactual error:   <nil>\n\n", testCase.errshould)
				} else if err.Error() != testCase.errshould {
					t.Errorf("\nexpected error: %s\nactual error:   %v\n\n", testCase.errshould, err)
				}
			}
		})
	}
}
