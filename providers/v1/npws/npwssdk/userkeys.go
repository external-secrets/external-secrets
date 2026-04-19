// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package npwssdk

import (
	"context"
	"encoding/base64"
	"fmt"
)

// UserKeyManager manages user encryption keys and handles decrypt/encrypt pipelines.
// Mirrors C# PsrApi.Internals.UserKeyManager.
type UserKeyManager struct {
	keys          []UserKey
	serviceClient *HTTPClient
	encryption    *EncryptionManager
	rights        *RightManager
	seals         *SealManager
	currentUserID string
}

// NewUserKeyManager creates a new UserKeyManager.
func NewUserKeyManager(serviceClient *HTTPClient, encryption *EncryptionManager, rights *RightManager, seals *SealManager) *UserKeyManager {
	return &UserKeyManager{
		serviceClient: serviceClient,
		encryption:    encryption,
		rights:        rights,
		seals:         seals,
	}
}

// SetUserKeys stores the user's private keys received after authentication.
func (ukm *UserKeyManager) SetUserKeys(keys []UserKey) {
	ukm.keys = keys
}

// SetCurrentUserID sets the current user's ID for key lookup.
func (ukm *UserKeyManager) SetCurrentUserID(userID string) {
	ukm.currentUserID = userID
}

// GetCurrentUserID returns the current user's ID.
func (ukm *UserKeyManager) GetCurrentUserID() string {
	return ukm.currentUserID
}

// GetUserPrivateKey returns the current user's private key.
// Matches C# UserKeyManager.GetUserPrivateKey().
func (ukm *UserKeyManager) GetUserPrivateKey() ([]byte, error) {
	key := ukm.getCurrentUserKey()
	if key == nil {
		return nil, fmt.Errorf("UserKeyManager: no key found for current user")
	}
	return key.PrivateKey, nil
}

// DecryptContainerItem decrypts a container item's encrypted value.
// Matches C# UserKeyManager.DecryptContainerItem(item, reason) exactly.
func (ukm *UserKeyManager) DecryptContainerItem(ctx context.Context, item *PsrContainerItem, reason string) (string, error) {
	containerItemIsSealed := false

	if !item.IsEncrypted() {
		return "", nil
	}

	activeID := item.ID

	// Step 1: Get data rights
	dataRights, err := ukm.rights.GetLegitimateDataRights(ctx, activeID, false, false)
	if err != nil {
		return "", &PsrAPIError{Code: ExceptionRightNoKey, Message: "no rights found"}
	}
	if len(dataRights) == 0 {
		return "", &PsrAPIError{Code: ExceptionRightNoKey, Message: "no rights found"}
	}

	// Step 2: Try normal decryption (unsealed rights)
	dataKey := ukm.decryptDataRights(dataRights)

	// Step 3: If normal decryption failed, try seal-based decryption
	if dataKey == nil {
		for i := range dataRights {
			right := &dataRights[i]
			if !right.IsSealed() {
				continue
			}

			currentUserID := ukm.currentUserID

			seal, err := ukm.seals.BreakSeal(ctx, right.SealID)
			if err != nil || seal == nil {
				continue
			}

			// C# uses GetSealOpenType (not BySealId) — sends full seal object
			openType, err := ukm.seals.GetSealOpenType(ctx, seal, item.ID, currentUserID)
			if err != nil {
				continue
			}
			if openType != SealOpenTypeNone {
				containerItemIsSealed = true
			}
			if openType == SealOpenTypeBrokenExpired {
				return "", fmt.Errorf("the seal release is expired")
			}

			hasRelease, err := ukm.seals.HasRelease(ctx, seal, currentUserID)
			if err != nil || !hasRelease {
				continue
			}

			dataKey = ukm.decryptDataRightWithSealKeyRelease(dataRights, seal, currentUserID)
			if dataKey != nil {
				break
			}
		}
	}

	// Step 4: Check if we got a key
	if dataKey == nil {
		if containerItemIsSealed {
			return "", &PsrAPIError{Code: ExceptionContainerItemIsSealed, Message: "container item is sealed"}
		}
		return "", &PsrAPIError{Code: ExceptionRightNoKey, Message: "no decryptable right key found"}
	}

	// Step 5: Fetch encrypted value from server
	var secretItem PsrContainerItem
	err = ukm.serviceClient.Post(ctx, "GetContainerItemWithSecretValue", map[string]interface{}{
		"itemId": item.ID,
		"reason": reason,
	}, &secretItem)
	if err != nil {
		return "", fmt.Errorf("DecryptContainerItem: fetching secret value: %w", err)
	}
	if secretItem.Value == "" {
		return "", nil
	}

	// Step 6: Decrypt the value
	encryptedValue, err := base64.StdEncoding.DecodeString(secretItem.Value)
	if err != nil {
		return "", fmt.Errorf("DecryptContainerItem: decoding base64: %w", err)
	}

	plaintext, err := ukm.encryption.Decrypt(dataKey, encryptedValue)
	if err != nil {
		return "", fmt.Errorf("DecryptContainerItem: decrypting: %w", err)
	}

	return string(plaintext), nil
}

// EncryptContainerItem encrypts a container item with the given plaintext.
// Returns the new decryption key if one was created.
// Matches C# UserKeyManager.EncryptContainerItem(item, plaintext).
func (ukm *UserKeyManager) EncryptContainerItem(item *PsrContainerItem, plaintext string) ([]byte, error) {
	plainBytes := []byte(plaintext)

	var getKey func() ([]byte, error)
	if item.ID != "" {
		getKey = func() ([]byte, error) {
			return ukm.GetDecryptedRightKey(context.Background(), item.ID)
		}
	}

	result, err := ukm.encryption.EncryptContainerItem(item, plainBytes, getKey)
	if err != nil {
		return nil, err
	}

	return result.NewDecryptionKey, nil
}

// GetDecryptedRightKey retrieves and decrypts the right key for a data item.
// Matches C# UserKeyManager.GetDecryptedRightKey(dataId).
func (ukm *UserKeyManager) GetDecryptedRightKey(ctx context.Context, dataID string) ([]byte, error) {
	rights, err := ukm.rights.GetLegitimateDataRights(ctx, dataID, false, false)
	if err != nil {
		return nil, err
	}
	dataKey := ukm.decryptDataRights(rights)
	return dataKey, nil
}

// DecryptDataRight decrypts a single data right using the matching user key.
// Returns nil if no matching key found (not an error).
// Matches C# UserKeyManager.DecryptDataRight(right).
func (ukm *UserKeyManager) DecryptDataRight(right *PsrDataRight) []byte {
	if right == nil || !right.HasRightKey() {
		return nil
	}
	key := ukm.getKeyByID(right.LegitimateID)
	if key == nil {
		return nil
	}
	decrypted, err := ukm.decryptWithKey(key, right.RightKey)
	if err != nil {
		return nil
	}
	return decrypted
}

// DecryptRightKeyWithCurrentUserKey decrypts a right key with the current user's private key.
// Matches C# UserKeyManager.DecryptRightKeyWithCurrentUserKey(encryptedRightKey).
func (ukm *UserKeyManager) DecryptRightKeyWithCurrentUserKey(encryptedRightKey []byte) ([]byte, error) {
	currentUserKey := ukm.getCurrentUserKey()
	if currentUserKey == nil {
		return nil, fmt.Errorf("current user key not found, please log out and login again")
	}
	return ukm.decryptWithKey(currentUserKey, encryptedRightKey)
}

// EncryptDataRightKey encrypts a right key with a public key.
// Matches C# UserKeyManager.EncryptDataRightKey(rightKey, publicKey).
func (ukm *UserKeyManager) EncryptDataRightKey(rightKey, publicKey []byte) ([]byte, error) {
	if rightKey == nil {
		return nil, nil
	}
	return ukm.encryption.EncryptWithPublicKey(publicKey, rightKey)
}

// DecryptDataRightWithSeal decrypts a data right protected by a seal.
// Matches C# UserKeyManager.DecryptDataRightWithSeal(data, right).
func (ukm *UserKeyManager) DecryptDataRightWithSeal(ctx context.Context, dataID string, right *PsrDataRight) ([]byte, error) {
	userID := ukm.currentUserID

	seal, err := ukm.seals.BreakSeal(ctx, right.SealID)
	if err != nil || seal == nil {
		return nil, nil
	}

	openType, err := ukm.seals.GetSealOpenTypeBySealID(ctx, seal.ID, dataID, userID)
	if err != nil {
		return nil, nil
	}
	if openType == SealOpenTypeBrokenExpired {
		return nil, fmt.Errorf("the seal release expired")
	}

	hasRelease, err := ukm.seals.HasRelease(ctx, seal, userID)
	if err != nil || !hasRelease {
		return nil, nil
	}

	result := ukm.decryptDataRightWithSealKeyRelease([]PsrDataRight{*right}, seal, userID)
	return result, nil
}

// EncryptRightKeysAndReturn encrypts the right key for each legitimate's public key.
// Returns tuples of (DataID, LegitimateID, EncryptedKey).
// Matches C# UserKeyManager.EncryptRightKeysAndReturn(data, privateKey).
func (ukm *UserKeyManager) EncryptRightKeysAndReturn(ctx context.Context, dataID string, privateKey []byte) ([]RightKeyTuple, error) {
	rights, err := ukm.rights.GetLegitimateDataRights(ctx, dataID, false, false)
	if err != nil {
		return nil, err
	}

	var result []RightKeyTuple
	for _, right := range rights {
		if len(right.LegitimatePublicKey) == 0 {
			continue
		}
		encrypted, err := ukm.encryption.EncryptWithPublicKey(right.LegitimatePublicKey, privateKey)
		if err != nil {
			return nil, fmt.Errorf("EncryptRightKeys: encrypting for %s: %w", right.LegitimateID, err)
		}
		result = append(result, RightKeyTuple{
			DataID:       dataID,
			LegitimateID: right.LegitimateID,
			EncryptedKey: encrypted,
		})
	}

	return result, nil
}

// RightKeyTuple holds an encrypted right key for a specific legitimate.
type RightKeyTuple struct {
	DataID       string
	LegitimateID string
	EncryptedKey []byte
}

// --- Private helpers ---

func (ukm *UserKeyManager) getCurrentUserKey() *UserKey {
	for i := range ukm.keys {
		if ukm.keys[i].ID == ukm.currentUserID {
			return &ukm.keys[i]
		}
	}
	if len(ukm.keys) == 1 {
		return &ukm.keys[0]
	}
	return nil
}

func (ukm *UserKeyManager) getKeyByID(id string) *UserKey {
	for i := range ukm.keys {
		if ukm.keys[i].ID == id {
			return &ukm.keys[i]
		}
	}
	return nil
}

func (ukm *UserKeyManager) decryptWithKey(key *UserKey, encryptedValue []byte) ([]byte, error) {
	return ukm.encryption.Decrypt(key.PrivateKey, encryptedValue)
}

// decryptDataRights tries to find and decrypt a right key from the list.
// Returns nil if no matching unsealed right found (not an error).
// Matches C# UserKeyManager.DecryptDataRights(rights).
func (ukm *UserKeyManager) decryptDataRights(rights []PsrDataRight) []byte {
	if rights == nil || ukm.keys == nil {
		return nil
	}

	// Find first right that has a key, is not sealed, and matches one of our keys
	for _, right := range rights {
		if !right.HasRightKey() || right.IsSealed() {
			continue
		}
		key := ukm.getKeyByID(right.LegitimateID)
		if key == nil {
			continue
		}
		decrypted, err := ukm.decryptWithKey(key, right.RightKey)
		if err == nil {
			return decrypted
		}
	}
	return nil
}

// decryptDataRightWithSealKeyRelease decrypts the right key using a seal key release.
// Matches C# UserKeyManager.DecryptDataRightWithSealKeyRelease(rights, seal, userId).
func (ukm *UserKeyManager) decryptDataRightWithSealKeyRelease(rights []PsrDataRight, seal *PsrSeal, userID string) []byte {
	if seal == nil {
		return nil
	}

	// Find SealKey where any KeyRelease has LegitimateSealKey and matches one of our keys
	var sk *PsrSealKey
	for i := range seal.Keys {
		for _, r := range seal.Keys[i].KeyReleases {
			if len(r.LegitimateSealKey) > 0 && ukm.getKeyByID(r.LegitimateID) != nil {
				sk = &seal.Keys[i]
				break
			}
		}
		if sk != nil {
			break
		}
	}
	if sk == nil {
		return nil
	}

	// Find release for this user with LegitimateSealKey
	var release *PsrSealKeyRelease
	for i := range sk.KeyReleases {
		r := &sk.KeyReleases[i]
		if len(r.LegitimateSealKey) > 0 && r.LegitimateID == userID {
			release = r
			break
		}
	}
	if release == nil {
		return nil
	}

	// Get user key matching the release
	key := ukm.getKeyByID(release.LegitimateID)
	if key == nil {
		return nil
	}

	// Find right matching this seal where we have a key
	var right *PsrDataRight
	for i := range rights {
		r := &rights[i]
		if len(r.RightKey) > 0 && r.SealID == seal.ID && ukm.getKeyByID(r.LegitimateID) != nil {
			right = r
			break
		}
	}
	if right == nil {
		return nil
	}

	// Decrypt: user key → seal key → right key
	sealKey, err := ukm.decryptWithKey(key, release.LegitimateSealKey)
	if err != nil {
		return nil
	}
	dataKey, err := ukm.encryption.Decrypt(sealKey, right.RightKey)
	if err != nil {
		return nil
	}
	return dataKey
}
