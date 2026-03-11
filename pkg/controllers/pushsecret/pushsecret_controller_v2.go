// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/clientmanager"
)

// isV2SecretStore checks if the referenced SecretStore is a v2 API version.
func (r *Reconciler) isV2SecretStore(ctx context.Context, storeRef esv1alpha1.PushSecretStoreRef, namespace string) bool {
	if storeRef.Kind == esapi.ProviderKindStr || storeRef.Kind == esapi.ClusterProviderKindStr {
		if storeRef.APIVersion == "" {
			return true
		}
		return storeRef.APIVersion == esapi.SchemeGroupVersion.String()
	}

	// Check the apiVersion field first if specified
	if storeRef.APIVersion != "" {
		return false
	}

	if storeRef.Kind == esapi.ClusterProviderKindStr {
		var store esapi.ClusterProvider
		err := r.Client.Get(ctx, types.NamespacedName{Name: storeRef.Name}, &store)
		return err == nil
	}

	// For backwards compatibility, try to fetch as namespaced v2 Provider.
	var store esapi.Provider
	storeKey := types.NamespacedName{Name: storeRef.Name, Namespace: namespace}
	err := r.Client.Get(ctx, storeKey, &store)
	return err == nil
}

// GetSecretStoresV2 retrieves both v1 and v2 Providers.
func (r *Reconciler) GetSecretStoresV2(ctx context.Context, ps esv1alpha1.PushSecret) (map[esv1alpha1.PushSecretStoreRef]interface{}, error) {
	stores := make(map[esv1alpha1.PushSecretStoreRef]interface{})

	for _, refStore := range ps.Spec.SecretStoreRefs {
		if refStore.LabelSelector != nil {
			labelSelector, err := metav1.LabelSelectorAsSelector(refStore.LabelSelector)
			if err != nil {
				return nil, fmt.Errorf("could not convert labels: %w", err)
			}

			if refStore.Kind == esapi.ClusterSecretStoreKind {
				clusterSecretStoreList := esapi.ClusterSecretStoreList{}
				err = r.List(ctx, &clusterSecretStoreList, &client.ListOptions{LabelSelector: labelSelector})
				if err != nil {
					return nil, fmt.Errorf("could not list cluster Secret Stores: %w", err)
				}
				for k, v := range clusterSecretStoreList.Items {
					key := esv1alpha1.PushSecretStoreRef{
						Name: v.Name,
						Kind: esapi.ClusterSecretStoreKind,
					}
					stores[key] = &clusterSecretStoreList.Items[k]
				}
				continue
			}

			secretStoreList := esapi.SecretStoreList{}
			err = r.List(ctx, &secretStoreList, &client.ListOptions{LabelSelector: labelSelector, Namespace: ps.Namespace})
			if err != nil {
				return nil, fmt.Errorf("could not list Secret Stores: %w", err)
			}
			for k, v := range secretStoreList.Items {
				key := esv1alpha1.PushSecretStoreRef{
					Name: v.Name,
					Kind: esapi.SecretStoreKind,
				}
				stores[key] = &secretStoreList.Items[k]
			}
			continue
		}

		// Check if this is a v2 Provider
		if r.isV2SecretStore(ctx, refStore, ps.Namespace) {
			if refStore.Kind == esapi.ClusterProviderKindStr {
				var store esapi.ClusterProvider
				storeKey := types.NamespacedName{Name: refStore.Name}
				if err := r.Client.Get(ctx, storeKey, &store); err != nil {
					return nil, fmt.Errorf("failed to get v2 ClusterProvider %s: %w", refStore.Name, err)
				}
				stores[refStore] = &store
				continue
			}

			var store esapi.Provider
			storeKey := types.NamespacedName{Name: refStore.Name, Namespace: ps.Namespace}
			if err := r.Client.Get(ctx, storeKey, &store); err != nil {
				return nil, fmt.Errorf("failed to get v2 Provider %s: %w", refStore.Name, err)
			}
			stores[refStore] = &store
			continue
		}

		// Get v1 SecretStore (existing implementation).
		store, err := r.getSecretStoreFromName(ctx, refStore, ps.Namespace)
		if err != nil {
			return nil, err
		}
		stores[refStore] = store
	}

	return stores, nil
}

// PushSecretToProvidersV2 pushes secret data to both v1 stores and v2 providers.
func (r *Reconciler) PushSecretToProvidersV2(
	ctx context.Context,
	stores map[esv1alpha1.PushSecretStoreRef]interface{},
	ps esv1alpha1.PushSecret,
	secret *corev1.Secret,
	mgr *clientmanager.Manager,
) (esv1alpha1.SyncedPushSecretsMap, error) {
	out := make(esv1alpha1.SyncedPushSecretsMap)
	for ref := range stores {
		var err error
		out, err = r.handlePushSecretDataForStore(ctx, ps, secret, out, mgr, ref.Name, ref.Kind)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// DeleteSecretFromProvidersV2 removes secrets from v2 providers when they're no longer needed.
func (r *Reconciler) DeleteSecretFromProvidersV2(
	ctx context.Context,
	ps *esv1alpha1.PushSecret,
	newMap esv1alpha1.SyncedPushSecretsMap,
	_ map[esv1alpha1.PushSecretStoreRef]interface{},
) (esv1alpha1.SyncedPushSecretsMap, error) {
	out := mergeSecretState(newMap, ps.Status.SyncedPushSecrets)
	mgr := clientmanager.NewManager(r.Client, r.ControllerClass, false)
	defer func() {
		_ = mgr.Close(ctx)
	}()

	for storeName, oldData := range ps.Status.SyncedPushSecrets {
		// Parse store name format "Kind/Name"
		parts := strings.Split(storeName, "/")
		if len(parts) != 2 {
			continue
		}
		storeKind := parts[0]
		storeNameOnly := parts[1]

		secretClient, err := mgr.Get(ctx, esapi.SecretStoreRef{
			Name: storeNameOnly,
			Kind: storeKind,
		}, ps.Namespace, nil)
		if err != nil {
			return out, fmt.Errorf("could not get secrets client for store %v: %w", storeName, err)
		}

		newData, ok := newMap[storeName]
		if !ok {
			err = r.DeleteAllSecretsFromStore(ctx, secretClient, oldData)
			if err != nil {
				return out, err
			}
			delete(out, storeName)
			continue
		}
		for oldEntry, oldRef := range oldData {
			if _, stillExists := newData[oldEntry]; !stillExists {
				err = r.DeleteSecretFromStore(ctx, secretClient, oldRef)
				if err != nil {
					return out, err
				}
				delete(out[storeName], oldEntry)
			}
		}
	}

	return out, nil
}
