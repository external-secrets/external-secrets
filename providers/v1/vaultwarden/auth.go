//go:build vaultwarden || all_providers

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

package vaultwarden

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/vaultwarden/internal/crypto"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// vaultwardenTokenResponse is the JSON response from /identity/connect/token.
type vaultwardenTokenResponse struct {
	AccessToken    string `json:"access_token"`
	Key            string `json:"Key"`   // encrypted symmetric key
	Kdf            int    `json:"Kdf"`
	KdfIterations  int    `json:"KdfIterations"`
	KdfMemory      *int   `json:"KdfMemory"`
	KdfParallelism *int   `json:"KdfParallelism"`
}

// vaultwardenProfile is the JSON response from /api/accounts/profile.
type vaultwardenProfile struct {
	Email          string `json:"email"`          // lowercase in profile response
	Key            string `json:"key"`            // lowercase
	Kdf            int    `json:"kdf"`
	KdfIterations  int    `json:"kdfIterations"`
	KdfMemory      *int   `json:"kdfMemory"`
	KdfParallelism *int   `json:"kdfParallelism"`
}

// resolveSecretKeyRef is a helper that reads a K8s secret value using a SecretKeySelector.
func (c *Client) resolveSecretKeyRef(ctx context.Context, sel esmeta.SecretKeySelector) (string, error) {
	storeKind := c.store.GetObjectKind().GroupVersionKind().Kind
	return resolvers.SecretKeyRef(ctx, c.crClient, storeKind, c.namespace, &sel)
}

// fetchToken authenticates with Vaultwarden's identity endpoint and returns the token response.
// It reads clientID, clientSecret and masterPassword from Kubernetes secrets referenced in the store.
func (c *Client) fetchToken(ctx context.Context) (*vaultwardenTokenResponse, error) {
	secretRef := c.provider.Auth.SecretRef

	clientID, err := c.resolveSecretKeyRef(ctx, secretRef.ClientID)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: reading clientID: %w", err)
	}

	clientSecret, err := c.resolveSecretKeyRef(ctx, secretRef.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: reading clientSecret: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "api")
	form.Set("DeviceIdentifier", uuid.New().String())
	form.Set("DeviceType", "21")
	form.Set("DeviceName", "eso-provider")

	tokenURL := strings.TrimRight(c.provider.URL, "/") + "/identity/connect/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vaultwarden: token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tokenResp vaultwardenTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("vaultwarden: decoding token response: %w", err)
	}
	return &tokenResp, nil
}

// fetchProfile retrieves the account profile using the provided bearer token.
func (c *Client) fetchProfile(ctx context.Context, accessToken string) (*vaultwardenProfile, error) {
	profileURL := strings.TrimRight(c.provider.URL, "/") + "/api/accounts/profile"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: building profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vaultwarden: profile request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vaultwarden: profile endpoint returned HTTP %d", resp.StatusCode)
	}

	var profile vaultwardenProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("vaultwarden: decoding profile response: %w", err)
	}
	return &profile, nil
}

// getSymKey fetches (or returns a cached) access token and symmetric key material.
// It authenticates with Vaultwarden, fetches the account profile, derives the master key,
// and decrypts the user's symmetric encryption key. Results are cached for 5 minutes.
func (c *Client) getSymKey(ctx context.Context) (accessToken string, symEncKey, symMacKey []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cache != nil && time.Now().Before(c.cache.expiresAt) {
		return c.cache.accessToken, c.cache.symEncKey, c.cache.symMacKey, nil
	}

	tokenResp, err := c.fetchToken(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	profile, err := c.fetchProfile(ctx, tokenResp.AccessToken)
	if err != nil {
		return "", nil, nil, err
	}

	masterPassword, err := c.resolveSecretKeyRef(ctx, c.provider.Auth.SecretRef.MasterPassword)
	if err != nil {
		return "", nil, nil, fmt.Errorf("vaultwarden: reading masterPassword: %w", err)
	}

	memVal, parVal := 0, 0
	if profile.KdfMemory != nil {
		memVal = *profile.KdfMemory
	}
	if profile.KdfParallelism != nil {
		parVal = *profile.KdfParallelism
	}

	masterKey, err := crypto.DeriveKey(masterPassword, profile.Email, profile.Kdf, profile.KdfIterations, memVal, parVal)
	if err != nil {
		return "", nil, nil, fmt.Errorf("vaultwarden: deriving master key: %w", err)
	}

	stretchedEnc, stretchedMac, err := crypto.StretchKey(masterKey)
	if err != nil {
		return "", nil, nil, fmt.Errorf("vaultwarden: stretching master key: %w", err)
	}

	profileKeyES, err := crypto.ParseEncString(profile.Key)
	if err != nil {
		return "", nil, nil, fmt.Errorf("vaultwarden: parsing profile key: %w", err)
	}
	symKeyBytes, err := crypto.Decrypt(profileKeyES, stretchedEnc, stretchedMac)
	if err != nil {
		return "", nil, nil, fmt.Errorf("vaultwarden: decrypting symmetric key: %w", err)
	}
	if len(symKeyBytes) < 64 {
		return "", nil, nil, fmt.Errorf("vaultwarden: symmetric key too short (%d bytes)", len(symKeyBytes))
	}

	c.cache = &cachedToken{
		accessToken: tokenResp.AccessToken,
		symEncKey:   symKeyBytes[0:32],
		symMacKey:   symKeyBytes[32:64],
		expiresAt:   time.Now().Add(5 * time.Minute),
	}

	return c.cache.accessToken, c.cache.symEncKey, c.cache.symMacKey, nil
}

// getProvider extracts the VaultwardenProvider config from a generic store.
func getProvider(store esv1.GenericStore) (*esv1.VaultwardenProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Vaultwarden == nil {
		return nil, fmt.Errorf(errUnexpectedStoreSpec)
	}
	return spc.Provider.Vaultwarden, nil
}

