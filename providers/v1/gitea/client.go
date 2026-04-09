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
	"github.com/external-secrets/external-secrets/runtime/find"
)

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
	getVariableFn    func(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (string, error)
	listVariablesFn  func(ctx context.Context) (map[string][]byte, error)
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

// GetSecret reads a Gitea Actions Variable by name.
// If ref.Property is set, the variable value must be a JSON object and the specified property is extracted.
func (g *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	value, err := g.getVariableFn(ctx, ref)
	if err != nil {
		return nil, err
	}
	if ref.Property != "" {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(value), &obj); err != nil {
			return nil, fmt.Errorf("property %q requested but variable value is not a JSON object: %w", ref.Property, err)
		}
		if raw, ok := obj[ref.Property]; ok {
			return jsonRawToBytes(raw), nil
		}
		return nil, fmt.Errorf("property %q not found in variable", ref.Property)
	}
	return []byte(value), nil
}

// GetSecretMap reads a Gitea Actions Variable and returns its value parsed as a JSON object.
func (g *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	value, err := g.getVariableFn(ctx, ref)
	if err != nil {
		return nil, err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &obj); err != nil {
		return nil, fmt.Errorf("variable %q value is not a JSON object: %w", ref.Key, err)
	}
	result := make(map[string][]byte, len(obj))
	for k, raw := range obj {
		result[k] = jsonRawToBytes(raw)
	}
	return result, nil
}

// GetAllSecrets lists all Gitea Actions Variables, optionally filtered by ref.Name regexp.
func (g *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	all, err := g.listVariablesFn(ctx)
	if err != nil {
		return nil, err
	}
	if ref.Name == nil {
		return all, nil
	}
	matcher, err := find.New(*ref.Name)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte)
	for k, v := range all {
		if matcher.MatchName(k) {
			out[k] = v
		}
	}
	return out, nil
}

// Close is a no-op for this provider.
func (g *Client) Close(_ context.Context) error {
	return nil
}

// jsonRawToBytes converts a json.RawMessage to a byte slice.
// JSON strings are returned unquoted; all other JSON values (numbers, booleans,
// objects, arrays) are returned as their raw JSON representation.
func jsonRawToBytes(raw json.RawMessage) []byte {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []byte(s)
	}
	return []byte(raw)
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
