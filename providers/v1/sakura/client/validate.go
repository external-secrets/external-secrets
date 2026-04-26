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
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const validationTimeout = 5 * time.Second

// Validate checks if the client is configured correctly and is able to retrieve secrets from the provider.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), validationTimeout)
	defer cancel()

	if _, err := c.api.List(ctx); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("failed to validate client: %w", err)
	}

	return esv1.ValidationResultReady, nil
}
