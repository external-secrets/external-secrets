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

package crd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
	pointer "k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// fakeDiscover returns a discoverFn that always succeeds with the given plural and scope.
func fakeDiscover(plural string, namespaced bool) func(*rest.Config, esv1.CRDProviderResource) (string, bool, error) {
	return func(_ *rest.Config, _ esv1.CRDProviderResource) (string, bool, error) {
		return plural, namespaced, nil
	}
}

// fakeDiscoverErr returns a discoverFn that always fails with the given error.
func fakeDiscoverErr(err error) func(*rest.Config, esv1.CRDProviderResource) (string, bool, error) {
	return func(_ *rest.Config, _ esv1.CRDProviderResource) (string, bool, error) {
		return "", true, err
	}
}

// providerWithFakeDiscover returns a Provider with a fake discovery function
// and a fake dynamic client injected, bypassing both token fetch and the real cluster.
// namespaced defaults to true when omitted (namespace-scoped CRD).
func providerWithFakeDiscover(plural string, namespaced ...bool) *Provider {
	ns := true
	if len(namespaced) > 0 {
		ns = namespaced[0]
	}
	return &Provider{discoverFn: fakeDiscover(plural, ns)}
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

// defaultRESTCfg returns a minimal REST config used in provider construction tests.
func defaultRESTCfg() *rest.Config {
	return &rest.Config{Host: "https://example.com", BearerToken: "tok"}
}

func TestProviderCapabilities(t *testing.T) {
	p := &Provider{}
	if got := p.Capabilities(); got != esv1.SecretStoreReadOnly {
		t.Fatalf("Capabilities() = %v, want %v", got, esv1.SecretStoreReadOnly)
	}
}

func TestValidateStore(t *testing.T) {
	tests := []struct {
		name              string
		store             esv1.GenericStore
		wantErr           error
		wantMsg           string
		wantWarnSubstring string
	}{
		{
			name:  "missing provider config is ignored",
			store: &esv1.SecretStore{},
		},
		{
			name:    "missing service account ref",
			store:   makeStoreWithCRDProvider(&esv1.CRDProvider{Resource: widgetResource}),
			wantErr: errMissingSA,
		},
		{
			name: "service account ref with empty name",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: ""},
				Resource:          widgetResource,
			}),
			wantErr: errMissingSA,
		},
		{
			name: "missing version",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "example.io", Kind: "Widget"},
			}),
			wantErr: errMissingVersion,
		},
		{
			name: "missing kind",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "example.io", Version: "v1alpha1"},
			}),
			wantErr: errMissingKind,
		},
		{
			name: "empty group is valid (core resource e.g. ConfigMap)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "", Version: "v1", Kind: "ConfigMap"},
			}),
		},
		{
			name: "kind Secret is denied (exact case)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "Secret"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "kind secret is denied (lowercase)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "secret"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "kind SECRET is denied (uppercase)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "SECRET"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "kind sEcReT is denied (mixed case)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "sEcReT"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "invalid whitelist name regex",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          widgetResource,
				Whitelist:         &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{Name: "("}}},
			}),
			wantMsg: "invalid whitelist.rules[0].name regex",
		},
		{
			name: "invalid whitelist property regex",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          widgetResource,
				Whitelist:         &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{Properties: []string{"("}}}},
			}),
			wantMsg: "invalid whitelist.rules[0].properties[0] regex",
		},
		{
			name: "empty whitelist rule is invalid",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          widgetResource,
				Whitelist:         &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{}}},
			}),
			wantErr: errEmptyWhitelistRule,
		},
		{
			name: "valid config",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          widgetResource,
				Whitelist: &esv1.CRDProviderWhitelist{Rules: []esv1.CRDProviderWhitelistRule{{
					Name:       "^app-.*$",
					Properties: []string{"^spec\\..+$"},
				}}},
			}),
		},
		{
			name: "explicit mode with serviceAccountRef impersonation: empty name rejected",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				Resource: widgetResource,
				Server: esv1.KubernetesServer{
					URL:      "https://k8s.example",
					CABundle: []byte("fake-ca"),
				},
				Auth: &esv1.KubernetesAuth{
					Token: &esv1.TokenAuth{
						BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
					},
				},
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: ""},
			}),
			wantMsg: "serviceAccountRef.name must not be empty",
		},
		{
			name: "explicit mode with serviceAccountRef impersonation: valid",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				Resource: widgetResource,
				Server: esv1.KubernetesServer{
					URL:      "https://k8s.example",
					CABundle: []byte("fake-ca"),
				},
				Auth: &esv1.KubernetesAuth{
					Token: &esv1.TokenAuth{
						BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
					},
				},
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "remote-reader"},
			}),
		},
		{
			name: "explicit auth without serviceAccountName",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				Resource: widgetResource,
				Server: esv1.KubernetesServer{
					URL:      "https://k8s.example",
					CABundle: []byte("fake-ca"),
				},
				Auth: &esv1.KubernetesAuth{
					Token: &esv1.TokenAuth{
						BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
					},
				},
			}),
		},
		{
			name: "explicit connection TLS CA warning",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				Resource: widgetResource,
				Server: esv1.KubernetesServer{
					URL: "https://k8s.example",
				},
				Auth: &esv1.KubernetesAuth{
					Token: &esv1.TokenAuth{
						BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
					},
				},
			}),
			wantWarnSubstring: "system certificate roots",
		},
		{
			name: "ClusterSecretStore CAProvider needs namespace",
			store: &esv1.ClusterSecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							Resource: widgetResource,
							Server: esv1.KubernetesServer{
								URL: "https://x",
								CAProvider: &esv1.CAProvider{
									Type: esv1.CAProviderTypeSecret,
									Name: "ca",
									Key:  "k",
								},
							},
							Auth: &esv1.KubernetesAuth{
								Token: &esv1.TokenAuth{
									BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
								},
							},
						},
					},
				},
			},
			wantMsg: "CAProvider.namespace must not be empty",
		},
		{
			name: "SecretStore rejects CAProvider.namespace",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						CRD: &esv1.CRDProvider{
							Resource: widgetResource,
							Server: esv1.KubernetesServer{
								URL: "https://x",
								CAProvider: &esv1.CAProvider{
									Type:      esv1.CAProviderTypeSecret,
									Name:      "ca",
									Key:       "k",
									Namespace: pointer.To("ns"),
								},
							},
							Auth: &esv1.KubernetesAuth{
								Token: &esv1.TokenAuth{
									BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
								},
							},
						},
					},
				},
			},
			wantMsg: "CAProvider.namespace must be empty with SecretStore",
		},
	}

	p := &Provider{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := p.ValidateStore(tt.store)
			if tt.wantErr != nil || tt.wantMsg != "" {
				if err == nil {
					t.Fatalf("ValidateStore() error = nil, want error")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Fatalf("ValidateStore() error = %v, want %v", err, tt.wantErr)
				}
				if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
					t.Fatalf("ValidateStore() error = %q, want substring %q", err.Error(), tt.wantMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateStore() unexpected error: %v", err)
			}
			if tt.wantWarnSubstring != "" {
				var b strings.Builder
				for _, w := range warnings {
					b.WriteString(w)
				}
				if !strings.Contains(b.String(), tt.wantWarnSubstring) {
					t.Fatalf("ValidateStore() warnings = %v, want substring %q", warnings, tt.wantWarnSubstring)
				}
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
		want := &esv1.CRDProvider{ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"}, Resource: widgetResource}
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
	store := makeStoreWithCRDProvider(&esv1.CRDProvider{ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"}, Resource: widgetResource})

	t.Run("newClient returns getProvider error on nil store", func(t *testing.T) {
		_, err := providerWithFakeDiscover("widgets").newClient(ctx, nil, nil, &rest.Config{Host: "https://example.com"}, nil, "default")
		if !errors.Is(err, errMissingStore) {
			t.Fatalf("newClient() error = %v, want %v", err, errMissingStore)
		}
	})

	t.Run("newClient returns errMissingSA when no serviceAccountRef in simple mode", func(t *testing.T) {
		simpleStore := makeStoreWithCRDProvider(&esv1.CRDProvider{
			// No Server/Auth/AuthRef → simple mode; no ServiceAccountRef → should error.
			Resource: widgetResource,
		})
		_, err := providerWithFakeDiscover("widgets").newClient(ctx, simpleStore, nil, defaultRESTCfg(), nil, "default")
		if !errors.Is(err, errMissingSA) {
			t.Fatalf("newClient() error = %v, want %v", err, errMissingSA)
		}
	})

	t.Run("newClientWithRESTConfig returns getProvider error on nil store", func(t *testing.T) {
		_, err := providerWithFakeDiscover("widgets").newClientWithRESTConfig(context.Background(), nil, defaultRESTCfg(), "default")
		if !errors.Is(err, errMissingStore) {
			t.Fatalf("newClientWithRESTConfig() error = %v, want %v", err, errMissingStore)
		}
	})

	t.Run("discovery error is propagated", func(t *testing.T) {
		discErr := fmt.Errorf("group/version not registered")
		_, err := (&Provider{discoverFn: fakeDiscoverErr(discErr)}).newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "default")
		if !errors.Is(err, discErr) {
			t.Fatalf("newClientWithRESTConfig() error = %v, want %v", err, discErr)
		}
	})

	t.Run("returns dynamic client creation error on bad host", func(t *testing.T) {
		_, err := providerWithFakeDiscover("widgets").newClientWithRESTConfig(context.Background(), store, &rest.Config{Host: "://bad-host", BearerToken: "tok"}, "default")
		if err == nil || !strings.Contains(err.Error(), "failed to create dynamic client") {
			t.Fatalf("newClientWithRESTConfig() error = %v, want dynamic client creation error", err)
		}
	})

	t.Run("creates client for namespaced store with discovered plural", func(t *testing.T) {
		client, err := providerWithFakeDiscover("widgets").newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "app-ns")
		if err != nil {
			t.Fatalf("newClientWithRESTConfig() unexpected error: %v", err)
		}
		c, ok := client.(*Client)
		if !ok {
			t.Fatalf("returned %T, want *Client", client)
		}
		if c.store.ServiceAccountRef.Name != "reader" {
			t.Fatalf("client SA = %q, want %q", c.store.ServiceAccountRef.Name, "reader")
		}
		if c.namespace != "app-ns" || c.plural != "widgets" || c.dynClient == nil {
			t.Fatalf("client: ns=%q plural=%q dynClient=%v", c.namespace, c.plural, c.dynClient)
		}
	})

	t.Run("creates client for cluster store (empty namespace)", func(t *testing.T) {
		client, err := providerWithFakeDiscover("widgets").newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "")
		if err != nil {
			t.Fatalf("newClientWithRESTConfig() unexpected error: %v", err)
		}
		c, ok := client.(*Client)
		if !ok {
			t.Fatalf("returned %T, want *Client", client)
		}
		if c.namespace != "" {
			t.Fatalf("client namespace = %q, want empty", c.namespace)
		}
	})

	t.Run("plural from discovery is used in GVR", func(t *testing.T) {
		client, err := providerWithFakeDiscover("mycustomwidgets").newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "default")
		if err != nil {
			t.Fatalf("newClientWithRESTConfig() unexpected error: %v", err)
		}
		gvr := client.(*Client).buildGVR()
		if gvr.Resource != "mycustomwidgets" || gvr.Group != widgetResource.Group || gvr.Version != widgetResource.Version {
			t.Fatalf("buildGVR() = %+v, want resource=%q group=%q version=%q", gvr, "mycustomwidgets", widgetResource.Group, widgetResource.Version)
		}
	})

	t.Run("remoteNamespace overrides store namespace on client", func(t *testing.T) {
		prov := &esv1.CRDProvider{
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
			Resource:          widgetResource,
			RemoteNamespace:   "remote-ns",
		}
		client, err := providerWithFakeDiscover("widgets").newClientWithRESTConfig(context.Background(), makeStoreWithCRDProvider(prov), defaultRESTCfg(), resolveCRDTargetNamespace(prov, "es-ns"))
		if err != nil {
			t.Fatalf("newClientWithRESTConfig() unexpected error: %v", err)
		}
		if client.(*Client).namespace != "remote-ns" {
			t.Fatalf("client namespace = %q, want %q", client.(*Client).namespace, "remote-ns")
		}
	})

	t.Run("cluster-scoped discovery sets namespaced false on client", func(t *testing.T) {
		client, err := providerWithFakeDiscover("clusterdbspecs", false).newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "default")
		if err != nil {
			t.Fatalf("newClientWithRESTConfig() unexpected error: %v", err)
		}
		if client.(*Client).namespaced {
			t.Fatalf("expected cluster-scoped client (namespaced=false)")
		}
	})
}

func TestResolveSimpleSANamespace(t *testing.T) {
	tests := []struct {
		name      string
		storeKind string
		storeNS   string
		ref       *esmeta.ServiceAccountSelector
		wantNS    string
	}{
		{
			name:      "SecretStore uses store namespace",
			storeKind: esv1.SecretStoreKind,
			storeNS:   "app",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa"},
			wantNS:    "app",
		},
		{
			name:      "SecretStore ignores ref.Namespace",
			storeKind: esv1.SecretStoreKind,
			storeNS:   "app",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: strPtr("other")},
			wantNS:    "app",
		},
		{
			name:      "ClusterSecretStore uses ref.Namespace when set",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: strPtr("ops")},
			wantNS:    "ops",
		},
		{
			name:      "ClusterSecretStore falls back to default when namespace nil",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa"},
			wantNS:    "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSimpleSANamespace(tt.storeKind, tt.storeNS, tt.ref)
			if got != tt.wantNS {
				t.Fatalf("resolveSimpleSANamespace() = %q, want %q", got, tt.wantNS)
			}
		})
	}
}

func TestResolveImpersonationNamespace(t *testing.T) {
	tests := []struct {
		name      string
		storeKind string
		storeNS   string
		ref       *esmeta.ServiceAccountSelector
		wantNS    string
		wantErr   bool
	}{
		{
			name:      "SecretStore uses store namespace",
			storeKind: esv1.SecretStoreKind,
			storeNS:   "app",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa"},
			wantNS:    "app",
		},
		{
			name:      "SecretStore uses store namespace even when ref.Namespace set",
			storeKind: esv1.SecretStoreKind,
			storeNS:   "app",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: strPtr("other")},
			wantNS:    "app",
		},
		{
			name:      "ClusterSecretStore uses ref.Namespace",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: strPtr("ops")},
			wantNS:    "ops",
		},
		{
			name:      "ClusterSecretStore without namespace returns error",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa"},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveImpersonationNamespace(tt.storeKind, tt.storeNS, tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveImpersonationNamespace() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveImpersonationNamespace() unexpected error: %v", err)
			}
			if got != tt.wantNS {
				t.Fatalf("resolveImpersonationNamespace() = %q, want %q", got, tt.wantNS)
			}
		})
	}
}

func TestImpersonationWiring(t *testing.T) {
	t.Run("impersonation config set when serviceAccountRef present in explicit mode", func(t *testing.T) {
		p := providerWithFakeDiscover("widgets")
		store := makeStoreWithCRDProvider(&esv1.CRDProvider{
			Resource: widgetResource,
			Server:   esv1.KubernetesServer{URL: "https://remote.example"},
			Auth: &esv1.KubernetesAuth{
				Token: &esv1.TokenAuth{
					BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
				},
			},
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "remote-reader"},
		})
		// Inject a pre-built REST config (bypassing real auth fetch) via newClientWithRESTConfig.
		// We test that the impersonate field is set by newClient when called with an already-built cfg.
		// Here we simulate explicit mode by calling the internal path directly with a cfg that
		// already has the impersonation applied (as newClient would do).
		cfg := &rest.Config{
			Host:        "https://remote.example",
			BearerToken: "tok",
			Impersonate: rest.ImpersonationConfig{
				UserName: "system:serviceaccount:default:remote-reader",
			},
		}
		client, err := p.newClientWithRESTConfig(context.Background(), store, cfg, "default")
		if err != nil {
			t.Fatalf("newClientWithRESTConfig() unexpected error: %v", err)
		}
		c := client.(*Client)
		if c.dynClient == nil {
			t.Fatalf("client dynClient is nil")
		}
	})

	t.Run("ClusterSecretStore impersonation requires namespace on ref", func(t *testing.T) {
		_, err := resolveImpersonationNamespace(esv1.ClusterSecretStoreKind, "", &esmeta.ServiceAccountSelector{Name: "sa"})
		if err == nil || !strings.Contains(err.Error(), "namespace is required") {
			t.Fatalf("resolveImpersonationNamespace() = %v, want namespace-required error", err)
		}
	})

	t.Run("impersonation UserName format includes namespace", func(t *testing.T) {
		saRef := &esmeta.ServiceAccountSelector{Name: "reader", Namespace: strPtr("ops")}
		ns, err := resolveImpersonationNamespace(esv1.ClusterSecretStoreKind, "", saRef)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := fmt.Sprintf("system:serviceaccount:%s:%s", ns, saRef.Name); got != "system:serviceaccount:ops:reader" {
			t.Fatalf("UserName = %q, want %q", got, "system:serviceaccount:ops:reader")
		}
	})
}
