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

// Package fakes contains fake implementations for testing purposes.
package fakes

import (
	"sync"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// PushRemoteRef is a fake implementation of the PushRemoteRef interface for testing.
type PushRemoteRef struct {
	GetRemoteKeyStub        func() string
	getRemoteKeyMutex       sync.RWMutex
	getRemoteKeyArgsForCall []struct {
	}
	getRemoteKeyReturns struct {
		result1 string
	}
	getRemoteKeyReturnsOnCall map[int]struct {
		result1 string
	}
	invocations      map[string][][]any
	invocationsMutex sync.RWMutex
}

// GetRemoteKey returns a string representing the remote key.
func (fake *PushRemoteRef) GetRemoteKey() string {
	fake.getRemoteKeyMutex.Lock()
	ret, specificReturn := fake.getRemoteKeyReturnsOnCall[len(fake.getRemoteKeyArgsForCall)]
	fake.getRemoteKeyArgsForCall = append(fake.getRemoteKeyArgsForCall, struct {
	}{})
	stub := fake.GetRemoteKeyStub
	fakeReturns := fake.getRemoteKeyReturns
	fake.recordInvocation("GetRemoteKey", []any{})
	fake.getRemoteKeyMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

// GetProperty returns the property value as a string.
func (fake *PushRemoteRef) GetProperty() string {
	return ""
}

// GetRemoteKeyCallCount returns the number of times GetRemoteKey has been called.
func (fake *PushRemoteRef) GetRemoteKeyCallCount() int {
	fake.getRemoteKeyMutex.RLock()
	defer fake.getRemoteKeyMutex.RUnlock()
	return len(fake.getRemoteKeyArgsForCall)
}

// GetRemoteKeyCalls sets a custom stub function for the GetRemoteKey method.
func (fake *PushRemoteRef) GetRemoteKeyCalls(stub func() string) {
	fake.getRemoteKeyMutex.Lock()
	defer fake.getRemoteKeyMutex.Unlock()
	fake.GetRemoteKeyStub = stub
}

// GetRemoteKeyReturns sets return values that will be returned by GetRemoteKey.
func (fake *PushRemoteRef) GetRemoteKeyReturns(result1 string) {
	fake.getRemoteKeyMutex.Lock()
	defer fake.getRemoteKeyMutex.Unlock()
	fake.GetRemoteKeyStub = nil
	fake.getRemoteKeyReturns = struct {
		result1 string
	}{result1}
}

// GetRemoteKeyReturnsOnCall sets return values for specific calls to GetRemoteKey.
func (fake *PushRemoteRef) GetRemoteKeyReturnsOnCall(i int, result1 string) {
	fake.getRemoteKeyMutex.Lock()
	defer fake.getRemoteKeyMutex.Unlock()
	fake.GetRemoteKeyStub = nil
	if fake.getRemoteKeyReturnsOnCall == nil {
		fake.getRemoteKeyReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.getRemoteKeyReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

// Invocations returns a map recording the calls to methods on this fake.
func (fake *PushRemoteRef) Invocations() map[string][][]any {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getRemoteKeyMutex.RLock()
	defer fake.getRemoteKeyMutex.RUnlock()
	copiedInvocations := map[string][][]any{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *PushRemoteRef) recordInvocation(key string, args []any) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]any{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]any{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ esv1.PushSecretRemoteRef = new(PushRemoteRef)
