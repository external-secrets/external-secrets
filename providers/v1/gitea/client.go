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

package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	giteasdk "code.gitea.io/sdk/gitea"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const errWriteOnlyProvider = "not implemented - this provider supports write-only operations"

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}

// Client implements esv1.SecretsClient for Gitea Actions secrets.
type Client struct {
	crClient         client.Client
	store            esv1.GenericStore
	provider         *esv1.GiteaProvider
	baseClient       *giteasdk.Client
	namespace        string
	storeKind        string
	createOrUpdateFn func(ctx context.Context, name, value string) error
	listSecretsFn    func(ctx context.Context) ([]*giteasdk.Secret, error)
	deleteSecretFn   func(ctx context.Context, ref esv1.PushSecretRemoteRef) error
	getSecretFn      func(ctx context.Context, ref esv1.PushSecretRemoteRef) (*giteasdk.Secret, error)
}

// DeleteSecret deletes a secret from Gitea Actions.
func (g *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	if err := g.deleteSecretFn(ctx, remoteRef); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

// SecretExists checks if a secret exists in Gitea Actions.
func (g *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	secret, err := g.getSecretFn(ctx, ref)
	if err != nil {
		return false, fmt.Errorf("error fetching secret: %w", err)
	}
	return secret != nil, nil
}

// PushSecret pushes a secret to Gitea Actions.
// When remoteRef.GetSecretKey() is set, that specific key's value is pushed;
// otherwise the entire secret data map is JSON-marshalled and pushed.
func (g *Client) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	var value []byte
	if remoteRef.GetSecretKey() != "" {
		var ok bool
		value, ok = secret.Data[remoteRef.GetSecretKey()]
		if !ok {
			return fmt.Errorf("key %s not found in secret", remoteRef.GetSecretKey())
		}
	} else {
		var err error
		value, err = json.Marshal(secret.Data)
		if err != nil {
			return fmt.Errorf("json.Marshal failed: %w", err)
		}
	}

	if err := g.createOrUpdateFn(ctx, remoteRef.GetRemoteKey(), string(value)); err != nil {
		return fmt.Errorf("failed to push secret: %w", err)
	}
	return nil
}

// GetAllSecrets is not implemented — this provider is write-only.
func (g *Client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf(errWriteOnlyProvider)
}

// GetSecret is not implemented — this provider is write-only.
func (g *Client) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf(errWriteOnlyProvider)
}

// GetSecretMap is not implemented — this provider is write-only.
func (g *Client) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf(errWriteOnlyProvider)
}

// Close is a no-op for this provider.
func (g *Client) Close(_ context.Context) error {
	return nil
}

// Validate checks that the client can communicate with the Gitea API.
// For ClusterSecretStore we skip the live check and return Unknown.
func (g *Client) Validate() (esv1.ValidationResult, error) {
	if g.store.GetKind() == esv1.ClusterSecretStoreKind {
		return esv1.ValidationResultUnknown, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := g.listSecretsFn(ctx)
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("store is not allowed to list secrets: %w", err)
	}
	return esv1.ValidationResultReady, nil
}
