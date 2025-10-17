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
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// WatchTracker tracks which GroupVersionKinds are currently being watched.
// This interface allows for different implementations, such as sharding-aware trackers.
type WatchTracker interface {
	// IsWatched checks if a GVK is already being watched
	IsWatched(gvk schema.GroupVersionKind) bool

	// MarkWatched marks a GVK as being watched
	MarkWatched(gvk schema.GroupVersionKind)
}

// InMemoryWatchTracker is a simple in-memory implementation of WatchTracker.
// It uses a sync.Map for concurrent access.
type InMemoryWatchTracker struct {
	watches sync.Map // map[string]bool
}

// NewInMemoryWatchTracker creates a new in-memory watch tracker.
func NewInMemoryWatchTracker() *InMemoryWatchTracker {
	return &InMemoryWatchTracker{}
}

// IsWatched checks if a GVK is already being watched.
func (t *InMemoryWatchTracker) IsWatched(gvk schema.GroupVersionKind) bool {
	_, ok := t.watches.Load(gvk.String())
	return ok
}

// MarkWatched marks a GVK as being watched.
func (t *InMemoryWatchTracker) MarkWatched(gvk schema.GroupVersionKind) {
	t.watches.Store(gvk.String(), true)
}
