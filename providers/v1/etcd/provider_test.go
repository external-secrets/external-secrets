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

package etcd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestCapabilities(t *testing.T) {
	p := &Provider{}
	caps := p.Capabilities()
	assert.Equal(t, esv1.SecretStoreReadWrite, caps, "etcd provider should support ReadWrite")
}

func TestValidateStore(t *testing.T) {
	testCases := []struct {
		name        string
		store       esv1.GenericStore
		expectedErr string
	}{
		{
			name: "valid store with endpoints",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{"https://etcd:2379"},
						},
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "missing endpoints",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{},
						},
					},
				},
			},
			expectedErr: errMissingEndpoints,
		},
		{
			name: "missing etcd spec",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{},
				},
			},
			expectedErr: errMissingEtcdSpec,
		},
		{
			name: "valid store with username/password auth",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{"https://etcd:2379"},
							Auth: &esv1.EtcdAuth{
								SecretRef: &esv1.EtcdAuthSecretRef{
									Username: esmeta.SecretKeySelector{
										Name: "etcd-creds",
										Key:  "username",
									},
									Password: esmeta.SecretKeySelector{
										Name: "etcd-creds",
										Key:  "password",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "missing username in secretRef",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{"https://etcd:2379"},
							Auth: &esv1.EtcdAuth{
								SecretRef: &esv1.EtcdAuthSecretRef{
									Password: esmeta.SecretKeySelector{
										Name: "etcd-creds",
										Key:  "password",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: errMissingUsername,
		},
		{
			name: "missing password in secretRef",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{"https://etcd:2379"},
							Auth: &esv1.EtcdAuth{
								SecretRef: &esv1.EtcdAuthSecretRef{
									Username: esmeta.SecretKeySelector{
										Name: "etcd-creds",
										Key:  "username",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: errMissingPassword,
		},
		{
			name: "valid store with TLS auth",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{"https://etcd:2379"},
							Auth: &esv1.EtcdAuth{
								TLS: &esv1.EtcdTLSAuth{
									ClientCert: esmeta.SecretKeySelector{
										Name: "etcd-tls",
										Key:  "cert",
									},
									ClientKey: esmeta.SecretKeySelector{
										Name: "etcd-tls",
										Key:  "key",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "missing client cert in TLS auth",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{"https://etcd:2379"},
							Auth: &esv1.EtcdAuth{
								TLS: &esv1.EtcdTLSAuth{
									ClientKey: esmeta.SecretKeySelector{
										Name: "etcd-tls",
										Key:  "key",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: errMissingClientCert,
		},
		{
			name: "missing client key in TLS auth",
			store: &esv1.SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-store",
					Namespace: "default",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Etcd: &esv1.EtcdProvider{
							Endpoints: []string{"https://etcd:2379"},
							Auth: &esv1.EtcdAuth{
								TLS: &esv1.EtcdTLSAuth{
									ClientCert: esmeta.SecretKeySelector{
										Name: "etcd-tls",
										Key:  "cert",
									},
								},
							},
						},
					},
				},
			},
			expectedErr: errMissingClientKey,
		},
	}

	p := &Provider{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.ValidateStore(tc.store)
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	assert.NotNil(t, p)
	assert.IsType(t, &Provider{}, p)
}

func TestProviderSpec(t *testing.T) {
	spec := ProviderSpec()
	assert.NotNil(t, spec)
	assert.NotNil(t, spec.Etcd)
}

func TestMaintenanceStatus(t *testing.T) {
	status := MaintenanceStatus()
	assert.Equal(t, esv1.MaintenanceStatusMaintained, status)
}
