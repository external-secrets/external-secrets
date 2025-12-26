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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestNewDVLSClient_CrossNamespaceSecurityConstraint(t *testing.T) {
	otherNamespace := "other-namespace"

	tests := []struct {
		name        string
		storeKind   string
		namespace   string
		secretNS    *string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "ClusterSecretStore can access cross-namespace secrets",
			storeKind:   esv1.ClusterSecretStoreKind,
			namespace:   testNamespace,
			secretNS:    &otherNamespace,
			expectError: false,
		},
		{
			name:        "SecretStore cannot access cross-namespace secrets",
			storeKind:   esv1.SecretStoreKind,
			namespace:   testNamespace,
			secretNS:    &otherNamespace,
			expectError: true,
			errorMsg:    "cannot get Kubernetes secret",
		},
		{
			name:        "SecretStore can access same-namespace secrets",
			storeKind:   esv1.SecretStoreKind,
			namespace:   testNamespace,
			secretNS:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetNS := testNamespace
			if tt.secretNS != nil {
				targetNS = *tt.secretNS
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: targetNS,
				},
				Data: map[string][]byte{
					appIDKey:     []byte(testAppID),
					appSecretKey: []byte(testAppSecret),
				},
			}

			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

			provider := &esv1.DVLSProvider{
				ServerURL: testServerURL,
				Auth: esv1.DVLSAuth{
					SecretRef: esv1.DVLSAuthSecretRef{
						AppID: esmeta.SecretKeySelector{
							Name:      secretName,
							Key:       appIDKey,
							Namespace: tt.secretNS,
						},
						AppSecret: esmeta.SecretKeySelector{
							Name:      secretName,
							Key:       appSecretKey,
							Namespace: tt.secretNS,
						},
					},
				},
			}

			client, err := NewDVLSClient(context.Background(), kube, tt.storeKind, tt.namespace, provider)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, client)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				// Verify Kubernetes secret access succeeded (DVLS connection will fail due to fake server).
				assert.NotContains(t, err.Error(), "failed to get appId")
				assert.NotContains(t, err.Error(), "failed to get appSecret")
				assert.NotContains(t, err.Error(), "cannot get Kubernetes secret")
			}
		})
	}
}
