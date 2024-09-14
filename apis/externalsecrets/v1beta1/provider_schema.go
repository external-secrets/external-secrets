//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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
