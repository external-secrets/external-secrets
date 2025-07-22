package plugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/plugin/proto"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Common exported errors for plugin provider
var (
	ErrInvalidStoreType                 = errors.New("invalid store type")
	ErrPluginConfigNil                  = errors.New("plugin provider configuration is nil")
	ErrEndpointRequired                 = errors.New("endpoint is required")
	ErrTimeoutEmpty                     = errors.New("timeout cannot be empty")
	ErrUnixSocketNotAbsolute            = errors.New("unix socket path must be absolute")
	ErrUnsupportedScheme                = errors.New("unsupported scheme")
	ErrCapabilityProviderNotImplemented = errors.New("client does not implement CapabilityProvider interface")
	ErrPluginNotConnected               = errors.New("plugin client not connected")
	ErrTCPSchemeRequiresHostPort        = errors.New("tcp scheme requires host:port")
)

// PluginProvider implements the external-secrets Provider interface for plugin-based providers
type PluginProvider struct{}

// CapabilityProvider is an interface that plugin clients can implement
// to provide dynamic capability detection
type CapabilityProvider interface {
	// GetCapabilities returns the plugin's actual capabilities by querying it
	GetCapabilities(ctx context.Context) (*proto.GetInfoResponse, error)
}

// NewClient creates a new plugin client
func (p *PluginProvider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	secretStore, ok := store.(*esv1.SecretStore)
	if !ok {
		clusterSecretStore, ok := store.(*esv1.ClusterSecretStore)
		if !ok {
			return nil, ErrInvalidStoreType
		}

		// Convert ClusterSecretStore to SecretStore for processing
		secretStore = &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: clusterSecretStore.Spec.Provider,
			},
		}
		secretStore.Name = clusterSecretStore.Name
		secretStore.Namespace = namespace // Use the namespace where the external secret is created
	}

	return NewPluginClient(ctx, secretStore, kube, namespace)
}

// ValidateStore validates the plugin store configuration and performs capability discovery
func (p *PluginProvider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store.GetSpec().Provider.Plugin == nil {
		return nil, ErrPluginConfigNil
	}

	cfg := store.GetSpec().Provider.Plugin

	// Validate endpoint
	if cfg.Endpoint == "" {
		return nil, ErrEndpointRequired
	}

	// Validate endpoint format
	endpoint := cfg.Endpoint
	if _, _, err := buildDialOptions(endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint format: %w", err)
	}

	// Validate timeout
	if cfg.Timeout != nil && *cfg.Timeout == "" {
		return nil, ErrTimeoutEmpty
	}

	// Perform capability discovery during validation
	// This will update the store status with actual plugin capabilities
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	capabilities, err := p.GetPluginCapabilities(ctx, store, nil, "")
	if err != nil {
		// Don't fail validation due to capability discovery failure
		// Log warning and continue with conservative default
		return admission.Warnings{
			fmt.Sprintf("Failed to discover plugin capabilities: %v. Using read-only default.", err),
		}, nil
	}

	// Update store status with discovered capabilities if it's a SecretStore or ClusterSecretStore
	if secretStore, ok := store.(*esv1.SecretStore); ok {
		if secretStore.Status.Capabilities != capabilities {
			secretStore.Status.Capabilities = capabilities
		}
	} else if clusterStore, ok := store.(*esv1.ClusterSecretStore); ok {
		if clusterStore.Status.Capabilities != capabilities {
			clusterStore.Status.Capabilities = capabilities
		}
	}

	return nil, nil
}

// Capabilities returns the provider capabilities by querying the plugin via gRPC
// Note: For plugins, this returns a conservative default since capability discovery
// requires connecting to the specific plugin. The actual capabilities should be
// discovered during store validation or when creating clients.
func (p *PluginProvider) Capabilities() esv1.SecretStoreCapabilities {
	// Return conservative default for plugin providers
	// Actual capability discovery happens in GetPluginCapabilities() or during client creation
	return esv1.SecretStoreReadOnly
}

// GetPluginCapabilities queries the actual plugin for its capabilities
// This method performs real capability discovery by connecting to the plugin
func (p *PluginProvider) GetPluginCapabilities(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretStoreCapabilities, error) {
	// Create a temporary client to query capabilities
	client, err := p.NewClient(ctx, store, kube, namespace)
	if err != nil {
		return esv1.SecretStoreReadOnly, fmt.Errorf("failed to create plugin client for capability discovery: %w", err)
	}
	defer func() {
		if closer, ok := client.(interface{ Close(context.Context) error }); ok {
			closer.Close(ctx)
		}
	}()

	// Check if client implements CapabilityProvider interface
	capabilityProvider, ok := client.(CapabilityProvider)
	if !ok {
		// Fallback: assume read-only for safety
		return esv1.SecretStoreReadOnly, ErrCapabilityProviderNotImplemented
	}

	// Query the plugin for its capabilities
	info, err := capabilityProvider.GetCapabilities(ctx)
	if err != nil {
		// If we can't get capabilities, assume read-only for safety
		return esv1.SecretStoreReadOnly, fmt.Errorf("failed to discover plugin capabilities: %w", err)
	}

	// Convert plugin capabilities to external-secrets capabilities
	capabilities := convertPluginCapabilities(info)

	return capabilities, nil
}

// DiscoverCapabilities is a convenience method for capability discovery without creating a full client
// This can be used by the external-secrets controller to discover capabilities before creating the store
func (p *PluginProvider) DiscoverCapabilities(ctx context.Context, endpoint string, timeout string) (esv1.SecretStoreCapabilities, error) {
	// Create a minimal store configuration for capability discovery
	tempStore := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Plugin: &esv1.PluginProvider{
					Endpoint: endpoint,
					Timeout:  &timeout,
				},
			},
		},
	}

	// Use GetPluginCapabilities with the temporary store
	return p.GetPluginCapabilities(ctx, tempStore, nil, "default")
}

// convertPluginCapabilities converts plugin info response to SecretStoreCapabilities
// Maps plugin capabilities to external-secrets' three capability types:
//   - esv1.SecretStoreReadOnly: read-only operations
//   - esv1.SecretStoreWriteOnly: write-only operations
//   - esv1.SecretStoreReadWrite: both read and write operations
func convertPluginCapabilities(info *proto.GetInfoResponse) esv1.SecretStoreCapabilities {
	if info == nil {
		return esv1.SecretStoreReadOnly
	}

	// First check enum-based capabilities (preferred)
	if len(info.Capabilities) > 0 {
		for _, cap := range info.Capabilities {
			switch cap {
			case proto.Capability_CAPABILITY_READ_ONLY:
				return esv1.SecretStoreReadOnly
			case proto.Capability_CAPABILITY_WRITE_ONLY:
				return esv1.SecretStoreWriteOnly
			case proto.Capability_CAPABILITY_READ_WRITE:
				return esv1.SecretStoreReadWrite
			}
		}
	}

	// Default to read-only for safety
	return esv1.SecretStoreReadOnly
}

// Helper functions for plugin developers to create proper capability responses

// NewReadOnlyCapabilityResponse creates a GetInfoResponse indicating read-only capabilities
func NewReadOnlyCapabilityResponse(name, version string) *proto.GetInfoResponse {
	return &proto.GetInfoResponse{
		Name:         name,
		Version:      version,
		Capabilities: []proto.Capability{proto.Capability_CAPABILITY_READ_ONLY},
	}
}

// NewWriteOnlyCapabilityResponse creates a GetInfoResponse indicating write-only capabilities
func NewWriteOnlyCapabilityResponse(name, version string) *proto.GetInfoResponse {
	return &proto.GetInfoResponse{
		Name:         name,
		Version:      version,
		Capabilities: []proto.Capability{proto.Capability_CAPABILITY_WRITE_ONLY},
	}
}

// NewReadWriteCapabilityResponse creates a GetInfoResponse indicating read-write capabilities
func NewReadWriteCapabilityResponse(name, version string) *proto.GetInfoResponse {
	return &proto.GetInfoResponse{
		Name:         name,
		Version:      version,
		Capabilities: []proto.Capability{proto.Capability_CAPABILITY_READ_WRITE},
	}
}

func init() {
	esv1.Register(&PluginProvider{}, &esv1.SecretStoreProvider{
		Plugin: &esv1.PluginProvider{},
	}, esv1.MaintenanceStatusNotMaintained)
}
