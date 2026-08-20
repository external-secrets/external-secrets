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

package safeguard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	sg "github.com/OneIdentity/safeguard-go"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	credentialTypePassword    = "password"
	credentialTypePrivateKey  = "privatekey"
	credentialTypeAPIKey      = "apikey"
	accountIDPrefix           = "accountId:"
	accountLookupSeparator    = "/"
)

type secretsClient struct {
	a2a a2aAPI
}

type a2aAPI interface {
	RetrievePassword(ctx context.Context, apiKey sg.Secret) (sg.Secret, error)
	RetrievePrivateKey(ctx context.Context, apiKey sg.Secret, format sg.KeyFormat) (sg.Secret, error)
	RetrieveAPIKey(ctx context.Context, apiKey sg.Secret) ([]sg.APIKey, error)
	SetPassword(ctx context.Context, apiKey sg.Secret, newPassword sg.Secret) error
	GetRetrievableAccounts(ctx context.Context, filter string) ([]sg.A2ARetrievableAccount, error)
	Close() error
}

var _ esv1.SecretsClient = &secretsClient{}

func (c *secretsClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Version != "" {
		return nil, errors.New("specifying a version is not supported")
	}

	apiKey, err := c.resolveAPIKey(ctx, ref.Key)
	if err != nil {
		return nil, err
	}
	defer apiKey.Zero()

	credType, format, subProperty := parseCredentialProperty(ref.Property)
	switch credType {
	case credentialTypePassword:
		secret, err := c.a2a.RetrievePassword(ctx, apiKey)
		if err != nil {
			return nil, mapNotFound(err)
		}
		defer secret.Zero()
		if secret.IsZero() {
			return nil, esv1.NoSecretError{}
		}
		return []byte(secret.ExposeString()), nil
	case credentialTypePrivateKey:
		secret, err := c.a2a.RetrievePrivateKey(ctx, apiKey, format)
		if err != nil {
			return nil, mapNotFound(err)
		}
		defer secret.Zero()
		if secret.IsZero() {
			return nil, esv1.NoSecretError{}
		}
		return []byte(secret.ExposeString()), nil
	case credentialTypeAPIKey:
		keys, err := c.a2a.RetrieveAPIKey(ctx, apiKey)
		if err != nil {
			return nil, mapNotFound(err)
		}
		if len(keys) == 0 {
			return nil, esv1.NoSecretError{}
		}
		if subProperty == "" {
			payload, err := encodeAPIKeys(keys)
			if err != nil {
				return nil, err
			}
			return payload, nil
		}
		value, err := selectAPIKeyValue(keys, subProperty)
		if err != nil {
			return nil, err
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported property %q", ref.Property)
	}
}

func (c *secretsClient) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	value, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	credType, _, subProperty := parseCredentialProperty(ref.Property)
	if credType != credentialTypeAPIKey || subProperty != "" {
		key := ref.Property
		if key == "" {
			key = "value"
		}
		return map[string][]byte{key: value}, nil
	}

	data := make(map[string]any)
	if err := json.Unmarshal(value, &data); err != nil {
		return nil, errors.New("failed to unmarshal api key payload")
	}
	out := make(map[string][]byte, len(data))
	for k := range data {
		out[k], err = esutils.GetByteValueFromMap(data, k)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *secretsClient) GetAllSecrets(context.Context, esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("GetAllSecrets is not supported by the Safeguard provider")
}

func (c *secretsClient) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if data.GetRemoteKey() == "" {
		return errors.New("remote key must be defined")
	}

	value, err := esutils.ExtractSecretData(data, secret)
	if err != nil {
		return fmt.Errorf("failed to extract secret data: %w", err)
	}

	credType, _, _ := parseCredentialProperty(data.GetProperty())
	if credType != "" && credType != credentialTypePassword {
		return fmt.Errorf("push is only supported for password credentials, got %q", data.GetProperty())
	}

	apiKey, err := c.resolveAPIKey(ctx, data.GetRemoteKey())
	if err != nil {
		return err
	}
	defer apiKey.Zero()

	newPassword := sg.NewSecretString(string(value))
	defer newPassword.Zero()

	if err := c.a2a.SetPassword(ctx, apiKey, newPassword); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}
	return nil
}

func (c *secretsClient) DeleteSecret(context.Context, esv1.PushSecretRemoteRef) error {
	return errors.New("DeleteSecret is not supported by the Safeguard provider")
}

func (c *secretsClient) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	apiKey, err := c.resolveAPIKey(ctx, ref.GetRemoteKey())
	if err != nil {
		return false, err
	}
	defer apiKey.Zero()

	secret, err := c.a2a.RetrievePassword(ctx, apiKey)
	if err != nil {
		var notFound *sg.NotFoundError
		if errors.As(mapNotFound(err), &notFound) {
			return false, nil
		}
		return false, err
	}
	defer secret.Zero()
	return !secret.IsZero(), nil
}

func (c *secretsClient) Validate() (esv1.ValidationResult, error) {
	if c.a2a == nil {
		return esv1.ValidationResultError, errors.New("safeguard client is not initialized")
	}
	if _, err := c.a2a.GetRetrievableAccounts(context.Background(), ""); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("failed to validate Safeguard credentials: %w", err)
	}
	return esv1.ValidationResultReady, nil
}

func (c *secretsClient) Close(context.Context) error {
	if c.a2a == nil {
		return nil
	}
	return c.a2a.Close()
}

func (c *secretsClient) resolveAPIKey(ctx context.Context, key string) (sg.Secret, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return sg.Secret{}, errors.New("remote key must not be empty")
	}
	if strings.HasPrefix(key, accountIDPrefix) {
		id, err := strconv.Atoi(strings.TrimPrefix(key, accountIDPrefix))
		if err != nil || id <= 0 {
			return sg.Secret{}, fmt.Errorf("invalid account id key %q", key)
		}
		return c.lookupAPIKeyByAccountID(ctx, id)
	}
	if strings.Contains(key, accountLookupSeparator) {
		return c.lookupAPIKeyByAccountName(ctx, key)
	}
	return sg.NewSecretString(key), nil
}

func (c *secretsClient) lookupAPIKeyByAccountID(ctx context.Context, accountID int) (sg.Secret, error) {
	accounts, err := c.a2a.GetRetrievableAccounts(ctx, "")
	if err != nil {
		return sg.Secret{}, err
	}
	for _, account := range accounts {
		if account.AccountID == accountID {
			return cloneSecret(account.APIKey), nil
		}
	}
	return sg.Secret{}, esv1.NoSecretError{}
}

func (c *secretsClient) lookupAPIKeyByAccountName(ctx context.Context, key string) (sg.Secret, error) {
	parts := strings.SplitN(key, accountLookupSeparator, 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return sg.Secret{}, fmt.Errorf("invalid account lookup key %q, expected accountName/assetName", key)
	}
	accountName := parts[0]
	assetName := parts[1]

	accounts, err := c.a2a.GetRetrievableAccounts(ctx, "")
	if err != nil {
		return sg.Secret{}, err
	}
	var matches []sg.A2ARetrievableAccount
	for _, account := range accounts {
		if account.AccountName == accountName && account.AssetName == assetName {
			matches = append(matches, account)
		}
	}
	switch len(matches) {
	case 0:
		return sg.Secret{}, esv1.NoSecretError{}
	case 1:
		return cloneSecret(matches[0].APIKey), nil
	default:
		return sg.Secret{}, fmt.Errorf("multiple retrievable accounts found for %q", key)
	}
}

func parseCredentialProperty(property string) (credType string, format sg.KeyFormat, subProperty string) {
	property = strings.TrimSpace(property)
	if property == "" {
		return credentialTypePassword, "", ""
	}

	lower := strings.ToLower(property)
	switch {
	case lower == credentialTypePassword:
		return credentialTypePassword, "", ""
	case strings.HasPrefix(lower, credentialTypePrivateKey):
		formatPart := strings.TrimPrefix(lower, credentialTypePrivateKey)
		formatPart = strings.TrimPrefix(formatPart, ".")
		return credentialTypePrivateKey, parseKeyFormat(formatPart), ""
	case strings.HasPrefix(lower, credentialTypeAPIKey):
		subProperty = strings.TrimPrefix(lower, credentialTypeAPIKey)
		subProperty = strings.TrimPrefix(subProperty, ".")
		return credentialTypeAPIKey, "", subProperty
	default:
		return lower, "", ""
	}
}

func parseKeyFormat(format string) sg.KeyFormat {
	switch strings.ToLower(format) {
	case "ssh2":
		return sg.KeyFormatSSH2
	case "putty":
		return sg.KeyFormatPuTTY
	default:
		return sg.KeyFormatOpenSSH
	}
}

func encodeAPIKeys(keys []sg.APIKey) ([]byte, error) {
	type apiKeyPayload struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Description  string `json:"description,omitempty"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret,omitempty"`
	}
	out := make([]apiKeyPayload, len(keys))
	for i, key := range keys {
		out[i] = apiKeyPayload{
			ID:           key.ID,
			Name:         key.Name,
			Description:  key.Description,
			ClientID:     key.ClientID,
			ClientSecret: key.ClientSecret.ExposeString(),
		}
		key.ClientSecret.Zero()
	}
	return json.Marshal(out)
}

func selectAPIKeyValue(keys []sg.APIKey, subProperty string) ([]byte, error) {
	switch strings.ToLower(subProperty) {
	case "clientid", "client_id":
		if len(keys) == 0 {
			return nil, esv1.NoSecretError{}
		}
		return []byte(keys[0].ClientID), nil
	case "clientsecret", "client_secret":
		if len(keys) == 0 {
			return nil, esv1.NoSecretError{}
		}
		value := keys[0].ClientSecret.ExposeString()
		keys[0].ClientSecret.Zero()
		if value == "" {
			return nil, esv1.NoSecretError{}
		}
		return []byte(value), nil
	}
	for _, key := range keys {
		if strings.EqualFold(key.Name, subProperty) {
			value := key.ClientSecret.ExposeString()
			key.ClientSecret.Zero()
			if value == "" {
				return nil, esv1.NoSecretError{}
			}
			return []byte(value), nil
		}
	}
	return nil, esv1.NoSecretError{}
}

func cloneSecret(secret sg.Secret) sg.Secret {
	return sg.NewSecretString(secret.ExposeString())
}

func mapNotFound(err error) error {
	var notFound *sg.NotFoundError
	if errors.As(err, &notFound) {
		return esv1.NoSecretError{}
	}
	return err
}
