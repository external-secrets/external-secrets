package plugin

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/plugin/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ProviderName           = "plugin"
	PluginMagicCookieKey   = "ESO_PLUGIN"
	PluginMagicCookieValue = "external-secrets-operator"
)

// PluginClient implements the SecretsClient interface for plugin-based providers
type PluginClient struct {
	store     *esv1.SecretStore
	kube      client.Client
	namespace string

	// Connection details
	endpoint   string
	timeout    time.Duration
	grpcClient proto.SecretsPluginServiceClient
	grpcConn   *grpc.ClientConn
}

// NewPluginClient creates a new plugin client with direct endpoint connection
func NewPluginClient(ctx context.Context, store *esv1.SecretStore, kube client.Client, namespace string) (*PluginClient, error) {
	if store.Spec.Provider.Plugin == nil {
		return nil, fmt.Errorf("plugin provider configuration is nil")
	}

	cfg := store.Spec.Provider.Plugin

	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("plugin endpoint is required")
	}

	// Parse timeout
	timeout := 30 * time.Second // default
	if cfg.Timeout != nil {
		if parsed, err := time.ParseDuration(*cfg.Timeout); err == nil {
			timeout = parsed
		}
	}

	client := &PluginClient{
		store:     store,
		kube:      kube,
		namespace: namespace,
		endpoint:  cfg.Endpoint,
		timeout:   timeout,
	}

	// Establish connection to the plugin
	err := client.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to plugin at %s: %w", cfg.Endpoint, err)
	}

	return client, nil
}

// connect establishes a connection to the plugin endpoint with gRPC
func (c *PluginClient) connect(ctx context.Context) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	target, dialOptions, err := c.buildDialOptions()
	if err != nil {
		return fmt.Errorf("failed to build dial options: %w", err)
	}

	// Create the connection
	conn, err := grpc.DialContext(ctxWithTimeout, target, dialOptions...)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", target, err)
	}

	c.grpcConn = conn
	c.grpcClient = proto.NewSecretsPluginServiceClient(conn)

	// Test connection with GetInfo
	if err := c.healthCheck(ctxWithTimeout); err != nil {
		conn.Close()
		return fmt.Errorf("plugin health check failed: %w", err)
	}

	return nil
}

// buildDialOptions creates appropriate dial options based on endpoint type
func (c *PluginClient) buildDialOptions() (string, []grpc.DialOption, error) {
	parsedURL, err := url.Parse(c.endpoint)

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Enable keepalive for long-lived connections
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // Send keepalive ping every 10 seconds
			Timeout:             3 * time.Second,  // Wait 3 seconds for ping ack
			PermitWithoutStream: true,             // Send pings even without active streams
		}),
		// Set max message sizes
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
			grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
		),
		// Add connection state monitoring
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1.0 * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   120 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
	}

	var target string

	// Handle URL parsing errors or no scheme
	if err != nil || parsedURL.Scheme == "" {
		target = c.endpoint
		// For host:port format, validate it
		if err == nil && parsedURL.Path != "" && (parsedURL.Path[0] == '/' || (len(parsedURL.Path) >= 2 && parsedURL.Path[:2] == "./")) {
			// Unix socket path
			target = parsedURL.Path
			dialOptions = append(dialOptions, grpc.WithContextDialer(c.createUnixDialer(target)))
		}
		return target, dialOptions, nil
	}

	switch parsedURL.Scheme {
	case "unix":
		target = parsedURL.Path
		if !filepath.IsAbs(target) {
			return "", nil, fmt.Errorf("unix socket path must be absolute: %s", target)
		}
		dialOptions = append(dialOptions, grpc.WithContextDialer(c.createUnixDialer(target)))
	case "tcp":
		target = parsedURL.Host
		if target == "" {
			return "", nil, fmt.Errorf("tcp scheme requires host:port")
		}
	default:
		return "", nil, fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	return target, dialOptions, nil
}

// createUnixDialer creates a dialer for Unix sockets
func (c *PluginClient) createUnixDialer(socketPath string) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout: c.timeout,
		}
		return dialer.DialContext(ctx, "unix", socketPath)
	}
}

// healthCheck performs a health check on the connection
func (c *PluginClient) healthCheck(ctx context.Context) error {
	if c.grpcClient == nil {
		return fmt.Errorf("grpc client not initialized")
	}

	_, err := c.grpcClient.GetInfo(ctx, &proto.GetInfoRequest{})
	if err != nil {
		// Check if it's a gRPC error and provide better context
		if stat := status.Convert(err); stat != nil {
			switch stat.Code() {
			case codes.Unavailable:
				return fmt.Errorf("plugin service unavailable: %w", err)
			case codes.Unimplemented:
				return fmt.Errorf("plugin does not implement GetInfo: %w", err)
			case codes.DeadlineExceeded:
				return fmt.Errorf("plugin health check timeout: %w", err)
			default:
				return fmt.Errorf("plugin health check failed with code %s: %w", stat.Code(), err)
			}
		}
		return err
	}

	return nil
}

// GetCapabilities returns the capabilities of the connected plugin
func (c *PluginClient) GetCapabilities(ctx context.Context) (*proto.GetInfoResponse, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("plugin client not connected")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.grpcClient.GetInfo(ctxWithTimeout, &proto.GetInfoRequest{})
}

// buildMetadata builds metadata for requests
func (c *PluginClient) buildMetadata() map[string]string {
	metadata := make(map[string]string)

	// Add store information
	metadata["store.name"] = c.store.Name
	metadata["store.namespace"] = c.store.Namespace
	metadata["client.namespace"] = c.namespace

	// Add auth information if configured
	if c.store.Spec.Provider.Plugin != nil && c.store.Spec.Provider.Plugin.Auth != nil {
		auth := c.store.Spec.Provider.Plugin.Auth
		if auth.SecretRef != nil {
			// Add generic secret reference metadata
			metadata["auth.secretRef.configured"] = "true"
		}
	}

	return metadata
}

// Close closes the plugin connection gracefully
func (c *PluginClient) Close(ctx context.Context) error {
	if c.grpcConn != nil {
		// Check connection state before closing
		state := c.grpcConn.GetState()
		if state != connectivity.Shutdown {
			// Gracefully close the connection
			return c.grpcConn.Close()
		}
	}
	return nil
}

// isConnected checks if the gRPC connection is in a good state
func (c *PluginClient) isConnected() bool {
	if c.grpcConn == nil {
		return false
	}
	state := c.grpcConn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// ensureConnection ensures the connection is ready, reconnecting if necessary
func (c *PluginClient) ensureConnection(ctx context.Context) error {
	if c.isConnected() {
		return nil
	}

	// Connection is not ready, attempt to reconnect
	if c.grpcConn != nil {
		c.grpcConn.Close()
	}

	return c.connect(ctx)
}

// GetSecret retrieves a secret from the configured plugin with connection resilience
func (c *PluginClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if err := c.ensureConnection(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure connection: %w", err)
	}

	// Convert the reference to proto format
	request := &proto.GetSecretRequest{
		Key:      ref.Key,
		Version:  ref.Version,
		Property: ref.Property,
		Auth:     c.buildMetadata(),
	}

	// Call the plugin with timeout context
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.grpcClient.GetSecret(ctxWithTimeout, request)
	if err != nil {
		// Check for connection errors and potentially retry
		if stat := status.Convert(err); stat != nil && stat.Code() == codes.Unavailable {
			// Try to reconnect and retry once
			if reconnectErr := c.connect(ctx); reconnectErr == nil {
				ctxRetry, cancelRetry := context.WithTimeout(ctx, c.timeout)
				defer cancelRetry()
				response, err = c.grpcClient.GetSecret(ctxRetry, request)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("plugin failed to get secret: %w", err)
		}
	}

	return response.Value, nil
}

// GetSecretMap retrieves multiple secrets from the configured plugin
func (c *PluginClient) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("plugin client not connected")
	}

	// Convert the reference to proto format
	request := &proto.GetSecretMapRequest{
		Key:      ref.Key,
		Version:  ref.Version,
		Property: ref.Property,
		Auth:     c.buildMetadata(),
	}

	// Call the plugin
	response, err := c.grpcClient.GetSecretMap(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("plugin failed to get secret map: %w", err)
	}

	// Convert response to map[string][]byte
	return response.Values, nil
}

// GetAllSecrets retrieves all secrets from the configured plugin
func (c *PluginClient) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if c.grpcClient == nil {
		return nil, fmt.Errorf("plugin client not connected")
	}

	// Convert the find request to proto format
	request := &proto.GetAllSecretsRequest{
		Auth: c.buildMetadata(),
	}

	// Call the plugin
	response, err := c.grpcClient.GetAllSecrets(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("plugin failed to get all secrets: %w", err)
	}

	// Convert response to map[string][]byte
	return response.Values, nil
}

// PushSecret pushes a secret to the configured plugin
func (c *PluginClient) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if c.grpcClient == nil {
		return fmt.Errorf("plugin client not connected")
	}

	// Extract secret data from the secret
	var secretValue []byte
	if secret != nil && secret.Data != nil {
		// For simplicity, use the first key-value pair
		for _, value := range secret.Data {
			secretValue = value
			break
		}
	}

	// Convert the secret data to proto format
	request := &proto.PushSecretRequest{
		Key:   data.GetSecretKey(),
		Value: secretValue,
		Auth:  c.buildMetadata(),
	}

	// Call the plugin
	_, err := c.grpcClient.PushSecret(ctx, request)
	if err != nil {
		return fmt.Errorf("plugin failed to push secret: %w", err)
	}

	return nil
}

// DeleteSecret deletes a secret from the configured plugin
func (c *PluginClient) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	if c.grpcClient == nil {
		return fmt.Errorf("plugin client not connected")
	}

	// Convert the reference to proto format
	request := &proto.DeleteSecretRequest{
		Key:  remoteRef.GetRemoteKey(),
		Auth: c.buildMetadata(),
	}

	// Call the plugin
	_, err := c.grpcClient.DeleteSecret(ctx, request)
	if err != nil {
		return fmt.Errorf("plugin failed to delete secret: %w", err)
	}

	return nil
}

// SecretExists checks if a secret exists in the configured plugin
func (c *PluginClient) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	if c.grpcClient == nil {
		return false, fmt.Errorf("plugin client not connected")
	}

	// Convert the reference to proto format
	request := &proto.SecretExistsRequest{
		Key:  remoteRef.GetRemoteKey(),
		Auth: c.buildMetadata(),
	}

	// Call the plugin
	response, err := c.grpcClient.SecretExists(ctx, request)
	if err != nil {
		return false, fmt.Errorf("plugin failed to check secret existence: %w", err)
	}

	return response.Exists, nil
}

// Validate validates the plugin configuration with connection resilience
func (c *PluginClient) Validate() (esv1.ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if err := c.ensureConnection(ctx); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("failed to ensure connection: %w", err)
	}

	// Call the plugin validate method
	request := &proto.ValidateRequest{
		Auth: c.buildMetadata(),
	}

	response, err := c.grpcClient.Validate(ctx, request)
	if err != nil {
		// Handle specific gRPC errors
		if stat := status.Convert(err); stat != nil {
			switch stat.Code() {
			case codes.Unimplemented:
				// Plugin doesn't implement validation, assume valid
				return esv1.ValidationResultReady, nil
			case codes.Unavailable:
				return esv1.ValidationResultError, fmt.Errorf("plugin service unavailable: %w", err)
			case codes.DeadlineExceeded:
				return esv1.ValidationResultError, fmt.Errorf("plugin validation timeout: %w", err)
			}
		}
		return esv1.ValidationResultError, fmt.Errorf("plugin validation failed: %w", err)
	}

	if !response.Valid {
		return esv1.ValidationResultError, fmt.Errorf("plugin validation failed: configuration is invalid")
	}

	return esv1.ValidationResultReady, nil
}

// GetSecretsByMetadata retrieves secrets by metadata from the configured plugin
func (c *PluginClient) GetSecretsByMetadata(ctx context.Context, metadata map[string]string) (map[string][]byte, error) {
	// This method is not commonly used but required by the interface
	// We'll implement it as a GetAllSecrets call for now
	ref := esv1.ExternalSecretFind{}
	return c.GetAllSecrets(ctx, ref)
}
