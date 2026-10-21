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

package secretmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google/externalaccount"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestServiceAccountImpersonationURL(t *testing.T) {
	const email = "eso@example-project.eu0.iam.gserviceaccount.com"
	assert.Equal(t,
		"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/"+email+":generateAccessToken",
		serviceAccountImpersonationURL("", email))
	assert.Equal(t,
		"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/"+email+":generateAccessToken",
		serviceAccountImpersonationURL(defaultUniverseDomain, email))
	assert.Equal(t,
		"https://iamcredentials.apis-berlin-build0.goog/v1/projects/-/serviceAccounts/"+email+":generateAccessToken",
		serviceAccountImpersonationURL(testDedicatedUniverse, email))
}

// universeCredConfig is an external_account JSON that declares its own universe.
func universeCredConfig(universe string) string {
	config := map[string]any{
		"type":               externalAccountCredentialType,
		"audience":           testAudience,
		"subject_token_type": workloadIdentitySubjectTokenType,
		"universe_domain":    universe,
		"token_url":          fmt.Sprintf(workloadIdentityTokenURLFormat, universe),
		"token_info_url":     fmt.Sprintf(workloadIdentityTokenInfoURLFormat, universe),
		"credential_source": map[string]any{
			"file": "/var/run/secrets/oidc_token",
		},
	}
	data, _ := json.Marshal(config)
	return string(data)
}

// A spec universeDomain overrides the one of the credential file: the endpoints the file derived from its
// former universe follow the override, endpoints on any other host are rejected.
func TestWorkloadIdentityFederationUniverseOverrideRebasesCredFileEndpoints(t *testing.T) {
	credConfigWith := func(tokenURL, tokenInfoURL, impersonationURL string) *corev1.ConfigMap {
		data, _ := json.Marshal(map[string]any{
			"type":                              externalAccountCredentialType,
			"audience":                          testAudience,
			"subject_token_type":                workloadIdentitySubjectTokenType,
			"token_url":                         tokenURL,
			"token_info_url":                    tokenInfoURL,
			"service_account_impersonation_url": impersonationURL,
			"credential_source": map[string]any{
				"file": "/var/run/secrets/oidc_token",
			},
		})

		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testConfigMapName, Namespace: testNamespace},
			Data:       map[string]string{testConfigMapKey: string(data)},
		}
	}
	read := func(cm *corev1.ConfigMap) (*externalaccount.Config, error) {
		wif := &workloadIdentityFederation{
			kubeClient: clientfake.NewClientBuilder().WithObjects(cm).Build(),
			config: &esv1.GCPWorkloadIdentityFederation{
				Audience:       testAudience,
				UniverseDomain: testDedicatedUniverse,
				CredConfig:     &esv1.ConfigMapReference{Name: testConfigMapName, Key: testConfigMapKey, Namespace: testNamespace},
			},
			isClusterKind: true,
			namespace:     testNamespace,
		}

		return wif.readCredConfig(context.Background())
	}

	t.Run("endpoints of the former universe follow the override", func(t *testing.T) {
		cfg, err := read(credConfigWith(
			testTokenURL,
			testTokenInfoURL,
			testServiceAccountImpersonationURL,
		))
		require.NoError(t, err)
		assert.Equal(t, testDedicatedUniverse, cfg.UniverseDomain)
		assert.Equal(t, "https://sts.apis-berlin-build0.goog/v1/token", cfg.TokenURL)
		assert.Equal(t, "https://sts.apis-berlin-build0.goog/v1/introspect", cfg.TokenInfoURL)
		assert.Equal(t,
			"https://iamcredentials.apis-berlin-build0.goog/v1/projects/-/serviceAccounts/test@test.iam.gserviceaccount.com:generateAccessToken",
			cfg.ServiceAccountImpersonationURL)
	})

	t.Run("endpoints on an unrelated host are rejected", func(t *testing.T) {
		_, err := read(credConfigWith(
			"https://sts.other.example/v1/token",
			testTokenInfoURL,
			"",
		))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token_url")
	})
}

// The universe domain must drive the STS and IAM Credentials hosts built by the provider, for all the
// ways a federation can be configured, without any network call.
func TestWorkloadIdentityFederationUniverseDomain(t *testing.T) {
	saRef := &esmeta.ServiceAccountSelector{
		Name:      testServiceAccount,
		Namespace: &testNamespace,
		Audiences: []string{testAudience},
	}
	plainSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount, Namespace: testNamespace},
	}
	annotatedSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testServiceAccount,
			Namespace:   testNamespace,
			Annotations: map[string]string{gcpSAAnnotation: testDedicatedServiceAccountEmail},
		},
	}
	credConfigMap := func(universe string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testConfigMapName, Namespace: testNamespace},
			Data:       map[string]string{testConfigMapKey: universeCredConfig(universe)},
		}
	}
	credConfigRef := &esv1.ConfigMapReference{Name: testConfigMapName, Key: testConfigMapKey, Namespace: testNamespace}

	tests := []struct {
		name               string
		wif                *esv1.GCPWorkloadIdentityFederation
		kubeObjects        []client.Object
		wantUniverse       string
		wantTokenURL       string
		wantTokenInfoURL   string
		wantImpersonateURL string
	}{
		{
			name: "serviceAccountRef with universeDomain and gcpServiceAccountEmail",
			wif: &esv1.GCPWorkloadIdentityFederation{
				Audience:               testAudience,
				UniverseDomain:         testDedicatedUniverse,
				GCPServiceAccountEmail: testDedicatedServiceAccountEmail,
				ServiceAccountRef:      saRef,
			},
			kubeObjects:        []client.Object{plainSA},
			wantUniverse:       testDedicatedUniverse,
			wantTokenURL:       "https://sts.apis-berlin-build0.goog/v1/token",
			wantTokenInfoURL:   "https://sts.apis-berlin-build0.goog/v1/introspect",
			wantImpersonateURL: serviceAccountImpersonationURL(testDedicatedUniverse, testDedicatedServiceAccountEmail),
		},
		{
			name: "serviceAccountRef with universeDomain and annotated service account",
			wif: &esv1.GCPWorkloadIdentityFederation{
				Audience:          testAudience,
				UniverseDomain:    testDedicatedUniverse,
				ServiceAccountRef: saRef,
			},
			kubeObjects:        []client.Object{annotatedSA},
			wantUniverse:       testDedicatedUniverse,
			wantTokenURL:       "https://sts.apis-berlin-build0.goog/v1/token",
			wantTokenInfoURL:   "https://sts.apis-berlin-build0.goog/v1/introspect",
			wantImpersonateURL: serviceAccountImpersonationURL(testDedicatedUniverse, testDedicatedServiceAccountEmail),
		},
		{
			name: "universe taken from credConfig when the spec does not set one",
			wif: &esv1.GCPWorkloadIdentityFederation{
				Audience:               testAudience,
				GCPServiceAccountEmail: testDedicatedServiceAccountEmail,
				CredConfig:             credConfigRef,
			},
			kubeObjects:        []client.Object{credConfigMap(testDedicatedUniverse)},
			wantUniverse:       testDedicatedUniverse,
			wantTokenURL:       "https://sts.apis-berlin-build0.goog/v1/token",
			wantTokenInfoURL:   "https://sts.apis-berlin-build0.goog/v1/introspect",
			wantImpersonateURL: serviceAccountImpersonationURL(testDedicatedUniverse, testDedicatedServiceAccountEmail),
		},
		{
			name: "no universe anywhere keeps the default googleapis.com endpoints",
			wif: &esv1.GCPWorkloadIdentityFederation{
				Audience:               testAudience,
				GCPServiceAccountEmail: testGCPServiceAccountEmail,
				ServiceAccountRef:      saRef,
			},
			kubeObjects:        []client.Object{plainSA},
			wantUniverse:       defaultUniverseDomain,
			wantTokenURL:       "https://sts.googleapis.com/v1/token",
			wantTokenInfoURL:   "https://sts.googleapis.com/v1/introspect",
			wantImpersonateURL: testServiceAccountImpersonationURL,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wif := &workloadIdentityFederation{
				kubeClient:       clientfake.NewClientBuilder().WithObjects(tc.kubeObjects...).Build(),
				saTokenGenerator: &fakeSATokenGen{GenerateFunc: defaultSATokenGenerator},
				config:           tc.wif,
				isClusterKind:    true,
				namespace:        testNamespace,
			}

			cfg, err := wif.readCredConfig(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tc.wantUniverse, cfg.UniverseDomain)
			assert.Equal(t, tc.wantTokenURL, cfg.TokenURL)
			assert.Equal(t, tc.wantTokenInfoURL, cfg.TokenInfoURL)
			assert.Equal(t, tc.wantImpersonateURL, cfg.ServiceAccountImpersonationURL)
		})
	}
}
