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

// Package ngrok provides integration with the ngrok API for secret management
package ngrok

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ngrok/ngrok-api-go/v7"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/utils/ptr"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
)

const (
	defaultDescription = "Managed by External Secrets Operator"
	defaultListTimeout = 1 * time.Minute
)

var (
	errWriteOnlyOperations     = errors.New("not implemented - the ngrok provider only supports write operations")
	errVaultDoesNotExist       = errors.New("vault does not exist")
	errVaultSecretDoesNotExist = errors.New("vault secret does not exist")
)

// PushSecretMetadataSpec defines the structure for metadata used when pushing secrets to ngrok.
type PushSecretMetadataSpec struct {
	// The description of the secret in the ngrok API.
	Description string `json:"description,omitempty"`
	// Custom metadata to be merged with generated metadata for the secret in the ngrok API.
	// This metadata is different from Kubernetes metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// VaultClient defines interface for interactions with ngrok vault API.
type VaultClient interface {
	Create(context.Context, *ngrok.VaultCreate) (*ngrok.Vault, error)
	Get(context.Context, string) (*ngrok.Vault, error)
	GetSecretsByVault(string, *ngrok.Paging) ngrok.Iter[*ngrok.Secret]
	List(*ngrok.Paging) ngrok.Iter[*ngrok.Vault]
}

// SecretsClient defines interface for interactions with ngrok secrets API.
type SecretsClient interface {
	Create(context.Context, *ngrok.SecretCreate) (*ngrok.Secret, error)
	Delete(context.Context, string) error
	Get(context.Context, string) (*ngrok.Secret, error)
	List(*ngrok.Paging) ngrok.Iter[*ngrok.Secret]
	Update(context.Context, *ngrok.SecretUpdate) (*ngrok.Secret, error)
}

type client struct {
	vaultClient   VaultClient
	secretsClient SecretsClient
	vaultName     string
	vaultID       string
	vaultIDMu     sync.RWMutex
}

func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	// First, make sure the vault name still matches the ID we have stored. If not, we have to look it up again.
	err := c.verifyVaultNameStillMatchesID(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify vault name still matches ID: %w", err)
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

	// Calculate the checksum of the value to add to metadata
	valueChecksum := sha256.Sum256(value)

	psmd, err := parseAndDefaultMetadata(data.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}

	psmd.Metadata["_sha256"] = hex.EncodeToString(valueChecksum[:])
	metadataJSON, err := json.Marshal(psmd.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata for ngrok api: %w", err)
	}

	// Check if the secret already exists in the vault
	existingSecret, err := c.getSecretByVaultIDAndName(ctx, c.getVaultID(), data.GetRemoteKey())
	if err != nil {
		if !errors.Is(err, errVaultSecretDoesNotExist) {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		// If the secret does not exist, create it
		_, err = c.secretsClient.Create(ctx, &ngrok.SecretCreate{
			VaultID:     c.getVaultID(),
			Name:        data.GetRemoteKey(),
			Value:       string(value),
			Metadata:    string(metadataJSON),
			Description: psmd.Description,
		})
		return err
	}

	// If the secret exists, update it
	_, err = c.secretsClient.Update(ctx, &ngrok.SecretUpdate{
		ID:          existingSecret.ID,
		Value:       ptr.To(string(value)),
		Metadata:    ptr.To(string(metadataJSON)),
		Description: ptr.To(psmd.Description),
	})
	return err
}

func (c *client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	err := c.verifyVaultNameStillMatchesID(ctx)
	if errors.Is(err, errVaultDoesNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Implementation for checking if a secret exists in ngrok
	secret, err := c.getSecretByVaultIDAndName(ctx, c.getVaultID(), ref.GetRemoteKey())
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
	err := c.verifyVaultNameStillMatchesID(ctx)
	if errors.Is(err, errVaultDoesNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	secret, err := c.getSecretByVaultIDAndName(ctx, c.getVaultID(), ref.GetRemoteKey())
	if errors.Is(err, errVaultDoesNotExist) || errors.Is(err, errVaultSecretDoesNotExist) {
		// If the secret or vault do not exist, we can consider it deleted.
		return nil
	}

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

func (c *client) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	// Implementation for getting a secret from ngrok
	return nil, errWriteOnlyOperations
}

func (c *client) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// Implementation for getting a map of secrets from ngrok
	return nil, errWriteOnlyOperations
}

func (c *client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	// Implementation for getting all secrets from ngrok
	return nil, errWriteOnlyOperations
}

func (c *client) Close(_ context.Context) error {
	return nil
}

func (c *client) verifyVaultNameStillMatchesID(ctx context.Context) error {
	vaultID := c.getVaultID()
	if vaultID == "" {
		return c.refreshVaultID(ctx)
	}

	vault, err := c.vaultClient.Get(ctx, vaultID)
	if err != nil || vault.Name != c.vaultName {
		return c.refreshVaultID(ctx)
	}

	return nil
}

// getVaultID safely retrieves the current vault ID.
func (c *client) getVaultID() string {
	c.vaultIDMu.RLock()
	defer c.vaultIDMu.RUnlock()
	return c.vaultID
}

// setVaultID safely sets the vault ID.
func (c *client) setVaultID(vaultID string) {
	c.vaultIDMu.Lock()
	defer c.vaultIDMu.Unlock()
	c.vaultID = vaultID
}

func (c *client) refreshVaultID(ctx context.Context) error {
	v, err := c.getVaultByName(ctx, c.vaultName)
	if err != nil {
		return fmt.Errorf("failed to refresh vault ID: %w", err)
	}

	c.setVaultID(v.ID)
	return nil
}

func (c *client) getVaultByName(ctx context.Context, name string) (*ngrok.Vault, error) {
	listCtx, cancel := context.WithTimeout(ctx, defaultListTimeout)
	defer cancel()

	iter := c.vaultClient.List(nil)
	for iter.Next(listCtx) {
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

// getSecretByVaultIDAndName retrieves a secret by its vault ID and secret name.
func (c *client) getSecretByVaultIDAndName(ctx context.Context, vaultID, name string) (*ngrok.Secret, error) {
	iter := c.vaultClient.GetSecretsByVault(vaultID, nil)
	for iter.Next(ctx) {
		secret := iter.Item()
		if secret.Name == name {
			return secret, nil
		}
	}

	if iter.Err() != nil {
		return nil, iter.Err()
	}

	return nil, fmt.Errorf("secret '%s' does not exist: %w", name, errVaultSecretDoesNotExist)
}

func parseAndDefaultMetadata(data *v1.JSON) (PushSecretMetadataSpec, error) {
	def := PushSecretMetadataSpec{
		Description: defaultDescription,
		Metadata:    make(map[string]string),
	}

	res, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data)
	if err != nil {
		return def, err
	}

	if res == nil {
		return def, nil
	}

	if res.Spec.Description != "" {
		def.Description = res.Spec.Description
	}

	if res.Spec.Metadata != nil {
		def.Metadata = res.Spec.Metadata
	}

	return def, nil
}
