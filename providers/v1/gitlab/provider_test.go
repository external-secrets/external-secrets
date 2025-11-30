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

// /*
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

package gitlab

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestGetClientWithCABundle(t *testing.T) {
	// Create a mock TLS server that asserts a client certificate is present
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We expect a GET request to the variables API
		assert.Equal(t, "/api/v4/projects/1234/variables/test-secret", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Define the GitLab provider with the CABundle
	provider := &esv1.GitlabProvider{
		URL:       server.URL,
		ProjectID: "1234",
		CABundle:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}),
		Auth: esv1.GitlabAuth{
			SecretRef: esv1.GitlabSecretRef{
				AccessToken: esmeta.SecretKeySelector{
					Name: "gitlab-secret",
					Key:  "token",
				},
			},
		},
	}

	// Create a fake Kubernetes client with the required secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitlab-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("test-token"),
		},
	}
	fakeClient := clientfake.NewClientBuilder().WithObjects(secret).Build()

	// Create the gitlabBase struct
	gl := &gitlabBase{
		kube:      fakeClient,
		store:     provider,
		namespace: "default",
	}

	// We need to initialize the gitlab clients inside our gitlabBase struct
	client, err := gl.getClient(context.Background(), provider)
	assert.NoError(t, err)
	gl.projectsClient = client.Projects
	gl.projectVariablesClient = client.ProjectVariables
	gl.groupVariablesClient = client.GroupVariables

	// Call getVariables to trigger a network request to the mock server.
	// The request will only succeed if the custom CA is correctly configured.
	_, err = gl.getVariables(esv1.ExternalSecretDataRemoteRef{Key: "test-secret"}, nil)
	assert.NoError(t, err, "getVariables should succeed with the correct CA")
}

func TestGetClientWithInvalidCABundle(t *testing.T) {
	provider := &esv1.GitlabProvider{
		CABundle: []byte("invalid-ca-bundle"),
		Auth: esv1.GitlabAuth{
			SecretRef: esv1.GitlabSecretRef{
				AccessToken: esmeta.SecretKeySelector{
					Name: "gitlab-secret",
					Key:  "token",
				},
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitlab-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("test-token"),
		},
	}
	fakeClient := clientfake.NewClientBuilder().WithObjects(secret).Build()

	gl := &gitlabBase{
		kube:      fakeClient,
		store:     provider,
		namespace: "default",
	}

	_, err := gl.getClient(context.Background(), provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read ca bundle")
}

func TestGetClientWithCAProviderSecret(t *testing.T) {
	// Create a mock TLS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	caSecret := makeFakeCASource(t, "Secret", "ca-secret", "default", certPEM)

	// Define the GitLab provider with the CAProvider
	provider := &esv1.GitlabProvider{
		URL:       server.URL,
		ProjectID: "1234",
		Auth: esv1.GitlabAuth{
			SecretRef: esv1.GitlabSecretRef{
				AccessToken: esmeta.SecretKeySelector{
					Name: "gitlab-secret",
					Key:  "token",
				},
			},
		},
		CAProvider: &esv1.CAProvider{
			Type: esv1.CAProviderTypeSecret,
			Name: "ca-secret",
			Key:  "tls.crt",
		},
	}

	// Create a fake Kubernetes client with the required secrets
	accessTokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitlab-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("test-token"),
		},
	}
	fakeClient := clientfake.NewClientBuilder().WithObjects(accessTokenSecret, caSecret).Build()

	// Create the gitlabBase struct
	gl := &gitlabBase{
		kube:      fakeClient,
		store:     provider,
		namespace: "default",
	}

	// We need to initialize the gitlab clients inside our gitlabBase struct
	client, err := gl.getClient(context.Background(), provider)
	assert.NoError(t, err)
	gl.projectsClient = client.Projects
	gl.projectVariablesClient = client.ProjectVariables
	gl.groupVariablesClient = client.GroupVariables

	// Call getVariables to trigger a network request to the mock server.
	_, err = gl.getVariables(esv1.ExternalSecretDataRemoteRef{Key: "test-secret"}, nil)
	assert.NoError(t, err, "getVariables should succeed with the correct CA from Secret")
}

func TestGetClientWithCAProviderConfigMap(t *testing.T) {
	// Create a mock TLS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	caCM := makeFakeCASource(t, "ConfigMap", "ca-cm", "default", certPEM)

	// Define the GitLab provider with the CAProvider
	provider := &esv1.GitlabProvider{
		URL:       server.URL,
		ProjectID: "1234",
		Auth: esv1.GitlabAuth{
			SecretRef: esv1.GitlabSecretRef{
				AccessToken: esmeta.SecretKeySelector{
					Name: "gitlab-secret",
					Key:  "token",
				},
			},
		},
		CAProvider: &esv1.CAProvider{
			Type: esv1.CAProviderTypeConfigMap,
			Name: "ca-cm",
			Key:  "ca.crt",
		},
	}

	// Create a fake Kubernetes client with the required secrets
	accessTokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitlab-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("test-token"),
		},
	}
	fakeClient := clientfake.NewClientBuilder().WithObjects(accessTokenSecret, caCM).Build()

	// Create the gitlabBase struct
	gl := &gitlabBase{
		kube:      fakeClient,
		store:     provider,
		namespace: "default",
	}

	// We need to initialize the gitlab clients inside our gitlabBase struct
	client, err := gl.getClient(context.Background(), provider)
	assert.NoError(t, err)
	gl.projectsClient = client.Projects
	gl.projectVariablesClient = client.ProjectVariables
	gl.groupVariablesClient = client.GroupVariables

	// Call getVariables to trigger a network request to the mock server.
	_, err = gl.getVariables(esv1.ExternalSecretDataRemoteRef{Key: "test-secret"}, nil)
	assert.NoError(t, err, "getVariables should succeed with the correct CA from ConfigMap")
}

func makeFakeCASource(t *testing.T, kind, name, namespace string, certData []byte) kclient.Object {
	t.Helper()
	switch kind {
	case "Secret":
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"tls.crt": certData,
			},
		}
	case "ConfigMap":
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string]string{
				"ca.crt": string(certData),
			},
		}
	}
	return nil
}
