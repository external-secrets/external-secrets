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

package dvls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Devolutions/go-dvls"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errNotImplemented         = "not implemented"
	errFailedToGetEntry       = "failed to get entry: %w"
	defaultDVLSRequestTimeout = 15 * time.Second
)

var _ esv1.SecretsClient = &Client{}

// Client implements the external-secrets SecretsClient interface for DVLS.
type Client struct {
	dvls    credentialClient
	timeout time.Duration
}

type credentialClient interface {
	GetByID(vaultID, entryID string) (dvls.Entry, error)
	Update(entry dvls.Entry) (dvls.Entry, error)
	DeleteByID(vaultID, entryID string) error
}

type realCredentialClient struct {
	cred *dvls.EntryCredentialService
}

func (r *realCredentialClient) GetByID(vaultID, entryID string) (dvls.Entry, error) {
	return r.cred.GetById(vaultID, entryID)
}

func (r *realCredentialClient) Update(entry dvls.Entry) (dvls.Entry, error) {
	return r.cred.Update(entry)
}

func (r *realCredentialClient) DeleteByID(vaultID, entryID string) error {
	return r.cred.DeleteById(vaultID, entryID)
}

// NewClient creates a new DVLS secrets client.
func NewClient(dvlsClient credentialClient) *Client {
	return &Client{
		dvls:    dvlsClient,
		timeout: defaultDVLSRequestTimeout,
	}
}

// GetSecret retrieves a secret from DVLS.
// The key format is: "<vault-id>/<entry-id>" for UUID-based lookup.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	vaultID, entryID, err := c.parseSecretRef(ref.Key)
	if err != nil {
		return nil, err
	}

	entry, err := c.getEntryWithTimeout(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetEntry, err)
	}

	// Extract the requested property from the entry
	secretMap, err := c.entryToSecretMap(entry)
	if err != nil {
		return nil, err
	}

	// If a specific property is requested, return just that value
	if ref.Property != "" {
		value, ok := secretMap[ref.Property]
		if !ok {
			return nil, fmt.Errorf("property %q not found in entry", ref.Property)
		}
		return value, nil
	}

	// Return the entire secret as JSON
	return json.Marshal(secretMap)
}

// GetSecretMap retrieves a secret and returns all its fields as a map.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	vaultID, entryID, err := c.parseSecretRef(ref.Key)
	if err != nil {
		return nil, err
	}

	entry, err := c.getEntryWithTimeout(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return nil, esv1.NoSecretErr
	}
	if err != nil {
		return nil, fmt.Errorf(errFailedToGetEntry, err)
	}

	return c.entryToSecretMap(entry)
}

// GetAllSecrets is not implemented for DVLS.
func (c *Client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// PushSecret creates or updates a secret in DVLS.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	vaultID, entryID, err := c.parseSecretRef(data.GetRemoteKey())
	if err != nil {
		return err
	}

	value, err := extractPushValue(secret, data)
	if err != nil {
		return err
	}

	existingEntry, err := c.getEntryWithTimeout(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return fmt.Errorf("entry %s not found in vault %s: %w", entryID, vaultID, err)
	}
	if err != nil {
		return fmt.Errorf(errFailedToGetEntry, err)
	}

	return c.updateEntryWithTimeout(ctx, existingEntry, value)
}

// DeleteSecret deletes a secret from DVLS.
func (c *Client) DeleteSecret(_ context.Context, ref esv1.PushSecretRemoteRef) error {
	vaultID, entryID, err := c.parseSecretRef(ref.GetRemoteKey())
	if err != nil {
		return err
	}

	return c.dvls.DeleteByID(vaultID, entryID)
}

// SecretExists checks if a secret exists in DVLS.
func (c *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	vaultID, entryID, err := c.parseSecretRef(ref.GetRemoteKey())
	if err != nil {
		return false, err
	}

	_, err = c.getEntryWithTimeout(ctx, vaultID, entryID)
	if isNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// Validate checks if the client is properly configured.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	if c.dvls == nil {
		return esv1.ValidationResultError, errors.New("DVLS client is not initialized")
	}
	return esv1.ValidationResultReady, nil
}

// Close is a no-op for the DVLS client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// parseSecretRef parses the secret reference key.
// Format: "<vault-id>/<entry-id>".
func (c *Client) parseSecretRef(key string) (vaultID, entryID string, err error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid key format: expected '<vault-id>/<entry-id>', got %q", key)
	}

	vaultID = strings.TrimSpace(parts[0])
	entryID = strings.TrimSpace(parts[1])

	if vaultID == "" {
		return "", "", errors.New("vault ID cannot be empty")
	}
	if entryID == "" {
		return "", "", errors.New("entry ID cannot be empty")
	}

	return vaultID, entryID, nil
}

// entryToSecretMap converts a DVLS entry to a map of secret values.
func (c *Client) entryToSecretMap(entry dvls.Entry) (map[string][]byte, error) {
	secretMap, err := entry.ToCredentialMap()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(secretMap))
	for k, v := range secretMap {
		result[k] = []byte(v)
	}

	return result, nil
}

func extractPushValue(secret *corev1.Secret, data esv1.PushSecretData) ([]byte, error) {
	if data.GetSecretKey() == "" {
		return nil, fmt.Errorf("secretKey is required for DVLS push")
	}

	value, ok := secret.Data[data.GetSecretKey()]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret %q", data.GetSecretKey(), secret.Name)
	}

	if len(value) == 0 {
		return nil, fmt.Errorf("key %q in secret %q is empty", data.GetSecretKey(), secret.Name)
	}

	return value, nil
}

func (c *Client) getEntryWithTimeout(ctx context.Context, vaultID, entryID string) (dvls.Entry, error) {
	return runWithTimeout(ctx, c.timeout, "get entry", func() (dvls.Entry, error) {
		return c.dvls.GetByID(vaultID, entryID)
	})
}

func (c *Client) updateEntryWithTimeout(ctx context.Context, entry dvls.Entry, value []byte) error {
	updatedEntry := entry
	if err := updatedEntry.SetCredentialSecret(string(value)); err != nil {
		return err
	}

	return runWithTimeoutErr(ctx, c.timeout, "update entry", func() error {
		_, err := c.dvls.Update(updatedEntry)
		return err
	})
}

func runWithTimeout[T any](ctx context.Context, timeout time.Duration, op string, fn func() (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result[T any] struct {
		val T
		err error
	}

	ch := make(chan result[T], 1)
	go func() {
		val, err := fn()
		ch <- result[T]{val: val, err: err}
	}()

	select {
	case <-ctx.Done():
		var zero T
		return zero, fmt.Errorf("%s timed out: %w", op, ctx.Err())
	case res := <-ch:
		return res.val, res.err
	}
}

func runWithTimeoutErr(ctx context.Context, timeout time.Duration, op string, fn func() error) error {
	_, err := runWithTimeout(ctx, timeout, op, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if dvls.IsNotFound(err) {
		return true
	}

	var reqErr dvls.RequestError
	if errors.As(err, &reqErr) && strings.Contains(reqErr.Error(), "404") {
		return true
	}

	return strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found")
}
