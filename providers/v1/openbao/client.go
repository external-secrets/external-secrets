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

package openbao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/find"
	"github.com/openbao/openbao/api/v2"
	v1 "k8s.io/api/core/v1"
)

var (
	_ esv1.SecretsClient = &Client{}
)

type Client struct {
	path   string
	useV1  bool
	client *api.Client
}

func (c *Client) Close(ctx context.Context) error {
	c.client = nil
	return nil
}

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	return errors.New("delete secret is not supported (the OpenBao provider is currently read only)")
}

func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, errors.New("tag based search is not implemented")
	}

	listPath := ""
	if ref.Path != nil {
		listPath = *ref.Path
	}

	if c.useV1 {
		listPath = path.Join(c.path, listPath)
	} else {
		listPath = path.Join(c.path, "metadata", listPath)
	}

	meta, err := c.client.Logical().ListWithContext(ctx, listPath) // TODO(@phil9909): raise PR against OpenBao: client.KVv1/2() should have a list method
	if err != nil {
		return nil, err
	}

	if meta == nil {
		return nil, nil
	}
	if _, ok := meta.Data["keys"]; !ok {
		return nil, nil
	}

	keysUntyped := meta.Data["keys"].([]any)
	keys := make([]string, 0, len(keysUntyped))
	for _, key := range keysUntyped {
		keys = append(keys, key.(string))
	}

	return c.findSecretsFromName(ctx, keys, *ref.Name)
}

func (c *Client) findSecretsFromName(ctx context.Context, candidates []string, ref esv1.FindName) (map[string][]byte, error) {
	secrets := make(map[string][]byte)
	matcher, err := find.New(ref)
	if err != nil {
		return nil, err
	}
	for _, name := range candidates {
		ok := matcher.MatchName(name)
		if ok {
			secret, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: name})
			if errors.Is(err, esv1.NoSecretError{}) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if secret != nil {
				secrets[name] = secret
			}
		}
	}
	return secrets, nil
}

func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var data *api.KVSecret
	var err error

	if c.useV1 {
		if ref.Version != "" {
			return nil, errors.New("OpenBao KVv1 secrets do not support versioning (use KVv2)") // TODO: improve
		}

		kv := c.client.KVv1(c.path)
		data, err = kv.Get(ctx, ref.Key)
		if err != nil {
			return nil, err
		}
	} else {
		kv := c.client.KVv2(c.path)
		if ref.Version != "" {
			version, err := strconv.Atoi(ref.Version)
			if err != nil {
				return nil, err // TODO: proper error
			}

			data, err = kv.GetVersion(ctx, ref.Key, version)
			if err != nil {
				return nil, err
			}
		} else {
			data, err = kv.Get(ctx, ref.Key)
			if err != nil {
				return nil, err
			}
		}
	}

	if ref.Property == "" {
		return json.Marshal(data.Data)
	}

	property, ok := data.Data[ref.Property]
	if !ok {
		return nil, esv1.NoSecretErr // TODO: improve and test
	}

	if property, ok := property.(string); ok {
		return []byte(property), nil
	} else {
		return nil, esv1.NoSecretErr // TODO: improve and test
	}

}

func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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
		byteMap[k], err = esutils.GetByteValueFromMap(secretData, k)
		if err != nil {
			return nil, err
		}
	}

	return byteMap, nil
}

func (c *Client) PushSecret(ctx context.Context, secret *v1.Secret, data esv1.PushSecretData) error {
	return errors.New("push secret is not supported (the OpenBao provider is currently read only)")
}

func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

func (c *Client) Validate() (esv1.ValidationResult, error) {
	mount, err := c.client.Sys().MountInfo(c.path)
	if err != nil {
		return esv1.ValidationResultError, err
	}

	if mount.Type != "kv" {
		return esv1.ValidationResultError, fmt.Errorf(`expected mount type "kv" found %q`, mount.Type)
	}

	actualVersion := mount.Options["version"]
	expectedVersion := "2"
	if c.useV1 {
		expectedVersion = "1"
	}
	if expectedVersion != actualVersion {
		return esv1.ValidationResultError, fmt.Errorf(`expected kv engine version %s found version %s`, expectedVersion, actualVersion)
	}

	return esv1.ValidationResultReady, nil
}
