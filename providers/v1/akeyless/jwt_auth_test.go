/*
Copyright © The ESO Authors

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

package akeyless

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akeylesslabs/akeyless-go/v4"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestResolveJWTAuthTokenFromSecret(t *testing.T) {
	sec := &corev1.Secret{
		Name:      "akeyless-jwt",
		Namespace: "default",
		Data: map[string][]byte{
			"token": []byte("header.payload.sig"),
		},
	}
	base := &akeylessBase{
		kube:      fake.NewClientBuilder().WithObjects(sec).Build(),
		namespace: "default",
		storeKind: esv1.SecretStoreKind,
	}

	got, err := base.resolveJWTAuthToken(t.Context(), &esv1.AkeylessJWTAuth{
		AccessID: "p-jwt",
		SecretRef: &esmeta.SecretKeySelector{
			Name: "akeyless-jwt",
			Key:  "token",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "header.payload.sig", got)
}

func TestResolveJWTAuthTokenRequiresSource(t *testing.T) {
	base := &akeylessBase{}
	_, err := base.resolveJWTAuthToken(t.Context(), &esv1.AkeylessJWTAuth{AccessID: "p-jwt"})
	require.ErrorContains(t, err, errJWTAuthTokenSource)
}

func TestResolveJWTAuthTokenRejectsBothSources(t *testing.T) {
	base := &akeylessBase{}
	_, err := base.resolveJWTAuthToken(t.Context(), &esv1.AkeylessJWTAuth{
		AccessID: "p-jwt",
		SecretRef: &esmeta.SecretKeySelector{
			Name: "akeyless-jwt",
			Key:  "token",
		},
		ServiceAccountRef: &esmeta.ServiceAccountSelector{
			Name: "sa",
		},
	})
	require.ErrorContains(t, err, errJWTAuthBothTokenSources)
}

type capturedTokenRequest struct {
	namespace string
	name      string
	spec      authv1.TokenRequestSpec
}

// newTokenRequestClientset returns a clientset whose TokenRequest subresource
// records each request and answers with token, or with err when it is set.
func newTokenRequestClientset(token string, err error) (*k8sfake.Clientset, *[]capturedTokenRequest) {
	var captured []capturedTokenRequest
	cs := k8sfake.NewClientset()
	cs.PrependReactor("create", "serviceaccounts", func(action clienttesting.Action) (bool, runtime.Object, error) {
		create, ok := action.(clienttesting.CreateActionImpl)
		if !ok || create.GetSubresource() != "token" {
			return false, nil, nil
		}
		tr := create.GetObject().(*authv1.TokenRequest)
		captured = append(captured, capturedTokenRequest{
			namespace: create.GetNamespace(),
			name:      create.Name,
			spec:      tr.Spec,
		})
		if err != nil {
			return true, nil, err
		}
		return true, &authv1.TokenRequest{Status: authv1.TokenRequestStatus{Token: token}}, nil
	})
	return cs, &captured
}

func TestResolveJWTAuthTokenFromServiceAccount(t *testing.T) {
	otherNS := "team-b"
	customExp := int64(3600)

	tests := []struct {
		name          string
		storeKind     string
		jwtAuth       esv1.AkeylessJWTAuth
		tokenErr      error
		wantNamespace string
		wantAudiences []string
		wantExp       int64
		wantErr       string
	}{
		{
			name:      "no audiences defaults to akeyless, not the API server",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa"},
			},
			wantNamespace: "default",
			wantAudiences: []string{defaultJWTAudience},
			wantExp:       defaultJWTTokenTTL,
		},
		{
			name:      "explicit audiences and expiration are passed through",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID: "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{
					Name:      "akeyless-sa",
					Audiences: []string{"https://example.akeyless.io"},
				},
				ExpirationSeconds: &customExp,
			},
			wantNamespace: "default",
			wantAudiences: []string{"https://example.akeyless.io"},
			wantExp:       customExp,
		},
		{
			name:      "SecretStore ignores serviceAccountRef namespace",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Namespace: &otherNS},
			},
			wantNamespace: "default",
			wantAudiences: []string{defaultJWTAudience},
			wantExp:       defaultJWTTokenTTL,
		},
		{
			name:      "ClusterSecretStore uses serviceAccountRef namespace",
			storeKind: esv1.ClusterSecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Namespace: &otherNS},
			},
			wantNamespace: otherNS,
			wantAudiences: []string{defaultJWTAudience},
			wantExp:       defaultJWTTokenTTL,
		},
		{
			name:      "TokenRequest error is returned",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa"},
			},
			tokenErr:      errors.New("forbidden"),
			wantNamespace: "default",
			wantAudiences: []string{defaultJWTAudience},
			wantExp:       defaultJWTTokenTTL,
			wantErr:       "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, captured := newTokenRequestClientset("sa.jwt.token", tt.tokenErr)
			base := &akeylessBase{
				corev1:    cs.CoreV1(),
				namespace: "default",
				storeKind: tt.storeKind,
			}

			got, err := base.resolveJWTAuthToken(t.Context(), &tt.jwtAuth)

			require.Len(t, *captured, 1)
			req := (*captured)[0]
			require.Equal(t, tt.wantNamespace, req.namespace)
			require.Equal(t, "akeyless-sa", req.name)
			require.Equal(t, tt.wantAudiences, req.spec.Audiences)
			require.NotNil(t, req.spec.ExpirationSeconds)
			require.Equal(t, tt.wantExp, *req.spec.ExpirationSeconds)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "sa.jwt.token", got)
		})
	}
}

// newAuthServer serves /auth, records the decoded request body, and returns
// an akeylessBase wired to it.
func newAuthServer(t *testing.T, base *akeylessBase) map[string]any {
	t.Helper()
	body := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"t-akeyless"}`))
	}))
	t.Cleanup(srv.Close)
	base.RestAPI = akeyless.NewAPIClient(&akeyless.Configuration{
		Servers: []akeyless.ServerConfiguration{{URL: srv.URL}},
	}).V2Api
	return body
}

func TestTokenFromJWTAuthSendsJWTAccessType(t *testing.T) {
	t.Run("secretRef", func(t *testing.T) {
		sec := &corev1.Secret{
			Name:      "akeyless-jwt",
			Namespace: "default",
			Data:      map[string][]byte{"token": []byte("idp.jwt.token\n")},
		}
		base := &akeylessBase{
			kube:      fake.NewClientBuilder().WithObjects(sec).Build(),
			namespace: "default",
			storeKind: esv1.SecretStoreKind,
		}
		body := newAuthServer(t, base)

		token, err := base.tokenFromJWTAuth(t.Context(), &esv1.AkeylessJWTAuth{
			AccessID:  "p-jwt",
			SecretRef: &esmeta.SecretKeySelector{Name: "akeyless-jwt"},
		})
		require.NoError(t, err)
		require.Equal(t, "t-akeyless", token)
		require.Equal(t, "p-jwt", body["access-id"])
		require.Equal(t, "jwt", body["access-type"])
		require.Equal(t, "idp.jwt.token", body["jwt"])
	})

	t.Run("serviceAccountRef", func(t *testing.T) {
		cs, captured := newTokenRequestClientset("sa.jwt.token", nil)
		base := &akeylessBase{
			corev1:    cs.CoreV1(),
			namespace: "default",
			storeKind: esv1.SecretStoreKind,
		}
		body := newAuthServer(t, base)

		token, err := base.tokenFromJWTAuth(t.Context(), &esv1.AkeylessJWTAuth{
			AccessID:          "p-jwt",
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa"},
		})
		require.NoError(t, err)
		require.Equal(t, "t-akeyless", token)
		require.Len(t, *captured, 1)
		require.Equal(t, "p-jwt", body["access-id"])
		require.Equal(t, "jwt", body["access-type"])
		require.Equal(t, "sa.jwt.token", body["jwt"])
	})
}

func TestGetTokenJWTRequiresToken(t *testing.T) {
	base := &akeylessBase{}
	_, err := base.GetToken(t.Context(), "p-jwt", akeylessJWTAccessType, "  ", nil)
	require.ErrorContains(t, err, errMissingJWT)
}
