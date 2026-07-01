/*
Copyright © The ESO Authors

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

package auth

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/openbao/openbao/api/v2"
)

// MockFactory is a mock implementation of [Factory].
// All factory methods return mock auth methods, which return a static dummy token.
// All calls are recorded and can be validated using [MockFactory.GetCalls].
type MockFactory struct {
	lock  sync.RWMutex
	calls []string
}

func (a *MockFactory) callf(format string, args ...any) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.calls = append(a.calls, fmt.Sprintf(format, args...))
}

// AppRole implements [Factory].
func (a *MockFactory) AppRole(id, secret, mount string) (api.AuthMethod, error) {
	a.callf("AppRole(%q, %q, %q)", id, secret, mount)
	return mockAuth{}, nil
}

// UserPass implements [Factory].
func (a *MockFactory) UserPass(username, password, mount string) (api.AuthMethod, error) {
	a.callf("UserPass(%q, %q, %q)", username, password, mount)
	return mockAuth{}, nil
}

// Kubernetes implements [Factory].
func (a *MockFactory) Kubernetes(role, jwt, mount string) (api.AuthMethod, error) {
	a.callf("Kubernetes(%q, %q, %v, %v)", role, jwt, mount)
	return mockAuth{}, nil
}

// GetCalls returns a list of all calls made to the mock (serialized as string), e.g.:
//
//	UserPass("user", "password", "mount")
func (a *MockFactory) GetCalls() []string {
	a.lock.Lock()
	defer a.lock.Unlock()
	return slices.Clone(a.calls)
}

var _ Factory = &MockFactory{}

type mockAuth struct{}

func (m mockAuth) Login(_ context.Context, _ *api.Client) (*api.Secret, error) {
	return &api.Secret{
		Auth: &api.SecretAuth{
			ClientToken: "mock",
		},
	}, nil
}

var _ api.AuthMethod = mockAuth{}
