//Copyright External Secrets Inc. All Rights Reserved

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
	WriteWithContext(ctx context.Context, path string, data map[string]any) (*vault.Secret, error)
	DeleteWithContext(ctx context.Context, path string) (*vault.Secret, error)
}

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

type VaultClient struct {
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

func (v VaultClient) AddHeader(key, value string) {
	v.AddHeaderFunc(key, value)
}

func (v VaultClient) Namespace() string {
	return v.NamespaceFunc()
}

func (v VaultClient) SetNamespace(namespace string) {
	v.SetNamespaceFunc(namespace)
}

func (v VaultClient) ClearToken() {
	v.ClearTokenFunc()
}

func (v VaultClient) Token() string {
	return v.TokenFunc()
}

func (v VaultClient) SetToken(token string) {
	v.SetTokenFunc(token)
}

func (v VaultClient) Auth() Auth {
	return v.AuthField
}

func (v VaultClient) AuthToken() Token {
	return v.AuthTokenField
}

func (v VaultClient) Logical() Logical {
	return v.LogicalField
}
