/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://wwg.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

var (
	githubProvider = &esv1beta1.GithubProvider{
		AppID:     "123456",
		InstallID: "123456789",
		Auth: esv1beta1.GithubAuth{
			PrivatKey: esv1beta1.GithubSecretRef{
				SecretRef: esmeta.SecretKeySelector{
					Name: "testName",
					Key:  "testKey",
				},
			},
		},
	}
	secretStore = &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: githubProvider.Auth.PrivatKey.SecretRef.Name,
		},
		Data: map[string][]byte{},
	}
	validResponce = []byte(`{
				"token": "ghs_16C7e42F292c6912E7710c838347Ae178B4a",
				"expires_at": "2016-07-11T22:14:10Z",
				"permissions": {
				  "issues": "write",
				  "contents": "read"
				},
				"repository_selection": "selected"
			  }`)
)

func TestGetInstallationToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err, "Failed to generate private key")
	appID := "123456"

	tkn, err := GetInstallationToken(key, appID)
	assert.NoError(t, err, "Should not error when generating token")

	// Validate the token string is not empty
	assert.NotEmpty(t, tkn, "Token string should not be empty")

	// Parse and validate the token
	token, err := jwt.Parse(tkn, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		assert.Equal(t, jwt.SigningMethodRS256, token.Method, "Token signing method mismatch")

		return &key.PublicKey, nil
	})

	assert.NoError(t, err, "Token should be valid")
	assert.NotNil(t, token, "Parsed token should not be nil")

	// Validate claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		assert.Equal(t, appID, claims["iss"], "Issuer (iss) claim should match the app ID")
		assert.WithinDuration(t, time.Now().Add(-time.Second*10), time.Unix(int64(claims["iat"].(float64)), 0), time.Second, "IssuedAt (iat) should be valid")
		assert.WithinDuration(t, time.Now().Add(time.Second*300), time.Unix(int64(claims["exp"].(float64)), 0), time.Second, "ExpiresAt (exp) should be valid")
	} else {
		t.Error("Failed to parse claims or token is invalid")
	}
}

func TestGetSecret(t *testing.T) {
	ref := esv1beta1.ExternalSecretDataRemoteRef{
		Key: "token",
	}
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "POST", req.Method, "Expected POST request")
		assert.Equal(t, req.URL.String(), fmt.Sprintf("/app/installations/%s/access_tokens", githubProvider.InstallID))

		assert.Empty(t, req.Body)
		assert.NotEmpty(t, req.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github.v3+json", req.Header.Get("Accept"))

		// Send response to be tested
		rw.Write(validResponce)
	}))
	defer server.Close()

	pk, err := os.ReadFile("github_test_pk.pem")
	assert.NoError(t, err, "Should not error when reading privatKey")

	secretStore.Data[githubProvider.Auth.PrivatKey.SecretRef.Key] = pk
	g := Github{
		store: &esv1beta1.SecretStore{
			Spec: esv1beta1.SecretStoreSpec{
				Provider: &esv1beta1.SecretStoreProvider{
					Github: githubProvider,
				},
			},
		},
		kube: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(secretStore).Build(),
		http: server.Client(),
		url:  fmt.Sprintf("%s/app/installations/%s/access_tokens", server.URL, githubProvider.InstallID),
	}
	tkn, err := g.GetSecret(context.Background(), ref)
	assert.Equal(t, "ghs_16C7e42F292c6912E7710c838347Ae178B4a", string(tkn))
	assert.NoError(t, err)
}
