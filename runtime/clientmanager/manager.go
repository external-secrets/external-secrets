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
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	adapterstore "github.com/external-secrets/external-secrets/providers/v2/adapter/store"
	"github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

const (
	errGetClusterSecretStore = "could not get ClusterSecretStore %q, %w"
	errGetSecretStore        = "could not get SecretStore %q, %w"
	errSecretStoreNotReady   = "%s %q is not ready"
	errClusterStoreMismatch  = "using cluster store %q is not allowed from namespace %q: denied by spec.condition"

	providerMetricsLabel                   = "provider"
	clusterProviderMetricsLabel            = "cluster-provider"
	cacheInvalidationGeneration            = "generation_change"
	cacheInvalidationMismatch              = "store_mismatch"
	runtimeRefCacheKeyType                 = "runtime-ref"
	errRuntimeRefProviderClassClusterStore = "ClusterSecretStore runtimeRef.kind must not be \"ProviderClass\""
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
	v2ProviderName         string
	v2ProviderNamespace    string
	runtimeSourceNamespace string
}

type clientVal struct {
	client esv1.SecretsClient
	store  esv1.GenericStore
}

func providerMetricsLabelForKey(key clientKey) string {
	if key.v2ProviderName == "" {
		return "unknown"
	}

	if key.v2ProviderNamespace == "" {
		return clusterProviderMetricsLabel
	}

	return providerMetricsLabel
}

func providerMetricsLabelForScope(isClusterScoped bool) string {
	if isClusterScoped {
		return clusterProviderMetricsLabel
	}
	return providerMetricsLabel
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
	if store.GetSpec().RuntimeRef != nil {
		return m.getRuntimeRefClient(ctx, store, namespace)
	}

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

func (m *Manager) getRuntimeRefClient(ctx context.Context, store esv1.GenericStore, namespace string) (esv1.SecretsClient, error) {
	runtimeRef := store.GetSpec().RuntimeRef
	cacheKey := runtimeRefStoreKey(store, namespace)
	if cached := m.getStoredRuntimeRefClient(ctx, cacheKey, store); cached != nil {
		return cached, nil
	}

	runtimeDetails, err := m.resolveRuntimeRef(ctx, store, runtimeRef)
	if err != nil {
		return nil, err
	}

	providerRef, err := buildProviderReference(store, namespace)
	if err != nil {
		return nil, err
	}

	tlsSecretNamespace := grpc.ResolveTLSSecretNamespace(runtimeDetails.address, "", "", "")
	tlsConfig, err := grpc.LoadClientTLSConfig(ctx, m.client, runtimeDetails.address, tlsSecretNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config for %s %q: %w", runtimeDetails.kind, runtimeRef.Name, err)
	}

	pool := getGlobalV2ConnectionPool()
	grpcClient, err := pool.Get(ctx, runtimeDetails.address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get gRPC client from pool for %s %q: %w", runtimeDetails.kind, runtimeRef.Name, err)
	}

	providerClient := adapterstore.NewClientWithCloser(grpcClient, providerRef, namespace, func(context.Context) error {
		pool.Release(runtimeDetails.address, tlsConfig)
		return nil
	})

	m.clientMap[cacheKey] = &clientVal{
		client: providerClient,
		store:  store,
	}

	return providerClient, nil
}

type runtimeRefDetails struct {
	kind    string
	address string
}

func (m *Manager) resolveRuntimeRef(ctx context.Context, store esv1.GenericStore, runtimeRef *esv1.StoreRuntimeRef) (*runtimeRefDetails, error) {
	runtimeKind, err := runtimeRefKindForStore(store, runtimeRef)
	if err != nil {
		return nil, err
	}

	switch runtimeKind {
	case esv1.StoreRuntimeRefKindProviderClass:
		var runtimeClass esv1alpha1.ProviderClass
		if err := m.client.Get(ctx, types.NamespacedName{Name: runtimeRef.Name, Namespace: store.GetNamespace()}, &runtimeClass); err != nil {
			return nil, fmt.Errorf("failed to get %s %q: %w", runtimeKind, runtimeRef.Name, err)
		}
		return &runtimeRefDetails{kind: runtimeKind, address: runtimeClass.Spec.Address}, nil
	case esv1.StoreRuntimeRefKindClusterProviderClass:
		var runtimeClass esv1alpha1.ClusterProviderClass
		if err := m.client.Get(ctx, types.NamespacedName{Name: runtimeRef.Name}, &runtimeClass); err != nil {
			return nil, fmt.Errorf("failed to get %s %q: %w", runtimeKind, runtimeRef.Name, err)
		}
		return &runtimeRefDetails{kind: runtimeKind, address: runtimeClass.Spec.Address}, nil
	default:
		return nil, fmt.Errorf("unsupported runtimeRef kind %q", runtimeKind)
	}
}

func runtimeRefKindForStore(store esv1.GenericStore, runtimeRef *esv1.StoreRuntimeRef) (string, error) {
	runtimeKind := runtimeRef.Kind
	if runtimeKind == "" {
		if store.GetKind() == esv1.ClusterSecretStoreKind {
			return esv1.StoreRuntimeRefKindClusterProviderClass, nil
		}
		return esv1.StoreRuntimeRefKindProviderClass, nil
	}
	if store.GetKind() == esv1.ClusterSecretStoreKind && runtimeKind == esv1.StoreRuntimeRefKindProviderClass {
		return "", fmt.Errorf(errRuntimeRefProviderClassClusterStore)
	}
	return runtimeKind, nil
}

func runtimeRefStoreKey(store esv1.GenericStore, sourceNamespace string) clientKey {
	return clientKey{
		providerType:           runtimeRefCacheKeyType + ":" + store.GetKind(),
		v2ProviderName:         store.GetName(),
		v2ProviderNamespace:    store.GetNamespace(),
		runtimeSourceNamespace: sourceNamespace,
	}
}

func (m *Manager) getStoredRuntimeRefClient(ctx context.Context, key clientKey, store esv1.GenericStore) esv1.SecretsClient {
	val, ok := m.clientMap[key]
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

	if val.store.GetObjectMeta().Generation == store.GetGeneration() &&
		valGVK == storeGVK &&
		val.store.GetName() == store.GetName() &&
		val.store.GetNamespace() == store.GetNamespace() {
		clientManagerMetrics.RecordCacheHit(providerMetricsLabelForKey(key))
		return val.client
	}

	_ = val.client.Close(ctx)
	delete(m.clientMap, key)

	reason := cacheInvalidationMismatch
	if val.store.GetObjectMeta().Generation != store.GetGeneration() {
		reason = cacheInvalidationGeneration
	}
	clientManagerMetrics.RecordCacheInvalidation(providerMetricsLabelForKey(key), reason)

	return nil
}

// Get returns a provider client from the given storeRef or sourceRef.secretStoreRef
// while sourceRef.SecretStoreRef takes precedence over storeRef.
// Do not close the client returned from this func, instead close
// the manager once you're done with recinciling the external secret.
func (m *Manager) Get(ctx context.Context, storeRef esv1.SecretStoreRef, namespace string, sourceRef *esv1.StoreGeneratorSourceRef) (esv1.SecretsClient, error) {
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
		clientManagerMetrics.RecordCacheHit(providerMetricsLabelForKey(idx))
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
	providerType := providerMetricsLabelForKey(idx)
	reason := cacheInvalidationMismatch
	if idx.v2ProviderName != "" {
		if val.store.GetObjectMeta().Generation != store.GetGeneration() {
			reason = cacheInvalidationGeneration
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

	// Close cached clients. Runtime-ref-backed clients release their pooled
	// connection through their Close implementation.
	for key, val := range m.clientMap {
		err := val.client.Close(ctx)
		if err != nil {
			errs = append(errs, err.Error())
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
	// Only check conditions for cluster-scoped resources.
	if store.GetKind() != esv1.ClusterSecretStoreKind {
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
