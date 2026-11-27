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

package keyvault

import (
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/azidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestValidateIdentityBindingStore(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*esv1.AzureKVProvider)
		wantErr string
	}{
		{name: "referenced account"},
		{
			name:   "controller account",
			change: func(p *esv1.AzureKVProvider) { p.ServiceAccountRef = nil },
		},
		{
			name: "standard remains the default",
			change: func(p *esv1.AzureKVProvider) {
				p.WorkloadIdentityMethod = ""
				p.UseAzureSDK = nil
				p.ServiceAccountRef.Audiences = []string{"custom"}
			},
		},
		{
			name:    "requires workload identity",
			change:  func(p *esv1.AzureKVProvider) { p.AuthType = new(esv1.AzureServicePrincipal) },
			wantErr: "IdentityBinding requires authType: WorkloadIdentity",
		},
		{
			name:    "requires modern SDK",
			change:  func(p *esv1.AzureKVProvider) { p.UseAzureSDK = new(false) },
			wantErr: "IdentityBinding requires useAzureSDK: true",
		},
		{
			name:    "rejects custom audiences",
			change:  func(p *esv1.AzureKVProvider) { p.ServiceAccountRef.Audiences = []string{"custom"} },
			wantErr: "IdentityBinding does not support custom serviceAccountRef audiences",
		},
		{
			name:    "rejects service account in another namespace",
			change:  func(p *esv1.AzureKVProvider) { p.ServiceAccountRef.Namespace = new("team-b") },
			wantErr: "invalid ServiceAccountRef",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &esv1.AzureKVProvider{
				AuthType:               new(esv1.AzureWorkloadIdentity),
				UseAzureSDK:            new(true),
				WorkloadIdentityMethod: esv1.AzureWorkloadIdentityMethodIdentityBinding,
				ServiceAccountRef:      &esmeta.ServiceAccountSelector{Name: "eso-keyvault"},
			}
			store := &esv1.SecretStore{
				Namespace: "team-a",
				Spec:      esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{AzureKV: provider}},
			}
			if tc.change != nil {
				tc.change(provider)
			}
			_, err := new(Azure).ValidateStore(store)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildMountedWorkloadIdentityCredential(t *testing.T) {
	t.Setenv("AZURE_CLIENT_ID", "controller-test-client")
	t.Setenv("AZURE_TENANT_ID", "controller-test-tenant")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", filepath.Join(t.TempDir(), "controller-token"))
	t.Setenv(azureKubernetesTokenProxy, "")
	t.Setenv(azureKubernetesCAData, "")
	t.Setenv(azureKubernetesCAFile, "")
	t.Setenv(azureKubernetesSNIName, "")
	az := &Azure{provider: &esv1.AzureKVProvider{}}

	_, err := buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.NoError(t, err, "the default method does not need a proxy")

	az.provider.WorkloadIdentityMethod = esv1.AzureWorkloadIdentityMethodIdentityBinding
	_, err = buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.ErrorContains(t, err, azureKubernetesTokenProxy)

	t.Setenv(azureKubernetesTokenProxy, "https://proxy.example.com")
	credential, err := buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.NoError(t, err)
	require.IsType(t, &azidentity.WorkloadIdentityCredential{}, credential)
}

func TestIdentityBindingProxyRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		endpoint  string
		caData    string
		caFile    string
		wantError string
	}{
		{name: "HTTP endpoint", endpoint: "http://proxy.example.com", wantError: "must be an HTTPS URL"},
		{name: "endpoint with user info", endpoint: "https://user@proxy.example.com", wantError: "must be an HTTPS URL"},
		{name: "endpoint with query", endpoint: "https://proxy.example.com?foo=bar", wantError: "must be an HTTPS URL"},
		{name: "invalid CA", endpoint: "https://proxy.example.com", caData: "invalid", wantError: "no valid PEM certificates"},
		{
			name:      "conflicting CA sources",
			endpoint:  "https://proxy.example.com",
			caFile:    "cert.pem",
			caData:    "invalid",
			wantError: "may be set",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(azureKubernetesTokenProxy, tc.endpoint)
			t.Setenv(azureKubernetesCAData, tc.caData)
			t.Setenv(azureKubernetesCAFile, tc.caFile)
			_, err := newIdentityBindingProxy()
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}

func TestBuildIdentityBindingCredentialForServiceAccount(t *testing.T) {
	const (
		account   = "eso-keyvault"
		tenant    = "test-tenant"
		assertion = "team-a-test-assertion"
		clientID  = "team-a-test-client"
		sni       = "example.com"
	)

	t.Setenv("AZURE_CLIENT_ID", "controller-test-client")
	t.Setenv("AZURE_TENANT_ID", "controller-test-tenant")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", filepath.Join(t.TempDir(), "unused-controller-token"))
	t.Setenv(azureKubernetesCAFile, "")
	t.Setenv(azureKubernetesSNIName, sni)

	proxy := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, sni, r.TLS.ServerName)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/proxy/common/discovery/instance":
			assert.NotEmpty(t, r.URL.Query().Get("authorization_endpoint"))
			_, _ = fmt.Fprintf(w, `{"tenant_discovery_endpoint":"https://%[1]s/%[2]s/v2.0/.well-known/openid-configuration","metadata":[{"preferred_network":"%[1]s","preferred_cache":"%[1]s","aliases":["%[1]s"]}]}`, r.Host, tenant)
		case "/proxy/" + tenant + "/v2.0/.well-known/openid-configuration":
			_, _ = fmt.Fprintf(w, `{"token_endpoint":"https://%[1]s/%[2]s/oauth2/v2.0/token","issuer":"https://%[1]s/%[2]s/v2.0","authorization_endpoint":"https://%[1]s/%[2]s/oauth2/v2.0/authorize"}`, r.Host, tenant)
		case "/proxy/" + tenant + "/oauth2/v2.0/token":
			if assert.NoError(t, r.ParseForm()) {
				assert.Equal(t, clientID, r.PostForm.Get("client_id"))
				assert.Equal(t, assertion, r.PostForm.Get("client_assertion"))
				assert.Equal(t, "client_credentials", r.PostForm.Get("grant_type"))
				assert.Contains(t, r.PostForm.Get("scope"), "https://vault.azure.net/.default")
			}
			_, _ = fmt.Fprint(w, `{"token_type":"Bearer","access_token":"team-a-test-access-token","expires_in":3600}`)
		default:
			t.Errorf("unexpected proxy request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(proxy.Close)
	t.Setenv(azureKubernetesCAData, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: proxy.Certificate().Raw})))

	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
		Name:      account,
		Namespace: "team-a",
		Annotations: map[string]string{
			AnnotationClientID: clientID,
			AnnotationTenantID: tenant,
		},
	}}
	kube := kubefake.NewClientset()
	kube.PrependReactor("create", "serviceaccounts", func(action kubetesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "token" {
			return false, nil, nil
		}
		return true, &authv1.TokenRequest{Status: authv1.TokenRequestStatus{Token: assertion}}, nil
	})
	az := &Azure{
		crClient:   clientfake.NewClientBuilder().WithObjects(sa).Build(),
		kubeClient: kube.CoreV1(),
		store:      &esv1.SecretStore{Namespace: "team-a"},
		namespace:  "external-secrets",
		provider: &esv1.AzureKVProvider{
			WorkloadIdentityMethod: esv1.AzureWorkloadIdentityMethodIdentityBinding,
			ServiceAccountRef:      &esmeta.ServiceAccountSelector{Name: account},
		},
	}

	t.Setenv(azureKubernetesTokenProxy, "")
	_, err := buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.ErrorContains(t, err, azureKubernetesTokenProxy)
	require.Empty(t, kube.Actions(), "do not mint an assertion without a proxy")

	t.Setenv(azureKubernetesTokenProxy, proxy.URL+"/proxy")
	credential, err := buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.NoError(t, err)
	token, err := credential.GetToken(t.Context(), policy.TokenRequestOptions{Scopes: []string{"https://vault.azure.net/.default"}})
	require.NoError(t, err)
	require.Equal(t, "team-a-test-access-token", token.Token)

	actions := kube.Actions()
	require.Len(t, actions, 1)
	action, ok := actions[0].(kubetesting.CreateActionImpl)
	require.True(t, ok)
	require.Equal(t, "team-a", action.GetNamespace())
	require.Equal(t, account, action.Name)
	require.Equal(t, "token", action.GetSubresource())
	request, ok := action.GetObject().(*authv1.TokenRequest)
	require.True(t, ok)
	require.Equal(t, []string{"api://AKSIdentityBinding"}, request.Spec.Audiences)
	require.Nil(t, request.Spec.BoundObjectRef)
}