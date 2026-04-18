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

package clientmanager

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	adapterstore "github.com/external-secrets/external-secrets/providers/v2/adapter/store"
	"github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

func (m *Manager) getV2ProviderStoreClient(ctx context.Context, storeName, callerNamespace string) (esv1.SecretsClient, error) {
	var store esv2alpha1.ProviderStore
	storeKey := types.NamespacedName{
		Name:      storeName,
		Namespace: callerNamespace,
	}
	if err := m.client.Get(ctx, storeKey, &store); err != nil {
		return nil, fmt.Errorf("failed to get ProviderStore %q: %w", storeName, err)
	}

	if m.enableFloodgate {
		if err := assertV2StoreIsUsable(&store); err != nil {
			return nil, err
		}
	}

	return m.getOrCreateProviderStoreClient(ctx, &store, callerNamespace, callerNamespace)
}

func (m *Manager) getV2ClusterProviderStoreClient(ctx context.Context, storeName, callerNamespace string) (esv1.SecretsClient, error) {
	var store esv2alpha1.ClusterProviderStore
	storeKey := types.NamespacedName{
		Name: storeName,
	}
	if err := m.client.Get(ctx, storeKey, &store); err != nil {
		return nil, fmt.Errorf("failed to get ClusterProviderStore %q: %w", storeName, err)
	}

	shouldProcess, err := m.validateProviderStoreNamespaceConditions(store.Spec.Conditions, callerNamespace)
	if err != nil {
		return nil, err
	}
	if !shouldProcess {
		return nil, fmt.Errorf(errClusterProviderStoreDenied, storeName, callerNamespace)
	}

	if m.enableFloodgate {
		if err := assertV2StoreIsUsable(&store); err != nil {
			return nil, err
		}
	}

	effectiveBackendNamespace := store.Spec.BackendRef.Namespace
	if effectiveBackendNamespace == "" {
		effectiveBackendNamespace = callerNamespace
	}

	return m.getOrCreateProviderStoreClient(ctx, &store, callerNamespace, effectiveBackendNamespace)
}

func (m *Manager) getOrCreateProviderStoreClient(ctx context.Context, store esv2alpha1.GenericStore, callerNamespace, effectiveBackendNamespace string) (esv1.SecretsClient, error) {
	cacheKeyType := v2ProviderStoreCacheKey
	isClusterScoped := false
	if store.GetKind() == esv1.ClusterProviderStoreKindStr {
		cacheKeyType = v2ClusterProviderStoreCache
		isClusterScoped = true
	}

	cacheKey := clientKey{
		providerType:        cacheKeyType,
		v2ProviderName:      store.GetName(),
		v2ProviderNamespace: callerNamespace,
	}

	if cached, ok := m.clientMap[cacheKey]; ok {
		if cached.v2ProviderGeneration == store.GetGeneration() {
			clientManagerMetrics.RecordCacheHit(providerMetricsLabelForScope(isClusterScoped))
			return cached.client, nil
		}
		clientManagerMetrics.RecordCacheInvalidation(providerMetricsLabelForScope(isClusterScoped), cacheInvalidationGeneration)
		delete(m.clientMap, cacheKey)
	}

	runtimeRef := store.GetRuntimeRef()
	runtimeKind := runtimeRef.Kind
	if runtimeKind == "" {
		runtimeKind = "ClusterProviderClass"
	}
	if runtimeKind != "ClusterProviderClass" {
		return nil, fmt.Errorf("unsupported runtimeRef kind %q", runtimeKind)
	}

	var runtimeClass esv1alpha1.ClusterProviderClass
	if err := m.client.Get(ctx, types.NamespacedName{Name: runtimeRef.Name}, &runtimeClass); err != nil {
		return nil, fmt.Errorf("failed to get ClusterProviderClass %q: %w", runtimeRef.Name, err)
	}

	if runtimeClass.Spec.Address == "" {
		return nil, fmt.Errorf("provider address is required in ClusterProviderClass %q", runtimeRef.Name)
	}

	tlsSecretNamespace := grpc.ResolveTLSSecretNamespace(runtimeClass.Spec.Address, "", "", effectiveBackendNamespace)
	tlsConfig, err := grpc.LoadClientTLSConfig(ctx, m.client, runtimeClass.Spec.Address, tlsSecretNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config for ClusterProviderClass %q: %w", runtimeRef.Name, err)
	}

	pool := getGlobalV2ConnectionPool()
	grpcClient, err := pool.Get(ctx, runtimeClass.Spec.Address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get gRPC client from pool for ClusterProviderClass %q: %w", runtimeRef.Name, err)
	}

	m.v2PooledConnections = append(m.v2PooledConnections, v2PooledConnection{
		address:   runtimeClass.Spec.Address,
		tlsConfig: tlsConfig,
	})

	backendRef := store.GetBackendRef()
	providerRef := &pb.ProviderReference{
		ApiVersion:   backendRef.APIVersion,
		Kind:         backendRef.Kind,
		Name:         backendRef.Name,
		Namespace:    effectiveBackendNamespace,
		StoreRefKind: store.GetKind(),
	}

	wrappedClient := adapterstore.NewClient(grpcClient, providerRef, callerNamespace)
	m.clientMap[cacheKey] = &clientVal{
		client:               wrappedClient,
		v2ProviderGeneration: store.GetGeneration(),
	}

	return wrappedClient, nil
}

func (m *Manager) validateProviderStoreNamespaceConditions(conditions []esv2alpha1.StoreNamespaceCondition, ns string) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	translated := make([]esv1.ClusterSecretStoreCondition, 0, len(conditions))
	for _, condition := range conditions {
		translated = append(translated, esv1.ClusterSecretStoreCondition{
			NamespaceSelector: condition.NamespaceSelector,
			Namespaces:        append([]string(nil), condition.Namespaces...),
			NamespaceRegexes:  append([]string(nil), condition.NamespaceRegexes...),
		})
	}

	return m.validateNamespaceConditions(translated, ns)
}

func assertV2StoreIsUsable(store esv2alpha1.GenericStore) error {
	if store == nil {
		return nil
	}

	condition := GetProviderStoreCondition(store.GetStoreStatus(), esv2alpha1.ProviderStoreReady)
	if condition == nil || condition.Status != corev1.ConditionTrue {
		return fmt.Errorf(errSecretStoreNotReady, store.GetKind(), store.GetName())
	}

	return nil
}

func GetProviderStoreCondition(status esv2alpha1.ProviderStoreStatus, condType esv2alpha1.ProviderStoreConditionType) *esv2alpha1.ProviderStoreCondition {
	for i := range status.Conditions {
		if status.Conditions[i].Type == condType {
			return &status.Conditions[i]
		}
	}
	return nil
}
