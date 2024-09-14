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

	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
)

// Implementation of common.SecretGetter.
type lockboxSecretGetter struct {
	lockboxClient client.LockboxClient
}

func newLockboxSecretGetter(lockboxClient client.LockboxClient) (common.SecretGetter, error) {
	return &lockboxSecretGetter{
		lockboxClient: lockboxClient,
	}, nil
}

func (g *lockboxSecretGetter) GetSecret(ctx context.Context, iamToken, resourceID, versionID, property string) ([]byte, error) {
	entries, err := g.lockboxClient.GetPayloadEntries(ctx, iamToken, resourceID, versionID)
	if err != nil {
		return nil, fmt.Errorf("unable to request secret payload to get secret: %w", err)
	}

	if property == "" {
		keyToValue := make(map[string]any, len(entries))
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

	entry, err := findEntryByKey(entries, property)
	if err != nil {
		return nil, err
	}
	return getValueAsBinary(entry)
}

func (g *lockboxSecretGetter) GetSecretMap(ctx context.Context, iamToken, resourceID, versionID string) (map[string][]byte, error) {
	entries, err := g.lockboxClient.GetPayloadEntries(ctx, iamToken, resourceID, versionID)
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

func getValueAsIs(entry *lockbox.Payload_Entry) (any, error) {
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
