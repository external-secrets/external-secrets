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

package secretserver

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type client struct {
	api secretAPI
}

var _ esv1beta1.SecretsClient = &client{}

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
func (c *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
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
	// If no property is defined return the full secret as raw json
	if ref.Property == "" {
		return jsonStr, nil
	}
	// extract first "field" i.e. Items.0.ItemValue, data from secret using gjson
	val := gjson.Get(string(jsonStr), "Items.0.ItemValue")
	if !val.Exists() {
		return nil, esv1beta1.NoSecretError{}
	}
	// extract specific value from data directly above using gjson
	out := gjson.Get(val.String(), ref.Property)
	if !out.Exists() {
		return nil, esv1beta1.NoSecretError{}
	}

	return []byte(out.String()), nil
}

// Not supported at this time.
func (c *client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	return errors.New("pushing secrets is not supported by Secret Server at this time")
}

// Not supported at this time.
func (c *client) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	return errors.New("deleting secrets is not supported by Secret Server at this time")
}

// Not supported at this time.
func (c *client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

// Not supported at this time.
func (c *client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (c *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	secretData := make(map[string]any)

	err = json.Unmarshal([]byte(secret.Fields[0].ItemValue), &secretData)
	if err != nil {
		return nil, err
	}

	data := make(map[string][]byte)
	for k, v := range secretData {
		data[k], err = utils.GetByteValue(v)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// Not supported at this time.
func (c *client) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("getting all secrets is not supported by Delinea Secret Server at this time")
}

func (c *client) Close(context.Context) error {
	return nil
}

// getSecret retrieves the secret referenced by ref from the Vault API.
func (c *client) getSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (*server.Secret, error) {
	if ref.Version != "" {
		return nil, errors.New("specifying a version is not supported")
	}
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
