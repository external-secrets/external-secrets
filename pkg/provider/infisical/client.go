/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/

package infisical

import (
	"context"
	"errors"
	"strings"

	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
)

var errNotImplemented = errors.New("not implemented")

// if GetSecret returns an error with type NoSecretError.
// then the secret entry will be deleted depending on the deletionPolicy.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := p.apiClient.GetSecretByKeyV3(api.GetSecretByKeyV3Request{
		EnvironmentSlug: p.apiScope.EnvironmentSlug,
		ProjectSlug:     p.apiScope.ProjectSlug,
		SecretPath:      p.apiScope.SecretPath,
		SecretKey:       ref.Key,
	})

	if err != nil {
		return nil, err
	}

	return []byte(secret), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secrets, err := p.apiClient.GetSecretsV3(api.GetSecretsV3Request{
		EnvironmentSlug: p.apiScope.EnvironmentSlug,
		ProjectSlug:     p.apiScope.ProjectSlug,
		SecretPath:      p.apiScope.SecretPath,
	})
	if err != nil {
		return nil, err
	}

	secretMap := make(map[string][]byte)
	for key, value := range secrets {
		secretMap[key] = []byte(value)
	}
	return secretMap, nil
}

// GetAllSecrets returns multiple k/v pairs from the provider.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	secrets, err := p.apiClient.GetSecretsV3(api.GetSecretsV3Request{
		EnvironmentSlug: p.apiScope.EnvironmentSlug,
		ProjectSlug:     p.apiScope.ProjectSlug,
		SecretPath:      p.apiScope.SecretPath,
	})
	if err != nil {
		return nil, err
	}

	secretMap := make(map[string][]byte)
	for key, value := range secrets {
		secretMap[key] = []byte(value)
	}
	if ref.Name == nil && ref.Path == nil {
		return secretMap, nil
	}

	var matcher *find.Matcher
	if ref.Name != nil {
		m, err := find.New(*ref.Name)
		if err != nil {
			return nil, err
		}
		matcher = m
	}

	selected := map[string][]byte{}
	for key, value := range secrets {
		if (matcher != nil && !matcher.MatchName(key)) || (ref.Path != nil && !strings.HasPrefix(key, *ref.Path)) {
			continue
		}
		selected[key] = []byte(value)
	}
	return selected, nil
}

// Validate checks if the client is configured correctly.
// and is able to retrieve secrets from the provider.
// If the validation result is unknown it will be ignored.
func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// PushSecret will write a single secret into the provider.
func (p *Provider) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	return errNotImplemented
}

// DeleteSecret will delete the secret from a provider.
func (p *Provider) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	return errNotImplemented
}

// SecretExists checks if a secret is already present in the provider at the given location.
func (p *Provider) SecretExists(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errNotImplemented
}

func (p *Provider) Close(ctx context.Context) error {
	return nil
}
