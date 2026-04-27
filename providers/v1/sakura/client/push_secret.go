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
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// PushSecret will write a single secret into the provider.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	value, err := esutils.ExtractSecretData(data, secret)
	if err != nil {
		return fmt.Errorf("failed to extract secret data: %w", err)
	}

	key := data.GetRemoteKey()
	property := data.GetProperty()

	if property == "" {
		if _, err := c.api.Create(ctx, v1.CreateSecret{
			Name:  key,
			Value: string(value),
		}); err != nil {
			return fmt.Errorf("failed to create/update secret: %w", err)
		}
		return nil
	}

	kv := make(map[string]json.RawMessage)

	exists, err := c.secretKeyExists(ctx, key)
	if err != nil {
		return err
	}

	if exists {
		existingData, err := c.unveilSecret(ctx, key, "", "")
		if err != nil {
			return err
		}

		if err := json.Unmarshal(existingData, &kv); err != nil {
			return fmt.Errorf("failed to unmarshal existing secret as JSON: %w", err)
		}
	}

	if !json.Valid(value) {
		value, err = json.Marshal(string(value))
		if err != nil {
			return fmt.Errorf("failed to marshal value as JSON string: %w", err)
		}
	}

	kv[property] = value

	value, err = json.Marshal(kv)
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
