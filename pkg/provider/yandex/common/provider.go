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

package common

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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	clock2 "github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const maxSecretsClientLifetime = 5 * time.Minute // supposed SecretsClient lifetime is quite short

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.Provider = &YandexCloudProvider{}

// Implementation of v1beta1.Provider.
type YandexCloudProvider struct {
	logger              logr.Logger
	clock               clock2.Clock
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

func InitYandexCloudProvider(
	logger logr.Logger,
	clock clock2.Clock,
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

type NewSecretSetterFunc func()
type AdaptInputFunc func(store esv1beta1.GenericStore) (*SecretsClientInput, error)
type NewSecretGetterFunc func(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (SecretGetter, error)
type NewIamTokenFunc func(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (*IamToken, error)

type IamToken struct {
	Token     string
	ExpiresAt time.Time
}

type SecretsClientInput struct {
	APIEndpoint   string
	AuthorizedKey esmeta.SecretKeySelector
	CACertificate *esmeta.SecretKeySelector
}

func (p *YandexCloudProvider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *YandexCloudProvider) Convert(_ esv1beta1.GenericStore) (kclient.Object, error) {
	// Makes default to normal SecretStore approach
	return nil, nil
}

func (p *YandexCloudProvider) ApplyReferent(spec kclient.Object, _ esmeta.ReferentCallOrigin, _ string) (kclient.Object, error) {
	return spec, nil
}
func (p *YandexCloudProvider) NewClientFromObj(_ context.Context, _ kclient.Object, _ kclient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

// NewClient constructs a Yandex.Cloud Provider.
func (p *YandexCloudProvider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	input, err := p.adaptInputFunc(store)
	if err != nil {
		return nil, err
	}

	key, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&input.AuthorizedKey,
	)
	if err != nil {
		return nil, err
	}

	var authorizedKey iamkey.Key
	err = json.Unmarshal([]byte(key), &authorizedKey)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal authorized key: %w", err)
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

	secretGetter, err := p.getOrCreateSecretGetter(ctx, input.APIEndpoint, &authorizedKey, caCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex.Cloud client: %w", err)
	}

	iamToken, err := p.getOrCreateIamToken(ctx, input.APIEndpoint, &authorizedKey, caCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM token: %w", err)
	}

	return &yandexCloudSecretsClient{secretGetter, nil, iamToken.Token}, nil
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
		p.logger.Info("creating IAM token", "authorizedKeyId", authorizedKey.Id)

		iamToken, err := p.newIamTokenFunc(ctx, apiEndpoint, authorizedKey, caCertificate)
		if err != nil {
			return nil, err
		}

		p.logger.Info("created IAM token", "authorizedKeyId", authorizedKey.Id, "expiresAt", iamToken.ExpiresAt)

		p.iamTokenMap[iamTokenKey] = iamToken
	}
	return p.iamTokenMap[iamTokenKey], nil
}

func (p *YandexCloudProvider) isIamTokenUsable(iamToken *IamToken) bool {
	now := p.clock.CurrentTime()
	return now.Add(maxSecretsClientLifetime).Before(iamToken.ExpiresAt)
}

func buildIamTokenKey(authorizedKey *iamkey.Key) iamTokenKey {
	privateKeyHash := sha256.Sum256([]byte(authorizedKey.PrivateKey))
	return iamTokenKey{
		authorizedKey.GetId(),
		authorizedKey.GetServiceAccountId(),
		hex.EncodeToString(privateKeyHash[:]),
	}
}

// Used for testing.
func (p *YandexCloudProvider) IsIamTokenCached(authorizedKey *iamkey.Key) bool {
	p.iamTokenMapMutex.Lock()
	defer p.iamTokenMapMutex.Unlock()

	_, ok := p.iamTokenMap[buildIamTokenKey(authorizedKey)]
	return ok
}

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

func (p *YandexCloudProvider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	_, err := p.adaptInputFunc(store) // adaptInputFunc validates the input store
	if err != nil {
		return nil, err
	}
	return nil, nil
}
