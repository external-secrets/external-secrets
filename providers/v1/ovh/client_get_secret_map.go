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

package ovh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const retrieveSecretError = "failed to retrieve secret at path"

// GetSecretMap retrieves a single secret from the provider.
// The created secret will have the same keys as the Secret Manager secret.
// You can specify a key, a property, and a version.
// If a property is provided, it should reference only nested values.
func (cl *ovhClient) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// Retrieve secret from KMS.
	secretDataBytes, _, err := getSecretWithOvhSDK(ctx, cl.okmsClient, cl.okmsID, ref)
	if err != nil && !errors.Is(err, esv1.NoSecretErr) {
		return map[string][]byte{}, fmt.Errorf("%s %q: %w", retrieveSecretError, ref.Key, err)
	} else if err != nil {
		return map[string][]byte{}, err
	}
	if len(secretDataBytes) == 0 {
		return map[string][]byte{}, nil
	}

	// Unmarshal the secret value into a map[string]any
	// so it can be passed to esutils.GetByteValueFromMap.
	var rawSecretDataMap map[string]any
	err = json.Unmarshal(secretDataBytes, &rawSecretDataMap)
	if err != nil {
		return map[string][]byte{}, fmt.Errorf("%s %q: %w", retrieveSecretError, ref.Key, err)
	}

	// Convert the map[string]any into map[string][]byte.
	secretDataMap := make(map[string][]byte, len(rawSecretDataMap))
	for key := range rawSecretDataMap {
		secretDataMap[key], err = esutils.GetByteValueFromMap(rawSecretDataMap, key)
		if err != nil {
			return map[string][]byte{}, fmt.Errorf("%s %q: %w", retrieveSecretError, ref.Key, err)
		}
	}

	return secretDataMap, nil
}
