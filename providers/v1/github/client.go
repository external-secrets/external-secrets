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

package github

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	github "github.com/google/go-github/v56/github"
	"golang.org/x/crypto/nacl/box"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const errWriteOnlyProvider = "not implemented - this provider supports write-only operations"

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}

// ActionsServiceClient defines the interface for interacting with GitHub Actions secrets.
type ActionsServiceClient interface {
	// CreateOrUpdateOrgSecret creates or updates an organization secret.
	CreateOrUpdateOrgSecret(ctx context.Context, org string, eSecret *github.EncryptedSecret) (response *github.Response, err error)
	// GetOrgSecret retrieves an organization secret.
	GetOrgSecret(ctx context.Context, org string, name string) (*github.Secret, *github.Response, error)
	// ListOrgSecrets lists all organization secrets.
	ListOrgSecrets(ctx context.Context, org string, opts *github.ListOptions) (*github.Secrets, *github.Response, error)
}

// Client implements the External Secrets Kubernetes provider for GitHub Actions secrets.
type Client struct {
	crClient         client.Client
	store            esv1.GenericStore
	provider         *esv1.GithubProvider
	baseClient       github.ActionsService
	namespace        string
	storeKind        string
	repoID           int64
	getSecretFn      func(ctx context.Context, ref esv1.PushSecretRemoteRef) (*github.Secret, *github.Response, error)
	getPublicKeyFn   func(ctx context.Context) (*github.PublicKey, *github.Response, error)
	createOrUpdateFn func(ctx context.Context, eSecret *github.EncryptedSecret) (*github.Response, error)
	listSecretsFn    func(ctx context.Context) (*github.Secrets, *github.Response, error)
	deleteSecretFn   func(ctx context.Context, ref esv1.PushSecretRemoteRef) (*github.Response, error)
}

// DeleteSecret deletes a secret from GitHub Actions.
func (g *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	_, err := g.deleteSecretFn(ctx, remoteRef)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

// SecretExists checks if a secret exists in GitHub Actions.
func (g *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	githubSecret, _, err := g.getSecretFn(ctx, ref)
	if err != nil {
		return false, fmt.Errorf("error fetching secret: %w", err)
	}
	if githubSecret != nil {
		return true, nil
	}
	return false, nil
}

// PushSecret pushes a new secret to GitHub Actions.
func (g *Client) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	githubSecret, response, err := g.getSecretFn(ctx, remoteRef)
	if err != nil && (response == nil || response.StatusCode != 404) {
		return fmt.Errorf("error fetching secret: %w", err)
	}

	// First at all, we need the organization public key to encrypt the secret.
	publicKey, _, err := g.getPublicKeyFn(ctx)
	if err != nil {
		return fmt.Errorf("error fetching public key: %w", err)
	}

	decodedPublicKey, err := base64.StdEncoding.DecodeString(publicKey.GetKey())
	if err != nil {
		return fmt.Errorf("unable to decode public key: %w", err)
	}

	var boxKey [32]byte
	copy(boxKey[:], decodedPublicKey)
	var ok bool
	// default to full secret.
	value, err := json.Marshal(secret.Data)
	if err != nil {
		return fmt.Errorf("json.Marshal failed with error %w", err)
	}
	// if key is specified, overwrite to key only
	if remoteRef.GetSecretKey() != "" {
		value, ok = secret.Data[remoteRef.GetSecretKey()]
		if !ok {
			return fmt.Errorf("key %s not found in secret", remoteRef.GetSecretKey())
		}
	}

	encryptedBytes, err := box.SealAnonymous([]byte{}, value, &boxKey, crypto_rand.Reader)
	if err != nil {
		return fmt.Errorf("box.SealAnonymous failed with error %w", err)
	}
	name := remoteRef.GetRemoteKey()
	visibility := "all"
	if githubSecret != nil {
		name = githubSecret.Name
		visibility = githubSecret.Visibility
	}
	encryptedString := base64.StdEncoding.EncodeToString(encryptedBytes)
	keyID := publicKey.GetKeyID()
	encryptedSecret := &github.EncryptedSecret{
		Name:           name,
		KeyID:          keyID,
		EncryptedValue: encryptedString,
		Visibility:     visibility,
	}

	if _, err := g.createOrUpdateFn(ctx, encryptedSecret); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// GetAllSecrets is not implemented as this provider is write-only.
func (g *Client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf(errWriteOnlyProvider)
}

// GetSecret is not implemented as this provider is write-only.
func (g *Client) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf(errWriteOnlyProvider)
}

// GetSecretMap is not implemented as this provider is write-only.
func (g *Client) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf(errWriteOnlyProvider)
}

// Close cleans up any resources held by the client. No-op for this provider.
func (g *Client) Close(_ context.Context) error {
	return nil
}

// Validate checks if the client is properly configured and has access to the GitHub Actions API.
func (g *Client) Validate() (esv1.ValidationResult, error) {
	if g.store.GetKind() == esv1.ClusterSecretStoreKind {
		return esv1.ValidationResultUnknown, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _, err := g.listSecretsFn(ctx)

	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("store is not allowed to list secrets: %w", err)
	}

	return esv1.ValidationResultReady, nil
}
