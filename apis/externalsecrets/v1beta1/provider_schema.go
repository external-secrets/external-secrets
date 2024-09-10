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

package v1beta1

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

var builder map[string]Provider
var buildlock sync.RWMutex

func init() {
	builder = make(map[string]Provider)
}

// Register a store backend type. Register panics if a
// backend with the same store is already registered.
func Register(s Provider, storeSpec *SecretStoreProvider) {
	storeName, err := getProviderName(storeSpec)
	if err != nil {
		panic(fmt.Sprintf("store error registering schema: %s", err.Error()))
	}

	buildlock.Lock()
	defer buildlock.Unlock()
	_, exists := builder[storeName]
	if exists {
		panic(fmt.Sprintf("store %q already registered", storeName))
	}

	builder[storeName] = s
}

func RegisterByName(s Provider, name string) {
	buildlock.Lock()
	defer buildlock.Unlock()
	_, exists := builder[name]
	if exists {
		panic(fmt.Sprintf("store %q already registered", name))
	}

	builder[name] = s
}

// ForceRegister adds to store schema, overwriting a store if
// already registered. Should only be used for testing.
func ForceRegister(s Provider, storeSpec *SecretStoreProvider) {
	storeName, err := getProviderName(storeSpec)
	if err != nil {
		panic(fmt.Sprintf("store error registering schema: %s", err.Error()))
	}

	buildlock.Lock()
	builder[storeName] = s
	buildlock.Unlock()
}

// GetProviderByName returns the provider implementation by name.
func GetProviderByName(name string) (Provider, bool) {
	buildlock.RLock()
	f, ok := builder[name]
	buildlock.RUnlock()
	return f, ok
}

// GetProviderByRef returns the provider by its kind.
func GetProviderByRef(ref esmeta.ProviderRef) (Provider, error) {
	buildlock.RLock()
	f, ok := builder[ref.Kind]
	buildlock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("provider %v not found", ref.Kind)
	}
	return f, nil
}

// GetProvider returns the provider from the generic store.
func GetProvider(s GenericStore) (Provider, error) {
	if s == nil {
		return nil, nil
	}
	spec := s.GetSpec()
	if spec == nil {
		// Note, this condition can never be reached, because
		// the Spec is not a pointer in Kubernetes. It will
		// always exist.
		return nil, fmt.Errorf("no spec found in %#v", s)
	}
	storeName, err := getProviderName(spec.Provider)
	if err != nil {
		return nil, fmt.Errorf("store error for %s: %w", s.GetName(), err)
	}

	buildlock.RLock()
	f, ok := builder[storeName]
	buildlock.RUnlock()

	if !ok {
		return nil, fmt.Errorf("failed to find registered store backend for type: %s, name: %s", storeName, s.GetName())
	}

	return f, nil
}

// getProviderName returns the name of the configured provider
// or an error if the provider is not configured.
func getProviderName(storeSpec *SecretStoreProvider) (string, error) {
	storeBytes, err := json.Marshal(storeSpec)
	if err != nil || storeBytes == nil {
		return "", fmt.Errorf("failed to marshal store spec: %w", err)
	}

	storeMap := make(map[string]any)
	err = json.Unmarshal(storeBytes, &storeMap)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal store spec: %w", err)
	}

	if len(storeMap) != 1 {
		return "", fmt.Errorf("secret stores must only have exactly one backend specified, found %d", len(storeMap))
	}

	for k := range storeMap {
		return k, nil
	}

	return "", errors.New("failed to find registered store backend")
}
