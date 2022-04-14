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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client/grpc"
)

const maxSecretsClientLifetime = 5 * time.Minute // supposed SecretsClient lifetime is quite short
const iamTokenCleanupDelay = 1 * time.Hour       // specifies how often cleanUpIamTokenMap() is performed

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

// NewClient constructs a Yandex Lockbox Provider.
func (p *lockboxProvider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.YandexLockbox == nil {
		return nil, fmt.Errorf("received invalid Yandex Lockbox SecretStore resource")
	}
	storeSpecYandexLockbox := storeSpec.Provider.YandexLockbox

	authorizedKeySecretName := storeSpecYandexLockbox.Auth.AuthorizedKey.Name
	if authorizedKeySecretName == "" {
		return nil, fmt.Errorf("invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name")
	}
	objectKey := types.NamespacedName{
		Name:      authorizedKeySecretName,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		if storeSpecYandexLockbox.Auth.AuthorizedKey.Namespace == nil {
			return nil, fmt.Errorf("invalid ClusterSecretStore: missing AuthorizedKey Namespace")
		}
		objectKey.Namespace = *storeSpecYandexLockbox.Auth.AuthorizedKey.Namespace
	}

	authorizedKeySecret := &corev1.Secret{}
	err := kube.Get(ctx, objectKey, authorizedKeySecret)
	if err != nil {
		return nil, fmt.Errorf("could not fetch AuthorizedKey secret: %w", err)
	}

	authorizedKeySecretData := authorizedKeySecret.Data[storeSpecYandexLockbox.Auth.AuthorizedKey.Key]
	if (authorizedKeySecretData == nil) || (len(authorizedKeySecretData) == 0) {
		return nil, fmt.Errorf("missing AuthorizedKey")
	}

	var authorizedKey iamkey.Key
	err = json.Unmarshal(authorizedKeySecretData, &authorizedKey)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal authorized key: %w", err)
	}

	var caCertificateData []byte

	if storeSpecYandexLockbox.CAProvider != nil {
		certObjectKey := types.NamespacedName{
			Name:      storeSpecYandexLockbox.CAProvider.Certificate.Name,
			Namespace: namespace,
		}

		if store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
			if storeSpecYandexLockbox.CAProvider.Certificate.Namespace == nil {
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing CA certificate Namespace")
			}
			certObjectKey.Namespace = *storeSpecYandexLockbox.CAProvider.Certificate.Namespace
		}

		caCertificateSecret := &corev1.Secret{}
		err := kube.Get(ctx, certObjectKey, caCertificateSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch CA certificate secret: %w", err)
		}

		caCertificateData = caCertificateSecret.Data[storeSpecYandexLockbox.CAProvider.Certificate.Key]
		if (caCertificateData == nil) || (len(caCertificateData) == 0) {
			return nil, fmt.Errorf("missing CA Certificate")
		}
	}

	lockboxClient, err := p.getOrCreateLockboxClient(ctx, storeSpecYandexLockbox.APIEndpoint, &authorizedKey, caCertificateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex Lockbox client: %w", err)
	}

	iamToken, err := p.getOrCreateIamToken(ctx, storeSpecYandexLockbox.APIEndpoint, &authorizedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM token: %w", err)
	}

	return &lockboxSecretsClient{lockboxClient, iamToken.Token}, nil
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
	return getValueAsBinary(entry)
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *lockboxSecretsClient) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	entries, err := c.lockboxClient.GetPayloadEntries(ctx, c.iamToken, ref.Key, ref.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to request secret payload to get secret map: %w", err)
	}

	secretMap := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		value, err := getValueAsBinary(entry)
		if err != nil {
			return nil, err
		}
		secretMap[entry.Key] = value
	}
	return secretMap, nil
}

func (c *lockboxSecretsClient) Close(ctx context.Context) error {
	return nil
}

func (c *lockboxSecretsClient) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func getValueAsIs(entry *lockbox.Payload_Entry) (interface{}, error) {
	switch entry.Value.(type) {
	case *lockbox.Payload_Entry_TextValue:
		return entry.GetTextValue(), nil
	case *lockbox.Payload_Entry_BinaryValue:
		return entry.GetBinaryValue(), nil
	default:
		return nil, fmt.Errorf("unsupported payload value type, key: %v", entry.Key)
	}
}

func getValueAsBinary(entry *lockbox.Payload_Entry) ([]byte, error) {
	switch entry.Value.(type) {
	case *lockbox.Payload_Entry_TextValue:
		return []byte(entry.GetTextValue()), nil
	case *lockbox.Payload_Entry_BinaryValue:
		return entry.GetBinaryValue(), nil
	default:
		return nil, fmt.Errorf("unsupported payload value type, key: %v", entry.Key)
	}
}

func findEntryByKey(entries []*lockbox.Payload_Entry, key string) (*lockbox.Payload_Entry, error) {
	for i := range entries {
		if entries[i].Key == key {
			return entries[i], nil
		}
	}
	return nil, fmt.Errorf("payload entry with key '%s' not found", key)
}

func init() {
	lockboxProvider := newLockboxProvider(&grpc.YandexCloudCreator{})

	go func() {
		for {
			time.Sleep(iamTokenCleanupDelay)
			lockboxProvider.cleanUpIamTokenMap()
		}
	}()

	esv1beta1.Register(
		lockboxProvider,
		&esv1beta1.SecretStoreProvider{
			YandexLockbox: &esv1beta1.YandexLockboxProvider{},
		},
	)
}
