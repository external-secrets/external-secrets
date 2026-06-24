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

type AuthFactoryMock struct {
	lock  sync.RWMutex
	calls []string
}

func (a *AuthFactoryMock) callf(format string, args ...any) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.calls = append(a.calls, fmt.Sprintf(format, args...))
}

func (a *AuthFactoryMock) AppRole(id, secret, mount string) (api.AuthMethod, error) {
	a.callf("AppRole(%q, %q, %q)", id, secret, mount)
	return mockAuth{}, nil
}

func (a *AuthFactoryMock) UserPass(username, password, mount string) (api.AuthMethod, error) {
	a.callf("UserPass(%q, %q, %q)", username, password, mount)
	return mockAuth{}, nil
}

func (a *AuthFactoryMock) GetCalls() []string {
	a.lock.Lock()
	defer a.lock.Unlock()
	return slices.Clone(a.calls)
}

var _ AuthMethodFactory = &AuthFactoryMock{}

type mockAuth struct{}

func (m mockAuth) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	return &api.Secret{
		Auth: &api.SecretAuth{
			ClientToken: "mock",
		},
	}, nil
}

var _ api.AuthMethod = mockAuth{}
