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
package resolvers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestResolveSecretKeyRef(t *testing.T) {
	ctx := context.TODO()
	c := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	testNamespace := "test-namespace"
	testSecret := "test-secret"
	testKey := "test-key"
	testValue := "test-value"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testSecret,
		},
		Data: map[string][]byte{
			testKey: []byte(testValue),
		},
	}
	err := c.Create(ctx, secret)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		namespace string
		storeKind string
		selector  *esmeta.SecretKeySelector
		expected  string
		err       error
	}{
		{
			name:      "namespaced secret store can access secret in same namespace",
			namespace: testNamespace,
			storeKind: "SecretStore",
			selector: &esmeta.SecretKeySelector{
				Name:      testSecret,
				Namespace: ptr.To(testNamespace),
				Key:       testKey,
			},
			expected: testValue,
			err:      nil,
		},
		{
			name:      "omitting namespace in secret store defaults to same namespace",
			namespace: testNamespace,
			storeKind: "SecretStore",
			selector: &esmeta.SecretKeySelector{
				Name: testSecret,
				Key:  testKey,
			},
			expected: testValue,
			err:      nil,
		},
		{
			name:      "namespaced secret store can not access secret in different namespace",
			namespace: "other-namespace",
			storeKind: "SecretStore",
			selector: &esmeta.SecretKeySelector{
				Name:      testSecret,
				Namespace: ptr.To(testNamespace),
				Key:       testKey,
			},
			err: errors.New(`cannot get Kubernetes secret "test-secret": secrets "test-secret" not found`),
		},
		{
			name:      "cluster secret store may access all namespaces",
			storeKind: "ClusterSecretStore",
			selector: &esmeta.SecretKeySelector{
				Name:      testSecret,
				Namespace: ptr.To(testNamespace),
				Key:       testKey,
			},
			expected: testValue,
			err:      nil,
		},
		{
			name:      "key not found in secret",
			namespace: testNamespace,
			storeKind: "SecretStore",
			selector: &esmeta.SecretKeySelector{
				Name:      testSecret,
				Namespace: ptr.To(testNamespace),
				Key:       "xxxxxxxx",
			},
			expected: "",
			err:      errors.New(`cannot find secret data for key: "xxxxxxxx"`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolvedValue, err := SecretKeyRef(ctx, c, tc.storeKind, tc.namespace, tc.selector)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expected, resolvedValue)
		})
	}
}
