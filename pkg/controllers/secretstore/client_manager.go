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

package secretstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errGetClusterSecretStore = "could not get ClusterSecretStore %q, %w"
	errGetSecretStore        = "could not get SecretStore %q, %w"
	errSecretStoreNotReady   = "the desired SecretStore %s is not ready"
	errClusterStoreMismatch  = "using cluster store %q is not allowed from namespace %q: denied by spec.condition"
)

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
}

type clientKey struct {
	providerType string
}

type clientVal struct {
	client esv1beta1.SecretsClient
	store  esv1beta1.GenericStore
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

func (m *Manager) GetFromStore(ctx context.Context, store esv1beta1.GenericStore, namespace string) (esv1beta1.SecretsClient, error) {
	storeProvider, err := esv1beta1.GetProvider(store)
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
func (m *Manager) Get(ctx context.Context, storeRef esv1beta1.SecretStoreRef, namespace string, sourceRef *esv1beta1.StoreGeneratorSourceRef) (esv1beta1.SecretsClient, error) {
	if sourceRef != nil && sourceRef.SecretStoreRef != nil {
		storeRef = *sourceRef.SecretStoreRef
	}
	store, err := m.getStore(ctx, &storeRef, namespace)
	if err != nil {
		return nil, err
	}
	// check if store should be handled by this controller instance
	if !ShouldProcessStore(store, m.controllerClass) {
		return nil, fmt.Errorf("can not reference unmanaged store")
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
func (m *Manager) getStoredClient(ctx context.Context, storeProvider esv1beta1.Provider, store esv1beta1.GenericStore) esv1beta1.SecretsClient {
	idx := storeKey(storeProvider)
	val, ok := m.clientMap[idx]
	if !ok {
		return nil
	}
	storeName := fmt.Sprintf("%s/%s", store.GetNamespace(), store.GetName())
	// return client if it points to the very same store
	if val.store.GetObjectMeta().Generation == store.GetGeneration() &&
		val.store.GetTypeMeta().Kind == store.GetTypeMeta().Kind &&
		val.store.GetName() == store.GetName() &&
		val.store.GetNamespace() == store.GetNamespace() {
		m.log.V(1).Info("reusing stored client",
			"provider", fmt.Sprintf("%T", storeProvider),
			"store", storeName)
		return val.client
	}
	m.log.V(1).Info("cleaning up client",
		"provider", fmt.Sprintf("%T", storeProvider),
		"store", storeName)
	// if we have a client, but it points to a different store
	// we must clean it up
	val.client.Close(ctx)
	delete(m.clientMap, idx)
	return nil
}

func storeKey(storeProvider esv1beta1.Provider) clientKey {
	return clientKey{
		providerType: fmt.Sprintf("%T", storeProvider),
	}
}

// getStore fetches the (Cluster)SecretStore from the kube-apiserver
// and returns a GenericStore representing it.
func (m *Manager) getStore(ctx context.Context, storeRef *esv1beta1.SecretStoreRef, namespace string) (esv1beta1.GenericStore, error) {
	ref := types.NamespacedName{
		Name: storeRef.Name,
	}
	if storeRef.Kind == esv1beta1.ClusterSecretStoreKind {
		var store esv1beta1.ClusterSecretStore
		err := m.client.Get(ctx, ref, &store)
		if err != nil {
			return nil, fmt.Errorf(errGetClusterSecretStore, ref.Name, err)
		}
		return &store, nil
	}
	ref.Namespace = namespace
	var store esv1beta1.SecretStore
	err := m.client.Get(ctx, ref, &store)
	if err != nil {
		return nil, fmt.Errorf(errGetSecretStore, ref.Name, err)
	}
	return &store, nil
}

// Close cleans up all clients.
func (m *Manager) Close(ctx context.Context) error {
	var errs []string
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

func (m *Manager) shouldProcessSecret(store esv1beta1.GenericStore, ns string) (bool, error) {
	if store.GetKind() != esv1beta1.ClusterSecretStoreKind {
		return true, nil
	}

	if len(store.GetSpec().Conditions) == 0 {
		return true, nil
	}

	namespace := v1.Namespace{}
	if err := m.client.Get(context.Background(), client.ObjectKey{Name: ns}, &namespace); err != nil {
		return false, fmt.Errorf("failed to get a namespace %q: %w", ns, err)
	}

	nsLabels := labels.Set(namespace.GetLabels())
	for _, condition := range store.GetSpec().Conditions {
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
	}

	return false, nil
}

// assertStoreIsUsable assert that the store is ready to use.
func assertStoreIsUsable(store esv1beta1.GenericStore) error {
	if store == nil {
		return nil
	}
	condition := GetSecretStoreCondition(store.GetStatus(), esv1beta1.SecretStoreReady)
	if condition == nil || condition.Status != v1.ConditionTrue {
		return fmt.Errorf(errSecretStoreNotReady, store.GetName())
	}
	return nil
}
