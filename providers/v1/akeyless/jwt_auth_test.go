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

const testJWTAudience = "akeyless.io"

func TestResolveJWTAuthTokenFromServiceAccount(t *testing.T) {
	otherNS := "team-b"
	customExp := int64(3600)
	aud := []string{testJWTAudience}

	tests := []struct {
		name          string
		storeKind     string
		jwtAuth       esv1.AkeylessJWTAuth
		tokenErr      error
		wantNoRequest bool
		wantNamespace string
		wantAudiences []string
		wantExp       int64
		wantErr       string
	}{
		{
			name:      "missing audiences is rejected before TokenRequest",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa"},
			},
			wantNoRequest: true,
			wantErr:       errJWTAuthMissingAudiences,
		},
		{
			name:      "audiences use the default expiration",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Audiences: aud},
			},
			wantNamespace: "default",
			wantAudiences: aud,
			wantExp:       defaultJWTTokenTTL,
		},
		{
			name:      "explicit expiration is passed through",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Audiences: aud},
				ExpirationSeconds: &customExp,
			},
			wantNamespace: "default",
			wantAudiences: aud,
			wantExp:       customExp,
		},
		{
			name:      "SecretStore ignores serviceAccountRef namespace",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Namespace: &otherNS, Audiences: aud},
			},
			wantNamespace: "default",
			wantAudiences: aud,
			wantExp:       defaultJWTTokenTTL,
		},
		{
			name:      "ClusterSecretStore uses serviceAccountRef namespace",
			storeKind: esv1.ClusterSecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Namespace: &otherNS, Audiences: aud},
			},
			wantNamespace: otherNS,
			wantAudiences: aud,
			wantExp:       defaultJWTTokenTTL,
		},
		{
			name:      "TokenRequest error is returned",
			storeKind: esv1.SecretStoreKind,
			jwtAuth: esv1.AkeylessJWTAuth{
				AccessID:          "p-jwt",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Audiences: aud},
			},
			tokenErr:      errors.New("forbidden"),
			wantNamespace: "default",
			wantAudiences: aud,
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

			if tt.wantNoRequest {
				require.Empty(t, *captured)
			} else {
				require.Len(t, *captured, 1)
				req := (*captured)[0]
				require.Equal(t, tt.wantNamespace, req.namespace)
				require.Equal(t, "akeyless-sa", req.name)
				require.Equal(t, tt.wantAudiences, req.spec.Audiences)
				require.NotNil(t, req.spec.ExpirationSeconds)
				require.Equal(t, tt.wantExp, *req.spec.ExpirationSeconds)
			}
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

		token, _, err := base.tokenFromJWTAuth(t.Context(), &esv1.AkeylessJWTAuth{
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

		token, _, err := base.tokenFromJWTAuth(t.Context(), &esv1.AkeylessJWTAuth{
			AccessID:          "p-jwt",
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "akeyless-sa", Audiences: []string{testJWTAudience}},
		})
		require.NoError(t, err)
		require.Equal(t, "t-akeyless", token)
		require.Len(t, *captured, 1)
		require.Equal(t, "p-jwt", body["access-id"])
		require.Equal(t, "jwt", body["access-type"])
		require.Equal(t, "sa.jwt.token", body["jwt"])
	})
}

// TestTokenFromSecretRefJWTRouting drives the store-level entry point, so both the
// jwtAuth dispatch and the generic credentials Secret with accessType=jwt are covered.
func TestTokenFromSecretRefJWTRouting(t *testing.T) {
	gwURL := "https://api.akeyless.io"
	jwtSecret := &corev1.Secret{
		Name:      "akeyless-jwt",
		Namespace: "default",
		Data:      map[string][]byte{"token": []byte("idp.jwt.token")},
	}
	credsWithJWT := &corev1.Secret{
		Name:      "akeyless-creds",
		Namespace: "default",
		Data: map[string][]byte{
			"accessId":        []byte("p-creds"),
			"accessType":      []byte("jwt"),
			"accessTypeParam": []byte("creds.jwt.token"),
		},
	}
	credsWithoutJWT := &corev1.Secret{
		Name:      "akeyless-creds",
		Namespace: "default",
		Data: map[string][]byte{
			"accessId":   []byte("p-creds"),
			"accessType": []byte("jwt"),
		},
	}
	credsRef := func(withParam bool) esv1.AkeylessAuthSecretRef {
		ref := esv1.AkeylessAuthSecretRef{
			AccessID:   esmeta.SecretKeySelector{Name: "akeyless-creds", Key: "accessId"},
			AccessType: esmeta.SecretKeySelector{Name: "akeyless-creds", Key: "accessType"},
		}
		if withParam {
			ref.AccessTypeParam = esmeta.SecretKeySelector{Name: "akeyless-creds", Key: "accessTypeParam"}
		}
		return ref
	}

	tests := []struct {
		name         string
		auth         esv1.AkeylessAuth
		secret       *corev1.Secret
		wantAccessID string
		wantJWT      string
		wantErr      string
	}{
		{
			name: "jwtAuth.secretRef",
			auth: esv1.AkeylessAuth{JWTAuth: &esv1.AkeylessJWTAuth{
				AccessID:  "p-jwt",
				SecretRef: &esmeta.SecretKeySelector{Name: "akeyless-jwt", Key: "token"},
			}},
			secret:       jwtSecret,
			wantAccessID: "p-jwt",
			wantJWT:      "idp.jwt.token",
		},
		{
			name:         "credentials Secret with accessType jwt",
			auth:         esv1.AkeylessAuth{SecretRef: credsRef(true)},
			secret:       credsWithJWT,
			wantAccessID: "p-creds",
			wantJWT:      "creds.jwt.token",
		},
		{
			name:    "credentials Secret with accessType jwt but no accessTypeParam",
			auth:    esv1.AkeylessAuth{SecretRef: credsRef(false)},
			secret:  credsWithoutJWT,
			wantErr: errMissingJWT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := tt.auth
			store := &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Akeyless: &esv1.AkeylessProvider{
					AkeylessGWApiURL: &gwURL,
					Auth:             &auth,
				}}},
			}
			base := &akeylessBase{
				kube:      fake.NewClientBuilder().WithObjects(tt.secret).Build(),
				store:     store,
				namespace: "default",
				storeKind: esv1.SecretStoreKind,
			}
			body := newAuthServer(t, base)

			token, err := base.TokenFromSecretRef(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Empty(t, body, "Akeyless /auth must not be called")
				return
			}
			require.NoError(t, err)
			require.Equal(t, "t-akeyless", token)
			require.Equal(t, tt.wantAccessID, body["access-id"])
			require.Equal(t, "jwt", body["access-type"])
			require.Equal(t, tt.wantJWT, body["jwt"])
		})
	}
}

func TestGetTokenJWTRequiresToken(t *testing.T) {
	base := &akeylessBase{}
	_, err := base.GetToken(t.Context(), "p-jwt", akeylessJWTAccessType, "  ", nil)
	require.ErrorContains(t, err, errMissingJWT)
}
