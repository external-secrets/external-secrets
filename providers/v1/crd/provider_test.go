/*
Copyright © 2025 ESO Maintainer Team

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

package crd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// fakeDiscover returns a discoverFn that always succeeds with the given plural name.
func fakeDiscover(plural string) func(*rest.Config, esv1.CRDProviderResource) (string, error) {
	return func(_ *rest.Config, _ esv1.CRDProviderResource) (string, error) {
		return plural, nil
	}
}

// fakeDiscoverErr returns a discoverFn that always fails with the given error.
func fakeDiscoverErr(err error) func(*rest.Config, esv1.CRDProviderResource) (string, error) {
	return func(_ *rest.Config, _ esv1.CRDProviderResource) (string, error) {
		return "", err
	}
}

// providerWithFakeDiscover returns a Provider with a fake discovery function
// and a fake dynamic client injected, bypassing both token fetch and the real cluster.
func providerWithFakeDiscover(plural string) *Provider {
	return &Provider{discoverFn: fakeDiscover(plural)}
}

func makeStoreWithCRDProvider(prov *esv1.CRDProvider) esv1.GenericStore {
	return &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				CRD: prov,
			},
		},
	}
}

// widgetResource is a valid CRDProviderResource used across tests.
var widgetResource = esv1.CRDProviderResource{
	Group:   "example.io",
	Version: "v1alpha1",
	Kind:    "Widget",
}

// newTestClient builds a Client directly, bypassing token fetch and discovery.
// Use this in client-behaviour tests that need a ready-to-use Client.
func newTestClient(store esv1.GenericStore, dynClient dynamic.Interface, plural, namespace string) (*Client, error) {
	provSpec, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	return &Client{
		store:     provSpec,
		dynClient: dynClient,
		namespace: namespace,
		plural:    plural,
	}, nil
}

// fakeDynClient returns a minimal fake dynamic client for a given scheme.
func fakeDynClient() dynamic.Interface {
	return dynfake.NewSimpleDynamicClient(runtime.NewScheme())
}

func TestProviderCapabilities(t *testing.T) {
	p := &Provider{}
	if got := p.Capabilities(); got != esv1.SecretStoreReadOnly {
		t.Fatalf("Capabilities() = %v, want %v", got, esv1.SecretStoreReadOnly)
	}
}

func TestValidateStore(t *testing.T) {
	tests := []struct {
		name    string
		store   esv1.GenericStore
		wantErr error
		wantMsg string
	}{
		{
			name:  "missing provider config is ignored",
			store: &esv1.SecretStore{},
		},
		{
			name:    "missing service account",
			store:   makeStoreWithCRDProvider(&esv1.CRDProvider{Resource: widgetResource}),
			wantErr: errMissingSA,
		},
		{
			name: "missing version",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           esv1.CRDProviderResource{Group: "example.io", Kind: "Widget"},
			}),
			wantErr: errMissingVersion,
		},
		{
			name: "missing kind",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           esv1.CRDProviderResource{Group: "example.io", Version: "v1alpha1"},
			}),
			wantErr: errMissingKind,
		},
		{
			name: "empty group is valid (core resource e.g. ConfigMap)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           esv1.CRDProviderResource{Group: "", Version: "v1", Kind: "ConfigMap"},
			}),
		},
		{
			name: "kind Secret is denied (exact case)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "Secret"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "kind secret is denied (lowercase)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "secret"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "kind SECRET is denied (uppercase)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "SECRET"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "kind sEcReT is denied (mixed case)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "sEcReT"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "invalid whitelist name regex",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           widgetResource,
				Whitelist:          &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{Name: "("}}},
			}),
			wantMsg: "invalid whitelist.rules[0].name regex",
		},
		{
			name: "invalid whitelist property regex",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           widgetResource,
				Whitelist:          &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{Properties: []string{"("}}}},
			}),
			wantMsg: "invalid whitelist.rules[0].properties[0] regex",
		},
		{
			name: "empty whitelist rule is invalid",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           widgetResource,
				Whitelist:          &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{}}},
			}),
			wantErr: errEmptyWhitelistRule,
		},
		{
			name: "valid config",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountName: "reader",
				Resource:           widgetResource,
				Whitelist: &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{
					Name:       "^app-.*$",
					Properties: []string{"^spec\\..+$"},
				}}},
			}),
		},
	}

	p := &Provider{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ValidateStore(tt.store)
			if tt.wantErr == nil && tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("ValidateStore() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateStore() error = nil, want error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateStore() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("ValidateStore() error = %q, want substring %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestGetProvider(t *testing.T) {
	tests := []struct {
		name    string
		store   esv1.GenericStore
		wantErr error
	}{
		{name: "nil store", store: nil, wantErr: errMissingStore},
		{name: "missing provider", store: &esv1.SecretStore{}, wantErr: errMissingCRDProvider},
		{name: "missing crd provider", store: &esv1.SecretStore{Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{}}}, wantErr: errMissingCRDProvider},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getProvider(tt.store)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("getProvider() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("valid store", func(t *testing.T) {
		want := &esv1.CRDProvider{ServiceAccountName: "reader", Resource: widgetResource}
		got, err := getProvider(makeStoreWithCRDProvider(want))
		if err != nil {
			t.Fatalf("getProvider() unexpected error: %v", err)
		}
		if got != want {
			t.Fatalf("getProvider() returned wrong provider pointer")
		}
	})
}

func TestProviderMetadata(t *testing.T) {
	if _, ok := NewProvider().(*Provider); !ok {
		t.Fatalf("NewProvider() did not return *Provider")
	}

	spec := ProviderSpec()
	if spec == nil || spec.CRD == nil {
		t.Fatalf("ProviderSpec() returned nil CRD provider")
	}

	if got := MaintenanceStatus(); got != esv1.MaintenanceStatusMaintained {
		t.Fatalf("MaintenanceStatus() = %v, want %v", got, esv1.MaintenanceStatusMaintained)
	}
}

func TestNewClientInternal(t *testing.T) {
	ctx := context.Background()

	t.Run("newClient returns getProvider error on nil store", func(t *testing.T) {
		p := providerWithFakeDiscover("widgets")
		_, err := p.newClient(ctx, nil, nil, &rest.Config{Host: "https://example.com"}, "default")
		if !errors.Is(err, errMissingStore) {
			t.Fatalf("newClient() error = %v, want %v", err, errMissingStore)
		}
	})

	t.Run("newClientFromToken returns getProvider error on nil store", func(t *testing.T) {
		p := providerWithFakeDiscover("widgets")
		_, err := p.newClientFromToken(nil, &rest.Config{Host: "https://example.com"}, "tok", "default")
		if !errors.Is(err, errMissingStore) {
			t.Fatalf("newClientFromToken() error = %v, want %v", err, errMissingStore)
		}
	})

	t.Run("discovery error is propagated", func(t *testing.T) {
		discErr := fmt.Errorf("group/version not registered")
		p := &Provider{discoverFn: fakeDiscoverErr(discErr)}
		store := makeStoreWithCRDProvider(&esv1.CRDProvider{ServiceAccountName: "reader", Resource: widgetResource})
		_, err := p.newClientFromToken(store, &rest.Config{Host: "https://example.com"}, "tok", "default")
		if !errors.Is(err, discErr) {
			t.Fatalf("newClientFromToken() error = %v, want %v", err, discErr)
		}
	})

	t.Run("returns dynamic client creation error on bad host", func(t *testing.T) {
		p := providerWithFakeDiscover("widgets")
		store := makeStoreWithCRDProvider(&esv1.CRDProvider{ServiceAccountName: "reader", Resource: widgetResource})
		_, err := p.newClientFromToken(store, &rest.Config{Host: "://bad-host"}, "tok", "default")
		if err == nil {
			t.Fatalf("newClientFromToken() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "failed to create dynamic client") {
			t.Fatalf("newClientFromToken() error = %q, want dynamic client creation error", err.Error())
		}
	})

	t.Run("creates client for namespaced store with discovered plural", func(t *testing.T) {
		p := providerWithFakeDiscover("widgets")
		store := makeStoreWithCRDProvider(&esv1.CRDProvider{ServiceAccountName: "reader", Resource: widgetResource})
		client, err := p.newClientFromToken(store, &rest.Config{Host: "https://example.com"}, "tok", "app-ns")
		if err != nil {
			t.Fatalf("newClientFromToken() unexpected error: %v", err)
		}
		c, ok := client.(*Client)
		if !ok {
			t.Fatalf("newClientFromToken() returned %T, want *Client", client)
		}
		if c.store.ServiceAccountName != "reader" {
			t.Fatalf("client store mismatch, got %q", c.store.ServiceAccountName)
		}
		if c.namespace != "app-ns" {
			t.Fatalf("client namespace = %q, want %q", c.namespace, "app-ns")
		}
		if c.plural != "widgets" {
			t.Fatalf("client plural = %q, want %q", c.plural, "widgets")
		}
		if c.dynClient == nil {
			t.Fatalf("client dynClient is nil")
		}
	})

	t.Run("creates client for cluster store (empty namespace)", func(t *testing.T) {
		p := providerWithFakeDiscover("widgets")
		store := makeStoreWithCRDProvider(&esv1.CRDProvider{ServiceAccountName: "reader", Resource: widgetResource})
		client, err := p.newClientFromToken(store, &rest.Config{Host: "https://example.com"}, "tok", "")
		if err != nil {
			t.Fatalf("newClientFromToken() unexpected error: %v", err)
		}
		c, ok := client.(*Client)
		if !ok {
			t.Fatalf("newClientFromToken() returned %T, want *Client", client)
		}
		if c.namespace != "" {
			t.Fatalf("client namespace = %q, want empty", c.namespace)
		}
	})

	t.Run("plural from discovery is used in GVR", func(t *testing.T) {
		// Verify the server-resolved plural (not a heuristic) reaches the Client.
		p := providerWithFakeDiscover("mycustomwidgets")
		store := makeStoreWithCRDProvider(&esv1.CRDProvider{ServiceAccountName: "reader", Resource: widgetResource})
		client, err := p.newClientFromToken(store, &rest.Config{Host: "https://example.com"}, "tok", "default")
		if err != nil {
			t.Fatalf("newClientFromToken() unexpected error: %v", err)
		}
		c := client.(*Client)
		gvr := c.buildGVR()
		if gvr.Resource != "mycustomwidgets" {
			t.Fatalf("buildGVR() resource = %q, want %q", gvr.Resource, "mycustomwidgets")
		}
		if gvr.Group != widgetResource.Group {
			t.Fatalf("buildGVR() group = %q, want %q", gvr.Group, widgetResource.Group)
		}
		if gvr.Version != widgetResource.Version {
			t.Fatalf("buildGVR() version = %q, want %q", gvr.Version, widgetResource.Version)
		}
	})
}
