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

package fake

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	bao "github.com/hashicorp/vault/api"

	baoutil "github.com/external-secrets/external-secrets/providers/v1/openbao/util"
)

// LoginFn is a function type that represents logging in to OpenBao using a specific authentication method.
type LoginFn func(ctx context.Context, authMethod bao.AuthMethod) (*bao.Secret, error)

// Auth is a mock implementation of the OpenBao authentication interface for testing purposes.
type Auth struct {
	LoginFn LoginFn
}

// Login logs in to OpenBao using the specified authentication method.
func (f Auth) Login(ctx context.Context, authMethod bao.AuthMethod) (*bao.Secret, error) {
	return f.LoginFn(ctx, authMethod)
}

type ReadWithDataWithContextFn func(ctx context.Context, path string, data map[string][]string) (*bao.Secret, error)
type ListWithContextFn func(ctx context.Context, path string) (*bao.Secret, error)
type WriteWithContextFn func(ctx context.Context, path string, data map[string]any) (*bao.Secret, error)
type DeleteWithContextFn func(ctx context.Context, path string) (*bao.Secret, error)
type Logical struct {
	ReadWithDataWithContextFn ReadWithDataWithContextFn
	ListWithContextFn         ListWithContextFn
	WriteWithContextFn        WriteWithContextFn
	DeleteWithContextFn       DeleteWithContextFn
}

func (f Logical) DeleteWithContext(ctx context.Context, path string) (*bao.Secret, error) {
	return f.DeleteWithContextFn(ctx, path)
}
func NewDeleteWithContextFn(secret map[string]any, err error) DeleteWithContextFn {
	return func(ctx context.Context, path string) (*bao.Secret, error) {
		bao := &bao.Secret{
			Data: secret,
		}
		return bao, err
	}
}

func buildDataResponse(secret map[string]any, err error) (*bao.Secret, error) {
	if secret == nil {
		return nil, err
	}
	return &bao.Secret{Data: secret}, err
}

func buildMetadataResponse(secret map[string]any, err error) (*bao.Secret, error) {
	if secret == nil {
		return nil, err
	}
	// If the secret already has the expected metadata structure, return as-is
	if _, hasCustomMetadata := secret["custom_metadata"]; hasCustomMetadata {
		return &bao.Secret{Data: secret}, err
	}
	// Otherwise, wrap in custom_metadata for backwards compatibility
	metadata := make(map[string]any)
	metadata["custom_metadata"] = secret
	return &bao.Secret{Data: metadata}, err
}

func NewReadWithContextFn(secret map[string]any, err error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*bao.Secret, error) {
		return buildDataResponse(secret, err)
	}
}

func NewReadMetadataWithContextFn(secret map[string]any, err error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*bao.Secret, error) {
		return buildMetadataResponse(secret, err)
	}
}

func NewReadWithDataAndMetadataFn(dataSecret, metadataSecret map[string]any, dataErr, metadataErr error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*bao.Secret, error) {
		// Check if this is a metadata path request
		if strings.Contains(path, "/metadata/") {
			return buildMetadataResponse(metadataSecret, metadataErr)
		}

		// This is a data path request
		return buildDataResponse(dataSecret, dataErr)
	}
}

func NewWriteWithContextFn(secret map[string]any, err error) WriteWithContextFn {
	return func(ctx context.Context, path string, data map[string]any) (*bao.Secret, error) {
		return &bao.Secret{Data: secret}, err
	}
}

func ExpectWriteWithContextValue(expected map[string]any) WriteWithContextFn {
	return func(ctx context.Context, path string, data map[string]any) (*bao.Secret, error) {
		if strings.Contains(path, "metadata") {
			return &bao.Secret{Data: data}, nil
		}
		if !reflect.DeepEqual(expected, data) {
			return nil, fmt.Errorf("expected: %v, got: %v", expected, data)
		}
		return &bao.Secret{Data: data}, nil
	}
}

func ExpectWriteWithContextNoCall() WriteWithContextFn {
	return func(_ context.Context, path string, data map[string]any) (*bao.Secret, error) {
		return nil, errors.New("fail")
	}
}

func ExpectDeleteWithContextNoCall() DeleteWithContextFn {
	return func(ctx context.Context, path string) (*bao.Secret, error) {
		return nil, errors.New("fail")
	}
}

// ReadWithDataWithContext reads the secret at the specified path in OpenBao with additional data.
func (f Logical) ReadWithDataWithContext(ctx context.Context, path string, data map[string][]string) (*bao.Secret, error) {
	return f.ReadWithDataWithContextFn(ctx, path, data)
}

// ListWithContext lists the secrets at the specified path in OpenBao.
func (f Logical) ListWithContext(ctx context.Context, path string) (*bao.Secret, error) {
	return f.ListWithContextFn(ctx, path)
}

// WriteWithContext writes data to the specified path in OpenBao.
func (f Logical) WriteWithContext(ctx context.Context, path string, data map[string]any) (*bao.Secret, error) {
	return f.WriteWithContextFn(ctx, path, data)
}

// RevokeSelfWithContextFn is a function type that represents revoking the OpenBao token associated with the current client.
type RevokeSelfWithContextFn func(ctx context.Context, token string) error

// LookupSelfWithContextFn is a function type that represents looking up the OpenBao token associated with the current client.
type LookupSelfWithContextFn func(ctx context.Context) (*bao.Secret, error)

// Token is a mock implementation of the OpenBao token interface for testing purposes.
type Token struct {
	RevokeSelfWithContextFn RevokeSelfWithContextFn
	LookupSelfWithContextFn LookupSelfWithContextFn
}

// RevokeSelfWithContext revokes the token associated with the current client.
func (f Token) RevokeSelfWithContext(ctx context.Context, token string) error {
	return f.RevokeSelfWithContextFn(ctx, token)
}

// LookupSelfWithContext looks up the token associated with the current client.
func (f Token) LookupSelfWithContext(ctx context.Context) (*bao.Secret, error) {
	return f.LookupSelfWithContextFn(ctx)
}

// MockSetTokenFn is a function type that represents setting the OpenBao token.
type MockSetTokenFn func(v string)

// MockTokenFn is a function type that represents getting the OpenBao token.
type MockTokenFn func() string

// MockClearTokenFn is a function type that represents clearing the OpenBao token.
type MockClearTokenFn func()

// MockNamespaceFn is a function type that represents getting the OpenBao namespace.
type MockNamespaceFn func() string

// MockSetNamespaceFn is a function type that represents setting the OpenBao namespace.
type MockSetNamespaceFn func(namespace string)

// MockAddHeaderFn is a function type that represents adding a header to the OpenBao client requests.
type MockAddHeaderFn func(key, value string)

// OpenBaoListResponse is a struct to represent the response from an OpenBao list operation.
type OpenBaoListResponse struct {
	Metadata *bao.Response
	Data     *bao.Response
}

// NewAuthTokenFn returns a MockAuthToken that always returns a nil secret and nil error.
func NewAuthTokenFn() Token {
	return Token{nil, func(context.Context) (*bao.Secret, error) {
		return &(bao.Secret{}), nil
	}}
}

// NewSetTokenFn returns a MockSetTokenFn that calls the provided functions in order.
func NewSetTokenFn(ofn ...func(v string)) MockSetTokenFn {
	return func(v string) {
		for _, fn := range ofn {
			fn(v)
		}
	}
}

// NewTokenFn returns a MockTokenFn that always returns the provided string.
func NewTokenFn(v string) MockTokenFn {
	return func() string {
		return v
	}
}

// OpenBaoClient is a mock implementation of the OpenBao client interface for testing purposes.
type OpenBaoClient struct {
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

// Logical returns the mock Logical.
func (c *OpenBaoClient) Logical() Logical {
	return c.MockLogical
}

// newOpenBaoLogical returns a new OpenBao Logical instance.
func newOpenBaoLogical() Logical {
	logical := Logical{
		ReadWithDataWithContextFn: func(context.Context, string, map[string][]string) (*bao.Secret, error) {
			return nil, nil
		},
		ListWithContextFn: func(context.Context, string) (*bao.Secret, error) {
			return nil, nil
		},
		WriteWithContextFn: func(context.Context, string, map[string]any) (*bao.Secret, error) {
			return nil, nil
		},
	}
	return logical
}

// Auth returns the mock authentication.
func (c *OpenBaoClient) Auth() Auth {
	return c.MockAuth
}

// newOpenBaoAuth returns a mock authentication Auth.
func newOpenBaoAuth() Auth {
	auth := Auth{
		LoginFn: func(context.Context, bao.AuthMethod) (*bao.Secret, error) {
			return nil, nil
		},
	}
	return auth
}

// AuthToken returns the mock authentication token interface.
func (c *OpenBaoClient) AuthToken() Token {
	return c.MockAuthToken
}

// SetToken sets the authentication token.
func (c *OpenBaoClient) SetToken(v string) {
	c.MockSetToken(v)
}

// Token returns the current authentication token.
func (c *OpenBaoClient) Token() string {
	return c.MockToken()
}

// ClearToken clears the current authentication token.
func (c *OpenBaoClient) ClearToken() {
	c.MockClearToken()
}

// Namespace returns the current OpenBao namespace.
func (c *OpenBaoClient) Namespace() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ns := c.namespace
	return ns
}

// SetNamespace sets the OpenBao namespace.
func (c *OpenBaoClient) SetNamespace(namespace string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.namespace = namespace
}

// AddHeader adds a header to the OpenBao client requests.
func (c *OpenBaoClient) AddHeader(key, value string) {
	c.MockAddHeader(key, value)
}

// ClientWithLoginMock returns a client with mocked login functionality.
func ClientWithLoginMock(config *bao.Config) (baoutil.Client, error) {
	return clientWithLoginMockOptions(config)
}

// ModifiableClientWithLoginMock returns a factory function that creates clients with customizable mock behavior.
func ModifiableClientWithLoginMock(opts ...func(cl *OpenBaoClient)) func(config *bao.Config) (baoutil.Client, error) {
	return func(config *bao.Config) (baoutil.Client, error) {
		return clientWithLoginMockOptions(config, opts...)
	}
}

func clientWithLoginMockOptions(_ *bao.Config, opts ...func(cl *OpenBaoClient)) (baoutil.Client, error) {
	cl := &OpenBaoClient{
		MockAuthToken: NewAuthTokenFn(),
		MockSetToken:  NewSetTokenFn(),
		MockToken:     NewTokenFn(""),
		MockAuth:      newOpenBaoAuth(),
		MockLogical:   newOpenBaoLogical(),
	}

	for _, opt := range opts {
		opt(cl)
	}

	return &baoutil.OpenBaoClient{
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
