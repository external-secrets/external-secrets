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
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/clientmanager"
)

// isV2SecretStore checks if the referenced SecretStore is a v2 API version.
func (r *Reconciler) isV2SecretStore(ctx context.Context, storeRef esv1alpha1.PushSecretStoreRef, namespace string) bool {
	_, ok, err := r.resolveV2Store(ctx, storeRef, namespace)
	return err == nil && ok
}

// GetSecretStoresV2 retrieves both v1 and v2 Providers.
func (r *Reconciler) GetSecretStoresV2(ctx context.Context, ps esv1alpha1.PushSecret) (map[esv1alpha1.PushSecretStoreRef]interface{}, error) {
	stores := make(map[esv1alpha1.PushSecretStoreRef]interface{})

	for _, refStore := range ps.Spec.SecretStoreRefs {
		if refStore.LabelSelector != nil {
			resolvedStores, err := r.getSecretStoresFromSelectorV2(ctx, refStore, ps.Namespace)
			if err != nil {
				return nil, err
			}
			for resolvedRef, store := range resolvedStores {
				stores[resolvedRef] = store
			}
			continue
		}

		if store, ok, err := r.resolveV2Store(ctx, refStore, ps.Namespace); err != nil {
			return nil, err
		} else if ok {
			stores[refStore] = store
			continue
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

func (r *Reconciler) getSecretStoresFromSelectorV2(ctx context.Context, storeRef esv1alpha1.PushSecretStoreRef, namespace string) (map[esv1alpha1.PushSecretStoreRef]interface{}, error) {
	selector, err := metav1.LabelSelectorAsSelector(storeRef.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("could not convert labels: %w", err)
	}

	listOptions := &client.ListOptions{LabelSelector: selector}
	stores := make(map[esv1alpha1.PushSecretStoreRef]interface{})

	switch storeRef.Kind {
	case esapi.ProviderKindStr:
		listOptions.Namespace = namespace
		var providerList esv1.ProviderList
		if err := r.List(ctx, &providerList, listOptions); err != nil {
			return nil, fmt.Errorf("could not list Providers: %w", err)
		}
		for i := range providerList.Items {
			store := &providerList.Items[i]
			stores[esv1alpha1.PushSecretStoreRef{Name: store.Name, Kind: esapi.ProviderKindStr}] = store
		}
	case esapi.ClusterProviderKindStr:
		var providerList esv1.ClusterProviderList
		if err := r.List(ctx, &providerList, listOptions); err != nil {
			return nil, fmt.Errorf("could not list ClusterProviders: %w", err)
		}
		for i := range providerList.Items {
			store := &providerList.Items[i]
			stores[esv1alpha1.PushSecretStoreRef{Name: store.Name, Kind: esapi.ClusterProviderKindStr}] = store
		}
	case esv1.ClusterSecretStoreKind:
		var storeList esv1.ClusterSecretStoreList
		if err := r.List(ctx, &storeList, listOptions); err != nil {
			return nil, fmt.Errorf("could not list cluster Secret Stores: %w", err)
		}
		for i := range storeList.Items {
			store := &storeList.Items[i]
			stores[esv1alpha1.PushSecretStoreRef{Name: store.Name, Kind: esv1.ClusterSecretStoreKind}] = store
		}
	default:
		listOptions.Namespace = namespace
		var storeList esv1.SecretStoreList
		if err := r.List(ctx, &storeList, listOptions); err != nil {
			return nil, fmt.Errorf("could not list Secret Stores: %w", err)
		}
		for i := range storeList.Items {
			store := &storeList.Items[i]
			stores[esv1alpha1.PushSecretStoreRef{Name: store.Name, Kind: esv1.SecretStoreKind}] = store
		}
	}

	return stores, nil
}

func (r *Reconciler) resolveV2Store(ctx context.Context, storeRef esv1alpha1.PushSecretStoreRef, namespace string) (interface{}, bool, error) {
	if storeRef.APIVersion != "" && storeRef.APIVersion != esapi.SchemeGroupVersion.String() {
		return nil, false, nil
	}
	if storeRef.Name == "" {
		return nil, false, nil
	}

	switch storeRef.Kind {
	case esapi.ClusterProviderKindStr:
		var store esapi.ClusterProvider
		storeKey := types.NamespacedName{Name: storeRef.Name}
		if err := r.Client.Get(ctx, storeKey, &store); err != nil {
			return nil, true, fmt.Errorf("failed to get v2 ClusterProvider %s: %w", storeRef.Name, err)
		}
		return &store, true, nil
	case esapi.ProviderKindStr:
		var store esapi.Provider
		storeKey := types.NamespacedName{Name: storeRef.Name, Namespace: namespace}
		if err := r.Client.Get(ctx, storeKey, &store); err != nil {
			return nil, true, fmt.Errorf("failed to get v2 Provider %s: %w", storeRef.Name, err)
		}
		return &store, true, nil
	case "":
		var provider esapi.Provider
		providerKey := types.NamespacedName{Name: storeRef.Name, Namespace: namespace}
		if err := r.Client.Get(ctx, providerKey, &provider); err == nil {
			return &provider, true, nil
		}

		var clusterProvider esapi.ClusterProvider
		clusterProviderKey := types.NamespacedName{Name: storeRef.Name}
		if err := r.Client.Get(ctx, clusterProviderKey, &clusterProvider); err == nil {
			return &clusterProvider, true, nil
		}
	}

	return nil, false, nil
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
	for ref, store := range stores {
		si, ok := resolvedStoreInfo(ref, store)
		if !ok {
			continue
		}

		var err error
		out, err = r.handlePushSecretDataForStore(ctx, ps, secret, out, mgr, si)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// DeleteSecretFromProvidersV2 removes secrets from v2 providers when they're no longer needed.
func (r *Reconciler) DeleteSecretFromProvidersV2(ctx context.Context, ps *esv1alpha1.PushSecret, newMap esv1alpha1.SyncedPushSecretsMap, stores map[esv1alpha1.PushSecretStoreRef]interface{}) (esv1alpha1.SyncedPushSecretsMap, error) {
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
