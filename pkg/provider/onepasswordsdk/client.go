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

package onepasswordsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Version != "" {
		return nil, errors.New(errVersionNotImplemented)
	}
	if err := checkKeyIsValid(ref.Key); err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}
	secret, err := p.client.Secrets().Resolve(ctx, ref.Key)
	if err != nil {
		return nil, err
	}
	return []byte(secret), nil
}

// Close closes the client connection.
func (p *Provider) Close(_ context.Context) error {
	return nil
}

// DeleteSecret Not Implemented.
func (p *Provider) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf(errOnePasswordSdkStore, errors.New(errNotImplemented))
}

// GetAllSecrets Not Implemented.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errNotImplemented))
}

// GetSecretMap implements v1beta1.SecretsClient.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if ref.Version != "" {
		return nil, errors.New(errVersionNotImplemented)
	}

	// Gets a secret as normal, expecting secret value to be a json object
	data, err := p.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}

	// Maps the json data to a string:string map
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// Converts values in K:V pairs into bytes, while leaving keys as strings
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}

	return secretData, nil
}

// PushSecret Not Implemented.
func (p *Provider) PushSecret(ctx context.Context, secret *v1.Secret, data esv1beta1.PushSecretData) error {
	return fmt.Errorf(errOnePasswordSdkStore, errors.New(errNotImplemented))
}

// SecretExists Not Implemented.
func (p *Provider) SecretExists(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(errOnePasswordSdkStore, errors.New(errNotImplemented))
}

// Validate checks if the client is configured correctly
// currently only checks if it is possible to list vaults.
func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	vaults, err := p.client.Vaults().ListAll(context.Background())
	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf("error listing vaults: %w", err)
	}
	_, err = vaults.Next()
	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf("no vaults found when listing: %w", err)
	}
	return esv1beta1.ValidationResultReady, nil
}

func checkKeyIsValid(key string) error {
	if !strings.HasPrefix(key, "op://") {
		return fmt.Errorf("key must start with op://")
	}

	return nil
}
