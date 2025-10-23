/*
Copyright © 2025 ESO Maintainer Team

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

// Package clientmanager provides a Manager for provider clients
package clientmanager

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	adapterstore "github.com/external-secrets/external-secrets/providers/v2/adapter/store"
	"github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

const (
	errGetClusterSecretStore = "could not get ClusterSecretStore %q, %w"
	errGetSecretStore        = "could not get SecretStore %q, %w"
	errSecretStoreNotReady   = "%s %q is not ready"
	errClusterStoreMismatch  = "using cluster store %q is not allowed from namespace %q: denied by spec.condition"
	errClusterProviderDenied = "using ClusterProvider %q is not allowed from namespace %q: denied by spec.conditions"
)

var (
	// globalV2ConnectionPool is a singleton connection pool for v2 gRPC providers.
	// It persists across all reconciles and Manager instances to enable connection reuse.
	// Initialized once on first use and shared globally.
	globalV2ConnectionPool     *grpc.ConnectionPool
	globalV2ConnectionPoolOnce sync.Once
	globalV2ConnectionPoolLog  logr.Logger
)

// initGlobalV2ConnectionPool initializes the global connection pool for v2 providers.
// This is called once on first use via sync.Once.
func initGlobalV2ConnectionPool() {
	globalV2ConnectionPoolLog = ctrl.Log.WithName("v2-connection-pool")
	poolConfig := grpc.DefaultPoolConfig()
	globalV2ConnectionPool = grpc.NewConnectionPool(poolConfig)
	globalV2ConnectionPoolLog.Info("global v2 connection pool initialized",
		"maxIdleTime", poolConfig.MaxIdleTime.String(),
		"maxLifetime", poolConfig.MaxLifetime.String(),
		"healthCheckInterval", poolConfig.HealthCheckInterval.String())
}

// getGlobalV2ConnectionPool returns the global connection pool, initializing it if needed.
func getGlobalV2ConnectionPool() *grpc.ConnectionPool {
	globalV2ConnectionPoolOnce.Do(initGlobalV2ConnectionPool)
	return globalV2ConnectionPool
}

// v2PooledConnection tracks connection info needed to release connections back to the pool.
type v2PooledConnection struct {
	address   string
	tlsConfig *grpc.TLSConfig
}

// Manager stores instances of provider clients
// At any given time we must have no more than one instance
// of a client (due to limitations in GCP / see mutexlock there)
// If the controller requests another instance of a given client
// we will close the old client first and then construct a new one.
type Manager struct {
	log             logr.Logger
	client          client.Client
	controllerClass string
	enableFloodgate bool

	// store clients by provider type
	clientMap map[clientKey]*clientVal

	// Track v2 provider connections for release back to pool
	v2PooledConnections []v2PooledConnection
}

type clientKey struct {
	providerType string
	// For v2 providers, store the provider name and namespace
	v2ProviderName      string
	v2ProviderNamespace string
}

type clientVal struct {
	client esv1.SecretsClient
	store  esv1.GenericStore
	// For v2 providers, store the generation for cache invalidation
	v2ProviderGeneration int64
}

// v2ProviderConfig contains configuration for creating a v2 provider client.
type v2ProviderConfig struct {
	name              string
	resourceNamespace string // empty for cluster-scoped resources
	manifestNamespace string // namespace of the ExternalSecret/PushSecret
	config            esv1.ProviderConfig
	generation        int64
	isClusterScoped   bool
	kindStr           string // "Provider" or "ClusterProvider"
}

// NewManager constructs a new manager with defaults.
func NewManager(ctrlClient client.Client, controllerClass string, enableFloodgate bool) *Manager {
	log := ctrl.Log.WithName("clientmanager")
	return &Manager{
		log:             log,
		client:          ctrlClient,
		controllerClass: controllerClass,
		enableFloodgate: enableFloodgate,
		clientMap:       make(map[clientKey]*clientVal),
	}
}

// GetFromStore returns a provider client from the given store.
// Do not close the client returned from this func, instead close
// the manager once you're done with reconciling the external secret.
func (m *Manager) GetFromStore(ctx context.Context, store esv1.GenericStore, namespace string) (esv1.SecretsClient, error) {
	storeProvider, err := esv1.GetProvider(store)
	if err != nil {
		return nil, err
	}
	secretClient := m.getStoredClient(ctx, storeProvider, store)
	if secretClient != nil {
		return secretClient, nil
	}
	m.log.V(1).Info("creating new client",
		"provider", fmt.Sprintf("%T", storeProvider),
		"store", fmt.Sprintf("%s/%s", store.GetNamespace(), store.GetName()))
	// secret client is created only if we are going to refresh
	// this skip an unnecessary check/request in the case we are not going to do anything
	secretClient, err = storeProvider.NewClient(ctx, store, m.client, namespace)
	if err != nil {
		return nil, err
	}
	idx := storeKey(storeProvider)
	m.clientMap[idx] = &clientVal{
		client: secretClient,
		store:  store,
	}
	return secretClient, nil
}

// Get returns a provider client from the given storeRef or sourceRef.secretStoreRef
// while sourceRef.SecretStoreRef takes precedence over storeRef.
// Do not close the client returned from this func, instead close
// the manager once you're done with recinciling the external secret.
func (m *Manager) Get(ctx context.Context, storeRef esv1.SecretStoreRef, namespace string, sourceRef *esv1.StoreGeneratorSourceRef) (esv1.SecretsClient, error) {
	if storeRef.Kind == esv1.ProviderKindStr {
		return m.getV2ProviderClient(ctx, storeRef.Name, namespace)
	}
	if storeRef.Kind == esv1.ClusterProviderKindStr {
		return m.getV2ClusterProviderClient(ctx, storeRef.Name, namespace)
	}
	if sourceRef != nil && sourceRef.SecretStoreRef != nil {
		storeRef = *sourceRef.SecretStoreRef
	}
	store, err := m.getStore(ctx, &storeRef, namespace)
	if err != nil {
		return nil, err
	}
	// check if store should be handled by this controller instance
	if !ShouldProcessStore(store, m.controllerClass) {
		return nil, errors.New("can not reference unmanaged store")
	}
	// when using ClusterSecretStore, validate the ClusterSecretStore namespace conditions
	shouldProcess, err := m.shouldProcessSecret(store, namespace)
	if err != nil {
		return nil, err
	}
	if !shouldProcess {
		return nil, fmt.Errorf(errClusterStoreMismatch, store.GetName(), namespace)
	}

	if m.enableFloodgate {
		err := assertStoreIsUsable(store)
		if err != nil {
			return nil, err
		}
	}
	return m.GetFromStore(ctx, store, namespace)
}

// getOrCreateV2Client is a shared helper for creating or retrieving v2 provider clients.
// It handles caching, connection pooling, and client lifecycle for both Provider and ClusterProvider.
func (m *Manager) getOrCreateV2Client(ctx context.Context, cfg v2ProviderConfig, authNamespace string) (esv1.SecretsClient, error) {
	// Determine cache key type based on resource type
	cacheKeyType := "v2-provider"
	if cfg.isClusterScoped {
		cacheKeyType = "v2-cluster-provider"
	}

	// Create cache key
	cacheKey := clientKey{
		providerType:        cacheKeyType,
		v2ProviderName:      cfg.name,
		v2ProviderNamespace: cfg.manifestNamespace,
	}

	// Check if we have a cached client
	if cached, ok := m.clientMap[cacheKey]; ok {
		if cached.v2ProviderGeneration == cfg.generation {
			m.log.V(1).Info("reusing cached v2 provider client",
				cfg.kindStr, cfg.name,
				"manifestNamespace", cfg.manifestNamespace,
				"authNamespace", authNamespace,
				"generation", cfg.generation)
			// Record cache hit
			providerType := "provider"
			if cfg.isClusterScoped {
				providerType = "cluster-provider"
			}
			clientManagerMetrics.RecordCacheHit(providerType)
			return cached.client, nil
		}
		// Cache is stale, invalidate
		m.log.V(1).Info("provider generation changed, invalidating cache",
			cfg.kindStr, cfg.name,
			"manifestNamespace", cfg.manifestNamespace,
			"oldGeneration", cached.v2ProviderGeneration,
			"newGeneration", cfg.generation)
		// Record cache invalidation
		providerType := "provider"
		if cfg.isClusterScoped {
			providerType = "cluster-provider"
		}
		clientManagerMetrics.RecordCacheInvalidation(providerType, "generation_change")
		delete(m.clientMap, cacheKey)
	}

	m.log.V(1).Info("getting v2 provider client from pool",
		cfg.kindStr, cfg.name,
		"manifestNamespace", cfg.manifestNamespace,
		"authNamespace", authNamespace,
		"address", cfg.config.Address)

	// Get provider address
	address := cfg.config.Address
	if address == "" {
		return nil, fmt.Errorf("provider address is required in %s %q", cfg.kindStr, cfg.name)
	}

	// Load TLS configuration
	tlsConfig, err := grpc.LoadClientTLSConfig(ctx, m.client, cfg.config.Address, "external-secrets-system")
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config for %s %q: %w", cfg.kindStr, cfg.name, err)
	}

	// Get connection from global pool
	pool := getGlobalV2ConnectionPool()
	grpcClient, err := pool.Get(ctx, address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get gRPC client from pool for %s %q: %w", cfg.kindStr, cfg.name, err)
	}

	// Track this connection for release when Manager closes
	m.v2PooledConnections = append(m.v2PooledConnections, v2PooledConnection{
		address:   address,
		tlsConfig: tlsConfig,
	})

	// Convert ProviderReference to protobuf format
	providerRef := &pb.ProviderReference{
		ApiVersion: cfg.config.ProviderRef.APIVersion,
		Kind:       cfg.config.ProviderRef.Kind,
		Name:       cfg.config.ProviderRef.Name,
		Namespace:  cfg.config.ProviderRef.Namespace,
	}

	// Wrap with V2ClientWrapper
	wrappedClient := adapterstore.NewClient(grpcClient, providerRef, authNamespace)

	// Cache the client for this Manager instance
	m.clientMap[cacheKey] = &clientVal{
		client:               wrappedClient,
		store:                nil, // v2 providers don't use GenericStore
		v2ProviderGeneration: cfg.generation,
	}

	m.log.Info("v2 provider client obtained from pool",
		cfg.kindStr, cfg.name,
		"manifestNamespace", cfg.manifestNamespace,
		"authNamespace", authNamespace,
		"address", address)

	return wrappedClient, nil
}

// getV2ProviderClient creates or retrieves a cached gRPC client for a v2 Provider.
// It uses the global connection pool to enable connection reuse across reconciles.
func (m *Manager) getV2ProviderClient(ctx context.Context, providerName, namespace string) (esv1.SecretsClient, error) {
	// Fetch the Provider resource
	var provider esv1.Provider
	providerKey := types.NamespacedName{
		Name:      providerName,
		Namespace: namespace,
	}
	if err := m.client.Get(ctx, providerKey, &provider); err != nil {
		return nil, fmt.Errorf("failed to get Provider %q: %w", providerName, err)
	}

	// Build configuration for the helper
	cfg := v2ProviderConfig{
		name:              providerName,
		resourceNamespace: namespace,
		manifestNamespace: namespace,
		config:            provider.Spec.Config,
		generation:        provider.Generation,
		isClusterScoped:   false,
		kindStr:           esv1.ProviderKindStr,
	}

	// For namespace-scoped Provider, auth namespace is always the manifest namespace
	return m.getOrCreateV2Client(ctx, cfg, namespace)
}

// getV2ClusterProviderClient creates or retrieves a cached gRPC client for a v2 ClusterProvider.
// It uses the global connection pool to enable connection reuse across reconciles.
func (m *Manager) getV2ClusterProviderClient(ctx context.Context, providerName, namespace string) (esv1.SecretsClient, error) {
	// Fetch the ClusterProvider resource (cluster-scoped)
	var clusterProvider esv1.ClusterProvider
	providerKey := types.NamespacedName{
		Name: providerName,
	}
	if err := m.client.Get(ctx, providerKey, &clusterProvider); err != nil {
		return nil, fmt.Errorf("failed to get ClusterProvider %q: %w", providerName, err)
	}

	// Validate namespace conditions
	shouldProcess, err := m.validateNamespaceConditions(clusterProvider.Spec.Conditions, namespace)
	if err != nil {
		return nil, err
	}
	if !shouldProcess {
		return nil, fmt.Errorf(errClusterProviderDenied, providerName, namespace)
	}

	// Determine authentication namespace based on authenticationScope
	authNamespace := namespace // default to ManifestNamespace
	if clusterProvider.Spec.AuthenticationScope == esv1.AuthenticationScopeProviderNamespace {
		// Use namespace from providerRef
		if clusterProvider.Spec.Config.ProviderRef.Namespace != "" {
			authNamespace = clusterProvider.Spec.Config.ProviderRef.Namespace
		} else {
			return nil, fmt.Errorf("ClusterProvider %q has authenticationScope=ProviderNamespace but spec.config.providerRef.namespace is empty", providerName)
		}
	}

	// Build configuration for the helper
	cfg := v2ProviderConfig{
		name:              providerName,
		resourceNamespace: "", // cluster-scoped
		manifestNamespace: namespace,
		config:            clusterProvider.Spec.Config,
		generation:        clusterProvider.Generation,
		isClusterScoped:   true,
		kindStr:           esv1.ClusterProviderKindStr,
	}

	return m.getOrCreateV2Client(ctx, cfg, authNamespace)
}

// returns a previously stored client from the cache if store and store-version match
// if a client exists for the same provider which points to a different store or store version
// it will be cleaned up.
func (m *Manager) getStoredClient(ctx context.Context, storeProvider esv1.ProviderInterface, store esv1.GenericStore) esv1.SecretsClient {
	idx := storeKey(storeProvider)
	val, ok := m.clientMap[idx]
	if !ok {
		return nil
	}
	valGVK, err := m.client.GroupVersionKindFor(val.store)
	if err != nil {
		return nil
	}
	storeGVK, err := m.client.GroupVersionKindFor(store)
	if err != nil {
		return nil
	}
	storeName := fmt.Sprintf("%s/%s", store.GetNamespace(), store.GetName())
	// return client if it points to the very same store
	if val.store.GetObjectMeta().Generation == store.GetGeneration() &&
		valGVK == storeGVK &&
		val.store.GetName() == store.GetName() &&
		val.store.GetNamespace() == store.GetNamespace() {
		m.log.V(1).Info("reusing stored client",
			"provider", fmt.Sprintf("%T", storeProvider),
			"store", storeName)
		// Record cache hit
		providerType := "unknown"
		if idx.v2ProviderName != "" {
			if idx.v2ProviderNamespace == "" {
				providerType = "cluster-provider"
			} else {
				providerType = "provider"
			}
		}
		clientManagerMetrics.RecordCacheHit(providerType)
		return val.client
	}
	m.log.V(1).Info("cleaning up client",
		"provider", fmt.Sprintf("%T", storeProvider),
		"store", storeName)
	// if we have a client, but it points to a different store
	// we must clean it up
	_ = val.client.Close(ctx)
	delete(m.clientMap, idx)

	// Record cache invalidation
	providerType := "unknown"
	reason := "store_mismatch"
	if idx.v2ProviderName != "" {
		if idx.v2ProviderNamespace == "" {
			providerType = "cluster-provider"
		} else {
			providerType = "provider"
		}
		if val.store.GetObjectMeta().Generation != store.GetGeneration() {
			reason = "generation_change"
		}
	}
	clientManagerMetrics.RecordCacheInvalidation(providerType, reason)

	return nil
}

func storeKey(storeProvider esv1.ProviderInterface) clientKey {
	return clientKey{
		providerType: fmt.Sprintf("%T", storeProvider),
	}
}

// getStore fetches the (Cluster)SecretStore from the kube-apiserver
// and returns a GenericStore representing it.
func (m *Manager) getStore(ctx context.Context, storeRef *esv1.SecretStoreRef, namespace string) (esv1.GenericStore, error) {
	ref := types.NamespacedName{
		Name: storeRef.Name,
	}
	if storeRef.Kind == esv1.ClusterSecretStoreKind {
		var store esv1.ClusterSecretStore
		err := m.client.Get(ctx, ref, &store)
		if err != nil {
			return nil, fmt.Errorf(errGetClusterSecretStore, ref.Name, err)
		}
		return &store, nil
	}
	ref.Namespace = namespace
	var store esv1.SecretStore
	err := m.client.Get(ctx, ref, &store)
	if err != nil {
		return nil, fmt.Errorf(errGetSecretStore, ref.Name, err)
	}
	return &store, nil
}

// Close cleans up all clients.
// For v1 providers, it closes the clients directly.
// For v2 providers, it releases connections back to the pool for reuse.
func (m *Manager) Close(ctx context.Context) error {
	var errs []string

	// Release v2 pooled connections back to the pool
	pool := getGlobalV2ConnectionPool()
	for _, pooledConn := range m.v2PooledConnections {
		pool.Release(pooledConn.address, pooledConn.tlsConfig)
		m.log.V(1).Info("released v2 connection back to pool",
			"address", pooledConn.address)
	}
	m.v2PooledConnections = nil

	// Close v1 provider clients (they don't use the pool)
	for key, val := range m.clientMap {
		// Only close v1 clients; v2 clients are managed by the pool
		if key.providerType != "v2-provider" {
			err := val.client.Close(ctx)
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
		delete(m.clientMap, key)
	}

	if len(errs) != 0 {
		return fmt.Errorf("errors while closing clients: %s", strings.Join(errs, ", "))
	}
	return nil
}

// validateNamespaceConditions checks if a namespace matches the given conditions.
// Returns true if the namespace is allowed, false if denied.
func (m *Manager) validateNamespaceConditions(conditions []esv1.ClusterSecretStoreCondition, ns string) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	namespace := v1.Namespace{}
	if err := m.client.Get(context.Background(), client.ObjectKey{Name: ns}, &namespace); err != nil {
		return false, fmt.Errorf("failed to get a namespace %q: %w", ns, err)
	}

	nsLabels := labels.Set(namespace.GetLabels())
	for _, condition := range conditions {
		var labelSelectors []*metav1.LabelSelector
		if condition.NamespaceSelector != nil {
			labelSelectors = append(labelSelectors, condition.NamespaceSelector)
		}
		for _, n := range condition.Namespaces {
			labelSelectors = append(labelSelectors, &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": n,
				},
			})
		}

		for _, ls := range labelSelectors {
			selector, err := metav1.LabelSelectorAsSelector(ls)
			if err != nil {
				return false, fmt.Errorf("failed to convert label selector into selector %v: %w", ls, err)
			}
			if selector.Matches(nsLabels) {
				return true, nil
			}
		}

		for _, reg := range condition.NamespaceRegexes {
			match, err := regexp.MatchString(reg, ns)
			if err != nil {
				// Should not happen since store validation already verified the regexes.
				return false, fmt.Errorf("failed to compile regex %v: %w", reg, err)
			}

			if match {
				return true, nil
			}
		}
	}

	return false, nil
}

// shouldProcessSecret validates if a secret should be processed based on namespace conditions.
// This is a wrapper around validateNamespaceConditions for backward compatibility with GenericStore.
func (m *Manager) shouldProcessSecret(store esv1.GenericStore, ns string) (bool, error) {
	// Only check conditions for cluster-scoped resources (ClusterSecretStore and ClusterProvider)
	if store.GetKind() != esv1.ClusterSecretStoreKind && store.GetKind() != esv1.ClusterProviderKind {
		return true, nil
	}

	return m.validateNamespaceConditions(store.GetSpec().Conditions, ns)
}

// assertStoreIsUsable asserts that the store is ready to use.
func assertStoreIsUsable(store esv1.GenericStore) error {
	if store == nil {
		return nil
	}
	condition := GetSecretStoreCondition(store.GetStatus(), esv1.SecretStoreReady)
	if condition == nil || condition.Status != v1.ConditionTrue {
		return fmt.Errorf(errSecretStoreNotReady, store.GetKind(), store.GetName())
	}
	return nil
}

// ShouldProcessStore returns true if the store should be processed.
func ShouldProcessStore(store esv1.GenericStore, class string) bool {
	if store == nil || store.GetSpec().Controller == "" || store.GetSpec().Controller == class {
		return true
	}

	return false
}

// GetSecretStoreCondition returns the condition with the provided type.
func GetSecretStoreCondition(status esv1.SecretStoreStatus, condType esv1.SecretStoreConditionType) *esv1.SecretStoreStatusCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}
