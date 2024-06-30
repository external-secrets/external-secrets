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

package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/constants"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errReadSecret                   = "cannot read secret data from Vault: %w"
	errDataField                    = "failed to find data field"
	errJSONUnmarshall               = "failed to unmarshall JSON"
	errPathInvalid                  = "provided Path isn't a valid kv v2 path"
	errUnsupportedMetadataKvVersion = "cannot perform metadata fetch operations with kv version v1"
	errNotFound                     = "secret not found"
	errSecretKeyFmt                 = "cannot find secret data for key: %q"
)

// GetSecret supports two types:
//  1. get the full secret as json-encoded value
//     by leaving the ref.Property empty.
//  2. get a key from the secret.
//     Nested values are supported by specifying a gjson expression
func (c *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var data map[string]any
	var err error
	if ref.MetadataPolicy == esv1beta1.ExternalSecretMetadataPolicyFetch {
		if c.store.Version == esv1beta1.VaultKVStoreV1 {
			return nil, errors.New(errUnsupportedMetadataKvVersion)
		}

		metadata, err := c.readSecretMetadata(ctx, ref.Key)
		if err != nil {
			return nil, err
		}
		if len(metadata) == 0 {
			return nil, nil
		}
		data = make(map[string]any, len(metadata))
		for k, v := range metadata {
			data[k] = v
		}
	} else {
		data, err = c.readSecret(ctx, ref.Key, ref.Version)
		if err != nil {
			return nil, err
		}
	}

	return getSecretValue(data, ref.Property)
}

// GetSecretMap supports two modes of operation:
// 1. get the full secret from the vault data payload (by leaving .property empty).
// 2. extract key/value pairs from a (nested) object.
func (c *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	var secretData map[string]any
	err = json.Unmarshal(data, &secretData)
	if err != nil {
		return nil, err
	}
	byteMap := make(map[string][]byte, len(secretData))
	for k := range secretData {
		byteMap[k], err = utils.GetByteValueFromMap(secretData, k)
		if err != nil {
			return nil, err
		}
	}

	return byteMap, nil
}

func (c *client) SecretExists(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) (bool, error) {
	path := c.buildPath(ref.GetRemoteKey())
	data, err := c.readSecret(ctx, path, "")
	if err != nil {
		if errors.Is(err, esv1beta1.NoSecretError{}) {
			return false, nil
		}
		return false, err
	}
	value, err := getSecretValue(data, ref.GetProperty())
	if err != nil {
		if errors.Is(err, esv1beta1.NoSecretError{}) || err.Error() == fmt.Sprintf(errSecretKeyFmt, ref.GetProperty()) {
			return false, nil
		}
		return false, err
	}
	return value != nil, nil
}

func (c *client) readSecret(ctx context.Context, path, version string) (map[string]any, error) {
	dataPath := c.buildPath(path)

	// path formated according to vault docs for v1 and v2 API
	// v1: https://www.vaultproject.io/api-docs/secret/kv/kv-v1#read-secret
	// v2: https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
	var params map[string][]string
	if version != "" {
		params = make(map[string][]string)
		params["version"] = []string{version}
	}
	vaultSecret, err := c.logical.ReadWithDataWithContext(ctx, dataPath, params)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultReadSecretData, err)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	if vaultSecret == nil {
		return nil, esv1beta1.NoSecretError{}
	}
	secretData := vaultSecret.Data
	if c.store.Version == esv1beta1.VaultKVStoreV2 {
		// Vault KV2 has data embedded within sub-field
		// reference - https://www.vaultproject.io/api/secret/kv/kv-v2#read-secret-version
		dataInt, ok := vaultSecret.Data["data"]
		if !ok {
			return nil, errors.New(errDataField)
		}
		if dataInt == nil {
			return nil, esv1beta1.NoSecretError{}
		}
		secretData, ok = dataInt.(map[string]any)
		if !ok {
			return nil, errors.New(errJSONUnmarshall)
		}
	}

	return secretData, nil
}

func getSecretValue(data map[string]any, property string) ([]byte, error) {
	if data == nil {
		return nil, esv1beta1.NoSecretError{}
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	// (1): return raw json if no property is defined
	if property == "" {
		return jsonStr, nil
	}

	// For backwards compatibility we want the
	// actual keys to take precedence over gjson syntax
	// (2): extract key from secret with property
	if _, ok := data[property]; ok {
		return utils.GetByteValueFromMap(data, property)
	}

	// (3): extract key from secret using gjson
	val := gjson.Get(string(jsonStr), property)
	if !val.Exists() {
		return nil, fmt.Errorf(errSecretKeyFmt, property)
	}
	return []byte(val.String()), nil
}

func (c *client) readSecretMetadata(ctx context.Context, path string) (map[string]string, error) {
	metadata := make(map[string]string)
	url, err := c.buildMetadataPath(path)
	if err != nil {
		return nil, err
	}
	secret, err := c.logical.ReadWithDataWithContext(ctx, url, nil)
	metrics.ObserveAPICall(constants.ProviderHCVault, constants.CallHCVaultReadSecretData, err)
	if err != nil {
		return nil, fmt.Errorf(errReadSecret, err)
	}
	if secret == nil {
		return nil, errors.New(errNotFound)
	}
	t, ok := secret.Data["custom_metadata"]
	if !ok {
		return nil, nil
	}
	d, ok := t.(map[string]any)
	if !ok {
		return metadata, nil
	}
	for k, v := range d {
		metadata[k] = v.(string)
	}
	return metadata, nil
}

func (c *client) buildMetadataPath(path string) (string, error) {
	var url string
	if c.store.Version == esv1beta1.VaultKVStoreV1 {
		url = fmt.Sprintf("%s/%s", *c.store.Path, path)
	} else { // KV v2 is used
		if c.store.Path == nil && !strings.Contains(path, "data") {
			return "", fmt.Errorf(errPathInvalid)
		}
		if c.store.Path == nil {
			path = strings.Replace(path, "data", "metadata", 1)
			url = path
		} else {
			url = fmt.Sprintf("%s/metadata/%s", *c.store.Path, path)
		}
	}
	return url, nil
}

/*
	 buildPath is a helper method to build the vault equivalent path
		 from ExternalSecrets and SecretStore manifests. the path build logic
		 varies depending on the SecretStore KV version:
		 Example inputs/outputs:
		 # simple build:
		 kv version == "v2":
			provider_path: "secret/path"
			input: "foo"
			output: "secret/path/data/foo" # provider_path and data are prepended
		 kv version == "v1":
			provider_path: "secret/path"
			input: "foo"
			output: "secret/path/foo" # provider_path is prepended
		 # inheriting paths:
		 kv version == "v2":
			provider_path: "secret/path"
			input: "secret/path/foo"
			output: "secret/path/data/foo" #data is prepended
		 kv version == "v2":
			provider_path: "secret/path"
			input: "secret/path/data/foo"
			output: "secret/path/data/foo" #noop
		 kv version == "v1":
			provider_path: "secret/path"
			input: "secret/path/foo"
			output: "secret/path/foo" #noop
		 # provider path not defined:
		 kv version == "v2":
			provider_path: nil
			input: "secret/path/foo"
			output: "secret/data/path/foo" # data is prepended to secret/
		 kv version == "v2":
			provider_path: nil
			input: "secret/path/data/foo"
			output: "secret/path/data/foo" #noop
		 kv version == "v1":
			provider_path: nil
			input: "secret/path/foo"
			output: "secret/path/foo" #noop
*/
func (c *client) buildPath(path string) string {
	optionalMount := c.store.Path
	out := path
	// if optionalMount is Set, remove it from path if its there
	if optionalMount != nil {
		cut := *optionalMount + "/"
		if strings.HasPrefix(out, cut) {
			// This current logic induces a bug when the actual secret resides on same path names as the mount path.
			_, out, _ = strings.Cut(out, cut)
			// if data succeeds optionalMount on v2 store, we should remove it as well
			if strings.HasPrefix(out, "data/") && c.store.Version == esv1beta1.VaultKVStoreV2 {
				_, out, _ = strings.Cut(out, "data/")
			}
		}
		buildPath := strings.Split(out, "/")
		buildMount := strings.Split(*optionalMount, "/")
		if c.store.Version == esv1beta1.VaultKVStoreV2 {
			buildMount = append(buildMount, "data")
		}
		buildMount = append(buildMount, buildPath...)
		out = strings.Join(buildMount, "/")
		return out
	}
	if !strings.Contains(out, "/data/") && c.store.Version == esv1beta1.VaultKVStoreV2 {
		buildPath := strings.Split(out, "/")
		buildMount := []string{buildPath[0], "data"}
		buildMount = append(buildMount, buildPath[1:]...)
		out = strings.Join(buildMount, "/")
		return out
	}
	return out
}
