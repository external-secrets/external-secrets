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

// Package baoutil provides utility types and functions for interacting with OpenBao.
package baoutil

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	bao "github.com/hashicorp/vault/api"
)

// JwtProviderFactory is a function type that creates a JWT credentials provider.
type JwtProviderFactory func(ctx context.Context, name, namespace, roleArn string, aud []string, region string) (aws.CredentialsProvider, error)

// Auth defines the interface for OpenBao authentication.
type Auth interface {
	Login(ctx context.Context, authMethod bao.AuthMethod) (*bao.Secret, error)
}

// Token defines the interface for OpenBao token operations.
type Token interface {
	RevokeSelfWithContext(ctx context.Context, token string) error
	LookupSelfWithContext(ctx context.Context) (*bao.Secret, error)
}

// Logical defines the interface for OpenBao's logical operations.
type Logical interface {
	ReadWithDataWithContext(ctx context.Context, path string, data map[string][]string) (*bao.Secret, error)
	ListWithContext(ctx context.Context, path string) (*bao.Secret, error)
	WriteWithContext(ctx context.Context, path string, data map[string]any) (*bao.Secret, error)
	DeleteWithContext(ctx context.Context, path string) (*bao.Secret, error)
}

// Client defines the interface for an OpenBao client with methods for token management,
// authentication, and secret operations.
type Client interface {
	SetToken(v string)
	Token() string
	ClearToken()
	Auth() Auth
	Logical() Logical
	AuthToken() Token
	Namespace() string
	SetNamespace(namespace string)
	AddHeader(key, value string)
}

// OpenBaoClient is a wrapper around the OpenBao API client that provides
// methods for authentication, token management, and secret operations.
type OpenBaoClient struct {
	SetTokenFunc     func(v string)
	TokenFunc        func() string
	ClearTokenFunc   func()
	AuthField        Auth
	LogicalField     Logical
	AuthTokenField   Token
	NamespaceFunc    func() string
	SetNamespaceFunc func(namespace string)
	AddHeaderFunc    func(key, value string)
}

// AddHeader adds a header to all requests using the provided key, value pair.
func (v OpenBaoClient) AddHeader(key, value string) {
	v.AddHeaderFunc(key, value)
}

// Namespace returns the current OpenBao namespace.
func (v OpenBaoClient) Namespace() string {
	return v.NamespaceFunc()
}

// SetNamespace sets the OpenBao namespace to use for requests.
func (v OpenBaoClient) SetNamespace(namespace string) {
	v.SetNamespaceFunc(namespace)
}

// ClearToken clears the OpenBao token.
func (v OpenBaoClient) ClearToken() {
	v.ClearTokenFunc()
}

// Token returns the current OpenBao token.
func (v OpenBaoClient) Token() string {
	return v.TokenFunc()
}

// SetToken sets the OpenBao token to use for requests.
func (v OpenBaoClient) SetToken(token string) {
	v.SetTokenFunc(token)
}

// Auth returns the Auth interface for authentication operations.
func (v OpenBaoClient) Auth() Auth {
	return v.AuthField
}

// AuthToken returns the Token interface for token operations.
func (v OpenBaoClient) AuthToken() Token {
	return v.AuthTokenField
}

// Logical returns the Logical interface for secret operations.
func (v OpenBaoClient) Logical() Logical {
	return v.LogicalField
}
