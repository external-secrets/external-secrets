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

package oidc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// noopExchanger satisfies TokenExchanger and returns a fixed token.
type noopExchanger struct{}

func (n *noopExchanger) ExchangeToken(_ context.Context, _ string) (string, time.Time, error) {
	return "exchanged-token", time.Now().Add(time.Hour), nil
}

// namespaceMismatchReactor intercepts serviceaccounts/token create calls and
// returns an error when the request body namespace differs from the URL namespace,
// mimicking the kube-apiserver's enforcement. fake.NewSimpleClientset() does not
// enforce this check, which is why the pre-fix code passed tests.
func namespaceMismatchReactor(action k8stesting.Action) (bool, runtime.Object, error) {
	ca, ok := action.(k8stesting.CreateAction)
	if !ok {
		return false, nil, nil
	}
	tr, ok := ca.GetObject().(*authv1.TokenRequest)
	if !ok {
		return false, nil, nil
	}
	urlNS := action.GetNamespace()
	bodyNS := tr.Namespace
	if bodyNS != "" && bodyNS != urlNS {
		return true, nil, fmt.Errorf(
			"the namespace of the provided object does not match "+
				"the namespace sent on the request: body=%q url=%q",
			bodyNS, urlNS,
		)
	}
	return true, &authv1.TokenRequest{
		Status: authv1.TokenRequestStatus{Token: "fake-sa-token"},
	}, nil
}

func TestCreateServiceAccountToken_NamespaceConsistency(t *testing.T) {
	tests := []struct {
		name          string
		storeKind     string
		esNamespace   string
		saRef         esmeta.ServiceAccountSelector
		wantNamespace string // expected URL (and body) namespace
	}{
		{
			name:          "SecretStore same-namespace",
			storeKind:     esv1.SecretStoreKind,
			esNamespace:   "default",
			saRef:         esmeta.ServiceAccountSelector{Name: "my-sa"},
			wantNamespace: "default",
		},
		{
			name:        "ClusterSecretStore cross-namespace regression",
			storeKind:   esv1.ClusterSecretStoreKind,
			esNamespace: "default",
			saRef:       esmeta.ServiceAccountSelector{Name: "oidc-sa", Namespace: new("external-secrets")},
			// Pre-fix: body namespace was "default", URL was "external-secrets"
			// -> kube-apiserver rejected with "namespace of provided object does not match".
			wantNamespace: "external-secrets",
		},
		{
			name:          "ClusterSecretStore SA namespace equals ES namespace",
			storeKind:     esv1.ClusterSecretStoreKind,
			esNamespace:   "external-secrets",
			saRef:         esmeta.ServiceAccountSelector{Name: "oidc-sa", Namespace: new("external-secrets")},
			wantNamespace: "external-secrets",
		},
		{
			name:          "ClusterSecretStore no explicit SA namespace falls back to ES namespace",
			storeKind:     esv1.ClusterSecretStoreKind,
			esNamespace:   "default",
			saRef:         esmeta.ServiceAccountSelector{Name: "oidc-sa"},
			wantNamespace: "default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capturedURLNS := ""
			capturedBodyNS := ""

			fc := fake.NewSimpleClientset()
			fc.Fake.PrependReactor("create", "serviceaccounts/token",
				func(action k8stesting.Action) (bool, runtime.Object, error) {
					capturedURLNS = action.GetNamespace()
					ca, ok := action.(k8stesting.CreateAction)
					if ok {
						if tr, ok := ca.GetObject().(*authv1.TokenRequest); ok {
							capturedBodyNS = tr.Namespace
						}
					}
					return namespaceMismatchReactor(action)
				},
			)

			m := NewBaseTokenManager(
				fc.CoreV1(), tc.esNamespace, tc.storeKind, "https://example.com", tc.saRef,
			)

			saToken, err := m.CreateServiceAccountToken(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "fake-sa-token", saToken)

			assert.Equal(t, tc.wantNamespace, capturedURLNS,
				"URL namespace must match the SA's namespace")
			assert.Equal(t, capturedURLNS, capturedBodyNS,
				"body namespace must equal URL namespace (apiserver enforces this)")
		})
	}
}

func TestGetToken_CachingAndRefresh(t *testing.T) {
	fc := fake.NewSimpleClientset()
	callCount := 0
	fc.Fake.PrependReactor("create", "serviceaccounts/token",
		func(_ k8stesting.Action) (bool, runtime.Object, error) {
			callCount++
			return true, &authv1.TokenRequest{
				Status: authv1.TokenRequestStatus{Token: fmt.Sprintf("token-%d", callCount)},
			}, nil
		},
	)

	saRef := esmeta.ServiceAccountSelector{Name: "test-sa"}
	m := NewBaseTokenManager(fc.CoreV1(), "default", esv1.SecretStoreKind, "https://example.com", saRef)
	m.Exchanger = &noopExchanger{}

	tok1, err := m.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "exchanged-token", tok1)
	assert.Equal(t, 1, callCount)

	// Second call should use the cached token without a new SA token request.
	tok2, err := m.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, tok1, tok2)
	assert.Equal(t, 1, callCount, "cached token returned without a new SA token request")
}

func TestBuildAudiences(t *testing.T) {
	tests := []struct {
		name         string
		saAudiences  []string
		extraAud     []string
		baseURL      string
		wantAudience []string
	}{
		{
			name:         "uses baseURL when no SA audiences configured",
			baseURL:      "https://api.example.com",
			wantAudience: []string{"https://api.example.com"},
		},
		{
			name:         "SA audiences override baseURL",
			saAudiences:  []string{"sts.amazonaws.com"},
			baseURL:      "https://api.example.com",
			wantAudience: []string{"sts.amazonaws.com"},
		},
		{
			name:         "extra audiences appended after SA audiences",
			saAudiences:  []string{"sts.amazonaws.com"},
			extraAud:     []string{"extra-aud"},
			baseURL:      "https://api.example.com",
			wantAudience: []string{"sts.amazonaws.com", "extra-aud"},
		},
		{
			name:         "extra audiences appended after baseURL fallback",
			extraAud:     []string{"extra-aud"},
			baseURL:      "https://api.example.com",
			wantAudience: []string{"https://api.example.com", "extra-aud"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saRef := esmeta.ServiceAccountSelector{
				Name:      "test-sa",
				Audiences: tc.saAudiences,
			}
			fc := fake.NewSimpleClientset()
			m := NewBaseTokenManager(fc.CoreV1(), "default", esv1.SecretStoreKind, tc.baseURL, saRef)
			m.ExtraAudiences = tc.extraAud
			assert.Equal(t, tc.wantAudience, m.BuildAudiences())
		})
	}
}
