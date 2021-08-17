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
	"encoding/json"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client/grpc"
)

// lockboxProvider is a provider for Yandex Lockbox.
type lockboxProvider struct {
	lockboxClientCreator client.LockboxClientCreator
}

// NewClient constructs a Yandex Lockbox Provider.
func (p *lockboxProvider) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube kclient.Client, namespace string) (provider.SecretsClient, error) {
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
	if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
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

	lb, err := p.lockboxClientCreator.Create(ctx, &authorizedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex.Cloud SDK: %w", err)
	}

	return &lockboxSecretsClient{lb}, nil
}

// lockboxSecretsClient is a secrets client for Yandex Lockbox.
type lockboxSecretsClient struct {
	lockboxClient client.LockboxClient
}

// GetSecret returns a single secret from the provider.
func (p *lockboxSecretsClient) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	entries, err := p.lockboxClient.GetPayloadEntries(ctx, ref.Key, ref.Version)
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
func (p *lockboxSecretsClient) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	entries, err := p.lockboxClient.GetPayloadEntries(ctx, ref.Key, ref.Version)
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

func (p *lockboxSecretsClient) Close(ctx context.Context) error {
	return p.lockboxClient.Close(ctx)
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
	schema.Register(
		&lockboxProvider{
			lockboxClientCreator: &grpc.LockboxClientCreator{},
		},
		&esv1alpha1.SecretStoreProvider{
			YandexLockbox: &esv1alpha1.YandexLockboxProvider{},
		},
	)
}
