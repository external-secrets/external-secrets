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

// Package delinea implements a provider for Delinea DevOps Secrets Vault.
// It provides functionality to interact with secrets stored in Delinea DSV,
// supporting operations like fetching secrets and managing secret lifecycles.
package delinea

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/DelineaXPM/dsv-sdk-go/v2/vault"
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
//  1. get the full secret as json-encoded value
//     by leaving the ref.Property empty.
//  2. get a key from the secret.
//     Nested values are supported by specifying a gjson expression
func (c *client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Return nil if secret value is null
	if secret.Data == nil {
		return nil, nil
	}
	jsonStr, err := json.Marshal(secret.Data)
	if err != nil {
		return nil, err
	}
	// return raw json if no property is defined
	if ref.Property == "" {
		return jsonStr, nil
	}
	// extract key from secret using gjson
	val := gjson.Get(string(jsonStr), ref.Property)
	if !val.Exists() {
		return nil, esv1.NoSecretError{}
	}
	return []byte(val.String()), nil
}

func (c *client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New("pushing secrets is not supported by Delinea DevOps Secrets Vault")
}

func (c *client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New("deleting secrets is not supported by Delinea DevOps Secrets Vault")
}

func (c *client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

func (c *client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// GetSecretMap retrieves all key-value pairs from the secret referenced by ref.
func (c *client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	byteMap := make(map[string][]byte, len(secret.Data))
	for k := range secret.Data {
		byteMap[k], err = esutils.GetByteValueFromMap(secret.Data, k)
		if err != nil {
			return nil, err
		}
	}

	return byteMap, nil
}

// GetAllSecrets lists secrets matching the given criteria and return their latest versions.
func (c *client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("getting all secrets is not supported by Delinea DevOps Secrets Vault")
}

func (c *client) Close(context.Context) error {
	return nil
}

// getSecret retrieves the secret referenced by ref from the Vault API.
func (c *client) getSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (*vault.Secret, error) {
	if ref.Version != "" {
		return nil, errors.New("specifying a version is not yet supported")
	}
	return c.api.Secret(ref.Key)
}
