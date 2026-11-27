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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	infisical "github.com/infisical/go-sdk"
	sdkErrors "github.com/infisical/go-sdk/packages/errors"
	"github.com/infisical/go-sdk/packages/models"
	"github.com/tidwall/gjson"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/constants"
	"github.com/external-secrets/external-secrets/runtime/find"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

var errPropertyNotFound = "property %s does not exist in secret %s"

const (
	getSecretsV3     = "GetSecretsV3"
	getSecretByKeyV3 = "GetSecretByKeyV3"

	// findTagsKey is the find.tags key whose comma-separated value lists the
	// Infisical tag slugs a secret must carry.
	findTagsKey = "infisical.com/tags"
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
	var matcher *find.Matcher
	if ref.Name != nil {
		var err error
		if matcher, err = find.New(*ref.Name); err != nil {
			return nil, err
		}
	}
	tagSlugs, metadata, err := parseFindTags(ref.Tags)
	if err != nil {
		return nil, err
	}
	// A recursive list keeps one secret per key unless told otherwise. Copies
	// of a key in different folders can carry different tags and metadata, so a
	// filtered search must see every copy before it selects one.
	filtered := len(tagSlugs) > 0 || len(metadata) > 0

	// A find path says where to look, not how far: whether subfolders are read
	// stays with the store's recursive setting. An empty one is not a path, and
	// passing it on would ask the SDK for the whole project.
	secretPath := p.apiScope.SecretPath
	if ref.Path != nil && *ref.Path != "" {
		secretPath = *ref.Path
	}

	secrets, err := p.sdkClient.Secrets().List(infisical.ListSecretsOptions{
		Environment:            p.apiScope.EnvironmentSlug,
		ProjectSlug:            p.apiScope.ProjectSlug,
		SecretPath:             secretPath,
		Recursive:              p.apiScope.Recursive,
		ExpandSecretReferences: p.apiScope.ExpandSecretReferences,
		IncludeImports:         true,
		SkipUniqueValidation:   filtered,
	})
	metrics.ObserveAPICall(constants.ProviderName, getSecretsV3, err)
	if err != nil {
		return nil, err
	}

	selected := make(map[string][]byte, len(secrets))
	selectedPaths := make(map[string]string, len(secrets))
	for _, secret := range secrets {
		if matcher != nil && !matcher.MatchName(secret.SecretKey) {
			continue
		}
		if !hasTags(secret, tagSlugs) || !hasMetadata(secret, metadata) {
			continue
		}
		// The list order of copies is not stable, so picking one would be
		// arbitrary. Copies with the same value are not ambiguous.
		value := []byte(secret.SecretValue)
		if prev, ok := selected[secret.SecretKey]; ok && !bytes.Equal(prev, value) {
			return nil, fmt.Errorf("secret %s matches in folders %s and %s with different values", secret.SecretKey, selectedPaths[secret.SecretKey], secret.SecretPath)
		}
		selected[secret.SecretKey] = value
		selectedPaths[secret.SecretKey] = secret.SecretPath
	}
	return selected, nil
}

// parseFindTags splits find.tags into the tag slugs listed under findTagsKey
// and the remaining entries, which match secret metadata. A findTagsKey value
// with no slugs is an error, so it cannot silently turn off tag filtering.
func parseFindTags(tags map[string]string) ([]string, map[string]string, error) {
	var slugs []string
	metadata := make(map[string]string, len(tags))
	for k, v := range tags {
		if k != findTagsKey {
			metadata[k] = v
			continue
		}
		for slug := range strings.SplitSeq(v, ",") {
			if slug = strings.TrimSpace(slug); slug != "" {
				slugs = append(slugs, slug)
			}
		}
		if len(slugs) == 0 {
			return nil, nil, fmt.Errorf("find.tags %s lists no tag slugs", findTagsKey)
		}
	}
	return slugs, metadata, nil
}

func hasTags(secret models.Secret, slugs []string) bool {
	for _, slug := range slugs {
		if !slices.ContainsFunc(secret.Tags, func(t models.SecretTag) bool { return t.Slug == slug }) {
			return false
		}
	}
	return true
}

func hasMetadata(secret models.Secret, metadata map[string]string) bool {
	for k, v := range metadata {
		if !slices.Contains(secret.SecretMetadata, models.SecretMetadata{Key: k, Value: v}) {
			return false
		}
	}
	return true
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
