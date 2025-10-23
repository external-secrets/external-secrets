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

package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	v2 "github.com/external-secrets/external-secrets/providers/v2/common"
)

const (
	// defaultTimeout is the default timeout for gRPC calls
	defaultTimeout = 30 * time.Second
)

// grpcProviderClient implements the v2.Provider interface using gRPC.
type grpcProviderClient struct {
	conn   *grpc.ClientConn
	client pb.SecretStoreProviderClient
	log    logr.Logger
}

// Ensure grpcProviderClient implements Provider interface
var _ v2.Provider = &grpcProviderClient{}

// GetSecret retrieves a single secret from the provider via gRPC.
func (c *grpcProviderClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef, providerRef *pb.ProviderReference, sourceNamespace string) ([]byte, error) {
	start := time.Now()
	var err error
	defer func() {
		clientMetrics.ObserveRequest("GetSecret", c.conn.Target(), err, time.Since(start))
	}()

	c.log.V(1).Info("getting secret via gRPC",
		"key", ref.Key,
		"version", ref.Version,
		"property", ref.Property,
		"connectionState", c.conn.GetState().String(),
		"providerRef", providerRef,
		"sourceNamespace", sourceNamespace)

	// Check connection state before call
	state := c.conn.GetState()
	if state != connectivity.Ready && state != connectivity.Idle {
		c.log.Info("connection not ready, attempting to reconnect",
			"state", state.String(),
			"target", c.conn.Target())
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Convert v1 reference to protobuf message
	pbRef := &pb.ExternalSecretDataRemoteRef{
		Key:              ref.Key,
		Version:          ref.Version,
		Property:         ref.Property,
		DecodingStrategy: string(ref.DecodingStrategy),
		MetadataPolicy:   string(ref.MetadataPolicy),
	}

	// Make gRPC call with provider reference
	req := &pb.GetSecretRequest{
		RemoteRef:       pbRef,
		ProviderRef:     providerRef,
		SourceNamespace: sourceNamespace,
	}

	c.log.V(1).Info("calling GetSecret RPC",
		"target", c.conn.Target(),
		"timeout", defaultTimeout.String())

	resp, err := c.client.GetSecret(ctx, req)
	if err != nil {
		c.log.Error(err, "GetSecret RPC failed",
			"key", ref.Key,
			"connectionState", c.conn.GetState().String(),
			"target", c.conn.Target())
		err = fmt.Errorf("failed to get secret via gRPC: %w", err)
		return nil, err
	}

	c.log.V(1).Info("GetSecret RPC succeeded",
		"key", ref.Key,
		"valueLength", len(resp.Value))

	return resp.Value, nil
}

// Validate checks if the provider is properly configured via gRPC.
func (c *grpcProviderClient) Validate(ctx context.Context, providerRef *pb.ProviderReference, sourceNamespace string) error {
	start := time.Now()
	var err error
	defer func() {
		clientMetrics.ObserveRequest("Validate", c.conn.Target(), err, time.Since(start))
	}()

	c.log.Info("validating provider via gRPC",
		"target", c.conn.Target(),
		"connectionState", c.conn.GetState().String(),
		"providerRef", providerRef,
		"sourceNamespace", sourceNamespace)

	// Check connection state before call
	state := c.conn.GetState()
	c.log.V(1).Info("connection details",
		"state", state.String(),
		"target", c.conn.Target(),
		"authority", c.conn.GetMethodConfig("").WaitForReady)

	if state != connectivity.Ready && state != connectivity.Idle {
		c.log.Info("connection not in ready/idle state, will attempt to connect",
			"state", state.String(),
			"target", c.conn.Target())
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Make gRPC call with provider reference
	req := &pb.ValidateRequest{
		ProviderRef:     providerRef,
		SourceNamespace: sourceNamespace,
	}

	c.log.V(1).Info("calling Validate RPC",
		"target", c.conn.Target(),
		"timeout", defaultTimeout.String())

	resp, err := c.client.Validate(ctx, req)
	if err != nil {
		c.log.Error(err, "Validate RPC failed",
			"connectionState", c.conn.GetState().String(),
			"target", c.conn.Target(),
			"errorType", fmt.Sprintf("%T", err))
		err = fmt.Errorf("failed to validate provider via gRPC: %w", err)
		return err
	}

	c.log.V(1).Info("Validate RPC completed",
		"valid", resp.Valid,
		"error", resp.Error)

	// Check for error in response
	if !resp.Valid {
		if resp.Error != "" {
			c.log.Error(fmt.Errorf("provider validation failed"), "validation response",
				"message", resp.Error)
			err = fmt.Errorf("provider validation failed: %s", resp.Error)
			return err
		}
		c.log.Error(fmt.Errorf("provider validation failed"), "validation response",
			"message", "no error message provided")
		err = fmt.Errorf("provider validation failed without error message")
		return err
	}

	c.log.Info("provider validation succeeded")
	return nil
}

// GetAllSecrets retrieves multiple secrets based on find criteria via gRPC.
func (c *grpcProviderClient) GetAllSecrets(ctx context.Context, find esv1.ExternalSecretFind, providerRef *pb.ProviderReference, sourceNamespace string) (map[string][]byte, error) {
	start := time.Now()
	var err error
	defer func() {
		clientMetrics.ObserveRequest("GetAllSecrets", c.conn.Target(), err, time.Since(start))
	}()

	c.log.V(1).Info("getting all secrets via gRPC",
		"tags", find.Tags,
		"connectionState", c.conn.GetState().String(),
		"providerRef", providerRef,
		"sourceNamespace", sourceNamespace)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Convert find criteria to protobuf
	pbFind := &pb.ExternalSecretFind{
		Tags:               find.Tags,
		ConversionStrategy: string(find.ConversionStrategy),
		DecodingStrategy:   string(find.DecodingStrategy),
	}

	if find.Path != nil {
		pbFind.Path = *find.Path
	}

	if find.Name != nil {
		pbFind.Name = &pb.FindName{
			Regexp: find.Name.RegExp,
		}
	}

	// Make gRPC call
	req := &pb.GetAllSecretsRequest{
		ProviderRef:     providerRef,
		Find:            pbFind,
		SourceNamespace: sourceNamespace,
	}

	c.log.V(1).Info("calling GetAllSecrets RPC",
		"target", c.conn.Target())

	resp, err := c.client.GetAllSecrets(ctx, req)
	if err != nil {
		c.log.Error(err, "GetAllSecrets RPC failed",
			"connectionState", c.conn.GetState().String(),
			"target", c.conn.Target())
		err = fmt.Errorf("failed to get all secrets via gRPC: %w", err)
		return nil, err
	}

	c.log.V(1).Info("GetAllSecrets RPC succeeded",
		"secretCount", len(resp.Secrets))

	return resp.Secrets, nil
}

// PushSecret writes a secret to the provider via gRPC.
func (c *grpcProviderClient) PushSecret(ctx context.Context, secretData map[string][]byte, pushSecretData *pb.PushSecretData, providerRef *pb.ProviderReference, sourceNamespace string) error {
	start := time.Now()
	var err error
	defer func() {
		clientMetrics.ObserveRequest("PushSecret", c.conn.Target(), err, time.Since(start))
	}()

	c.log.V(1).Info("pushing secret via gRPC",
		"remoteKey", pushSecretData.RemoteKey,
		"property", pushSecretData.Property,
		"connectionState", c.conn.GetState().String(),
		"providerRef", providerRef,
		"sourceNamespace", sourceNamespace)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Make gRPC call
	req := &pb.PushSecretRequest{
		ProviderRef:     providerRef,
		SecretData:      secretData,
		PushSecretData:  pushSecretData,
		SourceNamespace: sourceNamespace,
	}

	c.log.V(1).Info("calling PushSecret RPC",
		"target", c.conn.Target())

	_, err = c.client.PushSecret(ctx, req)
	if err != nil {
		c.log.Error(err, "PushSecret RPC failed",
			"connectionState", c.conn.GetState().String(),
			"target", c.conn.Target())
		err = fmt.Errorf("failed to push secret via gRPC: %w", err)
		return err
	}

	c.log.V(1).Info("PushSecret RPC succeeded",
		"remoteKey", pushSecretData.RemoteKey)

	return nil
}

// DeleteSecret deletes a secret from the provider via gRPC.
func (c *grpcProviderClient) DeleteSecret(ctx context.Context, remoteRef *pb.PushSecretRemoteRef, providerRef *pb.ProviderReference, sourceNamespace string) error {
	start := time.Now()
	var err error
	defer func() {
		clientMetrics.ObserveRequest("DeleteSecret", c.conn.Target(), err, time.Since(start))
	}()

	c.log.V(1).Info("deleting secret via gRPC",
		"remoteKey", remoteRef.RemoteKey,
		"property", remoteRef.Property,
		"connectionState", c.conn.GetState().String(),
		"providerRef", providerRef,
		"sourceNamespace", sourceNamespace)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Make gRPC call
	req := &pb.DeleteSecretRequest{
		ProviderRef:     providerRef,
		RemoteRef:       remoteRef,
		SourceNamespace: sourceNamespace,
	}

	c.log.V(1).Info("calling DeleteSecret RPC",
		"target", c.conn.Target())

	_, err = c.client.DeleteSecret(ctx, req)
	if err != nil {
		c.log.Error(err, "DeleteSecret RPC failed",
			"connectionState", c.conn.GetState().String(),
			"target", c.conn.Target())
		err = fmt.Errorf("failed to delete secret via gRPC: %w", err)
		return err
	}

	c.log.V(1).Info("DeleteSecret RPC succeeded",
		"remoteKey", remoteRef.RemoteKey)

	return nil
}

// SecretExists checks if a secret exists in the provider via gRPC.
func (c *grpcProviderClient) SecretExists(ctx context.Context, remoteRef *pb.PushSecretRemoteRef, providerRef *pb.ProviderReference, sourceNamespace string) (bool, error) {
	start := time.Now()
	var err error
	defer func() {
		clientMetrics.ObserveRequest("SecretExists", c.conn.Target(), err, time.Since(start))
	}()

	c.log.V(1).Info("checking if secret exists via gRPC",
		"remoteKey", remoteRef.RemoteKey,
		"property", remoteRef.Property,
		"connectionState", c.conn.GetState().String(),
		"providerRef", providerRef,
		"sourceNamespace", sourceNamespace)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Make gRPC call
	req := &pb.SecretExistsRequest{
		ProviderRef:     providerRef,
		RemoteRef:       remoteRef,
		SourceNamespace: sourceNamespace,
	}

	c.log.V(1).Info("calling SecretExists RPC",
		"target", c.conn.Target())

	resp, err := c.client.SecretExists(ctx, req)
	if err != nil {
		c.log.Error(err, "SecretExists RPC failed",
			"connectionState", c.conn.GetState().String(),
			"target", c.conn.Target())
		err = fmt.Errorf("failed to check if secret exists via gRPC: %w", err)
		return false, err
	}

	c.log.V(1).Info("SecretExists RPC succeeded",
		"remoteKey", remoteRef.RemoteKey,
		"exists", resp.Exists)

	return resp.Exists, nil
}

// Capabilities retrieves the capabilities of the provider via gRPC.
func (c *grpcProviderClient) Capabilities(ctx context.Context, providerRef *pb.ProviderReference, sourceNamespace string) (pb.SecretStoreCapabilities, error) {
	start := time.Now()
	var err error
	defer func() {
		clientMetrics.ObserveRequest("Capabilities", c.conn.Target(), err, time.Since(start))
	}()

	c.log.V(1).Info("getting provider capabilities via gRPC",
		"target", c.conn.Target(),
		"connectionState", c.conn.GetState().String(),
		"providerRef", providerRef,
		"sourceNamespace", sourceNamespace)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Make gRPC call with provider reference
	req := &pb.CapabilitiesRequest{
		ProviderRef:     providerRef,
		SourceNamespace: sourceNamespace,
	}

	c.log.V(1).Info("calling Capabilities RPC",
		"target", c.conn.Target())

	resp, err := c.client.Capabilities(ctx, req)
	if err != nil {
		c.log.Error(err, "Capabilities RPC failed",
			"connectionState", c.conn.GetState().String(),
			"target", c.conn.Target())
		err = fmt.Errorf("failed to get capabilities via gRPC: %w", err)
		return pb.SecretStoreCapabilities_READ_ONLY, err
	}

	c.log.V(1).Info("Capabilities RPC succeeded",
		"capabilities", resp.Capabilities)

	return resp.Capabilities, nil
}

// Close closes the gRPC connection.
func (c *grpcProviderClient) Close(ctx context.Context) error {
	if c.conn != nil {
		c.log.V(1).Info("closing gRPC connection",
			"target", c.conn.Target(),
			"state", c.conn.GetState().String())
		return c.conn.Close()
	}
	c.log.V(1).Info("no connection to close")
	return nil
}
