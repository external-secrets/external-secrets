/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package acr

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

func TestGenerate(t *testing.T) {
	const (
		testUsername = "11111111-2222-3333-4444-111111111111"
		testURL      = "example.azurecr.io"
	)
	type args struct {
		ctx                 context.Context
		jsonSpec            *apiextensions.JSON
		crClient            client.Client
		kubeClient          kubernetes.Interface
		namespace           string
		accessTokenFetcher  accessTokenFetcher
		refreshTokenFetcher refreshTokenFetcher
		clientSecretCreds   clientSecretCredentialFunc
	}
	tests := []struct {
		name    string
		g       *Generator
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{
			name: "no spec",
			args: args{
				jsonSpec: nil,
			},
			wantErr: true,
		},
		{
			name: "empty spec",
			args: args{
				jsonSpec: &apiextensions.JSON{},
			},
			wantErr: true,
		},
		{
			name: "return acr access token if scope is defined",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(fmt.Sprintf(`apiVersion: generators.external-secrets.io/v1alpha1
kind: ACRAccessToken
spec:
  tenantId: %s
  registry: %s
  scope: "repository:foo:pull,push"
  environmentType: "PublicCloud"
  auth:
    servicePrincipal:
      secretRef:
        clientSecret:
          name: az-secret
          key: clientsecret
        clientId:
          name: az-secret
          key: clientid`, testUsername, testURL)),
				},
				crClient: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "az-secret",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"clientsecret": []byte("foo"),
						"clientid":     []byte("bar"),
					},
				}).Build(),
				namespace: "foobar",
				ctx:       context.Background(),
				accessTokenFetcher: func(acrRefreshToken, tenantID, registryURL, scope string) (string, error) {
					assert.Equal(t, "acrrefreshtoken", acrRefreshToken)
					assert.Equal(t, tenantID, testUsername)
					assert.Equal(t, registryURL, testURL)
					assert.Equal(t, scope, "repository:foo:pull,push")
					return "acraccesstoken", nil
				},
				refreshTokenFetcher: func(aadAccessToken, tenantID, registryURL string) (string, error) {
					assert.Equal(t, "1234", aadAccessToken)
					assert.Equal(t, tenantID, testUsername)
					assert.Equal(t, registryURL, testURL)
					return "acrrefreshtoken", nil
				},
				clientSecretCreds: func(tenantID, clientID, clientSecret string, options *azidentity.ClientSecretCredentialOptions) (TokenGetter, error) {
					return &FakeTokenGetter{
						token: azcore.AccessToken{
							Token: "1234",
						},
					}, nil
				},
			},
			want: map[string][]byte{
				"username": []byte(defaultLoginUsername),
				"password": []byte("acraccesstoken"),
			},
		},
		{
			name: "return acr refresh token if scope is not defined",
			args: args{
				jsonSpec: &apiextensions.JSON{
					Raw: []byte(fmt.Sprintf(`apiVersion: generators.external-secrets.io/v1alpha1
kind: ACRAccessToken
spec:
  tenantId: %s
  registry: %s
  environmentType: "PublicCloud"
  auth:
    servicePrincipal:
      secretRef:
        clientSecret:
          name: az-secret
          key: clientsecret
        clientId:
          name: az-secret
          key: clientid`, testUsername, testURL)),
				},
				crClient: clientfake.NewClientBuilder().WithObjects(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "az-secret",
						Namespace: "foobar",
					},
					Data: map[string][]byte{
						"clientsecret": []byte("foo"),
						"clientid":     []byte("bar"),
					},
				}).Build(),
				namespace: "foobar",
				ctx:       context.Background(),
				accessTokenFetcher: func(acrRefreshToken, tenantID, registryURL, scope string) (string, error) {
					t.Fail()
					return "", nil
				},
				refreshTokenFetcher: func(aadAccessToken, tenantID, registryURL string) (string, error) {
					assert.Equal(t, "1234", aadAccessToken)
					assert.Equal(t, tenantID, testUsername)
					assert.Equal(t, registryURL, testURL)
					return "acrrefreshtoken", nil
				},
				clientSecretCreds: func(tenantID, clientID, clientSecret string, options *azidentity.ClientSecretCredentialOptions) (TokenGetter, error) {
					return &FakeTokenGetter{
						token: azcore.AccessToken{
							Token: "1234",
						},
					}, nil
				},
			},
			want: map[string][]byte{
				"username": []byte(defaultLoginUsername),
				"password": []byte("acrrefreshtoken"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				clientSecretCreds: tt.args.clientSecretCreds,
			}
			got, err := g.generate(
				tt.args.ctx,
				tt.args.jsonSpec,
				tt.args.crClient,
				tt.args.namespace,
				tt.args.kubeClient,
				tt.args.accessTokenFetcher,
				tt.args.refreshTokenFetcher,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generator.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Generator.Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}

type FakeTokenGetter struct {
	token azcore.AccessToken
	err   error
}

func (f *FakeTokenGetter) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return f.token, f.err
}
