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
