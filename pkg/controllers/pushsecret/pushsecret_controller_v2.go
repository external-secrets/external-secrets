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

package pushsecret

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	"github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

// isV2SecretStore checks if the referenced SecretStore is a v2 API version.
func (r *Reconciler) isV2SecretStore(ctx context.Context, storeRef esv1alpha1.PushSecretStoreRef, namespace string) bool {
	// Check the apiVersion field first if specified
	if storeRef.APIVersion != "" {
		return storeRef.APIVersion == "external-secrets.io/v1"
	}

	// For backwards compatibility, try to fetch as v2 Provider
	var store esapi.Provider
	storeKey := types.NamespacedName{
		Name:      storeRef.Name,
		Namespace: namespace,
	}
	err := r.Client.Get(ctx, storeKey, &store)
	return err == nil
}

// pushSecretToProviderV2 pushes a secret to a v2 provider via gRPC.
func (r *Reconciler) pushSecretToProviderV2(ctx context.Context, storeRef esv1alpha1.PushSecretStoreRef, ps esv1alpha1.PushSecret, secret *corev1.Secret) (map[string]esv1alpha1.PushSecretData, error) {
	// Get the v2 Provider
	var store esapi.Provider
	storeKey := types.NamespacedName{
		Name:      storeRef.Name,
		Namespace: ps.Namespace,
	}
	if err := r.Client.Get(ctx, storeKey, &store); err != nil {
		return nil, fmt.Errorf("failed to get Provider: %w", err)
	}

	// Get provider address
	address := store.Spec.Config.Address
	if address == "" {
		return nil, fmt.Errorf("provider address is required in Provider")
	}

	// Load TLS configuration
	tlsConfig, err := grpc.LoadClientTLSConfig(ctx, r.Client, store.Spec.Config.Address, "external-secrets-system")
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Create gRPC client with TLS
	grpcClient, err := grpc.NewClient(address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer func() { _ = grpcClient.Close(ctx) }()

	// Convert ProviderReference to protobuf format
	providerRef := &pb.ProviderReference{
		ApiVersion: store.Spec.Config.ProviderRef.APIVersion,
		Kind:       store.Spec.Config.ProviderRef.Kind,
		Name:       store.Spec.Config.ProviderRef.Name,
		Namespace:  store.Spec.Config.ProviderRef.Namespace,
	}

	// Push secrets
	syncedSecrets := make(map[string]esv1alpha1.PushSecretData)

	// Convert secret data to map[string][]byte
	secretData := make(map[string][]byte)
	for k, v := range secret.Data {
		secretData[k] = v
	}

	// Push each data item
	for _, data := range ps.Spec.Data {
		// Prepare push secret data
		var metadataJSON []byte
		if data.Metadata != nil {
			metadataJSON = data.Metadata.Raw
		}

		pushData := &pb.PushSecretData{
			Metadata:  metadataJSON,
			SecretKey: data.Match.SecretKey,
			RemoteKey: data.Match.RemoteRef.RemoteKey,
			Property:  data.Match.RemoteRef.Property,
		}

		remoteRef := &pb.PushSecretRemoteRef{
			RemoteKey: data.Match.RemoteRef.RemoteKey,
			Property:  data.Match.RemoteRef.Property,
		}

		// Handle UpdatePolicy
		switch ps.Spec.UpdatePolicy {
		case esv1alpha1.PushSecretUpdatePolicyIfNotExists:
			// Check if secret already exists
			exists, err := grpcClient.SecretExists(ctx, remoteRef, providerRef, ps.Namespace)
			if err != nil {
				return syncedSecrets, fmt.Errorf("could not verify if secret exists: %w", err)
			}
			if exists {
				// Secret exists, skip push but record it as synced
				syncedSecrets[data.Match.RemoteRef.RemoteKey] = data
				continue
			}
		case esv1alpha1.PushSecretUpdatePolicyReplace:
			// Always push (replace existing)
		default:
			// Default to replace
		}

		// Push the secret
		if err := grpcClient.PushSecret(ctx, secretData, pushData, providerRef, ps.Namespace); err != nil {
			return syncedSecrets, fmt.Errorf("failed to push secret for key %s: %w", data.Match.RemoteRef.RemoteKey, err)
		}

		// Record successful push
		syncedSecrets[data.Match.RemoteRef.RemoteKey] = data
	}

	return syncedSecrets, nil
}

// GetSecretStoresV2 retrieves both v1 and v2 Providers.
func (r *Reconciler) GetSecretStoresV2(ctx context.Context, ps esv1alpha1.PushSecret) (map[esv1alpha1.PushSecretStoreRef]interface{}, error) {
	stores := make(map[esv1alpha1.PushSecretStoreRef]interface{})

	for _, refStore := range ps.Spec.SecretStoreRefs {
		// Check if this is a v2 Provider
		if r.isV2SecretStore(ctx, refStore, ps.Namespace) {
			// Get v2 Provider
			var store esapi.Provider
			storeKey := types.NamespacedName{
				Name:      refStore.Name,
				Namespace: ps.Namespace,
			}
			if err := r.Client.Get(ctx, storeKey, &store); err != nil {
				return nil, fmt.Errorf("failed to get v2 Provider %s: %w", refStore.Name, err)
			}
			stores[refStore] = &store
		} else {
			// Get v1 SecretStore (existing implementation)
			store, err := r.getSecretStoreFromName(ctx, refStore, ps.Namespace)
			if err != nil {
				return nil, err
			}
			stores[refStore] = store
		}
	}

	return stores, nil
}

// handleV2Push determines if a store is v2 and pushes accordingly.
func (r *Reconciler) handleV2Push(ctx context.Context, storeRef esv1alpha1.PushSecretStoreRef, store interface{}, ps esv1alpha1.PushSecret, secret *corev1.Secret) (map[string]esv1alpha1.PushSecretData, error) {
	// Check if this is a v2 store
	if _, ok := store.(*esapi.Provider); ok {
		// Use v2 push
		return r.pushSecretToProviderV2(ctx, storeRef, ps, secret)
	}

	// Not a v2 store, return nil to indicate v1 handling should be used
	return nil, nil
}

// DeleteSecretFromProvidersV2 removes secrets from v2 providers when they're no longer needed.
func (r *Reconciler) DeleteSecretFromProvidersV2(ctx context.Context, ps *esv1alpha1.PushSecret, newMap esv1alpha1.SyncedPushSecretsMap, stores map[esv1alpha1.PushSecretStoreRef]interface{}) (esv1alpha1.SyncedPushSecretsMap, error) {
	out := mergeSecretState(newMap, ps.Status.SyncedPushSecrets)

	for storeName, oldData := range ps.Status.SyncedPushSecrets {
		// Parse store name format "Kind/Name"
		parts := strings.Split(storeName, "/")
		if len(parts) != 2 {
			continue
		}
		storeKind := parts[0]
		storeNameOnly := parts[1]

		// Find the matching store
		var matchingStore interface{}
		var found bool
		for ref, store := range stores {
			if ref.Kind == storeKind && ref.Name == storeNameOnly {
				matchingStore = store
				found = true
				break
			}
		}

		if !found {
			// Store no longer referenced, skip deletion
			continue
		}

		// Check if it's a v2 store
		if v2Store, ok := matchingStore.(*esapi.Provider); ok {
			// Create gRPC client
			tlsConfig, err := grpc.LoadClientTLSConfig(ctx, r.Client, v2Store.Spec.Config.Address, "external-secrets-system")
			if err != nil {
				return out, fmt.Errorf("failed to load TLS config: %w", err)
			}

			grpcClient, err := grpc.NewClient(v2Store.Spec.Config.Address, tlsConfig)
			if err != nil {
				return out, fmt.Errorf("failed to create gRPC client: %w", err)
			}
			defer func() { _ = grpcClient.Close(ctx) }()

			// Convert ProviderReference to protobuf format
			providerRef := &pb.ProviderReference{
				ApiVersion: v2Store.Spec.Config.ProviderRef.APIVersion,
				Kind:       v2Store.Spec.Config.ProviderRef.Kind,
				Name:       v2Store.Spec.Config.ProviderRef.Name,
				Namespace:  v2Store.Spec.Config.ProviderRef.Namespace,
			}

			// Check if store still exists in newMap
			newData, ok := newMap[storeName]
			if !ok {
				// Store removed entirely, delete all secrets
				for _, oldRef := range oldData {
					remoteRef := &pb.PushSecretRemoteRef{
						RemoteKey: oldRef.Match.RemoteRef.RemoteKey,
						Property:  oldRef.Match.RemoteRef.Property,
					}
					if err := grpcClient.DeleteSecret(ctx, remoteRef, providerRef, ps.Namespace); err != nil {
						return out, fmt.Errorf("failed to delete secret %s: %w", oldRef.Match.RemoteRef.RemoteKey, err)
					}
				}
				delete(out, storeName)
				continue
			}

			// Delete individual secrets that are no longer in the new data
			for oldEntry, oldRef := range oldData {
				if _, stillExists := newData[oldEntry]; !stillExists {
					remoteRef := &pb.PushSecretRemoteRef{
						RemoteKey: oldRef.Match.RemoteRef.RemoteKey,
						Property:  oldRef.Match.RemoteRef.Property,
					}
					if err := grpcClient.DeleteSecret(ctx, remoteRef, providerRef, ps.Namespace); err != nil {
						return out, fmt.Errorf("failed to delete secret %s: %w", oldRef.Match.RemoteRef.RemoteKey, err)
					}
					delete(out[storeName], oldRef.Match.RemoteRef.RemoteKey)
				}
			}
		}
		// If not v2, the v1 deletion logic will handle it
	}

	return out, nil
}
