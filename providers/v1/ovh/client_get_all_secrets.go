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

package ovh

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/ovh/okms-sdk-go/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// GetAllSecrets retrieves multiple secrets from the Secret Manager.
// You can optionally filter secrets by name using a regular expression.
// When path is set to "/" or left empty, the search starts from the Secret Manager root.
func (cl *ovhClient) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	// List Secret Manager secrets.
	secrets, err := getSecretsList(ctx, cl.okmsClient, cl.okmsID, ref.Path)
	if err != nil {
		return map[string][]byte{}, err
	}
	if len(secrets) == 0 {
		return map[string][]byte{}, errors.New("no secrets found in the secret manager")
	}

	// Compile the regular expression defined in ref.Name.RegExp, if present.
	var regex *regexp.Regexp

	if ref.Name != nil {
		regex, err = regexp.Compile(ref.Name.RegExp)
		if err != nil || regex == nil {
			return map[string][]byte{}, errors.New("failed to parse regexp")
		}
	}

	return filterSecretsListWithRegexp(ctx, cl, secrets, regex, ref)
}

// Retrieve secrets located under the specified path.
// If the path is omitted, all secrets from the Secret Manager are returned.
func getSecretsList(ctx context.Context, okmsClient OkmsClient, okmsID uuid.UUID, path *string) ([]string, error) {
	var formatPath string

	// if path ends with '/' (and is not "/"), returns an empty list.
	// Secrets are not supposed to begin with '/'.
	if path == nil || *path == "" {
		formatPath = ""
	} else if len(*path) > 1 &&
		(*path)[len(*path)-1] == '/' &&
		(*path)[len(*path)-2] == '/' {
		return []string{}, nil
	} else {
		formatPath = *path
	}

	// Ensure `formatPath` does not end with '/', otherwise, GetSecretsMetadata
	// will not be able to retrieve secrets as it should.
	if formatPath != "" && formatPath[len(formatPath)-1] == '/' {
		formatPath = formatPath[:len(formatPath)-1]
	}

	return recursivelyGetSecretsList(ctx, okmsClient, okmsID, formatPath)
}

// Recursively traverses the path to retrieve all secrets it contains.
//
// The recursion stops when the for loop finishes iterating over the list
// returned by GetSecretsMetadata, or when an error occurs.
//
// A recursive call is triggered whenever a key ends with '/'.
//
// Example:
// Given the secrets ["secret1", "path/secret", "path/to/secret"] stored in the
// Secret Manager, an initial call to recursivelyGetSecretsList with path="path"
// will cause GetSecretsMetadata to return ["secret", "to/"]
// (see Note below for details on this behavior).
//
// - "secret" is added to the local secret list.
// - "to/" triggers a recursive call with path="path/to".
//
// In the second call, GetSecretsMetadata returns ["secret"], which is added to
// the local list. Since no key ends with '/', the recursion stops and the list
// is returned and merged into the result of the first call.
//
// Note: OVH's SDK GetSecretsMetadata does not return full paths.
// It returns only the next element of the hierarchy, and adds a trailing '/'
// when the element is a directory (i.e., not the last component).
//
// Examples:
//
//	secret1 = "path/to/secret1"
//	secret2 = "path/secret2"
//	secret3 = "path/secrets/secret3"
//
// For the path "path", GetSecretsMetadata returns:
//
//	["to/", "secret2", "secrets/"]
func recursivelyGetSecretsList(ctx context.Context, okmsClient OkmsClient, okmsID uuid.UUID, path string) ([]string, error) {
	var secrets *types.GetMetadataResponse
	var err error

	// Retrieve the list of KMS secrets for the given path.
	// If no path is provided, retrieve all existing secrets from KMS.
	if path != "" && path[0] == '/' {
		return []string{}, nil
	}
	if secrets, err = okmsClient.GetSecretsMetadata(ctx, okmsID, path, true); err != nil {
		return nil, err
	}
	if secrets == nil || secrets.Data == nil || secrets.Data.Keys == nil || len(*secrets.Data.Keys) == 0 {
		return nil, nil
	}

	return secretListLoop(ctx, secrets, okmsClient, okmsID, path)
}

// Loop over each key under 'path'.
// If a key represents a directory (ends with '/')
// and is valid (does not begin with '/' and does not contain successive '/'),
// a recursive call is made.
// Otherwise, the key is a secret and is added to the result list.
func secretListLoop(ctx context.Context, secrets *types.GetMetadataResponse, okmsClient OkmsClient, okmsID uuid.UUID, path string) ([]string, error) {
	var secretsList []string

	for _, key := range *secrets.Data.Keys {
		if key == "" || key[0] == '/' {
			continue
		}

		var toAppend []string
		var err error
		if key[len(key)-1] == '/' {
			toAppend, err = recursivelyGetSecretsList(ctx, okmsClient, okmsID, joinPath(key[:len(key)-1], path))
			if err != nil {
				return nil, err
			}
		} else {
			toAppend = []string{
				joinPath(key, path),
			}
		}
		secretsList = append(secretsList, toAppend...)
	}

	return secretsList, nil
}

func joinPath(key, path string) string {
	if path != "" {
		return path + "/" + key
	}
	return key
}

// Filter the list of secrets using a regular expression.
func filterSecretsListWithRegexp(ctx context.Context, cl *ovhClient, secrets []string, regex *regexp.Regexp, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	secretsDataMap := make(map[string][]byte)
	for _, secret := range secrets {
		// Insert the secret if no regex is provided;
		// otherwise, insert only matching secrets.
		if ref.Name == nil || (regex != nil && regex.MatchString(secret)) {
			secretToInsert, err := cl.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
				Key:                secret,
				ConversionStrategy: ref.ConversionStrategy,
				DecodingStrategy:   ref.DecodingStrategy,
			})
			if err != nil && !errors.Is(err, esv1.NoSecretErr) {
				return map[string][]byte{}, err
			}
			if !errors.Is(err, esv1.NoSecretErr) {
				secretsDataMap[secret] = secretToInsert
			}
		}
	}
	if len(secretsDataMap) == 0 {
		return map[string][]byte{}, errors.New("no secrets could be retrieved")
	}
	return secretsDataMap, nil
}
