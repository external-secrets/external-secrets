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
	"fmt"
	"time"
)

const guidEmpty = "00000000-0000-0000-0000-000000000000"

// ContainerManager handles CRUD operations on NPWS containers.
type ContainerManager struct {
	serviceClient *HTTPClient
	userKeys      *UserKeyManager
	encryption    *EncryptionManager
	orgUnits      *OrganisationUnitManager
	rights        *RightManager
	genericRights *GenericRightManager
}

// NewContainerManager creates a new ContainerManager.
func NewContainerManager(
	serviceClient *HTTPClient,
	userKeys *UserKeyManager,
	encryption *EncryptionManager,
	orgUnits *OrganisationUnitManager,
	rights *RightManager,
	genericRights *GenericRightManager,
) *ContainerManager {
	return &ContainerManager{
		serviceClient: serviceClient,
		userKeys:      userKeys,
		encryption:    encryption,
		orgUnits:      orgUnits,
		rights:        rights,
		genericRights: genericRights,
	}
}

// GetContainerList retrieves a list of containers matching the filter.
func (cm *ContainerManager) GetContainerList(ctx context.Context, containerType ContainerType, filter *PsrContainerListFilter) ([]PsrContainer, error) {
	var result []PsrContainer
	err := cm.serviceClient.Post(ctx, "GetContainerList", map[string]interface{}{
		"containerType": containerType,
		"filter":        filter,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("GetContainerList: %w", err)
	}
	return result, nil
}

// GetContainerByName searches for a container by its display name using a content filter.
// Returns an error if no container or multiple containers match the name.
func (cm *ContainerManager) GetContainerByName(ctx context.Context, name string) (*PsrContainer, error) {
	filter := &PsrContainerListFilter{
		Type:          TypeContainerListFilter,
		ContainerType: ContainerTypePassword,
		DataStates:    StateActive,
		FilterGroups: []interface{}{
			PsrListFilterGroupContent{
				Type: "ListFilterGroupContent",
				SearchList: []PsrListFilterObjectContent{
					{
						Search:                  name,
						FilterActive:            true,
						ExactSearch:             true,
						SearchTags:              false,
						SearchOrganisationUnits: false,
					},
				},
			},
		},
	}

	containers, err := cm.GetContainerList(ctx, ContainerTypePassword, filter)
	if err != nil {
		return nil, fmt.Errorf("GetContainerByName: %w", err)
	}

	// Server may return matches on other fields — filter by DisplayName
	var matches []PsrContainer
	for i := range containers {
		if containers[i].GetDisplayName() == name {
			matches = append(matches, containers[i])
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("GetContainerByName: multiple containers (%d) found with name %q, use ID instead", len(matches), name)
	}
}

// GetContainer retrieves a single container by ID.
func (cm *ContainerManager) GetContainer(ctx context.Context, containerID string) (*PsrContainer, error) {
	var result *PsrContainer
	err := cm.serviceClient.Post(ctx, "GetContainer", map[string]interface{}{
		"id": containerID,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("GetContainer: %w", err)
	}
	return result, nil
}

// UpdateContainer updates an existing container.
// Matches C# ContainerManager.UpdateContainer(container, behaviors) exactly.
func (cm *ContainerManager) UpdateContainer(ctx context.Context, container *PsrContainer) (*PsrContainer, error) {
	container.TimeStampUtc = FlexTime{time.Now().UTC()}

	// Collect new items (Id == empty) and set their state to active
	var newContainerItemNames []string
	for i := range container.Items {
		if container.Items[i].ID == "" || container.Items[i].ID == guidEmpty {
			container.Items[i].ID = guidEmpty
			container.Items[i].TransactionID = guidEmpty
			container.Items[i].DataStates = StateActive
			newContainerItemNames = append(newContainerItemNames, container.Items[i].Name)
		}
		if container.Items[i].ContainerID == "" {
			container.Items[i].ContainerID = container.ID
		}
	}

	// PrepareContainerItems — encrypt items, collect private keys
	var privateKeys []privateKeyTuple
	if container.ContainerType == ContainerTypePassword {
		privateKeys = cm.prepareContainerItems(container)
	}

	// Send update to server
	var updatedContainer *PsrContainer
	err := cm.serviceClient.Post(ctx, "UpdateContainer", map[string]interface{}{
		"container": container,
	}, &updatedContainer)
	if err != nil {
		return nil, fmt.Errorf("UpdateContainer: %w", err)
	}
	if updatedContainer == nil {
		return nil, nil
	}

	// Sync IDs from server response back to local items
	for i := range container.Items {
		for j := range updatedContainer.Items {
			if updatedContainer.Items[j].Name == container.Items[i].Name {
				container.Items[i].ID = updatedContainer.Items[j].ID
				break
			}
		}
	}

	// Apply rights to new container items
	if len(newContainerItemNames) > 0 {
		// Resolve new item IDs from updated container
		var newItemIDs []string
		for _, name := range newContainerItemNames {
			for j := range updatedContainer.Items {
				if updatedContainer.Items[j].Name == name {
					newItemIDs = append(newItemIDs, updatedContainer.Items[j].ID)
					break
				}
			}
		}

		// Get container rights and apply to new items
		containerRights, err := cm.rights.GetLegitimateDataRights(ctx, container.ID, false, false)
		if err == nil && len(newItemIDs) > 0 {
			_ = cm.genericRights.SaveRights(ctx, newItemIDs, containerRights, false, false)
		}
	}

	// Encrypt password data rights for all encrypted items with new keys
	if container.ContainerType == ContainerTypePassword && len(privateKeys) > 0 {
		batchItems, err := cm.encryptPasswordDataRights(ctx, updatedContainer, privateKeys)
		if err == nil && len(batchItems) > 0 {
			_ = cm.rights.BatchUpdateRights(ctx, batchItems)
		}
	}

	return updatedContainer, nil
}

// AddContainer creates a new container.
// parentOrgUnitID is required for password containers.
// Matches C# ContainerManager.AddContainer(container, parentOrganisationUnitId, rightTemplates, templateGroupId).
func (cm *ContainerManager) AddContainer(ctx context.Context, container *PsrContainer, parentOrgUnitID string) (*PsrContainer, error) {
	if container.ContainerType == ContainerTypePassword && parentOrgUnitID == "" {
		return nil, fmt.Errorf("AddContainer: containers of type password must have a parent organization unit")
	}

	// Server generates IDs — send Guid.Empty
	container.ID = guidEmpty
	container.TransactionID = guidEmpty
	for i := range container.Items {
		container.Items[i].ID = guidEmpty
		container.Items[i].ContainerID = guidEmpty
		container.Items[i].TransactionID = guidEmpty
	}

	rightOpts := &RightInheritanceOptions{}

	if container.ContainerType == ContainerTypePassword {
		// PrepareContainerItems — encrypt each item, collect private keys
		privateKeys := cm.prepareContainerItems(container)

		// Get current user's public key (C#: _psrApi.CurrentUser.PublicKey)
		userPublicKey, err := cm.GetCurrentUserPublicKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("AddContainer: %w", err)
		}

		// Build AsymmetricRightKeys — encrypt each private key with user's public key
		for i := range container.Items {
			if !container.Items[i].IsEncrypted() {
				continue
			}
			var privateKey []byte
			for _, pk := range privateKeys {
				if pk.Name == container.Items[i].Name {
					privateKey = pk.Key
					break
				}
			}
			encryptedPrivateKey, err := cm.userKeys.EncryptDataRightKey(privateKey, userPublicKey)
			if err != nil {
				return nil, fmt.Errorf("AddContainer: encrypting right key for %s: %w", container.Items[i].Name, err)
			}
			rightOpts.Keys = append(rightOpts.Keys, AsymmetricRightKey{
				Identifier:          container.Items[i].Name,
				EncryptedPrivateKey: encryptedPrivateKey,
			})
		}
	}

	var result *PsrContainer
	err := cm.serviceClient.Post(ctx, "AddContainerV2", map[string]interface{}{
		"container":                container,
		"parentOrganisationUnitId": parentOrgUnitID,
		"rightInheritanceOptions":  rightOpts,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("AddContainer: %w", err)
	}
	return result, nil
}

// privateKeyTuple holds a name-key pair from PrepareContainerItems.
type privateKeyTuple struct {
	Name string
	Key  []byte
}

// DeleteContainer deletes a container.
func (cm *ContainerManager) DeleteContainer(ctx context.Context, container *PsrContainer) error {
	err := cm.serviceClient.Post(ctx, "DeleteContainer", map[string]interface{}{
		"container": container,
	}, nil)
	if err != nil {
		return fmt.Errorf("DeleteContainer: %w", err)
	}
	return nil
}

// GetCurrentUserPublicKey retrieves the current user's public key from the server
// via the OrganisationUnitManager.
func (cm *ContainerManager) GetCurrentUserPublicKey(ctx context.Context) ([]byte, error) {
	userID := cm.userKeys.currentUserID
	if userID == "" {
		return nil, fmt.Errorf("GetCurrentUserPublicKey: not logged in")
	}

	user, err := cm.orgUnits.GetOrganisationUnitUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetCurrentUserPublicKey: %w", err)
	}
	return user.PublicKey, nil
}

// DecryptContainerItem decrypts a container item's encrypted value.
func (cm *ContainerManager) DecryptContainerItem(ctx context.Context, item *PsrContainerItem, reason string) (string, error) {
	return cm.userKeys.DecryptContainerItem(ctx, item, reason)
}

// EncryptContainerItem encrypts a container item with the given plaintext.
func (cm *ContainerManager) EncryptContainerItem(item *PsrContainerItem, plaintext string) ([]byte, error) {
	return cm.userKeys.EncryptContainerItem(item, plaintext)
}

// prepareContainerItems encrypts all encrypted items and returns their private keys.
// Matches C# ContainerManager.PrepareContainerItems(items).
// Used by both UpdateContainer and AddContainer.
func (cm *ContainerManager) prepareContainerItems(container *PsrContainer) []privateKeyTuple {
	var privateKeys []privateKeyTuple

	for i := range container.Items {
		item := &container.Items[i]
		item.Position = i

		if !item.IsEncrypted() {
			continue
		}

		// Encrypted item is new — if no PlainTextValue and no Value, set empty
		if item.PlainTextValue == "" && item.Value == "" {
			item.PlainTextValue = ""
		}

		// Encrypted value should NOT be changed if a currently encrypted value exists
		// and the PlainTextValue is empty
		if item.PlainTextValue == "" && item.Value != "" {
			continue
		}

		privateKey, err := cm.userKeys.EncryptContainerItem(item, item.PlainTextValue)
		if err != nil {
			continue
		}

		privateKeys = append(privateKeys, privateKeyTuple{
			Name: item.Name,
			Key:  privateKey,
		})
	}

	return privateKeys
}

// encryptPasswordDataRights encrypts the right keys for all encrypted items.
// Matches C# ContainerManager.EncryptPasswordDataRights(password, privateKeys).
//
//nolint:unparam // error return kept for future use
func (cm *ContainerManager) encryptPasswordDataRights(ctx context.Context, container *PsrContainer, privateKeys []privateKeyTuple) ([]PsrBatchRightItem, error) {
	var batchItems []PsrBatchRightItem
	for i := range container.Items {
		if !container.Items[i].IsEncrypted() {
			continue
		}
		// Find private key for this item
		var pk []byte
		for _, ptv := range privateKeys {
			if ptv.Name == container.Items[i].Name && ptv.Key != nil {
				pk = ptv.Key
				break
			}
		}
		if pk == nil {
			continue
		}
		// Encrypt right keys for all legitimates
		keys, err := cm.userKeys.EncryptRightKeysAndReturn(ctx, container.Items[i].ID, pk)
		if err != nil {
			continue
		}
		for _, key := range keys {
			batchItems = append(batchItems, PsrBatchRightItem{
				ItemType:     BatchRightUpdateLegitimateDataRightKey,
				DataID:       key.DataID,
				LegitimateID: key.LegitimateID,
				RightKey:     key.EncryptedKey,
			})
		}
	}
	return batchItems, nil
}

// RightInheritanceOptions contains encrypted keys for container creation.
type RightInheritanceOptions struct {
	Keys            []AsymmetricRightKey `json:"Keys"`
	TemplateGroupID *string              `json:"TemplateGroupId"`
	RightTemplates  interface{}          `json:"RightTemplates"`
}

// AsymmetricRightKey holds an encrypted private key for a container item.
type AsymmetricRightKey struct {
	Identifier          string `json:"Identifier"`
	PublicKey           []byte `json:"PublicKey"`
	EncryptedPrivateKey []byte `json:"EncryptedPrivateKey"`
}
