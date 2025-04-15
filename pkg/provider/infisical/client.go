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
	"strings"

	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
)

var (
	errNotImplemented   = errors.New("not implemented")
	errPropertyNotFound = "property %s does not exist in secret %s"
)

func getPropertyValue(jsonData, propertyName, keyName string) ([]byte, error) {
	result := gjson.Get(jsonData, propertyName)
	if !result.Exists() {
		return nil, fmt.Errorf(errPropertyNotFound, propertyName, keyName)
	}
	return []byte(result.Str), nil
}

// split key in path and key
func splitKey(key string) (path string, name string) {
	key = strings.TrimSuffix(key, "/")

	// Handle empty or root
	if key == "" || key == "/" {
		return "", ""
	}

	// Split path
	lastSlash := strings.LastIndex(key, "/")

	if lastSlash == -1 {
		// No slashes → just a key
		return "", key
	}

	pathPart := key[:lastSlash]
	keyPart := key[lastSlash+1:]

	// Special case: root folder "/"
	if pathPart == "" && strings.HasPrefix(key, "/") {
		return "/", keyPart
	}

	return pathPart, keyPart
}

// Check if all tag filters are included in the metadata.
func tagFilter(secretTags []api.SecretMetadata, tagFilter map[string]string) bool {
	for k, v := range tagFilter {
		tagKeyFound := false
		tagKeyValueValid := false
		for _, sTag := range secretTags {
			if sTag.Key == k {
				tagKeyFound = true
			}
			if sTag.Key == k && sTag.Value == v {
				tagKeyValueValid = true
				break
			}
		}
		// All tags filters should be in the secret
		if !tagKeyFound || (tagKeyFound && !tagKeyValueValid) {
			return false
		}

	}
	return true
}

// if GetSecret returns an error with type NoSecretError.
// then the secret entry will be deleted depending on the deletionPolicy.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {

	path, key := splitKey(ref.Key)

	secretPath := p.apiScope.SecretPath
	// For absolute paths we check that the path is a subpath of the Secret Store Path
	if strings.HasPrefix(path, "/") && strings.HasPrefix(path, p.apiScope.SecretPath) {
		secretPath = path
	}
	// For relative paths, we concat it to the the Secret Store Path
	if !strings.HasPrefix(path, "/") {
		if !strings.HasSuffix(p.apiScope.SecretPath, "/") {
			secretPath = p.apiScope.SecretPath + "/"
		}
		secretPath += path
	}

	secret, err := p.apiClient.GetSecretByKeyV3(api.GetSecretByKeyV3Request{
		EnvironmentSlug:        p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		SecretKey:              key,
		SecretPath:             secretPath,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
	})

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
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

// GetAllSecrets returns multiple k/v pairs from the provider.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {

	secretPath := p.apiScope.SecretPath
	if ref.Path != nil {

		// For absolute paths we check that the path is a subpath of the Secret Store Path
		if strings.HasPrefix(*ref.Path, "/") && strings.HasPrefix(*ref.Path, p.apiScope.SecretPath) {
			secretPath = *ref.Path
		}
		// For relative paths, we concat it to the the Secret Store Path
		if !strings.HasPrefix(*ref.Path, "/") {
			if !strings.HasSuffix(p.apiScope.SecretPath, "/") {
				secretPath = p.apiScope.SecretPath + "/"
			}
			secretPath += *ref.Path
		}
	}

	secrets, err := p.apiClient.GetSecretsV3(api.GetSecretsV3Request{
		EnvironmentSlug:        p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		SecretPath:             secretPath,
		Recursive:              p.apiScope.Recursive,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
	})
	if err != nil {
		return nil, err
	}

	//Without filter
	if ref.Name == nil && ref.Path == nil && ref.Tags == nil {
		secretMap := make(map[string][]byte)
		var i string = "0"
		for _, value := range secrets {
			if secretMap[value.SecretKey] != nil {
				Logger.Info("Duplicate Secret Key:" + value.SecretKey + " in Project " + p.apiScope.ProjectSlug + " with path " + p.apiScope.SecretPath + ". Override value")
				secretMap[value.SecretKey+i] = []byte(value.SecretValue)
				i += "0"
			}
			secretMap[value.SecretKey] = []byte(value.SecretValue)
		}

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

	for _, value := range secrets {
		// if import secrets, their secretPath is empty
		if value.SecretPath == "" {
			if (matcher != nil && !matcher.MatchName(value.SecretKey)) || (ref.Tags != nil && !tagFilter(value.SecretMetadata, ref.Tags)) {
				continue

			}
		} else {
			// we check the SecretPath against the secretPath we use to filter
			if (matcher != nil && !matcher.MatchName(value.SecretKey)) || (ref.Path != nil && !strings.HasPrefix(value.SecretPath, secretPath)) || (ref.Tags != nil && !tagFilter(value.SecretMetadata, ref.Tags)) {
				continue
			}
		}
		if selected[value.SecretKey] != nil {
			Logger.Info("Duplicate Secret Key:" + value.SecretKey + " in Project " + p.apiScope.ProjectSlug + " with path " + p.apiScope.SecretPath + ". Override value")
		}
		selected[value.SecretKey] = []byte(value.SecretValue)
	}
	return selected, nil
}

// Validate checks if the client is configured correctly.
// and is able to retrieve secrets from the provider.
// If the validation result is unknown it will be ignored.
func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	// try to fetch the secrets to ensure provided credentials has access to read secrets
	_, err := p.apiClient.GetSecretsV3(api.GetSecretsV3Request{
		EnvironmentSlug:        p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		Recursive:              p.apiScope.Recursive,
		SecretPath:             p.apiScope.SecretPath,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
	})

	if err != nil {
		return esv1beta1.ValidationResultError, fmt.Errorf("cannot read secrets with provided project scope project:%s environment:%s secret-path:%s recursive:%t, %w", p.apiScope.ProjectSlug, p.apiScope.EnvironmentSlug, p.apiScope.SecretPath, p.apiScope.Recursive, err)
	}

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
