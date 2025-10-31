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

// Package ydxcommon contains shared functionality for Yandex.Cloud providers.
package ydxcommon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/yandex-cloud/go-sdk/iamkey"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/providers/v1/yandex/common/clock"
)

const maxSecretsClientLifetime = 5 * time.Minute // supposed SecretsClient lifetime is quite short

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.Provider = &YandexCloudProvider{}

// YandexCloudProvider implements the Provider interface for Yandex.Cloud services.
type YandexCloudProvider struct {
	logger              logr.Logger
	clock               clock.Clock
	adaptInputFunc      AdaptInputFunc
	newSecretGetterFunc NewSecretGetterFunc
	newIamTokenFunc     NewIamTokenFunc

	secretGetteMap       map[string]SecretGetter // apiEndpoint -> SecretGetter
	secretGetterMapMutex sync.Mutex
	iamTokenMap          map[iamTokenKey]*IamToken
	iamTokenMapMutex     sync.Mutex
}

type iamTokenKey struct {
	authorizedKeyID  string
	serviceAccountID string
	privateKeyHash   string
}

// InitYandexCloudProvider creates and initializes a new YandexCloudProvider instance.
func InitYandexCloudProvider(
	logger logr.Logger,
	clock clock.Clock,
	adaptInputFunc AdaptInputFunc,
	newSecretGetterFunc NewSecretGetterFunc,
	newIamTokenFunc NewIamTokenFunc,
	iamTokenCleanupDelay time.Duration,
) *YandexCloudProvider {
	provider := &YandexCloudProvider{
		logger:              logger,
		clock:               clock,
		adaptInputFunc:      adaptInputFunc,
		newSecretGetterFunc: newSecretGetterFunc,
		newIamTokenFunc:     newIamTokenFunc,
		secretGetteMap:      make(map[string]SecretGetter),
		iamTokenMap:         make(map[iamTokenKey]*IamToken),
	}

	if iamTokenCleanupDelay > 0 {
		go func() {
			for {
				time.Sleep(iamTokenCleanupDelay)
				provider.CleanUpIamTokenMap()
			}
		}()
	}

	return provider
}

// NewSecretSetterFunc defines a function type to create a new secret setter.
type NewSecretSetterFunc func()

// AdaptInputFunc defines a function type to adapt generic store to client input.
type AdaptInputFunc func(store esv1.GenericStore) (*SecretsClientInput, error)

// NewSecretGetterFunc defines a function type to create a new secret getter.
type NewSecretGetterFunc func(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (SecretGetter, error)

// NewIamTokenFunc defines a function type to create a new IAM token.
type NewIamTokenFunc func(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (*IamToken, error)

// IamToken represents an authentication token for Yandex Cloud API.
type IamToken struct {
	Token     string
	ExpiresAt time.Time
}

// SecretsClientInput contains the input parameters for creating a Yandex Cloud secrets client.
type SecretsClientInput struct {
	APIEndpoint     string
	AuthorizedKey   *esmeta.SecretKeySelector
	CACertificate   *esmeta.SecretKeySelector
	ResourceKeyType ResourceKeyType
	FolderID        string
}

// ResourceKeyType defines how the resource key should be interpreted.
type ResourceKeyType int

const (
	// ResourceKeyTypeID indicates the resource key is an ID.
	ResourceKeyTypeID ResourceKeyType = iota
	// ResourceKeyTypeName indicates the resource key is a name.
	ResourceKeyTypeName ResourceKeyType = iota
)

// Capabilities returns the esv1.SecretStoreCapabilities of the Yandex.Cloud provider.
func (p *YandexCloudProvider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient constructs a Yandex.Cloud Provider.
func (p *YandexCloudProvider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	input, err := p.adaptInputFunc(store)
	if err != nil {
		return nil, err
	}

	var authorizedKey *iamkey.Key
	if input.AuthorizedKey != nil {
		key, err := resolvers.SecretKeyRef(
			ctx,
			kube,
			store.GetKind(),
			namespace,
			input.AuthorizedKey,
		)
		if err != nil {
			return nil, err
		}

		authorizedKey = &iamkey.Key{}
		err = json.Unmarshal([]byte(key), authorizedKey)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal authorized key: %w", err)
		}
	}

	var caCertificateData []byte
	if input.CACertificate != nil {
		caCert, err := resolvers.SecretKeyRef(
			ctx,
			kube,
			store.GetKind(),
			namespace,
			input.CACertificate,
		)
		if err != nil {
			return nil, err
		}
		caCertificateData = []byte(caCert)
	}

	secretGetter, err := p.getOrCreateSecretGetter(ctx, input.APIEndpoint, authorizedKey, caCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex.Cloud client: %w", err)
	}

	iamToken, err := p.getOrCreateIamToken(ctx, input.APIEndpoint, authorizedKey, caCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM token: %w", err)
	}

	return &yandexCloudSecretsClient{secretGetter, nil, iamToken.Token, input.ResourceKeyType, input.FolderID}, nil
}

func (p *YandexCloudProvider) getOrCreateSecretGetter(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (SecretGetter, error) {
	p.secretGetterMapMutex.Lock()
	defer p.secretGetterMapMutex.Unlock()

	if _, ok := p.secretGetteMap[apiEndpoint]; !ok {
		p.logger.Info("creating SecretGetter", "apiEndpoint", apiEndpoint)
		secretGetter, err := p.newSecretGetterFunc(ctx, apiEndpoint, authorizedKey, caCertificate)
		if err != nil {
			return nil, err
		}
		p.secretGetteMap[apiEndpoint] = secretGetter
	}
	return p.secretGetteMap[apiEndpoint], nil
}

func (p *YandexCloudProvider) getOrCreateIamToken(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (*IamToken, error) {
	p.iamTokenMapMutex.Lock()
	defer p.iamTokenMapMutex.Unlock()

	iamTokenKey := buildIamTokenKey(authorizedKey)
	if iamToken, ok := p.iamTokenMap[iamTokenKey]; !ok || !p.isIamTokenUsable(iamToken) {
		if authorizedKey != nil {
			p.logger.Info("creating IAM token", "authorizedKeyId", authorizedKey.Id)
		} else {
			p.logger.Info("creating instance SA IAM token")
		}

		iamToken, err := p.newIamTokenFunc(ctx, apiEndpoint, authorizedKey, caCertificate)
		if err != nil {
			return nil, err
		}

		if authorizedKey != nil {
			p.logger.Info("created IAM token", "authorizedKeyId", authorizedKey.Id, "expiresAt", iamToken.ExpiresAt)
		} else {
			p.logger.Info("created instance SA IAM token", "expiresAt", iamToken.ExpiresAt)
		}

		p.iamTokenMap[iamTokenKey] = iamToken
	}
	return p.iamTokenMap[iamTokenKey], nil
}

func (p *YandexCloudProvider) isIamTokenUsable(iamToken *IamToken) bool {
	now := p.clock.CurrentTime()
	return now.Add(maxSecretsClientLifetime).Before(iamToken.ExpiresAt)
}

func buildIamTokenKey(authorizedKey *iamkey.Key) iamTokenKey {
	if authorizedKey == nil {
		return iamTokenKey{}
	}

	privateKeyHash := sha256.Sum256([]byte(authorizedKey.PrivateKey))
	return iamTokenKey{
		authorizedKey.GetId(),
		authorizedKey.GetServiceAccountId(),
		hex.EncodeToString(privateKeyHash[:]),
	}
}

// IsIamTokenCached checks if the IAM token for the given authorized key is cached.
// Used for testing purposes.
func (p *YandexCloudProvider) IsIamTokenCached(authorizedKey *iamkey.Key) bool {
	p.iamTokenMapMutex.Lock()
	defer p.iamTokenMapMutex.Unlock()

	_, ok := p.iamTokenMap[buildIamTokenKey(authorizedKey)]
	return ok
}

// CleanUpIamTokenMap removes expired IAM tokens from the cache.
func (p *YandexCloudProvider) CleanUpIamTokenMap() {
	p.iamTokenMapMutex.Lock()
	defer p.iamTokenMapMutex.Unlock()

	for key, value := range p.iamTokenMap {
		if p.clock.CurrentTime().After(value.ExpiresAt) {
			p.logger.Info("deleting IAM token", "authorizedKeyId", key.authorizedKeyID)
			delete(p.iamTokenMap, key)
		}
	}
}

// ValidateStore validates the provider-specific configuration in the SecretStore resource.
func (p *YandexCloudProvider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	_, err := p.adaptInputFunc(store) // adaptInputFunc validates the input store
	if err != nil {
		return nil, err
	}
	return nil, nil
}
