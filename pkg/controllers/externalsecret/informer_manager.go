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
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
)

// InformerManager manages the lifecycle of informers for non-Secret target resources.
// It handles dynamic registration, reference counting, and cleanup of informers.
type InformerManager interface {
	// EnsureInformer ensures an informer exists for the given GVK.
	// Returns true if a new informer was created, false if it already existed.
	EnsureInformer(ctx context.Context, gvk schema.GroupVersionKind) (bool, error)

	// ReleaseInformer decrements the reference count for a GVK.
	// If the count reaches zero, the informer is stopped and removed.
	ReleaseInformer(ctx context.Context, gvk schema.GroupVersionKind) error

	// IsManaged returns true if the manager is currently managing an informer for the GVK.
	IsManaged(gvk schema.GroupVersionKind) bool

	// GetInformer returns the informer for a GVK if it exists.
	GetInformer(gvk schema.GroupVersionKind) (runtimecache.Informer, bool)
}

// informerEntry tracks an informer and its reference count.
type informerEntry struct {
	informer runtimecache.Informer
	refCount int
}

// DefaultInformerManager implements InformerManager using controller-runtime's cache.
type DefaultInformerManager struct {
	cache     runtimecache.Cache
	log       logr.Logger
	mu        sync.RWMutex
	informers map[string]*informerEntry // key: GVK string
}

// NewInformerManager creates a new InformerManager.
func NewInformerManager(cache runtimecache.Cache, log logr.Logger) InformerManager {
	return &DefaultInformerManager{
		cache:     cache,
		log:       log,
		informers: make(map[string]*informerEntry),
	}
}

// EnsureInformer ensures an informer exists for the given GVK.
func (m *DefaultInformerManager) EnsureInformer(ctx context.Context, gvk schema.GroupVersionKind) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := gvk.String()

	// Check if we already have an informer for this GVK
	if entry, exists := m.informers[key]; exists {
		entry.refCount++
		m.log.V(1).Info("incremented informer reference count",
			"gvk", key,
			"refCount", entry.refCount)
		return false, nil
	}

	// Get or create an informer for this GVK using PartialObjectMetadata to minimize memory usage
	// The cache will automatically start the informer if it doesn't exist
	informer, err := m.cache.GetInformerForKind(ctx, gvk)
	if err != nil {
		return false, fmt.Errorf("failed to get informer for %s: %w", key, err)
	}

	// Store the informer with initial reference count of 1
	m.informers[key] = &informerEntry{
		informer: informer,
		refCount: 1,
	}

	m.log.Info("registered dynamic informer for non-Secret target",
		"group", gvk.Group,
		"version", gvk.Version,
		"kind", gvk.Kind)

	return true, nil
}

// ReleaseInformer decrements the reference count for a GVK.
func (m *DefaultInformerManager) ReleaseInformer(ctx context.Context, gvk schema.GroupVersionKind) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := gvk.String()

	entry, exists := m.informers[key]
	if !exists {
		// Already removed or never existed
		return nil
	}

	entry.refCount--
	m.log.V(1).Info("decremented informer reference count",
		"gvk", key,
		"refCount", entry.refCount)

	// If no more references, remove the informer
	if entry.refCount <= 0 {
		// Create a PartialObjectMetadata instance to pass to RemoveInformer
		partial := &metav1.PartialObjectMetadata{}
		partial.SetGroupVersionKind(gvk)

		if err := m.cache.RemoveInformer(ctx, partial); err != nil {
			m.log.Error(err, "failed to remove informer, will clean up tracking anyway",
				"gvk", key)
		}

		delete(m.informers, key)

		m.log.Info("removed informer for non-Secret target",
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
