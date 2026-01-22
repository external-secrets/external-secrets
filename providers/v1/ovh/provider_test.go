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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeBuilder "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type EphemeralMTLS struct {
	Once    sync.Once
	keyPEM  string
	certPEM string
}

var (
	ephemeralMTLS = EphemeralMTLS{}
	namespace     = "namespace"
	scheme        = runtime.NewScheme()
	_             = corev1.AddToScheme(scheme)
	kube          = fakeBuilder.NewClientBuilder().
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
)

func (eph *EphemeralMTLS) SecretKeyRef(_ context.Context, _ kclient.Client, _, _ string, ref *esmeta.SecretKeySelector) (string, error) {
	if ref.Name == "Valid token auth" {
		return "Valid", nil
	}
	if ref.Name == "Valid mtls client certificate" || ref.Name == "Valid mtls client key" {
		var err error
		eph.Once.Do(func() {
			var privKey *rsa.PrivateKey
			privKey, err = rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return
			}
			eph.keyPEM = string(pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privKey),
			}))

			template := x509.Certificate{}
			var cert []byte
			cert, err = x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
			if err != nil {
				return
			}
			eph.certPEM = string(pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert,
			}))
		})

		if err != nil {
			return "", err
		}

		if ref.Name == "Valid mtls client certificate" {
			return eph.certPEM, nil
		}
		return eph.keyPEM, nil
	}
	return "", nil
}

func TestNewClient(t *testing.T) {
	tests := map[string]struct {
		should string
		kube   kclient.Client
		err    bool
		store  *esv1.SecretStore
	}{
		"Nil store": {
			should: "store is nil",
			err:    true,
			kube:   kube,
		},
		"Nil provider": {
			should: "store provider is nil",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: nil,
				},
			},
		},
		"Nil ovh provider": {
			should: "ovh store provider is nil",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
		},
		"Nil controller-runtime client": {
			should: "controller-runtime client is nil",
			err:    true,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Auth: esv1.OvhAuth{
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      "Valid token auth",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
							Server: "server",
							OkmsID: "okmsID",
						},
					},
				},
			},
		},
		"Authentication method conflict": {
			should: "only one authentication method allowed (mtls | token)",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Authentication method empty": {
			should: "missing authentication method",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth:   esv1.OvhAuth{},
						},
					},
				},
			},
		},
		"Valid token auth": {
			should: "",
			err:    false,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      "Valid token auth",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty token auth": {
			should: "ovh store auth.token.tokenSecretRef cannot be empty",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
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
			should: "",
			err:    false,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      "Valid mtls client certificate",
										Namespace: &namespace,
										Key:       "string",
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      "Valid mtls client key",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty mtls client certificate": {
			should: "missing tls certificate or key",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientKey: &esmeta.SecretKeySelector{
										Name:      "Valid mtls client key",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty mtls client key": {
			should: "missing tls certificate or key",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      "Valid mtls client certificate",
										Namespace: &namespace,
										Key:       "string",
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
				SecretKeyRef: ephemeralMTLS.SecretKeyRef,
			}
			_, err := provider.NewClient(ctx, testCase.store, testCase.kube, "namespace")
			if testCase.err == true {
				if err == nil {
					t.Error()
				} else if err.Error() != testCase.should {
					t.Error()
				}
			} else if err != nil {
				t.Error()
			}
		})
	}
}

func TestValidateStore(t *testing.T) {
	var namespace string = "namespace"
	tests := map[string]struct {
		should string
		err    bool
		kube   kclient.Client
		store  *esv1.SecretStore
	}{
		"Nil store": {
			should: "store provider is nil",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: nil,
				},
			},
		},
		"Nil ovh provider": {
			should: "ovh store provider is nil",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						AWS: &esv1.AWSProvider{},
					},
				},
			},
		},
		"Authentication method conflict": {
			should: "only one authentication method allowed (mtls | token)",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Valid token auth": {
			should: "",
			err:    false,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientToken: &esv1.OvhClientToken{
									ClientTokenSecret: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Valid mtls auth": {
			should: "",
			err:    false,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
									ClientKey: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Invalid mtls auth: missing client certificate": {
			should: "missing tls certificate or key",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientKey: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Invalid mtls auth: missing key certificate": {
			should: "missing tls certificate or key",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
							Auth: esv1.OvhAuth{
								ClientMTLS: &esv1.OvhClientMTLS{
									ClientCertificate: &esmeta.SecretKeySelector{
										Name:      "string",
										Namespace: &namespace,
										Key:       "string",
									},
								},
							},
						},
					},
				},
			},
		},
		"Empty auth": {
			should: "missing authentication method",
			err:    true,
			kube:   kube,
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ovh: &esv1.OvhProvider{
							Server: "string",
							OkmsID: "11111111-1111-1111-1111-111111111111",
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
			if testCase.err == true {
				if err == nil {
					t.Error()
				} else if err.Error() != testCase.should {
					t.Error()
				}
			} else if err != nil {
				t.Error()
			}
		})
	}
}
