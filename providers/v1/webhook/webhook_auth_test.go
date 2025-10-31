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
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package webhook

import (
	"context"
	b64 "encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type mockAuthTestPackage struct {
	Creds      mockCreds
	MockServer mockAuthTestServer
	Request    mockAuthRequest
	Expect     string
}

type mockCreds struct {
	UserName string
	Password string
}

type mockAuthTestServer func(
	serverCreds mockCreds,
	t *testing.T) *httptest.Server

type mockAuthRequest func(
	url string,
	creds mockCreds,
	t *testing.T) string

func TestWebhookAuth(t *testing.T) {
	// define test cases
	creds := mockCreds{"correctuser123", "correctpassword123"}
	basicAuthExpect := "Basic " + b64.StdEncoding.EncodeToString([]byte(creds.UserName+":"+creds.Password))
	ntlmExpect := "NTLM TlRMTVNTUAABAAAAAQCIoAAAAAAoAAAAAAAAACgAAAAGAbEdAAAADw=="
	negotiateExpect := "Negotiate TlRMTVNTUAABAAAAAQCIoAAAAAAoAAAAAAAAACgAAAAGAbEdAAAADw=="

	// due to integrated nature of GetSecret(), we use a mock server
	// to return relevant parts of a request, in this case, the auth header.
	testAuthHeaders := map[string]mockAuthTestPackage{
		"BasicAuth": {creds, basicAuthRequestEcho, basicAuthRequest, basicAuthExpect},
		"NTLM":      {creds, ntlmRequestEcho, ntlmRequest, ntlmExpect},
		"Negotiate": {creds, negotiateRequestEcho, ntlmRequest, negotiateExpect},
	}

	// execute test cases
	for _, p := range testAuthHeaders {
		server := p.MockServer(p.Creds, t)
		result := p.Request(server.URL, creds, t)
		server.Close()
		expect := p.Expect
		if result != expect {
			t.Errorf("Test failed. Result: '%s' / Expected:  '%s'", result, expect)
		}
	}
}

func ntlmRequestEcho(mockCreds, *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqAuthString := r.Header.Get("Authorization")
		if reqAuthString == "" {
			// go-ntlmssp first sends anonymous request, respond with 401
			w.Header().Add("WWW-Authenticate", "NTLM")
			w.WriteHeader(401)
		} else {
			w.Write([]byte(reqAuthString))
		}
	}))
	return server
}

func negotiateRequestEcho(mockCreds, *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqAuthString := r.Header.Get("Authorization")
		if reqAuthString == "" {
			// go-ntlmssp first sends anonymous request, respond with 401
			w.Header().Add("WWW-Authenticate", "Negotiate")
			w.WriteHeader(401)
		} else {
			w.Write([]byte(reqAuthString))
		}
	}))
	return server
}

func ntlmRequest(url string, creds mockCreds, t *testing.T) string {
	secretName := "ntlmTestAuthSecret"
	secretNamespace := "default"

	// ntlm clustersecretstore takes credentials from a secret,
	// so we need to mock k8s-client retrieval of secret.
	mockClient := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
			Labels: map[string]string{
				"external-secrets.io/type": "webhook",
			},
		},
		Data: map[string][]byte{
			"userName": []byte(creds.UserName),
			"password": []byte(creds.Password),
		},
	}).Build()

	// create clusteSecretStore
	ntlmAuthStore := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-store",
			Namespace: secretNamespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Webhook: &esv1.WebhookProvider{
					URL: url,
					Auth: &esv1.AuthorizationProtocol{
						NTLM: &esv1.NTLMProtocol{
							UserName: esmeta.SecretKeySelector{
								Name:      secretName,
								Namespace: &secretNamespace,
								Key:       "userName",
							},
							Password: esmeta.SecretKeySelector{
								Name:      secretName,
								Namespace: &secretNamespace,
								Key:       "password",
							},
						},
					},
				},
			},
		},
	}

	result := exerciseGetSecret(ntlmAuthStore, mockClient, t)
	return result
}

func basicAuthRequestEcho(mockCreds, *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqAuthString := r.Header.Get("Authorization")
		if reqAuthString == "" {
			w.Write([]byte("Empty Authorization header"))
		} else {
			w.Write([]byte(reqAuthString))
		}
	}))

	return server
}

func basicAuthRequest(url string, creds mockCreds, t *testing.T) string {
	reqAuthString := "Basic " + b64.StdEncoding.EncodeToString([]byte(creds.UserName+":"+creds.Password))

	// create ClusterSecretStore
	basicAuthStore := &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterSecretStore",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-store",
			Namespace: "default",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Webhook: &esv1.WebhookProvider{
					URL: url,
					Headers: map[string]string{
						"Authorization": reqAuthString,
					},
				},
			},
		},
	}
	result := exerciseGetSecret(basicAuthStore, nil, t)
	return result
}

func exerciseGetSecret(mockStore esv1.GenericStore, mockKubeClient client.Client, t *testing.T) string {
	mockProvider := &Provider{}
	client, err := mockProvider.NewClient(context.Background(), mockStore, mockKubeClient, "default")
	if err != nil {
		t.Errorf("Error creating client: %q", err)
		return "error"
	}

	// perform request, exercising GetSecret
	resp, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "dummy"})
	if err != nil {
		t.Errorf("Error retrieving secret:%s", err)
	}
	return string(resp)
}
