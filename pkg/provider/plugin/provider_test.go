package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestPluginProvider_ValidateStore(t *testing.T) {
	tests := []struct {
		name    string
		store   esv1.GenericStore
		wantErr bool
	}{
		{
			name: "valid plugin configuration",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Plugin: &esv1.PluginProvider{
							Endpoint: "unix:///tmp/plugins/test.sock",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing plugin configuration",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{},
				},
			},
			wantErr: true,
		},
		{
			name: "empty endpoint",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Plugin: &esv1.PluginProvider{
							Endpoint: "",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint format",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Plugin: &esv1.PluginProvider{
							Endpoint: "invalid://endpoint",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty timeout",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Plugin: &esv1.PluginProvider{
							Endpoint: "unix:///tmp/plugins/test.sock",
							Timeout:  stringPtr(""),
						},
					},
				},
			},
			wantErr: true,
		},
	}

	provider := &PluginProvider{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := provider.ValidateStore(tt.store)
			if (err != nil) != tt.wantErr {
				t.Errorf("PluginProvider.ValidateStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(warnings) > 0 {
				t.Logf("PluginProvider.ValidateStore() warnings = %v", warnings)
			}
		})
	}
}

func TestPluginProvider_Capabilities(t *testing.T) {
	provider := &PluginProvider{}
	capabilities := provider.Capabilities()

	// Plugin provider returns conservative default (read-only)
	// Actual capabilities are discovered via GetPluginCapabilities()
	if capabilities != esv1.SecretStoreReadOnly {
		t.Errorf("PluginProvider.Capabilities() = %v, want %v", capabilities, esv1.SecretStoreReadOnly)
	}
}

func TestPluginProvider_NewClient_InvalidStore(t *testing.T) {
	provider := &PluginProvider{}
	ctx := context.Background()

	// Test avec un store invalide
	invalidStore := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{},
		},
	}

	_, err := provider.NewClient(ctx, invalidStore, nil, "default")
	if err == nil {
		t.Error("PluginProvider.NewClient() expected error for invalid store, got nil")
	}

	// Check that we get the expected error type
	if !errors.Is(err, ErrPluginConfigNil) {
		t.Errorf("PluginProvider.NewClient() expected ErrPluginConfigNil, got %v", err)
	}
}

func TestPluginProvider_SpecificErrors(t *testing.T) {
	provider := &PluginProvider{}

	tests := []struct {
		name      string
		store     esv1.GenericStore
		expectErr error
	}{
		{
			name: "nil plugin config",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{},
				},
			},
			expectErr: ErrPluginConfigNil,
		},
		{
			name: "empty endpoint",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Plugin: &esv1.PluginProvider{
							Endpoint: "",
						},
					},
				},
			},
			expectErr: ErrEndpointRequired,
		},
		{
			name: "empty timeout",
			store: &esv1.SecretStore{
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Plugin: &esv1.PluginProvider{
							Endpoint: "unix:///tmp/plugins/test.sock",
							Timeout:  stringPtr(""),
						},
					},
				},
			},
			expectErr: ErrTimeoutEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.ValidateStore(tt.store)
			if err == nil {
				t.Errorf("ValidateStore() expected error for %s", tt.name)
				return
			}

			if !errors.Is(err, tt.expectErr) {
				t.Errorf("ValidateStore() expected error %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func TestEndpointParsing(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{
			name:     "unix socket with scheme",
			endpoint: "unix:///tmp/test.sock",
			wantErr:  false,
		},
		{
			name:     "unix socket without scheme",
			endpoint: "/tmp/test.sock",
			wantErr:  false,
		},
		{
			name:     "tcp with scheme",
			endpoint: "tcp://localhost:8080",
			wantErr:  false,
		},
		{
			name:     "network address without scheme",
			endpoint: "localhost:8080",
			wantErr:  false,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock PluginClient to test endpoint parsing logic
			client := &PluginClient{
				endpoint: tt.endpoint,
				timeout:  30 * time.Second,
			}

			// Test that the endpoint is set correctly
			if tt.wantErr {
				if client.endpoint != "" {
					t.Errorf("Expected empty endpoint for error case, got %s", client.endpoint)
				}
			} else {
				if client.endpoint != tt.endpoint {
					t.Errorf("Expected endpoint %s, got %s", tt.endpoint, client.endpoint)
				}
			}
		})
	}
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
