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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	clock2 "github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
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

// NewClient constructs a Yandex.Cloud Provider.
func (p *YandexCloudProvider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	input, err := p.adaptInputFunc(store)
	if err != nil {
		return nil, err
	}

	objectKey := types.NamespacedName{
		Name:      input.AuthorizedKey.Name,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		if input.AuthorizedKey.Namespace == nil {
			return nil, fmt.Errorf("invalid ClusterSecretStore: missing AuthorizedKey Namespace")
		}
		objectKey.Namespace = *input.AuthorizedKey.Namespace
	}

	authorizedKeySecret := &corev1.Secret{}
	err = kube.Get(ctx, objectKey, authorizedKeySecret)
	if err != nil {
		return nil, fmt.Errorf("could not fetch AuthorizedKey secret: %w", err)
	}

	authorizedKeySecretData := authorizedKeySecret.Data[input.AuthorizedKey.Key]
	if (authorizedKeySecretData == nil) || (len(authorizedKeySecretData) == 0) {
		return nil, fmt.Errorf("missing AuthorizedKey")
	}

	var authorizedKey iamkey.Key
	err = json.Unmarshal(authorizedKeySecretData, &authorizedKey)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal authorized key: %w", err)
	}

	var caCertificateData []byte

	if input.CACertificate != nil {
		certObjectKey := types.NamespacedName{
			Name:      input.CACertificate.Name,
			Namespace: namespace,
		}

		if store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
			if input.CACertificate.Namespace == nil {
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing CA certificate Namespace")
			}
			certObjectKey.Namespace = *input.CACertificate.Namespace
		}

		caCertificateSecret := &corev1.Secret{}
		err := kube.Get(ctx, certObjectKey, caCertificateSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch CA certificate secret: %w", err)
		}

		caCertificateData = caCertificateSecret.Data[input.CACertificate.Key]
		if (caCertificateData == nil) || (len(caCertificateData) == 0) {
			return nil, fmt.Errorf("missing CA Certificate")
		}
	}

	secretGetter, err := p.getOrCreateSecretGetter(ctx, input.APIEndpoint, &authorizedKey, caCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex.Cloud client: %w", err)
	}

	iamToken, err := p.getOrCreateIamToken(ctx, input.APIEndpoint, &authorizedKey, caCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM token: %w", err)
	}

	return &yandexCloudSecretsClient{secretGetter, iamToken.Token}, nil
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

func (p *YandexCloudProvider) ValidateStore(store esv1beta1.GenericStore) error {
	_, err := p.adaptInputFunc(store) // adaptInputFunc validates the input store
	if err != nil {
		return err
	}
	return nil
}
