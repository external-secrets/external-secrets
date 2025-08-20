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

package ngrok

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ngrok/ngrok-api-go/v7"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	defaultDescription = "Managed by External Secrets Operator"
)

var (
	errWriteOnlyOperations     = errors.New("not implemented - the ngrok provider supports write-only operations")
	errVaultDoesNotExist       = errors.New("vault does not exist")
	errVaultSecretDoesNotExist = errors.New("vault secret does not exist")
	errCannotPushNilSecret     = errors.New("cannot push nil secret")
)

type VaultClient interface {
	Create(context.Context, *ngrok.VaultCreate) (*ngrok.Vault, error)
	List(*ngrok.Paging) ngrok.Iter[*ngrok.Vault]
}

type SecretsClient interface {
	Create(context.Context, *ngrok.SecretCreate) (*ngrok.Secret, error)
	Update(context.Context, *ngrok.SecretUpdate) (*ngrok.Secret, error)
	Delete(context.Context, string) error
	List(*ngrok.Paging) ngrok.Iter[*ngrok.Secret]
}

type client struct {
	vaultClient   VaultClient
	secretsClient SecretsClient
	vaultName     string
}

func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if secret == nil {
		return errCannotPushNilSecret
	}

	// First, get the vault by name. If it doesn't exist, create it.
	vault, err := c.getVaultByName(ctx, c.vaultName)
	if err != nil {
		if !errors.Is(err, errVaultDoesNotExist) {
			return err
		}

		// Create the vault if it does not exist
		vault, err = c.vaultClient.Create(ctx, &ngrok.VaultCreate{
			Name:        c.vaultName,
			Description: defaultDescription,
		})
		if err != nil {
			return fmt.Errorf("failed to create vault: %w", err)
		}
	}

	// Prepare the secret data for pushing
	var value []byte

	// If key is specified, get the value from the secret data
	if data.GetSecretKey() != "" {
		var ok bool
		value, ok = secret.Data[data.GetSecretKey()]
		if !ok {
			return fmt.Errorf("key %s not found in secret", data.GetSecretKey())
		}
	} else { // otherwise, marshal the entire secret data as JSON
		value, err = json.Marshal(secret.Data)
		if err != nil {
			return fmt.Errorf("json.Marshal failed with error: %w", err)
		}
	}

	// Check if the secret already exists in the vault
	existingSecret, err := c.getSecretByVaultAndName(ctx, *vault, data.GetRemoteKey())
	if err != nil {
		if !errors.Is(err, errVaultSecretDoesNotExist) {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		// If the secret does not exist, create it
		_, err = c.secretsClient.Create(ctx, &ngrok.SecretCreate{
			VaultID:     vault.ID,
			Name:        data.GetRemoteKey(),
			Value:       string(value),
			Metadata:    data.GetMetadata().String(),
			Description: defaultDescription,
		})
		return err
	}

	// If the secret exists, update it
	_, err = c.secretsClient.Update(ctx, &ngrok.SecretUpdate{
		ID:          existingSecret.ID,
		Value:       ptr.To(string(value)),
		Metadata:    ptr.To(data.GetMetadata().String()),
		Description: ptr.To(string(defaultDescription)),
	})
	return err
}

func (c *client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	// Implementation for checking if a secret exists in ngrok
	secret, err := c.getSecretByVaultNameAndSecretName(ctx, c.vaultName, ref.GetRemoteKey())
	if errors.Is(err, errVaultDoesNotExist) || errors.Is(err, errVaultSecretDoesNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("error fetching secret: %w", err)
	}

	return (secret != nil), nil
}

// DeleteSecret deletes a secret from ngrok by its reference.
func (c *client) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	secret, err := c.getSecretByVaultNameAndSecretName(ctx, c.vaultName, ref.GetRemoteKey())
	if err != nil {
		return err
	}

	if secret == nil {
		return nil
	}

	return c.secretsClient.Delete(ctx, secret.ID)
}

func (c *client) Validate() (esv1.ValidationResult, error) {
	// Validate the client can list secrets with a timeout. If we
	// can list secrets, we assume the client is valid(API keys, URL, etc.)
	iter := c.secretsClient.List(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for iter.Next(ctx) {
		return esv1.ValidationResultReady, nil
	}

	if iter.Err() != nil {
		return esv1.ValidationResultError, fmt.Errorf("store is not allowed to list secrets: %w", iter.Err())
	}

	return esv1.ValidationResultReady, nil
}

func (c *client) GetSecret(ctx context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	// Implementation for getting a secret from ngrok
	return nil, errWriteOnlyOperations
}

func (c *client) GetSecretMap(ctx context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// Implementation for getting a map of secrets from ngrok
	return nil, errWriteOnlyOperations
}

func (c *client) GetAllSecrets(ctx context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	// Implementation for getting all secrets from ngrok
	return nil, errWriteOnlyOperations
}

func (c *client) Close(_ context.Context) error {
	return nil
}

func (c *client) getVaultByName(ctx context.Context, name string) (*ngrok.Vault, error) {
	iter := c.vaultClient.List(nil)
	for iter.Next(ctx) {
		vault := iter.Item()
		if vault.Name == name {
			return vault, nil
		}
	}

	if iter.Err() != nil {
		return nil, iter.Err()
	}

	return nil, errVaultDoesNotExist
}

func (c *client) getSecretByVaultAndName(ctx context.Context, vault ngrok.Vault, name string) (*ngrok.Secret, error) {
	iter := c.secretsClient.List(nil)
	for iter.Next(ctx) {
		secret := iter.Item()
		if secret.Vault.ID != vault.ID {
			continue
		}

		if secret.Name == name {
			return secret, nil
		}
	}

	if iter.Err() != nil {
		return nil, iter.Err()
	}

	return nil, fmt.Errorf("secret '%s' does not exist: %w", name, errVaultSecretDoesNotExist)
}

func (c *client) getSecretByVaultNameAndSecretName(ctx context.Context, vaultName, secretName string) (*ngrok.Secret, error) {
	vault, err := c.getVaultByName(ctx, vaultName)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault '%s' by name: %w", vaultName, err)
	}

	if vault == nil {
		return nil, errVaultDoesNotExist
	}

	return c.getSecretByVaultAndName(ctx, *vault, secretName)
}
