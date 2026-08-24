// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package auth

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// CredentialRequest represents a lazily resolved credential request.
type CredentialRequest struct {
	cacheKey string
	resolve  func(context.Context) (TokenExchangeCredentials, error)
}

// CacheKey returns the key used to cache the credential request.
func (r CredentialRequest) CacheKey() string {
	return r.cacheKey
}

// Resolve resolves the credential request.
func (r CredentialRequest) Resolve(ctx context.Context) (TokenExchangeCredentials, error) {
	return r.resolve(ctx)
}

// TokenExchangeCredentials represents credentials accepted by the IAM token exchanger.
type TokenExchangeCredentials interface {
	isTokenExchangeCredentials()
}

// TokenCredentials contains an IAM token.
type TokenCredentials struct {
	Token string
}

// GetTokenCredentials reads token credentials from a Kubernetes Secret.
func GetTokenCredentials(ctx context.Context, secret *esmeta.SecretKeySelector, store esv1.GenericStore, kube client.Client, namespace string) (TokenCredentials, error) {
	iamToken, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		secret,
	)
	if err != nil {
		return TokenCredentials{}, fmt.Errorf("read token secret %s/%s: %w", namespace, secret.Name, err)
	}
	return TokenCredentials{
		Token: strings.TrimSpace(iamToken),
	}, nil
}

// NewCredentialRequest creates a credential request with the provided cache key and resolver.
func NewCredentialRequest(cacheKey string, resolve func(context.Context) (TokenExchangeCredentials, error)) *CredentialRequest {
	return &CredentialRequest{
		cacheKey: cacheKey,
		resolve:  resolve,
	}
}

var _ TokenExchangeCredentials = &ResolvedServiceAccountCreds{}
var _ TokenExchangeCredentials = &ResolvedFederatedCredentials{}
