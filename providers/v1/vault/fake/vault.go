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

package fake

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	vault "github.com/hashicorp/vault/api"

	"github.com/external-secrets/external-secrets/providers/v1/vault/util"
)

// LoginFn is a function type that represents logging in to Vault using a specific authentication method.
type LoginFn func(ctx context.Context, authMethod vault.AuthMethod) (*vault.Secret, error)

// Auth is a mock implementation of the Vault authentication interface for testing purposes.
type Auth struct {
	LoginFn LoginFn
}

// Login logs in to Vault using the specified authentication method.
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

func buildDataResponse(secret map[string]any, err error) (*vault.Secret, error) {
	if secret == nil {
		return nil, err
	}
	return &vault.Secret{Data: secret}, err
}

func buildMetadataResponse(secret map[string]any, err error) (*vault.Secret, error) {
	if secret == nil {
		return nil, err
	}
	// If the secret already has the expected metadata structure, return as-is
	if _, hasCustomMetadata := secret["custom_metadata"]; hasCustomMetadata {
		return &vault.Secret{Data: secret}, err
	}
	// Otherwise, wrap in custom_metadata for backwards compatibility
	metadata := make(map[string]any)
	metadata["custom_metadata"] = secret
	return &vault.Secret{Data: metadata}, err
}

func NewReadWithContextFn(secret map[string]any, err error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
		return buildDataResponse(secret, err)
	}
}

func NewReadMetadataWithContextFn(secret map[string]any, err error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
		return buildMetadataResponse(secret, err)
	}
}

func NewReadWithDataAndMetadataFn(dataSecret, metadataSecret map[string]any, dataErr, metadataErr error) ReadWithDataWithContextFn {
	return func(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
		// Check if this is a metadata path request
		if strings.Contains(path, "/metadata/") {
			return buildMetadataResponse(metadataSecret, metadataErr)
		}

		// This is a data path request
		return buildDataResponse(dataSecret, dataErr)
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
		return nil, errors.New("fail")
	}
}

func ExpectDeleteWithContextNoCall() DeleteWithContextFn {
	return func(ctx context.Context, path string) (*vault.Secret, error) {
		return nil, errors.New("fail")
	}
}

// ReadWithDataWithContext reads the secret at the specified path in Vault with additional data.
func (f Logical) ReadWithDataWithContext(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error) {
	return f.ReadWithDataWithContextFn(ctx, path, data)
}

// ListWithContext lists the secrets at the specified path in Vault.
func (f Logical) ListWithContext(ctx context.Context, path string) (*vault.Secret, error) {
	return f.ListWithContextFn(ctx, path)
}

// WriteWithContext writes data to the specified path in Vault.
func (f Logical) WriteWithContext(ctx context.Context, path string, data map[string]any) (*vault.Secret, error) {
	return f.WriteWithContextFn(ctx, path, data)
}

// RevokeSelfWithContextFn is a function type that represents revoking the Vault token associated with the current client.
type RevokeSelfWithContextFn func(ctx context.Context, token string) error

// LookupSelfWithContextFn is a function type that represents looking up the Vault token associated with the current client.
type LookupSelfWithContextFn func(ctx context.Context) (*vault.Secret, error)

// Token is a mock implementation of the Vault token interface for testing purposes.
type Token struct {
	RevokeSelfWithContextFn RevokeSelfWithContextFn
	LookupSelfWithContextFn LookupSelfWithContextFn
}

// RevokeSelfWithContext revokes the token associated with the current client.
func (f Token) RevokeSelfWithContext(ctx context.Context, token string) error {
	return f.RevokeSelfWithContextFn(ctx, token)
}

// LookupSelfWithContext looks up the token associated with the current client.
func (f Token) LookupSelfWithContext(ctx context.Context) (*vault.Secret, error) {
	return f.LookupSelfWithContextFn(ctx)
}

// MockSetTokenFn is a function type that represents setting the Vault token.
type MockSetTokenFn func(v string)

// MockTokenFn is a function type that represents getting the Vault token.
type MockTokenFn func() string

// MockClearTokenFn is a function type that represents clearing the Vault token.
type MockClearTokenFn func()

// MockNamespaceFn is a function type that represents getting the Vault namespace.
type MockNamespaceFn func() string

// MockSetNamespaceFn is a function type that represents setting the Vault namespace.
type MockSetNamespaceFn func(namespace string)

// MockAddHeaderFn is a function type that represents adding a header to the Vault client requests.
type MockAddHeaderFn func(key, value string)

// VaultListResponse is a struct to represent the response from a Vault list operation.
type VaultListResponse struct {
	Metadata *vault.Response
	Data     *vault.Response
}

// NewAuthTokenFn returns a MockAuthToken that always returns a nil secret and nil error.
func NewAuthTokenFn() Token {
	return Token{nil, func(context.Context) (*vault.Secret, error) {
		return &(vault.Secret{}), nil
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

// VaultClient is a mock implementation of the Vault client interface for testing purposes.
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

// Logical returns the mock Logical.
func (c *VaultClient) Logical() Logical {
	return c.MockLogical
}

// NewVaultLogical returns a new vault Logical instance.
func NewVaultLogical() Logical {
	logical := Logical{
		ReadWithDataWithContextFn: func(context.Context, string, map[string][]string) (*vault.Secret, error) {
			return nil, nil
		},
		ListWithContextFn: func(context.Context, string) (*vault.Secret, error) {
			return nil, nil
		},
		WriteWithContextFn: func(context.Context, string, map[string]any) (*vault.Secret, error) {
			return nil, nil
		},
	}
	return logical
}

// Auth returns the mock authentication.
func (c *VaultClient) Auth() Auth {
	return c.MockAuth
}

// NewVaultAuth returns a mock authentication Auth.
func NewVaultAuth() Auth {
	auth := Auth{
		LoginFn: func(context.Context, vault.AuthMethod) (*vault.Secret, error) {
			return nil, nil
		},
	}
	return auth
}

// AuthToken returns the mock authentication token interface.
func (c *VaultClient) AuthToken() Token {
	return c.MockAuthToken
}

// SetToken sets the authentication token.
func (c *VaultClient) SetToken(v string) {
	c.MockSetToken(v)
}

// Token returns the current authentication token.
func (c *VaultClient) Token() string {
	return c.MockToken()
}

// ClearToken clears the current authentication token.
func (c *VaultClient) ClearToken() {
	c.MockClearToken()
}

// Namespace returns the current Vault namespace.
func (c *VaultClient) Namespace() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ns := c.namespace
	return ns
}

// SetNamespace sets the Vault namespace.
func (c *VaultClient) SetNamespace(namespace string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.namespace = namespace
}

// AddHeader adds a header to the Vault client requests.
func (c *VaultClient) AddHeader(key, value string) {
	c.MockAddHeader(key, value)
}

// ClientWithLoginMock returns a client with mocked login functionality.
func ClientWithLoginMock(config *vault.Config) (vaultutil.Client, error) {
	return clientWithLoginMockOptions(config)
}

// ModifiableClientWithLoginMock returns a factory function that creates clients with customizable mock behavior.
func ModifiableClientWithLoginMock(opts ...func(cl *VaultClient)) func(config *vault.Config) (vaultutil.Client, error) {
	return func(config *vault.Config) (vaultutil.Client, error) {
		return clientWithLoginMockOptions(config, opts...)
	}
}

func clientWithLoginMockOptions(_ *vault.Config, opts ...func(cl *VaultClient)) (vaultutil.Client, error) {
	cl := &VaultClient{
		MockAuthToken: NewAuthTokenFn(),
		MockSetToken:  NewSetTokenFn(),
		MockToken:     NewTokenFn(""),
		MockAuth:      NewVaultAuth(),
		MockLogical:   NewVaultLogical(),
	}

	for _, opt := range opts {
		opt(cl)
	}

	return &vaultutil.VaultClient{
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
