/*
Copyright © 2026 ESO Maintainer Team

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

package protonpass

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	passCLIBinary            = "pass-cli"
	defaultSubprocessTimeout = 30 * time.Second
)

// Proton Pass CLI errors.
var (
	errItemNotFound      = errors.New("item not found")
	errVaultNotFound     = errors.New("vault not found")
	errFieldNotFound     = errors.New("field not found in item")
	errLoginFailed       = errors.New("failed to login to Proton Pass")
	errCLINotFound       = errors.New("pass-cli binary not found")
	errCommandTimeout    = errors.New("pass-cli command timed out")
	errAmbiguousItemName = errors.New("multiple items share the same name; use item ID instead")
	errEnsureLogin       = errors.New("failed to ensure login")
)

// cli wraps the pass-cli binary for interacting with Proton Pass.
type cli struct {
	username      string
	password      string
	totpSecret    string
	extraPassword string
	vault         string
	homeDir       string

	mu        sync.Mutex
	loggedIn  bool
	itemCache map[string][]item   // vault -> items
	vaults    []vault             // cached vaults
	itemMap   map[string][]string // itemName -> itemIDs (within configured vault)
}

// newCLI creates a new CLI wrapper.
// homeDir sets the HOME directory for pass-cli session storage.
// When empty, it defaults to "/tmp".
func newCLI(username, password, totpSecret, extraPassword, vault, homeDir string) *cli {
	if homeDir == "" {
		homeDir = "/tmp"
	}
	return &cli{
		username:      username,
		password:      password,
		totpSecret:    totpSecret,
		extraPassword: extraPassword,
		vault:         vault,
		homeDir:       homeDir,
		itemCache:     make(map[string][]item),
		itemMap:       make(map[string][]string),
	}
}

// ensureLoggedIn ensures the CLI is logged in to Proton Pass.
func (c *cli) ensureLoggedIn(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.ensureLoggedInLocked(ctx)
}

// ensureLoggedInLocked is like ensureLoggedIn but assumes c.mu is already held.
func (c *cli) ensureLoggedInLocked(ctx context.Context) error {
	if c.loggedIn {
		return nil
	}

	return c.login(ctx)
}

// clearItemCache resets the item cache. Assumes c.mu is already held.
func (c *cli) clearItemCache() {
	c.itemCache = make(map[string][]item)
	c.itemMap = make(map[string][]string)
}

// clearCache resets all cached data. Assumes c.mu is already held.
func (c *cli) clearCache() {
	c.clearItemCache()
	c.vaults = nil
}

// appendVaultFlag appends --vault-name to the argument list.
func (c *cli) appendVaultFlag(args []string) []string {
	return append(args, "--vault-name", c.vault)
}

// login performs the login to Proton Pass.
func (c *cli) login(ctx context.Context) error {
	env := []string{
		fmt.Sprintf("PROTON_PASS_PASSWORD=%s", c.password),
	}

	if c.totpSecret != "" {
		totpCode, err := generateTOTP(c.totpSecret)
		if err != nil {
			return fmt.Errorf("failed to generate TOTP code: %w", err)
		}
		env = append(env, fmt.Sprintf("PROTON_PASS_TOTP=%s", totpCode))
	}

	if c.extraPassword != "" {
		env = append(env, fmt.Sprintf("PROTON_PASS_EXTRA_PASSWORD=%s", c.extraPassword))
	}

	_, err := c.runCommand(ctx, env, "login", "--interactive", c.username)
	if err != nil {
		// pass-cli returns "Already authenticated" when a session already
		// exists on disk. Treat this as success rather than an error.
		if strings.Contains(err.Error(), "Already authenticated") {
			c.loggedIn = true
			return nil
		}
		return fmt.Errorf("%w: %w", errLoginFailed, err)
	}

	c.loggedIn = true

	c.clearCache()

	return nil
}

// ListVaults returns all vaults.
func (c *cli) ListVaults(ctx context.Context) ([]vault, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureLoggedInLocked(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", errEnsureLogin, err)
	}

	if c.vaults != nil {
		return c.vaults, nil
	}

	output, err := c.runCommand(ctx, nil, "vault", "list", "--output", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	var vaults []vault
	if err := json.Unmarshal(output, &vaults); err != nil {
		return nil, fmt.Errorf("failed to parse vaults: %w", err)
	}

	c.vaults = vaults
	return vaults, nil
}

// GetVaultID returns the vault ID for the given vault name.
func (c *cli) GetVaultID(ctx context.Context, vaultName string) (string, error) {
	vaults, err := c.ListVaults(ctx)
	if err != nil {
		return "", err
	}

	for _, v := range vaults {
		if v.Name == vaultName || v.VaultID == vaultName {
			return v.VaultID, nil
		}
	}

	return "", fmt.Errorf("%w: %s", errVaultNotFound, vaultName)
}

// ListItems returns all items in the configured vault.
func (c *cli) ListItems(ctx context.Context) ([]item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureLoggedInLocked(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", errEnsureLogin, err)
	}

	return c.listItemsLocked(ctx)
}

// listItemsLocked is the internal implementation of ListItems.
// It assumes c.mu is already held by the caller.
func (c *cli) listItemsLocked(ctx context.Context) ([]item, error) {
	if items, ok := c.itemCache[c.vault]; ok {
		return items, nil
	}

	args := []string{"item", "list", "--output", "json", c.vault}

	output, err := c.runCommand(ctx, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	var response itemListResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse items: %w", err)
	}

	c.itemCache[c.vault] = response.Items

	// Update item name to ID map (append to track duplicates)
	for _, item := range response.Items {
		c.itemMap[item.Content.Title] = append(c.itemMap[item.Content.Title], item.ID)
	}

	return response.Items, nil
}

// GetItem returns the details of an item by ID.
func (c *cli) GetItem(ctx context.Context, itemID string) (*item, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", errEnsureLogin, err)
	}

	args := c.appendVaultFlag([]string{"item", "view", "--item-id", itemID, "--output", "json"})

	output, err := c.runCommand(ctx, nil, args...)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("%w: %s", errItemNotFound, itemID)
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	var item item
	if err := json.Unmarshal(output, &item); err != nil {
		return nil, fmt.Errorf("failed to parse item details: %w", err)
	}

	return &item, nil
}

// ResolveItemID resolves an item name to its ID.
func (c *cli) ResolveItemID(ctx context.Context, nameOrID string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureLoggedInLocked(ctx); err != nil {
		return "", fmt.Errorf("%w: %w", errEnsureLogin, err)
	}

	// List items to populate the cache (under the same lock)
	if _, err := c.listItemsLocked(ctx); err != nil {
		return "", err
	}

	// Check if it's already an ID (in the items we found)
	for _, items := range c.itemCache {
		for _, item := range items {
			if item.ID == nameOrID {
				return nameOrID, nil
			}
		}
	}

	// Check if it's a name in our map
	if ids, ok := c.itemMap[nameOrID]; ok {
		if len(ids) > 1 {
			return "", fmt.Errorf("%w: %q matches %d items", errAmbiguousItemName, nameOrID, len(ids))
		}
		return ids[0], nil
	}

	return "", fmt.Errorf("%w: %s", errItemNotFound, nameOrID)
}

// Logout logs out from Proton Pass.
func (c *cli) Logout(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.loggedIn {
		return nil
	}

	_, err := c.runCommand(ctx, nil, "logout")
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	c.loggedIn = false
	c.clearCache()

	return nil
}

// runCommand executes a pass-cli command with a subprocess timeout.
func (c *cli) runCommand(ctx context.Context, extraEnv []string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultSubprocessTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, passCLIBinary, args...)

	// Set up environment - use /tmp as HOME for writable session storage
	// Use filesystem key provider since containers don't have keyring access
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", c.homeDir), "PROTON_PASS_KEY_PROVIDER=fs")
	cmd.Env = append(cmd.Env, extraEnv...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %s %s", errCommandTimeout, passCLIBinary, strings.Join(args, " "))
		}
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errCLINotFound
		}
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("command failed: %s", errMsg)
		}
		return nil, fmt.Errorf("command failed: %w", err)
	}

	return stdout.Bytes(), nil
}

// InvalidateCache clears the item cache for the configured vault.
func (c *cli) InvalidateCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.clearItemCache()
}
