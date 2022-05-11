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
package lockbox

import (
	"context"
	"fmt"
	"time"

	"github.com/yandex-cloud/go-sdk/iamkey"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
)

var log = ctrl.Log.WithName("provider").WithName("yandex").WithName("lockbox")

type iamTokenKey struct {
	authorizedKeyID  string
	serviceAccountID string
	privateKeyHash   string
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &lockboxSecretsClient{}
var _ esv1beta1.Provider = &lockboxProvider{}

// lockboxProvider is a provider for Yandex Lockbox.
type lockboxProvider struct {
	yandexCloudCreator client.YandexCloudCreator

	lockboxClientMap      map[string]client.LockboxClient // apiEndpoint -> LockboxClient
	lockboxClientMapMutex sync.Mutex
	iamTokenMap           map[iamTokenKey]*client.IamToken
	iamTokenMapMutex      sync.Mutex
}

func newLockboxProvider(yandexCloudCreator client.YandexCloudCreator) *lockboxProvider {
	return &lockboxProvider{
		yandexCloudCreator: yandexCloudCreator,
		lockboxClientMap:   make(map[string]client.LockboxClient),
		iamTokenMap:        make(map[iamTokenKey]*client.IamToken),
	}
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *lockboxProvider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// NewClient constructs a Yandex Lockbox Provider.
func (p *lockboxProvider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.YandexLockbox == nil {
		return nil, fmt.Errorf("received invalid Yandex Lockbox SecretStore resource")
	}
	storeSpecYandexLockbox := storeSpec.Provider.YandexLockbox

	if storeSpecYandexLockbox.Auth.AuthorizedKey.Name == "" {
		return nil, fmt.Errorf("invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name")
	}

	var caCertificate *esmeta.SecretKeySelector
	if storeSpecYandexLockbox.CAProvider != nil {
		caCertificate = &storeSpecYandexLockbox.CAProvider.Certificate
	}

	return &common.SecretsClientInput{
		APIEndpoint:   storeSpecYandexLockbox.APIEndpoint,
		AuthorizedKey: storeSpecYandexLockbox.Auth.AuthorizedKey,
		CACertificate: caCertificate,
	}, nil
}

func (p *lockboxProvider) getOrCreateLockboxClient(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (client.LockboxClient, error) {
	p.lockboxClientMapMutex.Lock()
	defer p.lockboxClientMapMutex.Unlock()

	if _, ok := p.lockboxClientMap[apiEndpoint]; !ok {
		log.Info("creating LockboxClient", "apiEndpoint", apiEndpoint)

		lockboxClient, err := p.yandexCloudCreator.CreateLockboxClient(ctx, apiEndpoint, authorizedKey, caCertificate)
		if err != nil {
			return nil, err
		}
		p.lockboxClientMap[apiEndpoint] = lockboxClient
	}
	return p.lockboxClientMap[apiEndpoint], nil
}

func (p *lockboxProvider) getOrCreateIamToken(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key) (*client.IamToken, error) {
	p.iamTokenMapMutex.Lock()
	defer p.iamTokenMapMutex.Unlock()

	iamTokenKey := buildIamTokenKey(authorizedKey)
	if iamToken, ok := p.iamTokenMap[iamTokenKey]; !ok || !p.isIamTokenUsable(iamToken) {
		log.Info("creating IAM token", "authorizedKeyId", authorizedKey.Id)

		iamToken, err := p.yandexCloudCreator.CreateIamToken(ctx, apiEndpoint, authorizedKey)
		if err != nil {
			return nil, err
		}

		log.Info("created IAM token", "authorizedKeyId", authorizedKey.Id, "expiresAt", iamToken.ExpiresAt)

		p.iamTokenMap[iamTokenKey] = iamToken
	}
	return p.iamTokenMap[iamTokenKey], nil
}

func (p *lockboxProvider) isIamTokenUsable(iamToken *client.IamToken) bool {
	now := p.yandexCloudCreator.Now()
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
func (p *lockboxProvider) isIamTokenCached(authorizedKey *iamkey.Key) bool {
	p.iamTokenMapMutex.Lock()
	defer p.iamTokenMapMutex.Unlock()

	_, ok := p.iamTokenMap[buildIamTokenKey(authorizedKey)]
	return ok
}

func (p *lockboxProvider) cleanUpIamTokenMap() {
	p.iamTokenMapMutex.Lock()
	defer p.iamTokenMapMutex.Unlock()

	for key, value := range p.iamTokenMap {
		if p.yandexCloudCreator.Now().After(value.ExpiresAt) {
			log.Info("deleting IAM token", "authorizedKeyId", key.authorizedKeyID)
			delete(p.iamTokenMap, key)
		}
	}
}

func (p *lockboxProvider) ValidateStore(store esv1beta1.GenericStore) error {
	return nil
}

// lockboxSecretsClient is a secrets client for Yandex Lockbox.
type lockboxSecretsClient struct {
	lockboxClient client.LockboxClient
	iamToken      string
}

// Not Implemented SetSecret.
func (c *lockboxSecretsClient) SetSecret() error {
	return fmt.Errorf("not implemented")
}

// Empty GetAllSecrets.
func (c *lockboxSecretsClient) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (c *lockboxSecretsClient) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	entries, err := c.lockboxClient.GetPayloadEntries(ctx, c.iamToken, ref.Key, ref.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to request secret payload to get secret: %w", err)
	}

	if ref.Property == "" {
		keyToValue := make(map[string]interface{}, len(entries))
		for _, entry := range entries {
			value, err := getValueAsIs(entry)
			if err != nil {
				return nil, err
			}
			keyToValue[entry.Key] = value
		}
		out, err := json.Marshal(keyToValue)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal secret: %w", err)
		}
		return out, nil
	}

	entry, err := findEntryByKey(entries, ref.Property)
	if err != nil {
		return nil, err
	}
	return newLockboxSecretGetter(lockboxClient)
}

func init() {
	provider := common.InitYandexCloudProvider(
		log,
		clock.NewRealClock(),
		adaptInput,
		newSecretGetter,
		common.NewIamToken,
		time.Hour,
	)

	esv1beta1.Register(
		provider,
		&esv1beta1.SecretStoreProvider{
			YandexLockbox: &esv1beta1.YandexLockboxProvider{},
		},
	)
}
