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

package externalsecret

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// InformerManager manages the lifecycle of informers for generic target resources.
// It handles dynamic registration, tracking, and cleanup of informers.
type InformerManager interface {
	// EnsureInformer ensures an informer exists for the given GVK and registers the ExternalSecret as using it.
	// Returns true if a new informer was created, false if it already existed.
	EnsureInformer(ctx context.Context, gvk schema.GroupVersionKind, es types.NamespacedName) (bool, error)

	// ReleaseInformer unregisters the ExternalSecret from using this GVK.
	// If no more ExternalSecrets use this GVK, the informer is stopped and removed.
	ReleaseInformer(ctx context.Context, gvk schema.GroupVersionKind, es types.NamespacedName) error

	// IsManaged returns true if the manager is currently managing an informer for the GVK.
	IsManaged(gvk schema.GroupVersionKind) bool

	// GetInformer returns the informer for a GVK if it exists.
	GetInformer(gvk schema.GroupVersionKind) (runtimecache.Informer, bool)

	// Source returns a source.TypedSource that can be used with WatchesRawSource
	Source() source.TypedSource[reconcile.Request]

	// SetQueue binds the reconcile queue to the informer manager
	SetQueue(queue workqueue.TypedRateLimitingInterface[ctrl.Request]) error
}

// informerEntry tracks an informer and the ExternalSecrets using it.
type informerEntry struct {
	informer runtimecache.Informer
	// externalSecrets tracks the external secrets using a GVK. Once this list is empty, we
	// stop the informer and deregister it to free up resources. It is a map instead of just a number to prevent
	// duplicated reconcile ensures to increase the number on each reconcile.
	externalSecrets map[types.NamespacedName]struct{}
}

// DefaultInformerManager implements InformerManager using controller-runtime's cache.
type DefaultInformerManager struct {
	cache          runtimecache.Cache
	client         client.Client
	log            logr.Logger
	mu             sync.RWMutex
	informers      map[string]*informerEntry // key: GVK string
	queue          workqueue.TypedRateLimitingInterface[ctrl.Request]
	managerContext context.Context
}

// NewInformerManager creates a new InformerManager.
func NewInformerManager(ctx context.Context, cache runtimecache.Cache, client client.Client, log logr.Logger) InformerManager {
	return &DefaultInformerManager{
		managerContext: ctx,
		cache:          cache,
		client:         client,
		log:            log,
		informers:      make(map[string]*informerEntry),
	}
}

// EnsureInformer ensures an informer exists for the given GVK and registers the ExternalSecret.
func (m *DefaultInformerManager) EnsureInformer(ctx context.Context, gvk schema.GroupVersionKind, es types.NamespacedName) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := gvk.String()

	// check if we have this gvk in the list of informers already
	if entry, exists := m.informers[key]; exists {
		// register this ExternalSecret as using this informer (deduplicate);
		entry.externalSecrets[es] = struct{}{}
		m.log.Info("registered ExternalSecret with existing informer",
			"gvk", key,
			"externalSecret", es,
			"totalUsers", len(entry.externalSecrets))
		return false, nil
	}

	if m.queue == nil {
		return false, fmt.Errorf("queue not initialized, call SetQueue first")
	}

	// Get or create informer for this GVK
	informer, err := m.cache.GetInformerForKind(ctx, gvk)
	if err != nil {
		return false, fmt.Errorf("failed to get informer for %s: %w", key, err)
	}

	// Add event handler to the informer that enqueues reconcile requests
	_, err = informer.AddEventHandler(&enqueueHandler{
		managerContext: m.managerContext,
		gvk:            gvk,
		client:         m.client,
		queue:          m.queue,
		log:            m.log,
	})
	if err != nil {
		return false, fmt.Errorf("failed to add event handler for %s: %w", key, err)
	}

	// Store the informer with this ExternalSecret as the first user
	m.informers[key] = &informerEntry{
		informer:        informer,
		externalSecrets: map[types.NamespacedName]struct{}{es: {}},
	}

	m.log.Info("registered informer for generic target",
		"group", gvk.Group,
		"version", gvk.Version,
		"kind", gvk.Kind,
		"externalSecret", es)

	return true, nil
}

// enqueueHandler is an event handler that enqueues reconcile requests for ExternalSecrets
// that target the changed resource.
type enqueueHandler struct {
	managerContext context.Context
	gvk            schema.GroupVersionKind
	client         client.Client
	queue          workqueue.TypedRateLimitingInterface[ctrl.Request]
	log            logr.Logger
}

func (h *enqueueHandler) OnAdd(obj interface{}, _ bool) {
	h.enqueue(obj)
}

func (h *enqueueHandler) OnUpdate(_, newObj interface{}) {
	h.enqueue(newObj)
}

func (h *enqueueHandler) OnDelete(obj interface{}) {
	h.enqueue(obj)
}

func (h *enqueueHandler) enqueue(obj interface{}) {
	// Extract metadata
	meta, ok := obj.(metav1.Object)
	if !ok {
		h.log.Error(nil, "unexpected object type", "type", fmt.Sprintf("%T", obj))
		return
	}

	// Only process resources with the managed label
	labels := meta.GetLabels()
	if labels == nil {
		return
	}

	value, hasLabel := labels[esv1.LabelManaged]
	if !hasLabel || value != esv1.LabelManagedValue {
		return
	}

	// Find ExternalSecrets that target this resource
	externalSecretsList := &esv1.ExternalSecretList{}
	indexValue := fmt.Sprintf("%s/%s/%s/%s", h.gvk.Group, h.gvk.Version, h.gvk.Kind, meta.GetName())
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexESTargetResourceField, indexValue),
		Namespace:     meta.GetNamespace(),
	}

	if err := h.client.List(h.managerContext, externalSecretsList, listOps); err != nil {
		h.log.Error(err, "failed to list ExternalSecrets for resource",
			"gvk", h.gvk.String(),
			"name", meta.GetName(),
			"namespace", meta.GetNamespace())
		return
	}

	// Enqueue reconcile requests for each ExternalSecret
	for i := range externalSecretsList.Items {
		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      externalSecretsList.Items[i].GetName(),
				Namespace: externalSecretsList.Items[i].GetNamespace(),
			},
		}
		h.queue.Add(req)

		h.log.V(1).Info("enqueued reconcile request due to resource change",
			"externalSecret", req.NamespacedName,
			"targetGVK", h.gvk.String(),
			"targetResource", meta.GetName())
	}
}

// ReleaseInformer unregisters the ExternalSecret from using this GVK.
func (m *DefaultInformerManager) ReleaseInformer(ctx context.Context, gvk schema.GroupVersionKind, es types.NamespacedName) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := gvk.String()

	entry, exists := m.informers[key]
	if !exists {
		// Already removed or never existed; can happen if we had a bad start, failed to watch, or during other errors.
		// In that case, there is nothing else to do really.
		m.log.Info("informer not found for release",
			"gvk", key,
			"externalSecret", es)
		return nil
	}

	// remove the ES from the list of ESs using this GVK
	delete(entry.externalSecrets, es)
	m.log.Info("unregistered ExternalSecret from informer",
		"gvk", key,
		"externalSecret", es,
		"remainingUsers", len(entry.externalSecrets))

	// if no more ExternalSecrets are using this informer, remove it
	if len(entry.externalSecrets) == 0 {
		partial := &metav1.PartialObjectMetadata{}
		partial.SetGroupVersionKind(gvk)

		if err := m.cache.RemoveInformer(ctx, partial); err != nil {
			m.log.Error(err, "failed to remove informer, will clean up tracking anyway",
				"gvk", key)
		}

		delete(m.informers, key)

		m.log.Info("removed informer for generic target (no more users)",
			"group", gvk.Group,
			"version", gvk.Version,
			"kind", gvk.Kind)
	}

	return nil
}

// IsManaged returns true if the manager is currently managing an informer for the GVK.
func (m *DefaultInformerManager) IsManaged(gvk schema.GroupVersionKind) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.informers[gvk.String()]
	return exists
}

// GetInformer returns the informer for a GVK if it exists.
func (m *DefaultInformerManager) GetInformer(gvk schema.GroupVersionKind) (runtimecache.Informer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.informers[gvk.String()]
	if !exists {
		return nil, false
	}
	return entry.informer, true
}

// Source returns a source.TypedSource that binds the reconcile queue to this manager.
func (m *DefaultInformerManager) Source() source.TypedSource[reconcile.Request] {
	return source.Func(func(_ context.Context, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) error {
		// This dynamically binds the given queue to the informer manager
		// From this point on, the queue will receive events for all registered informers
		return m.SetQueue(queue)
	})
}

// SetQueue binds the reconcile queue to the informer manager.
func (m *DefaultInformerManager) SetQueue(queue workqueue.TypedRateLimitingInterface[ctrl.Request]) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.queue != nil {
		return fmt.Errorf("queue already set")
	}

	m.queue = queue
	m.log.Info("reconcile queue bound to informer manager")
	return nil
}
