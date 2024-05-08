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

package fake

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	vault "github.com/hashicorp/vault/api"

	util "github.com/external-secrets/external-secrets/pkg/provider/vault/util"
)

type LoginFn func(ctx context.Context, authMethod vault.AuthMethod) (*vault.Secret, error)
type Auth struct {
	LoginFn LoginFn
}

func (f Auth) Login(ctx context.Context, authMethod vault.AuthMethod) (*vault.Secret, error) {
	return f.LoginFn(ctx, authMethod)
}

type ReadWithDataWithContextFn func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error)
type ListWithContextFn func(ctx context.Context, path string) (*vault.Secret, error)
type WriteWithContextFn func(ctx context.Context, path string, data map[string]any) (*vault.Secret, error)
type DeleteWithContextFn func(ctx context.Context, path string) (*vault.Secret, error)
type Logical struct {
	ReadWithDataWithContextFn ReadWithDataWithContextFn
	ListWithContextFn         ListWithContextFn
	WriteWithContextFn        WriteWithContextFn
	DeleteWithContextFn       DeleteWithContextFn
}

func (f Logical) DeleteWithContext(ctx context.Context, path string) (*vault.Secret, error) {
	return f.DeleteWithContextFn(ctx, path)
}
func NewDeleteWithContextFn(secret map[string]any, err error) DeleteWithContextFn {
	return func(ctx context.Context, path string) (*vault.Secret, error) {
		vault := &vault.Secret{
			Data: secret,
		}
		return vault, err
	}
}

func NewReadWithContextFn(secret map[string]any, err error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
		if secret == nil {
			return nil, err
		}
		vault := &vault.Secret{
			Data: secret,
		}
		return vault, err
	}
}

func NewReadMetadataWithContextFn(secret map[string]any, err error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
		if secret == nil {
			return nil, err
		}
		metadata := make(map[string]any)
		metadata["custom_metadata"] = secret
		vault := &vault.Secret{
			Data: metadata,
		}
		return vault, err
	}
}

func NewWriteWithContextFn(secret map[string]any, err error) WriteWithContextFn {
	return func(ctx context.Context, path string, data map[string]any) (*vault.Secret, error) {
		return &vault.Secret{Data: secret}, err
	}
}

func ExpectWriteWithContextValue(expected map[string]any) WriteWithContextFn {
	return func(ctx context.Context, path string, data map[string]any) (*vault.Secret, error) {
		if strings.Contains(path, "metadata") {
			return &vault.Secret{Data: data}, nil
		}
		if !reflect.DeepEqual(expected, data) {
			return nil, fmt.Errorf("expected: %v, got: %v", expected, data)
		}
		return &vault.Secret{Data: data}, nil
	}
}

func ExpectWriteWithContextNoCall() WriteWithContextFn {
	return func(_ context.Context, path string, data map[string]any) (*vault.Secret, error) {
		return nil, fmt.Errorf("fail")
	}
}

func ExpectDeleteWithContextNoCall() DeleteWithContextFn {
	return func(ctx context.Context, path string) (*vault.Secret, error) {
		return nil, fmt.Errorf("fail")
	}
}
func WriteChangingReadContext(secret map[string]any, l Logical) WriteWithContextFn {
	v := &vault.Secret{
		Data: secret,
	}
	return func(ctx context.Context, path string, data map[string]any) (*vault.Secret, error) {
		l.ReadWithDataWithContextFn = func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
			return v, nil
		}
		return v, nil
	}
}

func (f Logical) ReadWithDataWithContext(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
	return f.ReadWithDataWithContextFn(ctx, path, data)
}
func (f Logical) ListWithContext(ctx context.Context, path string) (*vault.Secret, error) {
	return f.ListWithContextFn(ctx, path)
}
func (f Logical) WriteWithContext(ctx context.Context, path string, data map[string]any) (*vault.Secret, error) {
	return f.WriteWithContextFn(ctx, path, data)
}

type RevokeSelfWithContextFn func(ctx context.Context, token string) error
type LookupSelfWithContextFn func(ctx context.Context) (*vault.Secret, error)

type Token struct {
	RevokeSelfWithContextFn RevokeSelfWithContextFn
	LookupSelfWithContextFn LookupSelfWithContextFn
}

func (f Token) RevokeSelfWithContext(ctx context.Context, token string) error {
	return f.RevokeSelfWithContextFn(ctx, token)
}
func (f Token) LookupSelfWithContext(ctx context.Context) (*vault.Secret, error) {
	return f.LookupSelfWithContextFn(ctx)
}

type MockSetTokenFn func(v string)

type MockTokenFn func() string

type MockClearTokenFn func()

type MockNamespaceFn func() string

type MockSetNamespaceFn func(namespace string)

type MockAddHeaderFn func(key, value string)

type VaultListResponse struct {
	Metadata *vault.Response
	Data     *vault.Response
}

func NewAuthTokenFn() Token {
	return Token{nil, func(ctx context.Context) (*vault.Secret, error) {
		return &(vault.Secret{}), nil
	}}
}

func NewSetTokenFn(ofn ...func(v string)) MockSetTokenFn {
	return func(v string) {
		for _, fn := range ofn {
			fn(v)
		}
	}
}

func NewTokenFn(v string) MockTokenFn {
	return func() string {
		return v
	}
}

func NewClearTokenFn() MockClearTokenFn {
	return func() {}
}

func NewAddHeaderFn() MockAddHeaderFn {
	return func(key, value string) {}
}

type VaultClient struct {
	MockLogical      Logical
	MockAuth         Auth
	MockAuthToken    Token
	MockSetToken     MockSetTokenFn
	MockToken        MockTokenFn
	MockClearToken   MockClearTokenFn
	MockNamespace    MockNamespaceFn
	MockSetNamespace MockSetNamespaceFn
	MockAddHeader    MockAddHeaderFn

	namespace string
	lock      sync.RWMutex
}

func (c *VaultClient) Logical() Logical {
	return c.MockLogical
}

func NewVaultLogical() Logical {
	logical := Logical{
		ReadWithDataWithContextFn: func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
			return nil, nil
		},
		ListWithContextFn: func(ctx context.Context, path string) (*vault.Secret, error) {
			return nil, nil
		},
		WriteWithContextFn: func(ctx context.Context, path string, data map[string]any) (*vault.Secret, error) {
			return nil, nil
		},
	}
	return logical
}
func (c *VaultClient) Auth() Auth {
	return c.MockAuth
}

func NewVaultAuth() Auth {
	auth := Auth{
		LoginFn: func(ctx context.Context, authMethod vault.AuthMethod) (*vault.Secret, error) {
			return nil, nil
		},
	}
	return auth
}
func (c *VaultClient) AuthToken() Token {
	return c.MockAuthToken
}

func (c *VaultClient) SetToken(v string) {
	c.MockSetToken(v)
}

func (c *VaultClient) Token() string {
	return c.MockToken()
}

func (c *VaultClient) ClearToken() {
	c.MockClearToken()
}

func (c *VaultClient) Namespace() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ns := c.namespace
	return ns
}

func (c *VaultClient) SetNamespace(namespace string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.namespace = namespace
}

func (c *VaultClient) AddHeader(key, value string) {
	c.MockAddHeader(key, value)
}

func ClientWithLoginMock(_ *vault.Config) (util.Client, error) {
	cl := VaultClient{
		MockAuthToken: NewAuthTokenFn(),
		MockSetToken:  NewSetTokenFn(),
		MockToken:     NewTokenFn(""),
		MockAuth:      NewVaultAuth(),
		MockLogical:   NewVaultLogical(),
	}

	return &util.VaultClient{
		SetTokenFunc:     cl.SetToken,
		TokenFunc:        cl.Token,
		ClearTokenFunc:   cl.ClearToken,
		AuthField:        cl.Auth(),
		AuthTokenField:   cl.AuthToken(),
		LogicalField:     cl.Logical(),
		NamespaceFunc:    cl.Namespace,
		SetNamespaceFunc: cl.SetNamespace,
		AddHeaderFunc:    cl.AddHeader,
	}, nil
}
