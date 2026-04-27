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

// SecretExists checks if a secret is already present in the provider at the given location.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	key := remoteRef.GetRemoteKey()
	property := remoteRef.GetProperty()

	exists, err := c.secretKeyExists(ctx, key)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if property == "" {
		return true, nil
	}

	data, err := c.unveilSecret(ctx, key, "", "")
	if err != nil {
		return false, err
	}

	kv := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &kv); err != nil {
		return false, fmt.Errorf("failed to unmarshal secret as JSON: %w", err)
	}

	_, ok := kv[property]
	return ok, nil
}
