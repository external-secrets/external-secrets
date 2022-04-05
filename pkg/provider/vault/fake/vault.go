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

	vault "github.com/hashicorp/vault/api"
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
type WriteWithContextFn func(ctx context.Context, path string, data map[string]interface{}) (*vault.Secret, error)

type Logical struct {
	ReadWithDataWithContextFn ReadWithDataWithContextFn
	ListWithContextFn         ListWithContextFn
	WriteWithContextFn        WriteWithContextFn
}

func NewReadWithContextFn(secret map[string]interface{}, err error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
		vault := &vault.Secret{
			Data: secret,
		}
		return vault, err
	}
}

func (f Logical) ReadWithDataWithContext(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
	return f.ReadWithDataWithContextFn(ctx, path, data)
}
func (f Logical) ListWithContext(ctx context.Context, path string) (*vault.Secret, error) {
	return f.ListWithContextFn(ctx, path)
}
func (f Logical) WriteWithContext(ctx context.Context, path string, data map[string]interface{}) (*vault.Secret, error) {
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

type MockSetNamespaceFn func(namespace string)

type MockAddHeaderFn func(key, value string)

type VaultListResponse struct {
	Metadata *vault.Response
	Data     *vault.Response
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

func NewSetNamespaceFn() MockSetNamespaceFn {
	return func(namespace string) {}
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
	MockSetNamespace MockSetNamespaceFn
	MockAddHeader    MockAddHeaderFn
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
		WriteWithContextFn: func(ctx context.Context, path string, data map[string]interface{}) (*vault.Secret, error) {
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

func (c *VaultClient) SetNamespace(namespace string) {
	c.MockSetNamespace(namespace)
}

func (c *VaultClient) AddHeader(key, value string) {
	c.MockAddHeader(key, value)
}
