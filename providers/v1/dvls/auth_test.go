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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestGetSecretValue_Success(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			appIDKey:     []byte(testAppID),
			appSecretKey: []byte(testAppSecret),
		},
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	selector := esmeta.SecretKeySelector{
		Name: secretName,
		Key:  appIDKey,
	}

	value, err := getSecretValue(context.Background(), kube, "SecretStore", testNamespace, selector)

	assert.NoError(t, err)
	assert.Equal(t, testAppID, string(value))
}

func TestGetSecretValue_SecretNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).Build()

	selector := esmeta.SecretKeySelector{
		Name: "non-existent-secret",
		Key:  appIDKey,
	}

	_, err := getSecretValue(context.Background(), kube, "SecretStore", testNamespace, selector)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret")
}

func TestGetSecretValue_KeyNotFound(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			appIDKey: []byte(testAppID),
		},
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	selector := esmeta.SecretKeySelector{
		Name: secretName,
		Key:  "non-existent-key",
	}

	_, err := getSecretValue(context.Background(), kube, "SecretStore", testNamespace, selector)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key \"non-existent-key\" not found in secret")
}

func TestGetSecretValue_WithSelectorNamespace(t *testing.T) {
	otherNamespace := "other-namespace"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: otherNamespace,
		},
		Data: map[string][]byte{
			appSecretKey: []byte(testAppSecret),
		},
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	selector := esmeta.SecretKeySelector{
		Name:      secretName,
		Key:       appSecretKey,
		Namespace: &otherNamespace,
	}

	value, err := getSecretValue(context.Background(), kube, "ClusterSecretStore", testNamespace, selector)

	assert.NoError(t, err)
	assert.Equal(t, testAppSecret, string(value))
}
