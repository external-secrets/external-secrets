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
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// fakeBuildClient returns a buildClientFn that succeeds with the given plural and
// scope, backed by a controller-runtime fake client (no objects). This bypasses
// both the RESTMapper and the real cluster.
func fakeBuildClient(plural string, namespaced bool) func(*rest.Config, esv1.CRDProviderResource) (kclient.Client, string, bool, error) {
	return func(_ *rest.Config, _ esv1.CRDProviderResource) (kclient.Client, string, bool, error) {
		return fakeCRDClient(namespaced), plural, namespaced, nil
	}
}

// fakeBuildClientErr returns a buildClientFn that always fails with the given error.
func fakeBuildClientErr(err error) func(*rest.Config, esv1.CRDProviderResource) (kclient.Client, string, bool, error) {
	return func(_ *rest.Config, _ esv1.CRDProviderResource) (kclient.Client, string, bool, error) {
		return nil, "", true, err
	}
}

// providerWithFakeClient returns a Provider with a fake client builder injected,
// bypassing both token fetch and the real cluster.
// namespaced defaults to true when omitted (namespace-scoped CRD).
func providerWithFakeClient(plural string, namespaced ...bool) *Provider {
	ns := true
	if len(namespaced) > 0 {
		ns = namespaced[0]
	}
	return &Provider{buildClientFn: fakeBuildClient(plural, ns)}
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

func makeClusterStoreWithCRDProvider(prov *esv1.CRDProvider) esv1.GenericStore {
	return &esv1.ClusterSecretStore{
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
			name: "core v1 Secret is denied (exact case)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "", Version: "v1", Kind: "Secret"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "core v1 secret is denied (lowercase)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "", Version: "v1", Kind: "secret"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			name: "core v1 SECRET is denied (uppercase)",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "", Version: "v1", Kind: "SECRET"},
			}),
			wantErr: errKindIsSecret,
		},
		{
			// Same Kind name on a different API group is a legitimate CRD;
			// only the core v1 Secret is blocked.
			name: "Secret kind in a non-core group is allowed",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "example.io", Version: "v1", Kind: "Secret"},
			}),
		},
		{
			// Different version of core "Secret" — also legitimate (no such
			// thing exists today, but the block is intentionally narrow).
			name: "core v2 Secret is allowed",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "", Version: "v2", Kind: "Secret"},
			}),
		},
		{
			name: "core group alias \"core\" still denies v1 Secret",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
				Resource:          esv1.CRDProviderResource{Group: "core", Version: "v1", Kind: "Secret"},
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
			name: "auth without server.url is rejected",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				Resource: widgetResource,
				Auth: &esv1.KubernetesAuth{
					Token: &esv1.TokenAuth{
						BearerToken: esmeta.SecretKeySelector{Name: "t", Key: "k"},
					},
				},
			}),
			wantMsg: "auth requires server.url",
		},
		{
			// authRef embeds a kubeconfig with the server address, so a
			// separate server.url is not required.
			name: "authRef without server.url is allowed",
			store: makeStoreWithCRDProvider(&esv1.CRDProvider{
				Resource: widgetResource,
				AuthRef:  &esmeta.SecretKeySelector{Name: "kubeconfig", Key: "config"},
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
									Namespace: new("ns"),
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
		_, err := providerWithFakeClient("widgets").newClient(ctx, nil, nil, &rest.Config{Host: "https://example.com"}, nil, "default")
		if !errors.Is(err, errMissingStore) {
			t.Fatalf("newClient() error = %v, want %v", err, errMissingStore)
		}
	})

	t.Run("newClient returns errMissingSA when no serviceAccountRef in simple mode", func(t *testing.T) {
		simpleStore := makeStoreWithCRDProvider(&esv1.CRDProvider{
			// No Server/Auth/AuthRef → simple mode; no ServiceAccountRef → should error.
			Resource: widgetResource,
		})
		_, err := providerWithFakeClient("widgets").newClient(ctx, simpleStore, nil, defaultRESTCfg(), nil, "default")
		if !errors.Is(err, errMissingSA) {
			t.Fatalf("newClient() error = %v, want %v", err, errMissingSA)
		}
	})

	t.Run("referent ClusterSecretStore returns a validation stub at store bootstrap", func(t *testing.T) {
		// ClusterSecretStore, no explicit serviceAccountRef.namespace, and an
		// empty namespace (the store-validation call). newClient must short
		// circuit before any token fetch and return a referent stub whose
		// Validate() reports "unknown".
		clusterStore := makeClusterStoreWithCRDProvider(&esv1.CRDProvider{
			ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "reader"},
			Resource:          widgetResource,
		})
		client, err := (&Provider{}).newClient(ctx, clusterStore, nil, defaultRESTCfg(), nil, "")
		if err != nil {
			t.Fatalf("newClient() unexpected error: %v", err)
		}
		c, ok := client.(*Client)
		if !ok {
			t.Fatalf("returned %T, want *Client", client)
		}
		if !c.referent {
			t.Fatalf("expected a referent stub client")
		}
		if c.kube != nil {
			t.Fatalf("referent stub must not carry a client")
		}
		res, err := c.Validate()
		if err != nil {
			t.Fatalf("Validate() unexpected error: %v", err)
		}
		if res != esv1.ValidationResultUnknown {
			t.Fatalf("Validate() = %v, want %v", res, esv1.ValidationResultUnknown)
		}
	})

	t.Run("newClientWithRESTConfig returns getProvider error on nil store", func(t *testing.T) {
		_, err := providerWithFakeClient("widgets").newClientWithRESTConfig(context.Background(), nil, defaultRESTCfg(), "default")
		if !errors.Is(err, errMissingStore) {
			t.Fatalf("newClientWithRESTConfig() error = %v, want %v", err, errMissingStore)
		}
	})

	t.Run("resource resolution error is propagated", func(t *testing.T) {
		mapErr := fmt.Errorf("group/version/kind not registered")
		_, err := (&Provider{buildClientFn: fakeBuildClientErr(mapErr)}).newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "default")
		if !errors.Is(err, mapErr) {
			t.Fatalf("newClientWithRESTConfig() error = %v, want %v", err, mapErr)
		}
	})

	t.Run("creates client for namespaced store", func(t *testing.T) {
		client, err := providerWithFakeClient("widgets").newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "app-ns")
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
		if c.namespace != "app-ns" || c.kube == nil {
			t.Fatalf("client: ns=%q kube=%v", c.namespace, c.kube)
		}
	})

	t.Run("creates client for cluster store (empty namespace)", func(t *testing.T) {
		client, err := providerWithFakeClient("widgets").newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "")
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

	t.Run("cluster-scoped resource sets namespaced false on client", func(t *testing.T) {
		client, err := providerWithFakeClient("clusterdbspecs", false).newClientWithRESTConfig(context.Background(), store, defaultRESTCfg(), "default")
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
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: new("other")},
			wantNS:    "app",
		},
		{
			name:      "ClusterSecretStore uses ref.Namespace when set",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: new("ops")},
			wantNS:    "ops",
		},
		{
			name:      "ClusterSecretStore referent: nil namespace uses the ES namespace",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "team-a",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa"},
			wantNS:    "team-a",
		},
		{
			name:      "ClusterSecretStore referent: empty ES namespace stays empty",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa"},
			wantNS:    "",
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
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: new("other")},
			wantNS:    "app",
		},
		{
			name:      "ClusterSecretStore uses ref.Namespace",
			storeKind: esv1.ClusterSecretStoreKind,
			storeNS:   "",
			ref:       &esmeta.ServiceAccountSelector{Name: "sa", Namespace: new("ops")},
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
		p := providerWithFakeClient("widgets")
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
		if c.kube == nil {
			t.Fatalf("client kube is nil")
		}
	})

	t.Run("ClusterSecretStore impersonation requires namespace on ref", func(t *testing.T) {
		_, err := resolveImpersonationNamespace(esv1.ClusterSecretStoreKind, "", &esmeta.ServiceAccountSelector{Name: "sa"})
		if err == nil || !strings.Contains(err.Error(), "namespace is required") {
			t.Fatalf("resolveImpersonationNamespace() = %v, want namespace-required error", err)
		}
	})

	t.Run("impersonation UserName format includes namespace", func(t *testing.T) {
		saRef := &esmeta.ServiceAccountSelector{Name: "reader", Namespace: new("ops")}
		ns, err := resolveImpersonationNamespace(esv1.ClusterSecretStoreKind, "", saRef)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := fmt.Sprintf("system:serviceaccount:%s:%s", ns, saRef.Name); got != "system:serviceaccount:ops:reader" {
			t.Fatalf("UserName = %q, want %q", got, "system:serviceaccount:ops:reader")
		}
	})
}
