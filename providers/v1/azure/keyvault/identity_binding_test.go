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
	"path/filepath"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestValidateIdentityBindingStore(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*esv1.AzureKVProvider)
		wantErr string
	}{
		{
			name: "identity binding without service account is valid",
		},
		{
			name: "standard workload identity remains the default",
			change: func(p *esv1.AzureKVProvider) {
				p.WorkloadIdentityMethod = esv1.AzureWorkloadIdentityMethodDefault
				p.UseAzureSDK = nil
				p.AuthType = nil
				p.ServiceAccountRef = &esmeta.ServiceAccountSelector{Name: "eso-keyvault"}
			},
		},
		{
			name:    "requires workload identity auth type",
			change:  func(p *esv1.AzureKVProvider) { p.AuthType = new(esv1.AzureServicePrincipal) },
			wantErr: "IdentityBinding requires authType: WorkloadIdentity",
		},
		{
			name:    "requires modern SDK",
			change:  func(p *esv1.AzureKVProvider) { p.UseAzureSDK = new(false) },
			wantErr: "IdentityBinding requires useAzureSDK: true",
		},
		{
			name: "rejects serviceAccountRef",
			change: func(p *esv1.AzureKVProvider) {
				p.ServiceAccountRef = &esmeta.ServiceAccountSelector{Name: "eso-keyvault"}
			},
			wantErr: "IdentityBinding does not support serviceAccountRef",
		},
		{
			name:    "rejects unknown method",
			change:  func(p *esv1.AzureKVProvider) { p.WorkloadIdentityMethod = "SomethingElse" },
			wantErr: "invalid WorkloadIdentityMethod",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &esv1.AzureKVProvider{
				AuthType:               new(esv1.AzureWorkloadIdentity),
				UseAzureSDK:            new(true),
				WorkloadIdentityMethod: esv1.AzureWorkloadIdentityMethodIdentityBinding,
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

func TestBuildWorkloadIdentityCredentialIdentityBinding(t *testing.T) {
	// The webhook injects these environment variables into the controller pod.
	t.Setenv("AZURE_CLIENT_ID", "controller-test-client")
	t.Setenv("AZURE_TENANT_ID", "controller-test-tenant")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", filepath.Join(t.TempDir(), "controller-token"))

	az := &Azure{provider: &esv1.AzureKVProvider{
		AuthType: new(esv1.AzureWorkloadIdentity),
	}}

	// The default method does not require the identity binding proxy.
	credential, err := buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.NoError(t, err)
	require.IsType(t, &azidentity.WorkloadIdentityCredential{}, credential)

	// Identity binding without the proxy environment variable fails fast with a
	// clear error instead of silently falling back to standard token exchange.
	az.provider.WorkloadIdentityMethod = esv1.AzureWorkloadIdentityMethodIdentityBinding
	_, err = buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.ErrorContains(t, err, azureKubernetesTokenProxyEnv)

	// With the proxy configured, the credential enables the Azure proxy.
	t.Setenv(azureKubernetesTokenProxyEnv, "https://proxy.example.com")
	credential, err = buildWorkloadIdentityCredential(t.Context(), az, cloud.AzurePublic)
	require.NoError(t, err)
	require.IsType(t, &azidentity.WorkloadIdentityCredential{}, credential)
}
