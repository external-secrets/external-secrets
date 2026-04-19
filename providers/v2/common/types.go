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

// Package common provides the v2 provider interface for out-of-tree providers communicating via gRPC.
package common

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

// Provider is the interface that v2 out-of-tree providers must satisfy.
// Unlike v1 providers which are compiled into ESO, v2 providers run as separate services
// and communicate with ESO via gRPC.
type Provider interface {
	// GetSecret retrieves a single secret from the provider.
	// If the secret doesn't exist, it should return an error.
	// At least one of providerRef or compatibilityStore must be provided.
	// If both are present, compatibilityStore takes precedence for read operations.
	// sourceNamespace is the namespace of the ExternalSecret.
	GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef, providerRef *pb.ProviderReference, compatibilityStore *pb.CompatibilityStore, sourceNamespace string) ([]byte, error)

	// GetSecretMap retrieves multiple key/value pairs from a single secret object.
	// At least one of providerRef or compatibilityStore must be provided.
	// If both are present, compatibilityStore takes precedence for read operations.
	// sourceNamespace is the namespace of the ExternalSecret.
	GetSecretMap(
		ctx context.Context,
		ref esv1.ExternalSecretDataRemoteRef,
		providerRef *pb.ProviderReference,
		compatibilityStore *pb.CompatibilityStore,
		sourceNamespace string,
	) (map[string][]byte, error)

	// GetAllSecrets retrieves multiple secrets based on find criteria.
	// Returns a map of secret names to their byte values.
	// At least one of providerRef or compatibilityStore must be provided.
	// If both are present, compatibilityStore takes precedence for read operations.
	// sourceNamespace is the namespace of the ExternalSecret.
	GetAllSecrets(ctx context.Context, find esv1.ExternalSecretFind, providerRef *pb.ProviderReference, compatibilityStore *pb.CompatibilityStore, sourceNamespace string) (map[string][]byte, error)

	// PushSecret writes a secret to the provider.
	// The secret is the Kubernetes Secret object to push, and pushSecretData contains the push configuration.
	// At least one of providerRef or compatibilityStore must be provided.
	// sourceNamespace is the namespace of the PushSecret.
	PushSecret(
		ctx context.Context,
		secret *corev1.Secret,
		pushSecretData *pb.PushSecretData,
		providerRef *pb.ProviderReference,
		compatibilityStore *pb.CompatibilityStore,
		sourceNamespace string,
	) error

	// DeleteSecret deletes a secret from the provider.
	// At least one of providerRef or compatibilityStore must be provided.
	// sourceNamespace is the namespace of the PushSecret.
	DeleteSecret(ctx context.Context, remoteRef *pb.PushSecretRemoteRef, providerRef *pb.ProviderReference, compatibilityStore *pb.CompatibilityStore, sourceNamespace string) error

	// SecretExists checks if a secret exists in the provider.
	// At least one of providerRef or compatibilityStore must be provided.
	// sourceNamespace is the namespace of the PushSecret.
	SecretExists(ctx context.Context, remoteRef *pb.PushSecretRemoteRef, providerRef *pb.ProviderReference, compatibilityStore *pb.CompatibilityStore, sourceNamespace string) (bool, error)

	// Validate checks if the provider is properly configured and can communicate with the backend.
	// This is called by the SecretStore controller during reconciliation.
	// At least one of providerRef or compatibilityStore must be provided.
	// sourceNamespace is the namespace of the requesting store.
	Validate(ctx context.Context, providerRef *pb.ProviderReference, compatibilityStore *pb.CompatibilityStore, sourceNamespace string) error

	// Capabilities returns what operations the provider supports (ReadOnly, WriteOnly, ReadWrite).
	// The providerRef references the provider configuration CRD, and sourceNamespace is the namespace of the Provider.
	Capabilities(ctx context.Context, providerRef *pb.ProviderReference, sourceNamespace string) (pb.SecretStoreCapabilities, error)

	// Close cleans up any resources held by the provider client.
	Close(ctx context.Context) error
}
