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

package bitwarden

import (
	"context"
	"fmt"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	resp, err := p.bitwardenSdkClient.GetSecret(ctx, ref.Key)
	if err != nil {
		return nil, fmt.Errorf("error getting secret: %w", err)
	}

	return []byte(resp.Value), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetSecretMap() not implemented")
}

// GetAllSecrets gets multiple secrets from the provider and loads into a kubernetes secret.
// First load all secrets from secretStore path configuration
// Then, gets secrets from a matching name or matching custom_metadata.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}
