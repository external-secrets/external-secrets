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

package statemanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/generator/gc"
)

// Manager takes care of maintaining the state of the generators.
// It provides the ability to commit and rollback the state of the generators,
// which is needed when we have multiple generators that need to be created or
// other operations which can fail.
// The manager shall be used to modify the state of the generators on a given resource.
// The user can choose any key to store the state of the generator on the "latest" field.
// When state is moved to GC, manager will create a hash of the key and the generator state
// and store it in the "GC" field.
type Manager struct {
	resource      v1beta1.GeneratorStateManagingResource
	internalState []QueueItem
}

type QueueItem struct {
	Rollback func() error
	Commit   func() error
}

func New(resource v1beta1.GeneratorStateManagingResource) *Manager {
	return &Manager{
		resource: resource,
	}
}

// Rollback will rollback the state of the generators.
func (m *Manager) Rollback() error {
	var errs []error
	for _, item := range m.internalState {
		if err := item.Rollback(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Commit will commit the state of the generators.
func (m *Manager) Commit() error {
	var errs []error
	for _, item := range m.internalState {
		if err := item.Commit(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// FlagLatestStateForGC will flag the latest state for garbage collection after Commit.
// It will be cleaned up later by the garbage collector.
func (m *Manager) FlagLatestStateForGC(stateKey string) {
	m.internalState = append(m.internalState, QueueItem{
		Commit: func() error {
			genState := m.resource.GetGeneratorState()
			if genState.Latest == nil {
				return nil
			}
			latest, ok := genState.Latest[stateKey]
			if !ok {
				return nil
			}
			gen, err := v1alpha1.GetGenerator(latest.Resource)
			if err != nil {
				return err
			}
			return m.ImmediateMoveStateToGC(latest.Resource, stateKey, gen, latest.State)
		},
	})
}

// MoveStateToGC will move the generator state to GC.
// It will be cleaned up later by the garbage collector.
func (m *Manager) MoveStateToGC(resource *apiextensions.JSON, stateKey string, gen v1alpha1.Generator, state v1alpha1.GeneratorProviderState) {
	m.internalState = append(m.internalState, QueueItem{
		Commit: func() error {
			return m.ImmediateMoveStateToGC(resource, stateKey, gen, state)
		},
	})
}

// ImmediateMoveStateToGC will move the generator state to GC immediately.
func (m *Manager) ImmediateMoveStateToGC(resource *apiextensions.JSON, stateKey string, gen v1alpha1.Generator, state v1alpha1.GeneratorProviderState) error {
	genState := m.resource.GetGeneratorState()
	entry := gc.Entry{
		Resource: resource,
		Impl:     gen,
		State:    state,
	}
	if err := gc.Enqueue(entry); err != nil {
		return fmt.Errorf("unable to enqueue generator state for GC: %w", err)
	}
	if genState.GC == nil {
		genState.GC = make(map[string]v1beta1.GeneratorGCState)
	}
	genState.GC[gcGeneratorStateKey(entry, stateKey)] = v1beta1.GeneratorGCState{
		Resource:         resource,
		State:            state,
		FlaggedForGCTime: metav1.Now(),
	}
	return nil
}

func gcGeneratorStateKey(entry gc.Entry, key string) string {
	return fmt.Sprintf("[%s]-%s", key, entry.Key())
}

// SetLatest sets the latest state for the given key.
// It will commit the state on success or move the state to GC on failure.
func (m *Manager) SetLatest(ctx context.Context, kubeClient client.Client, stateKey, namespace string, resource *apiextensions.JSON, gen v1alpha1.Generator, state v1alpha1.GeneratorProviderState) {
	m.internalState = append(m.internalState, QueueItem{
		// Store state at .Latest[<key>] on success
		// or attempt to immediately delete the state on failure
		Commit: func() error {
			genState := m.resource.GetGeneratorState()
			if genState.Latest == nil {
				genState.Latest = make(map[string]*v1beta1.GeneratorResourceState)
			}
			genState.Latest[stateKey] = &v1beta1.GeneratorResourceState{
				Resource: resource,
				State:    state,
			}
			return nil
		},
		// Rollback by cleaning up the state.
		// In case of failure, move the state to GC so it will be cleaned up later.
		Rollback: func() error {
			err := gen.Cleanup(ctx, resource, state, kubeClient, namespace)
			if err == nil {
				return nil
			}
			return m.ImmediateMoveStateToGC(resource, fmt.Sprintf("rollback-%s", uuid.New().String()), gen, state)
		},
	})
}

// GetLatest returns the latest state for the given key.
func (m *Manager) GetLatest(key string) *apiextensions.JSON {
	state := m.resource.GetGeneratorState()
	if state.Latest == nil {
		return nil
	}
	latest := state.Latest[key]
	if latest == nil {
		return nil
	}
	return latest.State
}

// CleanupImmediate will cleanup the generator state immediately.
// This is useful when we want to cleanup the state immediately after deletion of the resource.
func CleanupImmediate(ctx context.Context, resource v1beta1.GeneratorStateManagingResource, kubeClient client.Client, namespace string) error {
	var errs []error
	generatorState := resource.GetGeneratorState()
	for _, gcState := range generatorState.GC {
		genImpl, err := v1alpha1.GetGenerator(gcState.Resource)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to get generator: %w", err))
			continue
		}
		err = genImpl.Cleanup(ctx, gcState.Resource, gcState.State, kubeClient, namespace)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to cleanup generator state: %w", err))
		}
	}
	return errors.Join(errs...)
}

// GarbageCollect will cleanup the generator state that is flagged for GC.
// It updates the generator state with the new GC state.
// Garbage collection is done in a best-effort manner, i.e. if an error occurs during cleanup of a generator state,
// it will be aggregated and returned at the end but the cleanup will continue for the remaining generator states.
func GarbageCollect(ctx context.Context, resource v1beta1.GeneratorStateManagingResource, kubeClient client.Client, namespace string) error {
	var errs []error
	generatorState := resource.GetGeneratorState()
	newGCState := make(map[string]v1beta1.GeneratorGCState)
	for idx, gcState := range generatorState.GC {
		genImpl, err := v1alpha1.GetGenerator(gcState.Resource)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to get generator: %w", err))
			continue
		}
		deleted, err := gc.Cleanup(ctx, gcState.FlaggedForGCTime.Time, gc.Entry{
			Resource: gcState.Resource,
			Impl:     genImpl,
			State:    gcState.State,
		}, kubeClient, namespace)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to cleanup generator state: %w", err))
			continue
		}
		if !deleted {
			newGCState[idx] = gcState
		}
	}
	generatorState.GC = newGCState
	resource.SetGeneratorState(generatorState)
	return errors.Join(errs...)
}
