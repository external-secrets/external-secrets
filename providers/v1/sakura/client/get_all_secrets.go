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
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/find"
)

// GetAllSecrets returns multiple k/v pairs from the provider
//
//	Only Name filter is supported
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Path != nil {
		return nil, fmt.Errorf("path filter is not supported by the Sakura provider")
	}
	if len(ref.Tags) > 0 {
		return nil, fmt.Errorf("tag filter is not supported by the Sakura provider")
	}

	secrets, err := c.api.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	var matcher *find.Matcher
	if ref.Name != nil {
		m, err := find.New(*ref.Name)
		if err != nil {
			return nil, err
		}

		matcher = m
	}

	secretMap := make(map[string][]byte)
	for _, s := range secrets {
		if matcher != nil && !matcher.MatchName(s.Name) {
			continue
		}

		res, err := c.unveilSecret(ctx, s.Name, "", "")
		if err != nil {
			return nil, err
		}

		secretMap[s.Name] = res
	}

	return secretMap, nil
}
