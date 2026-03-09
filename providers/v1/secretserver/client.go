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

package secretserver

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

type client struct {
	api secretAPI
}

var _ esv1.SecretsClient = &client{}

// GetSecret supports two types:
//  1. Get the secrets using the secret ID in ref.key i.e. key: 53974
//  2. Get the secret using the secret "name" i.e. key: "secretNameHere"
//     - Secret names must not contain spaces.
//     - If using the secret "name" and multiple secrets are found ...
//     the first secret in the array will be the secret returned.
//  3. get the full secret as json-encoded value
//     by leaving the ref.Property empty.
//  4. get a specific value by using a key from the json formatted secret in Items.0.ItemValue.
//     Nested values are supported by specifying a gjson expression
func (c *client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Return nil if secret contains no fields
	if secret.Fields == nil {
		return nil, nil
	}
	jsonStr, err := json.Marshal(secret)
	if err != nil {
		return nil, err
	}
	// Intentionally fetch and return the full secret as raw JSON when no specific property is provided.
	// This requires calling the API to retrieve the entire secret object.
	if ref.Property == "" {
		return jsonStr, nil
	}

	// extract first "field" i.e. Items.0.ItemValue, data from secret using gjson
	val := gjson.Get(string(jsonStr), "Items.0.ItemValue")
	if val.Exists() && gjson.Valid(val.String()) {
		// extract specific value from data directly above using gjson
		out := gjson.Get(val.String(), ref.Property)
		if out.Exists() {
			return []byte(out.String()), nil
		}
	}

	// More general case Fields is an array in DelineaXPM/tss-sdk-go/v3/server
	// https://github.com/DelineaXPM/tss-sdk-go/blob/571e5674a8103031ad6f873453db27959ec1ca67/server/secret.go#L23
	secretMap := make(map[string]string)
	for index := range secret.Fields {
		secretMap[secret.Fields[index].FieldName] = secret.Fields[index].ItemValue
		secretMap[secret.Fields[index].Slug] = secret.Fields[index].ItemValue
	}

	out, ok := secretMap[ref.Property]
	if !ok {
		return nil, esv1.NoSecretError{}
	}

	return []byte(out), nil
}

// PushSecret not supported at this time.
func (c *client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New("pushing secrets is not supported by Secret Server at this time")
}

// DeleteSecret not supported at this time.
func (c *client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New("deleting secrets is not supported by Secret Server at this time")
}

// SecretExists not supported at this time.
func (c *client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

// Validate not supported at this time.
func (c *client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// GetSecretMap retrieves the secret referenced by ref from the Secret Server API
// and returns it as a map of byte slices.
func (c *client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Ensure secret has fields before indexing into them
	if secret.Fields == nil || len(secret.Fields) == 0 {
		return nil, errors.New("secret contains no fields")
	}

	secretData := make(map[string]any)

	err = json.Unmarshal([]byte(secret.Fields[0].ItemValue), &secretData)
	if err != nil {
		// Do not return the raw error as json.Unmarshal errors may contain
		// sensitive secret data in the error message
		return nil, errors.New("failed to unmarshal secret: invalid JSON format")
	}

	data := make(map[string][]byte)
	for k, v := range secretData {
		data[k], err = esutils.GetByteValue(v)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// GetAllSecrets not supported at this time.
func (c *client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("getting all secrets is not supported by Delinea Secret Server at this time")
}

func (c *client) Close(context.Context) error {
	return nil
}

// getSecret retrieves the secret referenced by ref from the Vault API.
func (c *client) getSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (*server.Secret, error) {
	if ref.Version != "" {
		return nil, errors.New("specifying a version is not supported")
	}

	// If the ref.Key looks like a full path (starts with "/"), fetch by path.
	// Example: "/Folder/Subfolder/SecretName"
	if strings.HasPrefix(ref.Key, "/") {
		s, err := c.api.SecretByPath(ref.Key)
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	// Otherwise try converting it to an ID
	id, err := strconv.Atoi(ref.Key)
	if err != nil {
		s, err := c.api.Secrets(ref.Key, "Name")
		if err != nil {
			return nil, err
		}
		if len(s) == 0 {
			return nil, errors.New("unable to retrieve secret at this time")
		}

		return &s[0], nil
	}
	return c.api.Secret(id)
}
