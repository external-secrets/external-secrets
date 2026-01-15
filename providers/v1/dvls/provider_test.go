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

package dvls

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testNamespace = "default"
	testServerURL = "https://dvls.example.com"
	secretName    = "dvls-secret"
	appIDKey      = "app-id"
	appSecretKey  = "app-secret"
	otherNS       = "other"
	testAppID     = "test-app-id"
	testAppSecret = "test-app-secret"
)

func TestProvider_ValidateStore(t *testing.T) {
	p := &Provider{}

	t.Run("case 1: should return no error when valid", func(t *testing.T) {
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: testServerURL,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appIDKey,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.NoError(t, err)
	})

	t.Run("case 1b: cluster store requires namespace", func(t *testing.T) {
		store := &esv1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{Kind: "ClusterSecretStore"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "dvls-cluster-store",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: testServerURL,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appIDKey,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}

		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cluster scope requires namespace")
	})

	t.Run("case 1c: cluster store succeeds with namespace", func(t *testing.T) {
		otherNamespace := otherNS
		store := &esv1.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{Kind: "ClusterSecretStore"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "dvls-cluster-store",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: testServerURL,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name:      secretName,
									Key:       appIDKey,
									Namespace: &otherNamespace,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name:      secretName,
									Key:       appSecretKey,
									Namespace: &otherNamespace,
								},
							},
						},
					},
				},
			},
		}

		_, err := p.ValidateStore(store)
		assert.NoError(t, err)
	})

	t.Run("case 2: should return error when provider is nil", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: nil,
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DVLS provider configuration is missing")
	})

	t.Run("case 2b: http URL without insecure flag should fail", func(t *testing.T) {
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: "http://dvls.example.com",
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appIDKey,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http URLs require 'insecure: true'")
	})

	t.Run("case 2c: http URL with insecure flag should pass", func(t *testing.T) {
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: "http://dvls.example.com",
						Insecure:  true,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appIDKey,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.NoError(t, err)
	})

	t.Run("case 3: should return error when serverUrl is empty", func(t *testing.T) {
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: "",
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appIDKey,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "serverUrl is required")
	})

	t.Run("case 3b: should return error when appId key is missing", func(t *testing.T) {
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: testServerURL,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  "",
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "appId secret key is required")
	})

	t.Run("case 3c: should return error when appSecret name is missing", func(t *testing.T) {
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: testServerURL,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appIDKey,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: "",
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "appSecret secret name is required")
	})

	t.Run("case 4: should return error when AppID secret reference is invalid", func(t *testing.T) {
		otherNamespace := otherNS
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: testServerURL,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name:      secretName,
									Key:       appIDKey,
									Namespace: &otherNamespace,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appSecretKey,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid appId")
	})

	t.Run("case 5: should return error when AppSecret secret reference is invalid", func(t *testing.T) {
		otherNamespace := otherNS
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dvls-store",
				Namespace: testNamespace,
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					DVLS: &esv1.DVLSProvider{
						ServerURL: testServerURL,
						Auth: esv1.DVLSAuth{
							SecretRef: esv1.DVLSAuthSecretRef{
								AppID: esmeta.SecretKeySelector{
									Name: secretName,
									Key:  appIDKey,
								},
								AppSecret: esmeta.SecretKeySelector{
									Name:      secretName,
									Key:       appSecretKey,
									Namespace: &otherNamespace,
								},
							},
						},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid appSecret")
	})
}

func TestProvider_Capabilities(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, esv1.SecretStoreReadWrite, p.Capabilities())
}

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	assert.NotNil(t, p)
	_, ok := p.(*Provider)
	assert.True(t, ok)
}

func TestProviderSpec(t *testing.T) {
	spec := ProviderSpec()
	assert.NotNil(t, spec)
	assert.NotNil(t, spec.DVLS)
}

func TestMaintenanceStatus(t *testing.T) {
	status := MaintenanceStatus()
	assert.Equal(t, esv1.MaintenanceStatusMaintained, status)
}
