/*
Copyright © 2025 ESO Maintainer Team

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

// Package infisical implements a provider for retrieving secrets from Infisical.
package infisical

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	infisical "github.com/infisical/go-sdk"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/find"
	"github.com/external-secrets/external-secrets/runtime/metrics"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/constants"
)

var (
	errNotImplemented     = errors.New("not implemented")
	errPropertyNotFound   = "property %s does not exist in secret %s"
	errTagsNotImplemented = errors.New("find by tags not supported")
)

const (
	getSecretsV3     = "GetSecretsV3"
	getSecretByKeyV3 = "GetSecretByKeyV3"
)

func getPropertyValue(jsonData, propertyName, keyName string) ([]byte, error) {
	result := gjson.Get(jsonData, propertyName)
	if !result.Exists() {
		return nil, fmt.Errorf(errPropertyNotFound, propertyName, keyName)
	}
	return []byte(result.Str), nil
}

// getSecretAddress returns the path and key from the given key.
//
// Users can configure a root path, and when a SecretKey is provided with a slash we assume that it is
// within a path appended to the root path.
//
// If the key is not addressing a path at all (i.e. has no `/`), simply return the original
// path and key.
func getSecretAddress(defaultPath, key string) (string, string, error) {
	if !strings.Contains(key, "/") {
		return defaultPath, key, nil
	}

	// Check if `key` starts with a `/`, and throw and error if it does not.
	if !strings.HasPrefix(key, "/") {
		return "", "", fmt.Errorf("a secret key referencing a folder must start with a '/' as it is an absolute path, key: %s", key)
	}

	// Otherwise, take the prefix from `key` and use that as the path. We intentionally discard
	// `defaultPath`.
	lastIndex := strings.LastIndex(key, "/")
	return key[:lastIndex], key[lastIndex+1:], nil
}

// GetSecret retrieves a secret value from Infisical.
// If this returns an error with type NoSecretError then the secret entry will be deleted depending on the
// deletionPolicy.
func (p *Provider) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	path, key, err := getSecretAddress(p.apiScope.SecretPath, ref.Key)
	if err != nil {
		return nil, err
	}

	secret, err := p.sdkClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		Environment:            p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		SecretKey:              key,
		SecretPath:             path,
		IncludeImports:         true,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
	})
	metrics.ObserveAPICall(constants.ProviderName, getSecretByKeyV3, err)

	if err != nil {
		return nil, err
	}

	if ref.Property != "" {
		propertyValue, err := getPropertyValue(secret.SecretValue, ref.Property, ref.Key)
		if err != nil {
			return nil, err
		}

		return propertyValue, nil
	}

	return []byte(secret.SecretValue), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := p.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(secret, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
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

// GetAllSecrets retrieves all secrets matching the given criteria from Infisical.
func (p *Provider) GetAllSecrets(_ context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, errTagsNotImplemented
	}

	secrets, err := p.sdkClient.Secrets().List(infisical.ListSecretsOptions{
		Environment:            p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		SecretPath:             p.apiScope.SecretPath,
		Recursive:              p.apiScope.Recursive,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
		IncludeImports:         true,
	})
	metrics.ObserveAPICall(constants.ProviderName, getSecretsV3, err)
	if err != nil {
		return nil, err
	}

	secretMap := make(map[string][]byte)
	for _, secret := range secrets {
		secretMap[secret.SecretKey] = []byte(secret.SecretValue)
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
	for _, secret := range secrets {
		if (matcher != nil && !matcher.MatchName(secret.SecretKey)) || (ref.Path != nil && !strings.HasPrefix(secret.SecretKey, *ref.Path)) {
			continue
		}
		selected[secret.SecretKey] = []byte(secret.SecretValue)
	}
	return selected, nil
}

// Validate checks if the client is configured correctly.
// and is able to retrieve secrets from the provider.
// If the validation result is unknown it will be ignored.
func (p *Provider) Validate() (esv1.ValidationResult, error) {
	// try to fetch the secrets to ensure provided credentials has access to read secrets
	_, err := p.sdkClient.Secrets().List(infisical.ListSecretsOptions{
		Environment:            p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		Recursive:              p.apiScope.Recursive,
		SecretPath:             p.apiScope.SecretPath,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
	})
	metrics.ObserveAPICall(constants.ProviderName, getSecretsV3, err)

	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("cannot read secrets with provided project scope project:%s environment:%s secret-path:%s recursive:%t, %w", p.apiScope.ProjectSlug, p.apiScope.EnvironmentSlug, p.apiScope.SecretPath, p.apiScope.Recursive, err)
	}

	return esv1.ValidationResultReady, nil
}

// PushSecret will write a single secret into the provider.
// This is not implemented for this provider.
func (p *Provider) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errNotImplemented
}

// DeleteSecret will delete the secret from a provider.
// This is not implemented for this provider.
func (p *Provider) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errNotImplemented
}

// SecretExists checks if a secret is already present in the provider at the given location.
// This is not implemented for this provider.
func (p *Provider) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errNotImplemented
}
