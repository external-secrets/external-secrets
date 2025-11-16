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

package volcengine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	testNamespace       = "default"
	testAccessKeyID     = "test-access-key-id"
	testSecretAccessKey = "test-secret-access-key"
	testSessionToken    = "test-session-token"
	testRegion          = "cn-beijing"
	secretName          = "volcengine-secret"
	accessKeyIDKey      = "accessKeyID"
	secretAccessKeyKey  = "secretAccessKey"
	tokenKey            = "token"
	otherNamespace      = "other"
)

func TestProvider_NewClient(t *testing.T) {
	p := &Provider{}
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-store",
			Namespace: testNamespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Volcengine: &esv1.VolcengineProvider{
					Region: testRegion,
					Auth: &esv1.VolcengineAuth{
						SecretRef: &esv1.VolcengineAuthSecretRef{
							AccessKeyID: esmeta.SecretKeySelector{
								Name: secretName,
								Key:  accessKeyIDKey,
							},
							SecretAccessKey: esmeta.SecretKeySelector{
								Name: secretName,
								Key:  secretAccessKeyKey,
							},
						},
					},
				},
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			accessKeyIDKey:     []byte("test-access-key"),
			secretAccessKeyKey: []byte("test-secret-key"),
		},
	}

	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	t.Run("case 1: Successful case", func(t *testing.T) {
		c, err := p.NewClient(context.Background(), store, kube, testNamespace)
		assert.NoError(t, err)
		assert.NotNil(t, c)
		_, ok := c.(*Client)
		assert.True(t, ok)
	})

	t.Run("case 2: Volcengine provider is nil", func(t *testing.T) {
		store.Spec.Provider.Volcengine = nil
		c, err := p.NewClient(context.Background(), store, kube, testNamespace)
		assert.Error(t, err)
		assert.Nil(t, c)
		assert.EqualError(t, err, "volcengine provider is nil")
	})
}

func TestProvider_ValidateStore(t *testing.T) {
	p := &Provider{}

	t.Run("case 1: should return no error when using IRSA", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: &esv1.VolcengineProvider{
						Region: testRegion,
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.NoError(t, err)
	})

	t.Run("case 2: should return no error when using SecretRef", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: &esv1.VolcengineProvider{
						Region: testRegion,
						Auth: &esv1.VolcengineAuth{
							SecretRef: &esv1.VolcengineAuthSecretRef{
								AccessKeyID: esmeta.SecretKeySelector{
									Name: "test",
									Key:  accessKeyIDKey,
								},
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: "test",
									Key:  secretAccessKeyKey,
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

	t.Run("case 3: should return error when provider is nil", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: nil,
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.EqualError(t, err, "volcengine provider is nil")
	})

	t.Run("case 4: should return error when region is empty", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: &esv1.VolcengineProvider{},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.EqualError(t, err, "region is required")
	})

	t.Run("case 5: should return error when SecretRef is nil", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: &esv1.VolcengineProvider{
						Region: testRegion,
						Auth:   &esv1.VolcengineAuth{},
					},
				},
			},
		}
		_, err := p.ValidateStore(store)
		assert.Error(t, err)
		assert.EqualError(t, err, "SecretRef is required when using static credentials")
	})

	t.Run("case 6: should return error when AccessKeyID is invalid", func(t *testing.T) {
		otherNamespace := otherNamespace
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "volcengile-store",
				Namespace: "kms-system",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: &esv1.VolcengineProvider{
						Region: testRegion,
						Auth: &esv1.VolcengineAuth{
							SecretRef: &esv1.VolcengineAuthSecretRef{
								AccessKeyID: esmeta.SecretKeySelector{
									Name:      "",
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
		assert.Contains(t, err.Error(), "invalid AccessKeyID")
	})

	t.Run("case 7: should return error when SecretAccessKey is invalid", func(t *testing.T) {
		otherNamespace := otherNamespace
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "volcengile-store",
				Namespace: "kms-system",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: &esv1.VolcengineProvider{
						Region: testRegion,
						Auth: &esv1.VolcengineAuth{
							SecretRef: &esv1.VolcengineAuthSecretRef{
								AccessKeyID: esmeta.SecretKeySelector{
									Name: "test",
									Key:  accessKeyIDKey,
								},
								SecretAccessKey: esmeta.SecretKeySelector{
									Name:      "",
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
		assert.Contains(t, err.Error(), "invalid SecretAccessKey")
	})

	t.Run("case 8: should return error when Token is invalid", func(t *testing.T) {
		otherNamespace := otherNamespace
		store := &esv1.SecretStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "volcengile-store",
				Namespace: "kms-system",
			},
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					Volcengine: &esv1.VolcengineProvider{
						Region: testRegion,
						Auth: &esv1.VolcengineAuth{
							SecretRef: &esv1.VolcengineAuthSecretRef{
								AccessKeyID: esmeta.SecretKeySelector{
									Name: "test",
									Key:  accessKeyIDKey,
								},
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: "test",
									Key:  secretAccessKeyKey,
								},
								Token: &esmeta.SecretKeySelector{
									Name:      "",
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
		assert.Contains(t, err.Error(), "invalid Token")
	})
}

func TestProvider_Capabilities(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, esv1.SecretStoreReadOnly, p.Capabilities())
}
