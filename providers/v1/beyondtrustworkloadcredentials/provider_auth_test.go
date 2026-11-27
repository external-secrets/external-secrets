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

package beyondtrustworkloadcredentials

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials/fake"
	btwcutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustworkloadcredentials/util"
)

const (
	testSiteID      = "12345678-1234-4234-8234-123456789012"
	testAPIURL      = "https://api.beyondtrust.io/site"
	testNamespace   = "es-namespace"
	testServiceName = "eso-cluster"
	testSAName      = "eso-wlc"
)

func makeStore(kind string, auth *esv1.BeyondtrustWorkloadCredentialsAuth) esv1.GenericStore {
	spec := esv1.SecretStoreSpec{
		Provider: &esv1.SecretStoreProvider{
			BeyondtrustWorkloadCredentials: &esv1.BeyondtrustWorkloadCredentialsProvider{
				Auth: auth,
				Server: &esv1.BeyondtrustWorkloadCredentialsServer{
					APIURL: testAPIURL,
					SiteID: testSiteID,
				},
			},
		},
	}
	if kind == esv1.ClusterSecretStoreKind {
		return &esv1.ClusterSecretStore{
			Kind: esv1.ClusterSecretStoreKind,
			Spec: spec,
		}
	}
	return &esv1.SecretStore{
		Kind:      esv1.SecretStoreKind,
		Namespace: testNamespace,
		Spec:      spec,
	}
}

func apiKeyAuth() *esv1.BeyondtrustWorkloadCredentialsAuth {
	return &esv1.BeyondtrustWorkloadCredentialsAuth{
		APIKey: &esv1.BeyondtrustWorkloadCredentialsAuthSecretRef{
			Token: esmeta.SecretKeySelector{Name: "api-token", Key: "token"},
		},
	}
}

func oidcAuth(serviceName string, saRef esmeta.ServiceAccountSelector) *esv1.BeyondtrustWorkloadCredentialsAuth {
	return &esv1.BeyondtrustWorkloadCredentialsAuth{
		OIDC: &esv1.BeyondtrustWorkloadCredentialsOIDCAuth{
			ServiceName:       serviceName,
			ServiceAccountRef: saRef,
		},
	}
}

func TestValidateStoreAuth(t *testing.T) {
	tests := []struct {
		name    string
		store   esv1.GenericStore
		wantErr error
		anyErr  bool
	}{
		{
			name:  "api key",
			store: makeStore(esv1.SecretStoreKind, apiKeyAuth()),
		},
		{
			name:  "oidc",
			store: makeStore(esv1.SecretStoreKind, oidcAuth(testServiceName, esmeta.ServiceAccountSelector{Name: testSAName})),
		},
		{
			name:  "oidc with service account in the store namespace",
			store: makeStore(esv1.SecretStoreKind, oidcAuth(testServiceName, esmeta.ServiceAccountSelector{Name: testSAName, Namespace: new(testNamespace)})),
		},
		{
			name:  "oidc on a cluster store with a service account namespace",
			store: makeStore(esv1.ClusterSecretStoreKind, oidcAuth(testServiceName, esmeta.ServiceAccountSelector{Name: testSAName, Namespace: new("eso")})),
		},
		{
			name:    "auth is missing",
			store:   makeStore(esv1.SecretStoreKind, nil),
			wantErr: ErrNoAuth,
		},
		{
			name:    "no auth method is set",
			store:   makeStore(esv1.SecretStoreKind, &esv1.BeyondtrustWorkloadCredentialsAuth{}),
			wantErr: ErrNoAuth,
		},
		{
			name: "both auth methods are set",
			store: makeStore(esv1.SecretStoreKind, &esv1.BeyondtrustWorkloadCredentialsAuth{
				APIKey: apiKeyAuth().APIKey,
				OIDC:   oidcAuth(testServiceName, esmeta.ServiceAccountSelector{Name: testSAName}).OIDC,
			}),
			wantErr: ErrNoAuth,
		},
		{
			name:    "oidc service name is missing",
			store:   makeStore(esv1.SecretStoreKind, oidcAuth("", esmeta.ServiceAccountSelector{Name: testSAName})),
			wantErr: ErrNoServiceName,
		},
		{
			name:    "oidc service name has an invalid format",
			store:   makeStore(esv1.SecretStoreKind, oidcAuth("bad service name!", esmeta.ServiceAccountSelector{Name: testSAName})),
			wantErr: ErrInvalidServiceName,
		},
		{
			name:    "oidc service account name is missing",
			store:   makeStore(esv1.SecretStoreKind, oidcAuth(testServiceName, esmeta.ServiceAccountSelector{})),
			wantErr: ErrNoServiceAccount,
		},
		{
			name:   "oidc service account in another namespace is rejected for a namespaced store",
			store:  makeStore(esv1.SecretStoreKind, oidcAuth(testServiceName, esmeta.ServiceAccountSelector{Name: testSAName, Namespace: new("other")})),
			anyErr: true,
		},
		{
			name:   "oidc service account namespace is required for a cluster store",
			store:  makeStore(esv1.ClusterSecretStoreKind, oidcAuth(testServiceName, esmeta.ServiceAccountSelector{Name: testSAName})),
			anyErr: true,
		},
	}

	p := &Provider{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ValidateStore(tt.store)
			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateStore() error = %v, want %v", err, tt.wantErr)
				}
			case tt.anyErr:
				if err == nil {
					t.Error("ValidateStore() error = nil, want an error")
				}
			default:
				if err != nil {
					t.Errorf("ValidateStore() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewClientOIDC(t *testing.T) {
	tests := []struct {
		name          string
		storeKind     string
		saRef         esmeta.ServiceAccountSelector
		wantSA        string
		wantAudiences []string
	}{
		{
			name:          "audience defaults to the site ID",
			storeKind:     esv1.SecretStoreKind,
			saRef:         esmeta.ServiceAccountSelector{Name: testSAName},
			wantSA:        testNamespace + "/" + testSAName,
			wantAudiences: []string{testSiteID},
		},
		{
			name:          "explicit audiences are used as given",
			storeKind:     esv1.SecretStoreKind,
			saRef:         esmeta.ServiceAccountSelector{Name: testSAName, Audiences: []string{"custom"}},
			wantSA:        testNamespace + "/" + testSAName,
			wantAudiences: []string{"custom"},
		},
		{
			name:          "cluster store uses the service account namespace",
			storeKind:     esv1.ClusterSecretStoreKind,
			saRef:         esmeta.ServiceAccountSelector{Name: testSAName, Namespace: new("eso")},
			wantSA:        "eso/" + testSAName,
			wantAudiences: []string{testSiteID},
		},
		{
			name:          "namespaced store ignores the service account namespace",
			storeKind:     esv1.SecretStoreKind,
			saRef:         esmeta.ServiceAccountSelector{Name: testSAName, Namespace: new("eso")},
			wantSA:        testNamespace + "/" + testSAName,
			wantAudiences: []string{testSiteID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSA string
			var gotRequest *authenticationv1.TokenRequest
			kube := clientfake.NewClientBuilder().
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceCreate: func(_ context.Context, _ kclient.Client, subResourceName string, obj, subResource kclient.Object, _ ...kclient.SubResourceCreateOption) error {
						if subResourceName != "token" {
							t.Errorf("unexpected subresource %q", subResourceName)
						}
						if _, ok := obj.(*corev1.ServiceAccount); !ok {
							t.Errorf("unexpected object type %T", obj)
						}
						gotSA = obj.GetNamespace() + "/" + obj.GetName()
						gotRequest = subResource.(*authenticationv1.TokenRequest)
						gotRequest.Status.Token = "sa-jwt"
						return nil
					},
				}).Build()

			var gotServer, gotToken, gotServiceName string
			p := &Provider{
				NewBeyondtrustWorkloadCredentialsClient: func(server, token, serviceName string) (btwcutil.Client, error) {
					gotServer, gotToken, gotServiceName = server, token, serviceName
					return &fake.BeyondtrustWorkloadCredentialsClient{}, nil
				},
			}

			store := makeStore(tt.storeKind, oidcAuth(testServiceName, tt.saRef))
			if _, err := p.NewClient(context.Background(), store, kube, testNamespace); err != nil {
				t.Fatalf("NewClient() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.wantSA, gotSA); diff != "" {
				t.Errorf("service account mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantAudiences, gotRequest.Spec.Audiences); diff != "" {
				t.Errorf("audiences mismatch (-want +got):\n%s", diff)
			}
			if gotRequest.Spec.ExpirationSeconds == nil || *gotRequest.Spec.ExpirationSeconds != serviceAccountTokenExpirationSeconds {
				t.Errorf("expirationSeconds = %v, want %d", gotRequest.Spec.ExpirationSeconds, serviceAccountTokenExpirationSeconds)
			}
			if diff := cmp.Diff(siteServerURL(testAPIURL, testSiteID), gotServer); diff != "" {
				t.Errorf("server mismatch (-want +got):\n%s", diff)
			}
			if gotToken != "sa-jwt" {
				t.Errorf("token = %q, want %q", gotToken, "sa-jwt")
			}
			if gotServiceName != testServiceName {
				t.Errorf("serviceName = %q, want %q", gotServiceName, testServiceName)
			}
		})
	}
}

func TestNewClientAPIKeyHasNoServiceName(t *testing.T) {
	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		Name:      "api-token",
		Namespace: testNamespace,
		Data:      map[string][]byte{"token": []byte("the-api-key")},
	}).Build()

	var gotToken, gotServiceName string
	p := &Provider{
		NewBeyondtrustWorkloadCredentialsClient: func(_, token, serviceName string) (btwcutil.Client, error) {
			gotToken, gotServiceName = token, serviceName
			return &fake.BeyondtrustWorkloadCredentialsClient{}, nil
		},
	}

	if _, err := p.NewClient(context.Background(), makeStore(esv1.SecretStoreKind, apiKeyAuth()), kube, testNamespace); err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	if gotToken != "the-api-key" {
		t.Errorf("token = %q, want %q", gotToken, "the-api-key")
	}
	if gotServiceName != "" {
		t.Errorf("serviceName = %q, want empty", gotServiceName)
	}
}

func TestNewClientOIDCTokenRequestFails(t *testing.T) {
	kube := clientfake.NewClientBuilder().
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceCreate: func(_ context.Context, _ kclient.Client, _ string, _, _ kclient.Object, _ ...kclient.SubResourceCreateOption) error {
				return errors.New("forbidden")
			},
		}).Build()

	p := &Provider{
		NewBeyondtrustWorkloadCredentialsClient: func(_, _, _ string) (btwcutil.Client, error) {
			t.Fatal("client factory must not be called when the token request fails")
			return nil, nil
		},
	}

	store := makeStore(esv1.SecretStoreKind, oidcAuth(testServiceName, esmeta.ServiceAccountSelector{Name: testSAName}))
	_, err := p.NewClient(context.Background(), store, kube, testNamespace)
	want := "failed to load credentials: failed to create token for service account " + testNamespace + "/" + testSAName + ": forbidden"
	if err == nil || err.Error() != want {
		t.Errorf("NewClient() error = %v, want %q", err, want)
	}
}
