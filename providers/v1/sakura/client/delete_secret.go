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

	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// DeleteSecret will delete the secret from a provider.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	key := remoteRef.GetRemoteKey()
	property := remoteRef.GetProperty()

	exists, err := c.secretKeyExists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if property == "" {
		if err := c.api.Delete(ctx, v1.DeleteSecret{Name: key}); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
		return nil
	}

	existingData, err := c.unveilSecret(ctx, key, "", "")
	if err != nil {
		return err
	}

	kv := make(map[string]json.RawMessage)
	if err := json.Unmarshal(existingData, &kv); err != nil {
		return fmt.Errorf("failed to unmarshal existing secret as JSON: %w", err)
	}

	if _, ok := kv[property]; !ok {
		return nil
	}

	delete(kv, property)

	if len(kv) == 0 {
		if err := c.api.Delete(ctx, v1.DeleteSecret{Name: key}); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}
		return nil
	}

	value, err := json.Marshal(kv)
	if err != nil {
		return fmt.Errorf("failed to marshal merged secret as JSON: %w", err)
	}

	if _, err := c.api.Create(ctx, v1.CreateSecret{
		Name:  key,
		Value: string(value),
	}); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	return nil
}
