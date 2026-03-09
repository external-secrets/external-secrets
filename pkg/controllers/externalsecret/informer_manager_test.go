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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeInformer struct{}

func (f *fakeInformer) AddEventHandler(handler toolscache.ResourceEventHandler) (toolscache.ResourceEventHandlerRegistration, error) {
	return nil, nil
}

func (f *fakeInformer) AddEventHandlerWithResyncPeriod(handler toolscache.ResourceEventHandler, _ time.Duration) (toolscache.ResourceEventHandlerRegistration, error) {
	return nil, nil
}

func (f *fakeInformer) AddEventHandlerWithOptions(handler toolscache.ResourceEventHandler, _ toolscache.HandlerOptions) (toolscache.ResourceEventHandlerRegistration, error) {
	return nil, nil
}

func (f *fakeInformer) RemoveEventHandler(_ toolscache.ResourceEventHandlerRegistration) error {
	return nil
}

func (f *fakeInformer) AddIndexers(indexers toolscache.Indexers) error {
	return nil
}

func (f *fakeInformer) HasSynced() bool {
	return true
}

func (f *fakeInformer) IsStopped() bool {
	return false
}

type fakeCache struct {
	runtimecache.Cache
	getInformerCalled bool
	getInformerObj    client.Object
	getInformerErr    error
}

func (f *fakeCache) GetInformer(ctx context.Context, obj client.Object, opts ...runtimecache.InformerGetOption) (runtimecache.Informer, error) {
	f.getInformerCalled = true
	f.getInformerObj = obj
	if f.getInformerErr != nil {
		return nil, f.getInformerErr
	}
	return &fakeInformer{}, nil
}

func TestEnsureInformer_UsesUnstructured(t *testing.T) {
	fc := &fakeCache{}
	log := ctrl.Log.WithName("test")
	m := &DefaultInformerManager{
		managerContext: context.Background(),
		cache:          fc,
		log:            log,
		informers:      make(map[string]*informerEntry),
		queue:          workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]()),
	}

	gvk := schema.GroupVersionKind{
		Group:   "monitoring.example.io",
		Version: "v1alpha1",
		Kind:    "CustomNotifier",
	}
	es := types.NamespacedName{Name: "test-es", Namespace: "default"}

	created, err := m.EnsureInformer(context.Background(), gvk, es)

	require.NoError(t, err)
	assert.True(t, created)
	assert.True(t, fc.getInformerCalled, "GetInformer should be called")

	obj, ok := fc.getInformerObj.(*unstructured.Unstructured)
	require.True(t, ok, "GetInformer should be called with *unstructured.Unstructured")
	assert.Equal(t, gvk, obj.GroupVersionKind())
}

func TestEnsureInformer_DeduplicatesExistingGVK(t *testing.T) {
	fc := &fakeCache{}
	log := ctrl.Log.WithName("test")
	m := &DefaultInformerManager{
		managerContext: context.Background(),
		cache:          fc,
		log:            log,
		informers:      make(map[string]*informerEntry),
		queue:          workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]()),
	}

	gvk := schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Foo"}
	es1 := types.NamespacedName{Name: "es-1", Namespace: "default"}
	es2 := types.NamespacedName{Name: "es-2", Namespace: "default"}

	created, err := m.EnsureInformer(context.Background(), gvk, es1)
	require.NoError(t, err)
	assert.True(t, created)

	fc.getInformerCalled = false

	created, err = m.EnsureInformer(context.Background(), gvk, es2)
	require.NoError(t, err)
	assert.False(t, created)
	assert.False(t, fc.getInformerCalled, "GetInformer should not be called again for same GVK")

	entry := m.informers[gvk.String()]
	assert.Len(t, entry.externalSecrets, 2)
}

func TestEnsureInformer_ErrorWhenQueueNotSet(t *testing.T) {
	fc := &fakeCache{}
	log := ctrl.Log.WithName("test")
	m := &DefaultInformerManager{
		managerContext: context.Background(),
		cache:          fc,
		log:            log,
		informers:      make(map[string]*informerEntry),
	}

	gvk := schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Foo"}
	es := types.NamespacedName{Name: "es-1", Namespace: "default"}

	_, err := m.EnsureInformer(context.Background(), gvk, es)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue not initialized")
}

func TestEnsureInformer_PropagatesCacheError(t *testing.T) {
	fc := &fakeCache{
		getInformerErr: fmt.Errorf("CRD not found"),
	}
	log := ctrl.Log.WithName("test")
	m := &DefaultInformerManager{
		managerContext: context.Background(),
		cache:          fc,
		log:            log,
		informers:      make(map[string]*informerEntry),
		queue:          workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]()),
	}

	gvk := schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Foo"}
	es := types.NamespacedName{Name: "es-1", Namespace: "default"}

	_, err := m.EnsureInformer(context.Background(), gvk, es)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CRD not found")
}

func TestReleaseInformer_RemovesES(t *testing.T) {
	fc := &fakeCache{}
	log := ctrl.Log.WithName("test")
	m := &DefaultInformerManager{
		managerContext: context.Background(),
		cache:          fc,
		log:            log,
		informers:      make(map[string]*informerEntry),
		queue:          workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]()),
	}

	gvk := schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Foo"}
	es1 := types.NamespacedName{Name: "es-1", Namespace: "default"}
	es2 := types.NamespacedName{Name: "es-2", Namespace: "default"}

	_, err := m.EnsureInformer(context.Background(), gvk, es1)
	require.NoError(t, err)
	_, err = m.EnsureInformer(context.Background(), gvk, es2)
	require.NoError(t, err)

	err = m.ReleaseInformer(context.Background(), gvk, es1)
	require.NoError(t, err)

	assert.True(t, m.IsManaged(gvk))
	entry := m.informers[gvk.String()]
	assert.Len(t, entry.externalSecrets, 1)
}
