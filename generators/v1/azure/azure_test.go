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

package azure

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testTenant = "11111111-2222-3333-4444-111111111111"
	testToken  = "entra-access-token"
	// adoResource is the well-known Azure DevOps Entra application id, used here as a
	// representative resource value.
	adoResource = "499b84ac-1321-427f-aa17-267ca6975798"
	// managed-identity id fixtures for the identity-type inference cases.
	miClientID   = "22222222-3333-4444-5555-666666666666"
	miObjectID   = "33333333-4444-5555-6666-777777777777"
	miResourceID = "/subscriptions/sub/resourcegroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id"
)

func spSecretSpec() *apiextensions.JSON {
	return spSecretSpecResource(adoResource)
}

func spSecretSpecResource(resource string) *apiextensions.JSON {
	return &apiextensions.JSON{
		Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: AzureAccessToken
spec:
  tenantId: %s
  resource: %s
  environmentType: "PublicCloud"
  auth:
    servicePrincipal:
      secretRef:
        clientId:
          name: az-secret
          key: clientid
        clientSecret:
          name: az-secret
          key: clientsecret`, testTenant, resource),
	}
}

func spCertSpec() *apiextensions.JSON {
	return &apiextensions.JSON{
		Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: AzureAccessToken
spec:
  tenantId: %s
  resource: %s
  environmentType: "PublicCloud"
  auth:
    servicePrincipal:
      secretRef:
        clientId:
          name: az-secret
          key: clientid
        clientCertificate:
          name: az-secret
          key: cert`, testTenant, adoResource),
	}
}

func spBothSpec() *apiextensions.JSON {
	return &apiextensions.JSON{
		Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: AzureAccessToken
spec:
  tenantId: %s
  resource: %s
  auth:
    servicePrincipal:
      secretRef:
        clientId:
          name: az-secret
          key: clientid
        clientSecret:
          name: az-secret
          key: clientsecret
        clientCertificate:
          name: az-secret
          key: cert`, testTenant, adoResource),
	}
}

func noResourceSpec() *apiextensions.JSON {
	return &apiextensions.JSON{
		Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: AzureAccessToken
spec:
  tenantId: %s
  auth:
    servicePrincipal:
      secretRef:
        clientId:
          name: az-secret
          key: clientid
        clientSecret:
          name: az-secret
          key: clientsecret`, testTenant),
	}
}

func noAuthSpec() *apiextensions.JSON {
	return &apiextensions.JSON{
		Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: AzureAccessToken
spec:
  resource: %s
  auth: {}`, adoResource),
	}
}

// miSpec builds a managed-identity spec. identityType is omitted from the rendered
// YAML when empty, exercising the inference path.
func miSpec(identityID, identityType string) *apiextensions.JSON {
	idType := ""
	if identityType != "" {
		idType = fmt.Sprintf("\n      identityType: %s", identityType)
	}
	return &apiextensions.JSON{
		Raw: fmt.Appendf(nil, `apiVersion: generators.external-secrets.io/v1alpha1
kind: AzureAccessToken
spec:
  resource: %s
  environmentType: "PublicCloud"
  auth:
    managedIdentity:
      identityId: %s%s`, adoResource, identityID, idType),
	}
}

func secretFixture() client.Client {
	return clientfake.NewClientBuilder().WithObjects(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "az-secret",
			Namespace: "foobar",
		},
		Data: map[string][]byte{
			"clientid":     []byte("the-client-id"),
			"clientsecret": []byte("the-client-secret"),
			"cert":         []byte("-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----"),
		},
	}).Build()
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name                 string
		jsonSpec             *apiextensions.JSON
		crClient             client.Client
		kubeClient           kubernetes.Interface
		clientSecretCreds    clientSecretCredentialFunc
		clientCertCreds      clientCertificateCredentialFunc
		managedIdentityCreds managedIdentityCredentialFunc
		want                 map[string][]byte
		wantErr              bool
	}{
		{
			name:     "no spec",
			jsonSpec: nil,
			wantErr:  true,
		},
		{
			name:     "empty spec",
			jsonSpec: &apiextensions.JSON{},
			wantErr:  true,
		},
		{
			name:     "missing resource",
			jsonSpec: noResourceSpec(),
			crClient: secretFixture(),
			wantErr:  true,
		},
		{
			name:     "no auth method",
			jsonSpec: noAuthSpec(),
			crClient: secretFixture(),
			wantErr:  true,
		},
		{
			name:     "service principal with both secret and certificate",
			jsonSpec: spBothSpec(),
			crClient: secretFixture(),
			wantErr:  true,
		},
		{
			name:     "service principal client secret",
			jsonSpec: spSecretSpec(),
			crClient: secretFixture(),
			clientSecretCreds: func(tenantID, clientID, clientSecret string, _ *azidentity.ClientSecretCredentialOptions) (TokenGetter, error) {
				assert.Equal(t, testTenant, tenantID)
				assert.Equal(t, "the-client-id", clientID)
				assert.Equal(t, "the-client-secret", clientSecret)
				return &fakeTokenGetter{t: t, wantScope: adoResource + "/.default", token: azcore.AccessToken{Token: testToken}}, nil
			},
			want: map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			// A resource URI carrying a trailing slash must still yield a
			// single-slash "<resource>/.default" scope (regression for the
			// double-slash bug on the workload-identity path).
			name:     "service principal trailing-slash resource",
			jsonSpec: spSecretSpecResource("https://management.azure.com/"),
			crClient: secretFixture(),
			clientSecretCreds: func(_, _, _ string, _ *azidentity.ClientSecretCredentialOptions) (TokenGetter, error) {
				return &fakeTokenGetter{t: t, wantScope: "https://management.azure.com/.default", token: azcore.AccessToken{Token: testToken}}, nil
			},
			want: map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			name:     "service principal client certificate",
			jsonSpec: spCertSpec(),
			crClient: secretFixture(),
			clientCertCreds: func(tenantID, clientID string, certData []byte, _ *azidentity.ClientCertificateCredentialOptions) (TokenGetter, error) {
				assert.Equal(t, testTenant, tenantID)
				assert.Equal(t, "the-client-id", clientID)
				assert.NotEmpty(t, certData)
				return &fakeTokenGetter{t: t, wantScope: adoResource + "/.default", token: azcore.AccessToken{Token: testToken}}, nil
			},
			want: map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			// Explicit identityType is honored verbatim, regardless of the value's shape.
			name:                 "managed identity explicit clientID",
			jsonSpec:             miSpec(miClientID, "ClientID"),
			managedIdentityCreds: miCreds(t, azidentity.ClientID(miClientID)),
			want:                 map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			// An object id can only be selected explicitly; it is never inferred.
			name:                 "managed identity explicit objectID",
			jsonSpec:             miSpec(miObjectID, "ObjectID"),
			managedIdentityCreds: miCreds(t, azidentity.ObjectID(miObjectID)),
			want:                 map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			name:                 "managed identity explicit resourceID",
			jsonSpec:             miSpec(miResourceID, "ResourceID"),
			managedIdentityCreds: miCreds(t, azidentity.ResourceID(miResourceID)),
			want:                 map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			// No identityType: a value containing "/" is inferred as a resource id.
			name:                 "managed identity infer resourceID from slash",
			jsonSpec:             miSpec(miResourceID, ""),
			managedIdentityCreds: miCreds(t, azidentity.ResourceID(miResourceID)),
			want:                 map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			// No identityType: a bare GUID is inferred as a client id.
			name:                 "managed identity infer clientID from bare guid",
			jsonSpec:             miSpec(miClientID, ""),
			managedIdentityCreds: miCreds(t, azidentity.ClientID(miClientID)),
			want:                 map[string][]byte{tokenKey: []byte(testToken)},
		},
		{
			// No identityType and no identityId: leave opts.ID unset so the SDK falls
			// back to the system-assigned identity.
			name:                 "managed identity system-assigned empty id",
			jsonSpec:             miSpec("", ""),
			managedIdentityCreds: miCreds(t, nil),
			want:                 map[string][]byte{tokenKey: []byte(testToken)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				clientSecretCreds:    tt.clientSecretCreds,
				clientCertCreds:      tt.clientCertCreds,
				managedIdentityCreds: tt.managedIdentityCreds,
			}
			kubeClientFn := func() (kubernetes.Interface, error) {
				return tt.kubeClient, nil
			}
			got, _, err := g.generate(context.Background(), tt.jsonSpec, tt.crClient, "foobar", kubeClientFn)
			if (err != nil) != tt.wantErr {
				t.Errorf("generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScopeForResource(t *testing.T) {
	assert.Equal(t, adoResource+"/.default", scopeForResource(adoResource))
	// a trailing slash on a resource URI must not produce a double slash.
	assert.Equal(t, "https://management.azure.com/.default", scopeForResource("https://management.azure.com/"))
	assert.Equal(t, "https://management.azure.com/.default", scopeForResource("https://management.azure.com"))
}

// miCreds returns a managedIdentityCredentialFunc that asserts the credential options
// carry the expected identity id kind and value produced by the identity-type inference.
// A nil wantID asserts that no id was set (system-assigned identity).
func miCreds(t *testing.T, wantID azidentity.ManagedIDKind) managedIdentityCredentialFunc {
	return func(opts *azidentity.ManagedIdentityCredentialOptions) (TokenGetter, error) {
		if wantID == nil {
			assert.Nil(t, opts.ID)
		} else {
			assert.Equal(t, wantID, opts.ID)
		}
		return &fakeTokenGetter{t: t, wantScope: adoResource + "/.default", token: azcore.AccessToken{Token: testToken}}, nil
	}
}

// fakeTokenGetter asserts that the requested scope matches the expected value.
type fakeTokenGetter struct {
	t         *testing.T
	wantScope string
	token     azcore.AccessToken
	err       error
}

func (f *fakeTokenGetter) GetToken(_ context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if f.wantScope != "" {
		assert.Equal(f.t, []string{f.wantScope}, opts.Scopes)
	}
	return f.token, f.err
}
