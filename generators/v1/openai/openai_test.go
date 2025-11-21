// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func storeFakeSecret(kube client.Client, namespace, name, key, value string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			key: []byte(value),
		},
	}
	return kube.Create(context.Background(), secret)
}

func TestOpenAiGenerator_GenerateAndCleanup(t *testing.T) {
	mockProjectID := "test-project"

	// Simulate OpenAI service account create response
	serviceAccountResponse := genv1alpha1.OpenAiServiceAccount{
		ID:   "svc_test_123",
		Name: "mock-service-account",
		APIKey: genv1alpha1.OpenAiAPIKey{
			Value: "sk-test123",
		},
	}

	// Mock OpenAI Admin API Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(serviceAccountResponse)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create fake Kubernetes client with mocked secret for admin API key
	fakeKube := fake.NewClientBuilder().Build()

	// Store fake admin API key in fake secrets backend
	adminKey := "fake-admin-key"
	err := storeFakeSecret(fakeKube, "default", "openai-admin-key", "api-key", adminKey)
	require.NoError(t, err)

	// Prepare generator spec
	spec := genv1alpha1.OpenAI{
		Spec: genv1alpha1.OpenAISpec{
			Host:      mockServer.URL, // override to mock server
			ProjectID: mockProjectID,
			OpenAiAdminKey: esmeta.SecretKeySelector{
				Name: "openai-admin-key",
				Key:  "api-key",
			},
		},
	}

	specRaw, _ := json.Marshal(spec)

	// Initialize generator
	gen := &Generator{}

	// Call Generate()
	secrets, state, err := gen.Generate(context.Background(), &apiextensions.JSON{Raw: specRaw}, fakeKube, "default")
	require.NoError(t, err)
	require.NotNil(t, secrets)
	require.NotEmpty(t, state)

	assert.Contains(t, secrets, "api_key")
	assert.Equal(t, "sk-test123", string(secrets["api_key"]))

	// Call Cleanup()
	err = gen.Cleanup(context.Background(), &apiextensions.JSON{Raw: specRaw}, state, fakeKube, "default")
	require.NoError(t, err)
}

func TestOpenAiGenerator_LastActivityTime(t *testing.T) {
	mockProjectID := "test-project"

	// Simulate OpenAI api key retrieve response
	apikeyResponse := genv1alpha1.OpenAiAPIKey{
		ID:         "api-key-123",
		Value:      "sk-test123",
		Name:       "mock-service-account",
		CreatedAt:  123456789,
		LastUsedAt: 0,
	}

	// Mock OpenAI Admin API Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(apikeyResponse)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create fake Kubernetes client with mocked secret for admin API key
	fakeKube := fake.NewClientBuilder().Build()

	// Store fake admin API key in fake secrets backend
	adminKey := "fake-admin-key"
	err := storeFakeSecret(fakeKube, "default", "openai-admin-key", "api-key", adminKey)
	require.NoError(t, err)

	// Prepare generator spec
	spec := genv1alpha1.OpenAI{
		Spec: genv1alpha1.OpenAISpec{
			Host:      mockServer.URL, // override to mock server
			ProjectID: mockProjectID,
			OpenAiAdminKey: esmeta.SecretKeySelector{
				Name: "openai-admin-key",
				Key:  "api-key",
			},
		},
	}

	specRaw, _ := json.Marshal(spec)

	// Initialize generator
	gen := &Generator{}

	// Call LastActivityTime
	rawState, err := json.Marshal(&genv1alpha1.OpenAiServiceAccountState{
		ServiceAccountID: apikeyResponse.Name,
		APIKeyID:         apikeyResponse.ID,
	})
	require.NoError(t, err)
	state := &apiextensions.JSON{Raw: rawState}

	lastActivity, found, err := gen.LastActivityTime(context.Background(), &apiextensions.JSON{Raw: specRaw}, state, fakeKube, "default")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, time.Unix(0, 0), lastActivity)
}
