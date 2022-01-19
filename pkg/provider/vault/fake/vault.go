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

type MockNewRequestFn func(method, requestPath string) *vault.Request

type MockRawRequestWithContextFn func(ctx context.Context, r *vault.Request) (*vault.Response, error)

type MockSetTokenFn func(v string)

type MockTokenFn func() string

type MockClearTokenFn func()

type MockSetNamespaceFn func(namespace string)

type MockAddHeaderFn func(key, value string)

func NewMockNewRequestFn(req *vault.Request) MockNewRequestFn {
	return func(method, requestPath string) *vault.Request {
		return req
	}
}

// An RequestFn operates on the supplied Request. You might use an RequestFn to
// test or update the contents of an Request.
type RequestFn func(req *vault.Request) error

func NewMockRawRequestWithContextFn(res *vault.Response, err error, ofn ...RequestFn) MockRawRequestWithContextFn {
	return func(_ context.Context, r *vault.Request) (*vault.Response, error) {
		for _, fn := range ofn {
			if err := fn(r); err != nil {
				return res, err
			}
		}
		return res, err
	}
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
	MockNewRequest            MockNewRequestFn
	MockRawRequestWithContext MockRawRequestWithContextFn
	MockSetToken              MockSetTokenFn
	MockToken                 MockTokenFn
	MockClearToken            MockClearTokenFn
	MockSetNamespace          MockSetNamespaceFn
	MockAddHeader             MockAddHeaderFn
}

func (c *VaultClient) NewRequest(method, requestPath string) *vault.Request {
	return c.MockNewRequest(method, requestPath)
}

func (c *VaultClient) RawRequestWithContext(ctx context.Context, r *vault.Request) (*vault.Response, error) {
	return c.MockRawRequestWithContext(ctx, r)
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
