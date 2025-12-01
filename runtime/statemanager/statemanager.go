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

// Package statemanager provides functionality for managing state of generator operations.
package statemanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	genapi "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/feature"
)

// Manager takes care of maintaining the state of the generators.
// It provides the ability to commit and rollback the state of the generators,
// which is needed when we have multiple generators that need to be created or
// other operations which can fail.
type Manager struct {
	ctx           context.Context
	scheme        *runtime.Scheme
	client        client.Client
	namespace     string
	resource      genapi.StatefulResource
	cleanupPolicy *genapi.CleanupPolicy

	queue []QueueItem
}

// QueueItem represents a single item in the state manager's queue.
type QueueItem struct {
	Rollback func() error
	Commit   func() error
}

var defaultGCGracePeriod time.Duration

func init() {
	fs := pflag.NewFlagSet("gc", pflag.ExitOnError)
	fs.DurationVar(&defaultGCGracePeriod, "generator-gc-grace-period", time.Minute*2, "Duration after which generated secrets are cleaned up after they have been flagged for gc.")
	feature.Register(feature.Feature{
		Flags: fs,
	})
}

// New creates a new state manager instance with the given configuration.
func New(ctx context.Context, client client.Client, scheme *runtime.Scheme, namespace string,
	resource genapi.StatefulResource) *Manager {
	return &Manager{
		ctx:       ctx,
		scheme:    scheme,
		client:    client,
		namespace: namespace,
		resource:  resource,
		// Set default policy as RetainLatest
		cleanupPolicy: &genapi.CleanupPolicy{
			Type: genapi.RetainLatestPolicy,
			GracePeriod: metav1.Duration{
				Duration: defaultGCGracePeriod,
			},
		},
	}
}

// GetDefaultCleanupPolicy returns the default cleanup policy for the state manager.
func GetDefaultCleanupPolicy() *genapi.CleanupPolicy {
	return &genapi.CleanupPolicy{
		Type: genapi.RetainLatestPolicy,
		GracePeriod: metav1.Duration{
			Duration: defaultGCGracePeriod,
		},
	}
}

// SetCleanupPolicy sets the cleanup policy for the state manager.
func (m *Manager) SetCleanupPolicy(cleanupPolicy *genapi.CleanupPolicy) {
	if cleanupPolicy == nil {
		cleanupPolicy = GetDefaultCleanupPolicy()
	}

	m.cleanupPolicy = cleanupPolicy
}

// Rollback will rollback the enqueued operations.
func (m *Manager) Rollback() error {
	var errs []error
	for _, item := range m.queue {
		if item.Rollback == nil {
			continue
		}
		if err := item.Rollback(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Commit will apply the enqueued changes to the state of the generators.
func (m *Manager) Commit() error {
	var errs []error
	for _, item := range m.queue {
		if item.Commit == nil {
			continue
		}
		if err := item.Commit(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// EnqueueFlagLatestStateForGC will flag the latest state for garbage collection after Commit.
// It will be cleaned up later by the garbage collector.
func (m *Manager) EnqueueFlagLatestStateForGC(stateKey string) {
	m.queue = append(m.queue, QueueItem{
		Commit: func() error {
			return m.disposeState(stateKey, m.getGCGracePeriod())
		},
	})
}

// EnqueueCreateState enqueues a new state to be created.
func (m *Manager) EnqueueCreateState(stateKey, namespace string, resource *apiextensions.JSON, gen genapi.Generator, state genapi.GeneratorProviderState) {
	if state == nil {
		return
	}
	// Must be defined outside the closure
	gcDeadline := m.getGCGracePeriod()
	m.queue = append(m.queue,
		QueueItem{
			Commit: func() error {
				return m.disposeState(stateKey, gcDeadline)
			},
		},
		QueueItem{
			Commit: func() error {
				genState, err := m.createGeneratorState(resource, state, namespace, stateKey)
				if err != nil {
					return err
				}
				return m.client.Create(m.ctx, genState)
			},
			// Rollback by cleaning up the state.
			// In case of failure, create a new GeneratorState, so it will eventually be cleaned up.
			// If that also fails we're out of luck :(
			Rollback: func() error {
				err := gen.Cleanup(m.ctx, resource, state, m.client, namespace)
				if err == nil {
					return nil
				}
				genState, err := m.createGeneratorState(resource, state, namespace, stateKey)
				if err != nil {
					return err
				}
				genState.Spec.GarbageCollectionDeadline = &metav1.Time{
					Time: time.Now(),
				}
				return m.client.Create(m.ctx, genState)
			},
		},
	)
}

func (m *Manager) createGeneratorState(resource *apiextensions.JSON, state genapi.GeneratorProviderState, namespace, stateKey string) (*genapi.GeneratorState, error) {
	genState := &genapi.GeneratorState{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("gen-%s-%s-", strings.ToLower(m.resource.GetObjectKind().GroupVersionKind().Kind), m.resource.GetName()),
			Namespace:    namespace,
			Labels: map[string]string{
				genapi.GeneratorStateLabelOwnerKey: ownerKey(
					m.resource,
					stateKey,
				),
			},
		},
		Spec: genapi.GeneratorStateSpec{
			Resource: resource,
			State:    state,
		},
	}
	if err := controllerutil.SetOwnerReference(m.resource, genState, m.scheme); err != nil {
		return nil, err
	}
	return genState, nil
}

func ownerKey(resource genapi.StatefulResource, key string) string {
	return esutils.ObjectHash(fmt.Sprintf("%s-%s-%s-%s",
		resource.GetObjectKind().GroupVersionKind().Kind,
		resource.GetNamespace(),
		resource.GetName(),
		key),
	)
}

func (m *Manager) disposeState(key string, gcGracePeriod time.Duration) error {
	allStates, err := m.GetAllStates(key)
	if err != nil {
		return err
	}

	var errs []error
	for _, state := range allStates {
		if state.Spec.GarbageCollectionDeadline != nil {
			continue
		}
		state.Spec.GarbageCollectionDeadline = &metav1.Time{
			Time: time.Now().Add(gcGracePeriod),
		}
		if err := m.client.Update(m.ctx, &state); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) getGCGracePeriod() time.Duration {
	if m.cleanupPolicy == nil {
		return defaultGCGracePeriod
	}

	return m.cleanupPolicy.GracePeriod.Duration
}

// GetAllStates retrieves all the stored states for the given key.
func (m *Manager) GetAllStates(key string) ([]genapi.GeneratorState, error) {
	var stateList genapi.GeneratorStateList
	if err := m.client.List(m.ctx, &stateList, &client.MatchingLabels{
		genapi.GeneratorStateLabelOwnerKey: ownerKey(
			m.resource,
			key,
		),
	}, client.InNamespace(m.namespace)); err != nil {
		return nil, err
	}

	return stateList.Items, nil
}

// GetLatestState returns the latest state for the given key.
func (m *Manager) GetLatestState(key string) (*genapi.GeneratorState, error) {
	var stateList genapi.GeneratorStateList
	if err := m.client.List(m.ctx, &stateList, &client.MatchingLabels{
		genapi.GeneratorStateLabelOwnerKey: ownerKey(
			m.resource,
			key,
		),
	}, client.InNamespace(m.namespace)); err != nil {
		return nil, err
	}

	if latestState := getLatest(stateList.Items); latestState != nil {
		return latestState, nil
	}
	return nil, nil
}

func getLatest(stateList []genapi.GeneratorState) *genapi.GeneratorState {
	var latest *genapi.GeneratorState
	for _, state := range stateList {
		// if the state is already flagged for GC, skip it
		// It can happen that the latest based on creation timestamp is already flagged for GC.
		// That is the case when a rollback was performed.
		if state.Spec.GarbageCollectionDeadline != nil {
			continue
		}
		if latest == nil {
			latest = &state
			continue
		}
		if state.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = &state
		}
	}
	return latest
}
