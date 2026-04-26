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
	"strconv"

	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
)

// unveilSecret retrieves the secret value.
func (c *Client) unveilSecret(ctx context.Context, key, version, property string) ([]byte, error) {
	versionOpt := v1.OptNilInt{}
	if version != "" {
		versionInt, err := strconv.Atoi(version)
		if err != nil {
			return nil, fmt.Errorf("invalid version: %w", err)
		}

		versionOpt = v1.NewOptNilInt(versionInt)
	}

	res, err := c.api.Unveil(ctx, v1.Unveil{
		Name:    key,
		Version: versionOpt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unveil secret with key %q: %w", key, err)
	}

	data := []byte(res.GetValue())
	if property == "" {
		return data, nil
	}

	kv := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &kv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret with key %q as JSON: %w", key, err)
	}

	value, ok := kv[property]
	if !ok {
		return nil, fmt.Errorf("property %q not found in secret %q", property, key)
	}

	var strVal string
	if err := json.Unmarshal(value, &strVal); err == nil {
		return []byte(strVal), nil
	}

	return value, nil
}

// secretKeyExists checks if a secret with the given key exists.
func (c *Client) secretKeyExists(ctx context.Context, key string) (bool, error) {
	secrets, err := c.api.List(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list secrets: %w", err)
	}

	for _, s := range secrets {
		if s.Name == key {
			return true, nil
		}
	}

	return false, nil
}
