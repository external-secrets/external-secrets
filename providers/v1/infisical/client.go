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

// Package infisical implements a provider for retrieving secrets from Infisical.
package infisical

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	infisical "github.com/infisical/go-sdk"
	sdkErrors "github.com/infisical/go-sdk/packages/errors"
	"github.com/tidwall/gjson"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/constants"
	"github.com/external-secrets/external-secrets/runtime/find"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

var (
	errPropertyNotFound   = "property %s does not exist in secret %s"
	errTagsNotImplemented = errors.New("find by tags not supported")
)

const (
	getSecretsV3     = "GetSecretsV3"
	getSecretByKeyV3 = "GetSecretByKeyV3"
)

// isNotFoundError reports whether err is an Infisical API error with HTTP 404.
// The go-sdk wraps transport failures in *sdkErrors.APIError, which carries the
// upstream StatusCode.
func isNotFoundError(err error) bool {
	var apiErr *sdkErrors.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func getPropertyValue(jsonData, propertyName, keyName string) ([]byte, error) {
	result := gjson.Get(jsonData, propertyName)
	if !result.Exists() {
		return nil, fmt.Errorf(errPropertyNotFound, propertyName, keyName)
	}
	return []byte(result.Str), nil
}

// getSecretAddress returns the (folder, name) pair to look up in Infisical for the given key.
//
// Resolution rules:
//   - No slash in key: treat key as a bare secret name in defaultPath.
//     ("foo" + defaultPath="/scope")            -> ("/scope", "foo")
//   - Key starts with `/`: treat key as an absolute path; defaultPath is ignored.
//     ("/a/b/foo" + defaultPath="/scope")       -> ("/a/b", "foo")
//   - Otherwise (slash present, no leading `/`): treat key as a folder path relative to defaultPath.
//     ("sub/foo" + defaultPath="/scope")        -> ("/scope/sub", "foo")
func getSecretAddress(defaultPath, key string) (string, string) {
	if !strings.Contains(key, "/") {
		return defaultPath, key
	}

	lastIndex := strings.LastIndex(key, "/")
	folder, name := key[:lastIndex], key[lastIndex+1:]

	if strings.HasPrefix(key, "/") {
		return folder, name
	}

	return path.Join(defaultPath, folder), name
}

// GetSecret retrieves a secret value from Infisical.
// If this returns an error with type NoSecretError then the secret entry will be deleted depending on the
// deletionPolicy.
func (p *Provider) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	path, key := getSecretAddress(p.apiScope.SecretPath, ref.Key)
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
		// Translate a 404 into the NoSecret sentinel so deletionPolicy: Delete
		// can prune the entry and a missing key reports as not-found rather
		// than a generic sync error.
		if isNotFoundError(err) {
			return nil, esv1.NoSecretErr
		}
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
		if (matcher != nil && !matcher.MatchName(secret.SecretKey)) || (ref.Path != nil && !strings.HasPrefix(secret.SecretPath, *ref.Path)) {
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
		return esv1.ValidationResultError, fmt.Errorf(
			"cannot read secrets with provided project scope project:%s environment:%s secret-path:%s recursive:%t, %w",
			p.apiScope.ProjectSlug,
			p.apiScope.EnvironmentSlug,
			p.apiScope.SecretPath,
			p.apiScope.Recursive,
			err,
		)
	}

	return esv1.ValidationResultReady, nil
}
