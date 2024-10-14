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
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
)

var (
	errNotImplemented          = errors.New("not implemented")
	errMissingProperty         = errors.New("missing property")
	errTagsNotImplemented      = errors.New("find by tags not supported")
	errPushWholeNotImplemented = errors.New("push whole secret not implemented")
)

// if GetSecret returns an error with type NoSecretError.
// then the secret entry will be deleted depending on the deletionPolicy.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := p.apiClient.GetSecretByKeyV3(api.GetSecretByKeyV3Request{
		EnvironmentSlug: p.apiScope.EnvironmentSlug,
		ProjectSlug:     p.apiScope.ProjectSlug,
		SecretPath:      ref.Key,
		SecretKey:       ref.Property,
	})

	if err != nil {
		return nil, err
	}

	return []byte(secret), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := p.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(secret, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s/%s: %w", ref.Key, ref.Property, err)
	}
	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		err = json.Unmarshal(v, &strVal)
		if err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}
	return secretData, nil
}

// GetAllSecrets returns multiple k/v pairs from the provider.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, errTagsNotImplemented
	}

	secrets, err := p.apiClient.GetSecretsV3(api.GetSecretsV3Request{
		EnvironmentSlug: p.apiScope.EnvironmentSlug,
		ProjectSlug:     p.apiScope.ProjectSlug,
		SecretPath:      *ref.Path,
	})
	if err != nil {
		return nil, err
	}

	secretMap := make(map[string][]byte)
	for key, value := range secrets {
		secretMap[key] = []byte(value)
	}
	if ref.Name == nil {
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
		if matcher != nil && !matcher.MatchName(key) {
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
	// try to fetch the secrets to ensure provided credentials has access to read secrets
	_, err := p.apiClient.GetSecretsV3(api.GetSecretsV3Request{
		EnvironmentSlug: p.apiScope.EnvironmentSlug,
		ProjectSlug:     p.apiScope.ProjectSlug,
	})

	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf("cannot read secrets with provided project scope project:%s environment:%s, %w", p.apiScope.ProjectSlug, p.apiScope.EnvironmentSlug, err)
	}

	return esv1beta1.ValidationResultReady, nil
}

// PushSecret will write a single secret into the provider.
func (p *Provider) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	val, ok := secret.Data[data.GetSecretKey()]
	if !ok {
		return errPushWholeNotImplemented
	}

	key := data.GetProperty()
	if key == "" {
		return errMissingProperty
	}

	req := api.ChangeSecretV3Request{
		EnvironmentSlug: p.apiScope.EnvironmentSlug,
		ProjectSlug:     p.apiScope.ProjectSlug,
		SecretPath:      data.GetRemoteKey(),
		SecretKey:       key,
		SecretValue:     string(val),
	}

	err := p.apiClient.UpdateSecretV3(req)
	if errors.Is(err, esv1beta1.NoSecretErr) {
		if err = p.apiClient.CreateSecretV3(req); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// DeleteSecret will delete the secret from a provider.
func (p *Provider) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	return errNotImplemented
}

// SecretExists checks if a secret is already present in the provider at the given location.
func (p *Provider) SecretExists(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errNotImplemented
}
