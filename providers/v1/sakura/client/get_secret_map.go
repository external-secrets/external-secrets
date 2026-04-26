/*
Copyright © The ESO Authors

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

package client

import (
	"context"
	"encoding/json"
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// GetSecretMap returns multiple k/v pairs from the provider.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.unveilSecret(ctx, ref.Key, ref.Version, ref.Property)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	if err = json.Unmarshal(data, &kv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret %s as JSON: %w", ref.Key, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		if err = json.Unmarshal(v, &strVal); err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}

	return secretData, nil
}
