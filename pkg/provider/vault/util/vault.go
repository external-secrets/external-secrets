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

package util

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/credentials"
	vault "github.com/hashicorp/vault/api"
)

type JwtProviderFactory func(name, namespace, roleArn string, aud []string, region string) (credentials.Provider, error)

type Auth interface {
	Login(ctx context.Context, authMethod vault.AuthMethod) (*vault.Secret, error)
}

type Token interface {
	RevokeSelfWithContext(ctx context.Context, token string) error
	LookupSelfWithContext(ctx context.Context) (*vault.Secret, error)
}

type Logical interface {
	ReadWithDataWithContext(ctx context.Context, path string, data map[string][]string) (*vault.Secret, error)
	ListWithContext(ctx context.Context, path string) (*vault.Secret, error)
	WriteWithContext(ctx context.Context, path string, data map[string]interface{}) (*vault.Secret, error)
	DeleteWithContext(ctx context.Context, path string) (*vault.Secret, error)
}

type Client interface {
	SetToken(v string)
	Token() string
	ClearToken()
	Auth() Auth
	Logical() Logical
	AuthToken() Token
	SetNamespace(namespace string)
	AddHeader(key, value string)
}

type VClient struct {
	SetTokenFunc     func(v string)
	TokenFunc        func() string
	ClearTokenFunc   func()
	AuthField        Auth
	LogicalField     Logical
	AuthTokenField   Token
	SetNamespaceFunc func(namespace string)
	AddHeaderFunc    func(key, value string)
}

func (v VClient) AddHeader(key, value string) {
	v.AddHeaderFunc(key, value)
}

func (v VClient) SetNamespace(namespace string) {
	v.SetNamespaceFunc(namespace)
}

func (v VClient) ClearToken() {
	v.ClearTokenFunc()
}

func (v VClient) Token() string {
	return v.TokenFunc()
}

func (v VClient) SetToken(token string) {
	v.SetTokenFunc(token)
}

func (v VClient) Auth() Auth {
	return v.AuthField
}

func (v VClient) AuthToken() Token {
	return v.AuthTokenField
}

func (v VClient) Logical() Logical {
	return v.LogicalField
}
