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

// Package store adapts v1 provider implementations to the v2 gRPC SecretStoreProvider interface.
package store

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	v2 "github.com/external-secrets/external-secrets/providers/v2/common"
)

// Client wraps a v2.Provider (gRPC client) and exposes it as an esv1.SecretsClient.
// This allows v2 providers to be used with the existing client manager infrastructure.
type Client struct {
	v2Provider      v2.Provider
	providerRef     *pb.ProviderReference
	sourceNamespace string
}

// Ensure Client implements SecretsClient interface.
var _ esv1.SecretsClient = &Client{}

// NewClient creates a new wrapper that adapts a v2.Provider to esv1.SecretsClient.
func NewClient(v2Provider v2.Provider, providerRef *pb.ProviderReference, sourceNamespace string) esv1.SecretsClient {
	return &Client{
		v2Provider:      v2Provider,
		providerRef:     providerRef,
		sourceNamespace: sourceNamespace,
	}
}

// GetSecret retrieves a single secret from the provider.
func (w *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return w.v2Provider.GetSecret(ctx, ref, w.providerRef, w.sourceNamespace)
}

// GetSecretMap is not supported for v2 providers.
// V2 providers don't have this method as it's being phased out in favor of GetAllSecrets.
func (w *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetSecretMap not supported for v2 providers")
}

// GetAllSecrets retrieves multiple secrets based on find criteria.
func (w *Client) GetAllSecrets(ctx context.Context, find esv1.ExternalSecretFind) (map[string][]byte, error) {
	return w.v2Provider.GetAllSecrets(ctx, find, w.providerRef, w.sourceNamespace)
}

// PushSecret writes a secret to the provider.
func (w *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	// Extract secret data
	secretData := secret.Data

	// Convert metadata from *apiextensionsv1.JSON to []byte
	var metadata []byte
	if data.GetMetadata() != nil {
		metadata = data.GetMetadata().Raw
	}

	// Convert esv1.PushSecretData to pb.PushSecretData
	pushSecretData := &pb.PushSecretData{
		RemoteKey: data.GetRemoteKey(),
		SecretKey: data.GetSecretKey(),
		Property:  data.GetProperty(),
		Metadata:  metadata,
	}

	return w.v2Provider.PushSecret(ctx, secretData, pushSecretData, w.providerRef, w.sourceNamespace)
}

// DeleteSecret deletes a secret from the provider.
func (w *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	// Convert esv1.PushSecretRemoteRef to pb.PushSecretRemoteRef
	pbRemoteRef := &pb.PushSecretRemoteRef{
		RemoteKey: remoteRef.GetRemoteKey(),
		Property:  remoteRef.GetProperty(),
	}

	return w.v2Provider.DeleteSecret(ctx, pbRemoteRef, w.providerRef, w.sourceNamespace)
}

// SecretExists checks if a secret exists in the provider.
func (w *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	// Convert esv1.PushSecretRemoteRef to pb.PushSecretRemoteRef
	pbRemoteRef := &pb.PushSecretRemoteRef{
		RemoteKey: remoteRef.GetRemoteKey(),
		Property:  remoteRef.GetProperty(),
	}

	return w.v2Provider.SecretExists(ctx, pbRemoteRef, w.providerRef, w.sourceNamespace)
}

// Validate checks if the provider is properly configured.
func (w *Client) Validate() (esv1.ValidationResult, error) {
	err := w.v2Provider.Validate(context.Background(), w.providerRef, w.sourceNamespace)
	if err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

// Close cleans up any resources held by the provider client.
func (w *Client) Close(ctx context.Context) error {
	return w.v2Provider.Close(ctx)
}
