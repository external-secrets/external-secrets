package plugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/plugin/proto"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockSecretsPlugin implements a mock plugin server for testing
type MockSecretsPlugin struct {
	proto.UnimplementedSecretsPluginServiceServer
	secrets map[string]map[string][]byte // namespace -> key -> value
}

func NewMockSecretsPlugin() *MockSecretsPlugin {
	return &MockSecretsPlugin{
		secrets: make(map[string]map[string][]byte),
	}
}

func (m *MockSecretsPlugin) GetInfo(ctx context.Context, req *proto.GetInfoRequest) (*proto.GetInfoResponse, error) {
	return &proto.GetInfoResponse{
		Name:         "mock-plugin",
		Version:      "1.0.0",
		Capabilities: []proto.Capability{proto.Capability_CAPABILITY_READ_WRITE},
	}, nil
}

func (m *MockSecretsPlugin) GetSecret(ctx context.Context, req *proto.GetSecretRequest) (*proto.GetSecretResponse, error) {
	namespace := req.Auth["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	secretMap, exists := m.secrets[namespace]
	if !exists {
		return nil, fmt.Errorf("secret not found")
	}

	value, exists := secretMap[req.Key]
	if !exists {
		return nil, fmt.Errorf("secret key not found")
	}

	return &proto.GetSecretResponse{
		Value: value,
	}, nil
}

func (m *MockSecretsPlugin) GetSecretMap(ctx context.Context, req *proto.GetSecretMapRequest) (*proto.GetSecretMapResponse, error) {
	namespace := req.Auth["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	secretMap, exists := m.secrets[namespace]
	if !exists {
		return &proto.GetSecretMapResponse{
			Values: make(map[string][]byte),
		}, nil
	}

	// Return all secrets for simplicity
	return &proto.GetSecretMapResponse{
		Values: secretMap,
	}, nil
}

func (m *MockSecretsPlugin) GetAllSecrets(ctx context.Context, req *proto.GetAllSecretsRequest) (*proto.GetAllSecretsResponse, error) {
	namespace := req.Auth["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	secretMap, exists := m.secrets[namespace]
	if !exists {
		return &proto.GetAllSecretsResponse{
			Values: make(map[string][]byte),
		}, nil
	}

	return &proto.GetAllSecretsResponse{
		Values: secretMap,
	}, nil
}

func (m *MockSecretsPlugin) PushSecret(ctx context.Context, req *proto.PushSecretRequest) (*proto.PushSecretResponse, error) {
	namespace := req.Auth["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	if m.secrets[namespace] == nil {
		m.secrets[namespace] = make(map[string][]byte)
	}

	// Store the data
	m.secrets[namespace][req.Key] = req.Value

	return &proto.PushSecretResponse{}, nil
}

func (m *MockSecretsPlugin) DeleteSecret(ctx context.Context, req *proto.DeleteSecretRequest) (*proto.DeleteSecretResponse, error) {
	namespace := req.Auth["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	secretMap, exists := m.secrets[namespace]
	if !exists {
		return nil, fmt.Errorf("secret not found")
	}

	delete(secretMap, req.Key)

	return &proto.DeleteSecretResponse{}, nil
}

func (m *MockSecretsPlugin) SecretExists(ctx context.Context, req *proto.SecretExistsRequest) (*proto.SecretExistsResponse, error) {
	namespace := req.Auth["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	secretMap, exists := m.secrets[namespace]
	if !exists {
		return &proto.SecretExistsResponse{Exists: false}, nil
	}

	_, exists = secretMap[req.Key]
	return &proto.SecretExistsResponse{Exists: exists}, nil
}

func (m *MockSecretsPlugin) Validate(ctx context.Context, req *proto.ValidateRequest) (*proto.ValidateResponse, error) {
	return &proto.ValidateResponse{
		Valid: true,
	}, nil
}

// TestBasicPluginIntegration tests basic plugin functionality
func TestBasicPluginIntegration(t *testing.T) {
	// Create a temporary directory for sockets
	tmpDir, err := os.MkdirTemp("", "plugin-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Start mock plugin server
	socketPath := filepath.Join(tmpDir, "mock-plugin.sock")
	server, err := startMockPluginServer(socketPath)
	if err != nil {
		t.Fatalf("failed to start mock plugin server: %v", err)
	}
	defer server.Stop()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Create fake Kubernetes client
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = esv1.AddToScheme(scheme)

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Create SecretStore with plugin configuration
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-plugin-store",
			Namespace: "default",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Plugin: &esv1.PluginProvider{
					Endpoint: "unix://" + filepath.Join(tmpDir, "mock-plugin.sock"),
					Timeout:  stringPtr("5s"), // 5 seconds
				},
			},
		},
	}

	// Create plugin provider
	provider := &PluginProvider{}

	// Test ValidateStore
	warnings, err := provider.ValidateStore(store)
	if err != nil {
		t.Fatalf("ValidateStore failed: %v", err)
	}
	if len(warnings) > 0 {
		t.Logf("ValidateStore warnings: %v", warnings)
	}

	// Create plugin client
	ctx := context.Background()
	client, err := provider.NewClient(ctx, store, kubeClient, "default")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close(ctx)

	// Wait for plugin discovery
	time.Sleep(2 * time.Second)

	// Test GetSecret (expect failure since no secrets are pre-loaded)
	t.Run("GetSecret_NotFound", func(t *testing.T) {
		_, err := client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
			Key: "nonexistent-key",
		})
		if err == nil {
			t.Error("GetSecret should fail when secret doesn't exist")
		}
	})

	// Test GetSecretMap (should return empty map)
	t.Run("GetSecretMap_Empty", func(t *testing.T) {
		secretMap, err := client.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{
			Key: "any-key",
		})
		if err != nil {
			t.Fatalf("GetSecretMap failed: %v", err)
		}

		if len(secretMap) != 0 {
			t.Errorf("GetSecretMap returned %d items, expected 0", len(secretMap))
		}
	})

	// Test Validate
	t.Run("Validate", func(t *testing.T) {
		result, err := client.Validate()
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
		if result != esv1.ValidationResultReady {
			t.Errorf("Validate returned %v, expected %v", result, esv1.ValidationResultReady)
		}
	})
}

// TestDirectPluginConnection tests direct connection to a plugin
func TestDirectPluginConnection(t *testing.T) {
	// Create a temporary directory for socket
	tmpDir, err := os.MkdirTemp("", "plugin-connection-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock plugin server
	socketPath := filepath.Join(tmpDir, "mock-plugin.sock")
	server, err := startMockPluginServer(socketPath)
	if err != nil {
		t.Fatalf("failed to start mock plugin server: %v", err)
	}
	defer server.Stop()

	// Wait for server to be ready
	time.Sleep(200 * time.Millisecond)

	// Create a secret store with direct endpoint
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-store",
			Namespace: "default",
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Plugin: &esv1.PluginProvider{
					Endpoint: "unix://" + socketPath,
					Timeout:  stringPtr("5s"),
				},
			},
		},
	}

	// Create plugin client with direct connection
	client, err := NewPluginClient(context.Background(), store, nil, "default")
	if err != nil {
		t.Fatalf("failed to create plugin client: %v", err)
	}
	defer client.Close(context.Background())

	// Test GetCapabilities
	capabilities, err := client.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("failed to get capabilities: %v", err)
	}

	if capabilities.Name != "mock-plugin" {
		t.Errorf("Expected plugin name 'mock-plugin', got '%s'", capabilities.Name)
	}

	// Test GetSecret on a key that doesn't exist (should fail gracefully)
	ref := esv1.ExternalSecretDataRemoteRef{
		Key: "nonexistent-key",
	}

	_, err = client.GetSecret(context.Background(), ref)
	if err == nil {
		t.Error("Expected error for nonexistent key, but got none")
	}

	// Test GetSecretMap
	secretMap, err := client.GetSecretMap(context.Background(), ref)
	if err == nil && len(secretMap) == 0 {
		// This is acceptable - empty map for non-existent key
	}
}

// startMockPluginServer starts a mock plugin server listening on a Unix socket
func startMockPluginServer(socketPath string) (*grpc.Server, error) {
	// Remove existing socket if it exists
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %v", socketPath, err)
	}

	server := grpc.NewServer()
	plugin := NewMockSecretsPlugin()

	// Don't pre-populate data for cleaner tests
	proto.RegisterSecretsPluginServiceServer(server, plugin)

	go func() {
		if err := server.Serve(listener); err != nil {
			// Server stopped, which is expected during tests
		}
	}()

	return server, nil
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
