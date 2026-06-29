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
)

func spSecretSpec() *apiextensions.JSON {
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
          key: clientsecret`, testTenant, adoResource),
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
		name              string
		jsonSpec          *apiextensions.JSON
		crClient          client.Client
		kubeClient        kubernetes.Interface
		clientSecretCreds clientSecretCredentialFunc
		clientCertCreds   clientCertificateCredentialFunc
		want              map[string][]byte
		wantErr           bool
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
				return &fakeTokenGetter{t: t, token: azcore.AccessToken{Token: testToken}}, nil
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
				return &fakeTokenGetter{t: t, token: azcore.AccessToken{Token: testToken}}, nil
			},
			want: map[string][]byte{tokenKey: []byte(testToken)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				clientSecretCreds: tt.clientSecretCreds,
				clientCertCreds:   tt.clientCertCreds,
			}
			got, _, err := g.generate(context.Background(), tt.jsonSpec, tt.crClient, "foobar", tt.kubeClient)
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
}

// fakeTokenGetter asserts that the requested scope targets the configured resource.
type fakeTokenGetter struct {
	t     *testing.T
	token azcore.AccessToken
	err   error
}

func (f *fakeTokenGetter) GetToken(_ context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	assert.Equal(f.t, []string{adoResource + "/.default"}, opts.Scopes)
	return f.token, f.err
}
